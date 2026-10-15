import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  PartTypeInput,
  PartTypeListResponse,
  PartTypeOptionsResponse,
  PartTypeResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/tipe-sparepart'
const ROUTE_OPTIONS = '/api/master/tipe-sparepart/pilihan'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Penyaring statusnya ikut pula: ketiga tab layar adalah tiga kombinasi penyaring atas satu
 * endpoint, dan tanpa status di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string) {
  return ['master-tipe-sparepart', portal, token, status] as const
}

/**
 * Hook daftar Master Tipe Sparepart.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti daftar Pega yang dimuat
 * penuh ke klipboard lalu disaring di peramban. Yang dipakai layar sekarang adalah
 * pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian kedua di
 * kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function usePartTypeList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<PartTypeListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan Kategori.
 *
 * Endpoint tersendiri, bukan disisipkan pada jawaban daftar, karena keduanya berubah pada
 * irama yang berbeda: daftar tipe berubah setiap kali ada yang menyimpan, sedangkan daftar
 * kategori nyaris tidak pernah berubah selama satu sesi.
 *
 * `staleTime` panjang dengan alasan yang sama. Ia tetap dibuang dari cache begitu ada
 * penyimpanan yang GAGAL karena kategorinya hilang — lihat penanganan
 * `PartTypeErrorCode.categoryNotFound` pada form, yang menyarankan pengguna memuat ulang.
 */
export function usePartTypeOptions() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['master-tipe-sparepart-pilihan', portal, token],
    queryFn: () => callAPI<PartTypeOptionsResponse>(ROUTE_OPTIONS, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook penambahan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil, bukan hanya daftar tab yang sedang
 * dibuka: baris baru lahir berstatus menunggu, sehingga yang berubah adalah tab Waiting
 * Approval — tab yang justru TIDAK sedang dilihat pengguna saat ia menambah dari tab
 * Approve.
 */
export function useCreatePartType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PartTypeInput) =>
      callAPI<PartTypeResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-tipe-sparepart'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/UpdateTypeSparepart_act2` menetapkan APPROVAL := "0" tanpa
 * syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 *
 * Menyimpan juga dapat MEMINDAHKAN tipe ke kategori lain, sehingga baris itu berpindah
 * kelompok pada setiap layar yang menampilkannya.
 */
export function useSavePartType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PartTypeInput }) =>
      callAPI<PartTypeResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-tipe-sparepart'] })
    },
  })
}

/*
 * TIDAK ADA hook keputusan di modul ini.
 *
 * Ketiga tab layar Pega — Approve, Reject, dan Waiting Approval — nol tombol dan nol
 * `pyLocalAction`; dibaca dari ketiga section tab, bukan disimpulkan. Persetujuan di Pega
 * dikerjakan layar LAIN, `Section/ApprovalMasterTipeSparepartHE-Section.xml`, yang dipakai
 * Inbox Manager.
 *
 * Versi pertama modul ini memasang tombol Approve/Reject di dalam layar, mengikuti Master
 * Kategori Sparepart dan ketiga master HE lainnya. Work Owner mencabutnya pada 2026-10-04:
 * "jika tidak sama, hapus, ikuti perilaku PEGA".
 *
 * Endpoint `POST /api/master/tipe-sparepart/keputusan` TETAP ADA di backend — ia padanan
 * sah dari section Inbox Manager, dan dipakai begitu layar itu dibangun. Yang dicabut
 * hanyalah pemakaiannya dari layar ini.
 */
