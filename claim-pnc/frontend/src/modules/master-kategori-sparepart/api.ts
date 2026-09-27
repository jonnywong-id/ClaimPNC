import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  PartCategoryDecisionInput,
  PartCategoryDecisionResponse,
  PartCategoryInput,
  PartCategoryListResponse,
  PartCategoryResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/kategori-sparepart'
const ROUTE_DECISION = '/api/master/kategori-sparepart/keputusan'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Penyaring statusnya ikut pula: ketiga tab layar adalah tiga kombinasi penyaring atas
 * satu endpoint, dan tanpa status di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string) {
  return ['master-kategori-sparepart', portal, token, status] as const
}

/**
 * Hook daftar Master Kategori Sparepart.
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
export function usePartCategoryList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<PartCategoryListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
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
export function useCreatePartCategory() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PartCategoryInput) =>
      callAPI<PartCategoryResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-kategori-sparepart'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/UpdateKategoriSparepart_act2` menetapkan APPROVAL := "0"
 * tanpa syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSavePartCategory() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PartCategoryInput }) =>
      callAPI<PartCategoryResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-kategori-sparepart'] })
    },
  })
}

/**
 * Hook keputusan borongan.
 *
 * SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris. Bentuknya
 * mengikuti `Section/ApprovalMasterKategoriSparepartHE-Section.xml` — memecahnya menjadi
 * sederet permintaan akan mengubah operasi yang di Pega utuh menjadi sesuatu yang dapat
 * gagal separuh jalan.
 *
 * TANPA catatan: tabelnya tidak punya kolom penampungnya.
 */
export function useDecidePartCategory() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PartCategoryDecisionInput) =>
      callAPI<PartCategoryDecisionResponse>(ROUTE_DECISION, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-kategori-sparepart'] })
    },
  })
}
