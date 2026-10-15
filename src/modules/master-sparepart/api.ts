import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  SparepartDecisionInput,
  SparepartDecisionResponse,
  SparepartDocumentResponse,
  SparepartImportResponse,
  SparepartInput,
  SparepartListResponse,
  SparepartOptionsResponse,
  SparepartResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/sparepart'
const ROUTE_DECISION = '/api/master/sparepart/keputusan'
const ROUTE_OPTIONS = '/api/master/sparepart/pilihan'

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
  return ['master-sparepart', portal, token, status] as const
}

/**
 * optionsKey MENYERTAKAN portal — berbeda dari Master Panel.
 *
 * Isinya dibaca dari POOLDATA.GCNM_M_SPAREPART_CATEGORY dan GCNM_M_SPAREPART_TYPE, yang
 * ada di basis data setiap entitas dan isinya berbeda. Pada Master Panel daftarnya
 * konstanta yang ditanam di activity Pega, sehingga di sana portal memang tidak perlu ikut.
 */
function optionsKey(portal: string | null, token: string | null) {
  return ['master-sparepart-pilihan', portal, token] as const
}

/**
 * Hook daftar Master Sparepart.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti `pyMaxRecords=500` yang
 * memotong daftar Pega tanpa satu pun cara mempersempitnya. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian
 * kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function useSparepartList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<SparepartListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan Kategori dan Tipe.
 *
 * Ia MENUNGGU portal dipilih, karena endpoint-nya pun menuntutnya: isinya data entitas,
 * bukan konstanta. Itu berbeda dari Master Panel, yang form-nya dapat menggambar dropdown
 * bahkan sebelum entitas dipilih.
 */
export function useSparepartOptions() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: optionsKey(portal, token),
    queryFn: () => callAPI<SparepartOptionsResponse>(ROUTE_OPTIONS, { token, portal }),
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
export function useCreateSparepart() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: SparepartInput) =>
      callAPI<SparepartResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-sparepart'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/UpdateSparepartHE_act` menetapkan APPROVAL := "0" tanpa
 * syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSaveSparepart() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: SparepartInput }) =>
      callAPI<SparepartResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-sparepart'] })
    },
  })
}

/**
 * Hook keputusan borongan — **belum dipakai layar mana pun**.
 *
 * Ia menunggu **Inbox Manager**, layar yang di Pega memang memiliki persetujuan borongan
 * (`Section/ApprovalMasterSparepartHE`). Layar Master Sparepart sempat memakainya, lalu
 * dicabut pada 2026-10-03 karena ketiga tabnya di Pega hanya punya tombol SIMPAN dan Ubah —
 * nol checkbox, nol `pySelected`.
 *
 * Dibiarkan hidup dengan sengaja: endpoint-nya ada, teruji, dan kontraknya cocok. Menghapus
 * keduanya berarti menulis ulang hal yang sudah terbukti benar saat Inbox Manager tiba.
 *
 * SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris. Bentuknya
 * mengikuti `Activity/SetApprovalAllMaster` beserta layarnya
 * `Section/ApprovalMasterSparepartHE-Section.xml` — memecahnya menjadi sederet permintaan
 * akan mengubah operasi yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
 *
 * TANPA catatan: tabelnya tidak punya kolom penampungnya.
 */
export function useDecideSparepart() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: SparepartDecisionInput) =>
      callAPI<SparepartDecisionResponse>(ROUTE_DECISION, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-sparepart'] })
    },
  })
}

/**
 * documentKey menyertakan portal, token, dan ID sparepart-nya.
 *
 * Portal ikut dengan alasan yang sama seperti daftar: dokumen sebuah sparepart hidup di
 * basis data entitasnya, dan cache yang tidak membedakannya akan menampilkan dokumen badan
 * hukum lain tanpa satu pun tanda di layar (`R-20`).
 */
function documentKey(portal: string | null, token: string | null, id: string) {
  return ['master-sparepart-dokumen', portal, token, id] as const
}

/**
 * Hook pembaca dokumen satu sparepart.
 *
 * 404 adalah jawaban yang WAJAR di sini — sparepart yang belum punya dokumen menjawab
 * `dokumen_belum_ada` — sehingga kegagalannya tidak diperlakukan sebagai galat layar, dan
 * tidak diulang. Mengulanginya tiga kali hanya memperlambat form tanpa mengubah jawabannya.
 */
export function useSparepartDocument(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: documentKey(portal, token, id ?? ''),
    queryFn: () =>
      callAPI<SparepartDocumentResponse>(
        `${ROUTE}/${encodeURIComponent(id ?? '')}/dokumen`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && id !== null && id !== '',
    retry: false,
  })
}

/**
 * Hook unggah dokumen sparepart.
 *
 * Badannya `FormData`, bukan JSON base64 — base64 membengkakkan muatan sekitar sepertiga,
 * dan `callAPI` sudah menangani FormData termasuk membiarkan peramban yang menulis
 * `Content-Type` beserta boundary-nya.
 *
 * Yang dibuang dari cache SESUDAH berhasil ada dua, dan keduanya perlu:
 *
 *   dokumen sparepart itu  karena isinya memang berganti
 *   seluruh daftar         karena `DOKUMENID` adalah kolom pada baris sparepart-nya
 *                          sendiri, sehingga baris di daftar pun sudah basi
 *
 * Yang TIDAK terjadi: baris itu tidak berpindah ke tab Waiting Approval. Mengunggah dokumen
 * hanya menyentuh kolom `DOKUMENID`, berbeda dari menyimpan lewat form yang menyetel
 * `APPROVAL := "0"`.
 */
export function useUploadSparepartDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, file, note }: { id: string; file: File; note: string }) => {
      const body = new FormData()
      body.append('berkas', file)
      if (note.trim() !== '') body.append('catatan', note.trim())
      return callAPI<SparepartDocumentResponse>(
        `${ROUTE}/${encodeURIComponent(id)}/dokumen`,
        { metode: 'POST', body, token, portal },
      )
    },
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-sparepart-dokumen'] })
      client.invalidateQueries({ queryKey: ['master-sparepart'] })
    },
  })
}

const ROUTE_IMPORT = '/api/master/sparepart/unggah-csv'

/**
 * Hook unggah CSV master sparepart.
 *
 * Badannya `FormData`, sama seperti unggah dokumen, dan karena alasan yang sama: base64
 * membengkakkan muatan sekitar sepertiga.
 *
 * # Jawabannya 200 bahkan saat sebagian baris gagal
 *
 * Yang gagal adalah barisnya, bukan permintaannya — dan sebagian datanya SUDAH masuk.
 * Memperlakukannya sebagai galat akan membuat pengguna mengunggah ulang dan menimpa baris
 * yang sudah benar. Layar karena itu membaca `data.gagal`, bukan menunggu `onError`.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil: baris hasil unggah mendarat di tab
 * Waiting Approval — tab yang justru TIDAK sedang dilihat pengguna saat ia mengunggah dari
 * tab Approve.
 */
export function useImportSparepartCSV() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('berkas', file)
      return callAPI<SparepartImportResponse>(ROUTE_IMPORT, {
        metode: 'POST',
        body,
        token,
        portal,
      })
    },
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-sparepart'] })
    },
  })
}
