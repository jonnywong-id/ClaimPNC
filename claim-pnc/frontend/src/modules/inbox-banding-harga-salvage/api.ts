import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { queryPath, useQueueList, useScreenMetadata } from '@/api/inboxShared'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  DecisionRequest,
  DecisionResultResponse,
  DecisionsResponse,
  DocumentScope,
  DocumentsResponse,
  FilterForm,
  ListResponse,
  MetadataResponse,
  SummaryResponse,
} from './types'

const PATH = '/api/inbox-banding-harga-salvage'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: banding harga satu badan hukum
 * bukan milik badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan banding entitas sebelumnya (`R-20`).
 *
 * Token ikut pula, dan di layar ini alasannya lebih dari sekadar kebersihan: kedua tab
 * menyaring menurut NAMA KOMITE pemanggil, sehingga cache yang tidak mengandung identitas
 * akan menampilkan antrean pengguna sebelumnya kepada pengguna berikutnya di peramban yang
 * sama.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-banding-harga-salvage', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterForm, page: number) =>
    [
      'inbox-banding-harga-salvage',
      'daftar',
      portal,
      token,
      filter.tab,
      filter.cari,
      page,
    ] as const,

  summary: (portal: string | null, token: string | null) =>
    ['inbox-banding-harga-salvage', 'ringkas', portal, token] as const,

  decisions: (portal: string | null, token: string | null, claimNo: string) =>
    ['inbox-banding-harga-salvage', 'riwayat', portal, token, claimNo] as const,

  documents: (
    portal: string | null,
    token: string | null,
    scope: DocumentScope | null,
  ) =>
    [
      'inbox-banding-harga-salvage',
      'dokumen',
      portal,
      token,
      scope?.detail_object ?? '',
      scope?.id_salvage ?? '',
    ] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, selisih terencana, dan keterbatasan.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu tercatat
 * adalah backend (`internal/inboxbandinghargasalvage/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini hal itu lebih dari kerapian: judul kolomnya menyesatkan secara sistematis —
 * "Object Name" yang berisi jenis salvage, "Detail Object" yang berisi ID baris. Menyalinnya
 * dengan tangan berarti menunggu salah satunya bergeser satu kolom.
 */
export function useBandingHargaSalvageMetadata() {
  return useScreenMetadata<MetadataResponse>(keys.metadata, `${PATH}/tab`)
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena `POOLDATA.T_CLAIM_CHEKER_SALVAGE` tumbuh seiring setiap banding yang pernah
 * diajukan, dan barisnya memuat dua angka harga yang sedang dipertentangkan. Menyaringnya di
 * peramban berarti mengirim seluruh antrean ke setiap ketikan — termasuk baris milik komite
 * lain, yang seharusnya tidak pernah sampai ke sana.
 */
export function useBandingHargaSalvageList(
  filter: FilterForm,
  page: number,
  enabled: boolean,
) {
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, filter, page),
    buildPath(filter, page),
    enabled,
  )
}

/**
 * Hook tabel ringkas "Status Salvage / Jumlah".
 *
 * Permintaan TERSENDIRI, bukan bagian jawaban daftar, karena isinya tidak berubah saat
 * pengguna berpindah tab — dan menggabungkannya akan menjalankan kedua hitungannya setiap kali
 * tab dibuka.
 *
 * Angkanya SAMA dengan jumlah baris gridnya. Itu berbeda dari layar lama, tempat pencacah dan
 * daftarnya menghitung populasi yang berbeda; selisihnya dinyatakan di `selisih_terencana`.
 */
export function useBandingHargaSalvageSummary(enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.summary(portal, token),
    queryFn: () => callAPI<SummaryResponse>(`${PATH}/ringkas`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    staleTime: 15 * 1000,
  })
}

/**
 * buildPath menyusun parameter permintaan.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `tab` yang kedua akan menjadi tab bawaan
 * sementara yang pertama memang itu yang diinginkan.
 */
function buildPath(filter: FilterForm, page: number): string {
  return queryPath(PATH, [
    ['tab', filter.tab],
    ['cari', filter.cari.trim()],
    ['halaman', page > 1 && String(page)],
  ])
}

/**
 * Hook panel rincian pada grid "History Cheker" — seluruh keputusan satu klaim.
 *
 * # Kenapa ia permintaan tersendiri, bukan ikut di baris daftarnya
 *
 * Karena satu klaim dapat punya beberapa barang yang dibanding, sehingga panel ini DAFTAR —
 * bukan perluasan satu baris. Membawanya serta pada setiap baris grid berarti menjalankan
 * kuerinya sebanyak baris yang tampil, untuk panel yang hampir selalu tertutup.
 *
 * Ia hanya berjalan ketika panelnya benar-benar dibuka (`claimNo` terisi).
 */
export function useBandingHargaSalvageDecisions(claimNo: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.decisions(portal, token, claimNo),
    queryFn: () =>
      callAPI<DecisionsResponse>(
        `${PATH}/riwayat/${encodeURIComponent(claimNo)}`,
        { token, portal },
      ),
    enabled: claimNo !== '' && token !== null && portal !== null,

    // Keputusan yang sudah diambil tidak berubah lagi. Cache-nya karena itu lebih panjang
    // daripada daftar, yang justru berubah saat balai lelang mengajukan banding baru.
    staleTime: 60 * 1000,
  })
}

/**
 * Hook tombol **Approve** dan **Reject** pada kolom Action.
 *
 * # Kenapa SELURUH daftar disegarkan, bukan barisnya dibuang di peramban
 *
 * Karena yang terjadi di basis data lebih banyak daripada "satu baris hilang". Sebuah
 * penolakan dapat ikut menutup baris komite BERIKUTNYA atas barang yang sama, dan sebuah
 * persetujuan oleh jenjang terakhir mengubah harga barang yang tergambar di panel rincian.
 * Membuang barisnya sendiri di peramban akan menampilkan keadaan yang tidak pernah ada di
 * basis data.
 *
 * Ketiga kunci disegarkan menurut AWALANNYA, sehingga halaman keberapa pun yang sedang
 * terbuka ikut termuat ulang — termasuk tabel ringkas, yang angkanya ikut berkurang.
 *
 * # Yang TIDAK dikerjakan hook ini
 *
 * Ia tidak menghalangi penekanan kedua. Penghalangnya ada di server, pada penyaring
 * `TGLAPPROVE IS NULL` — dan itu satu-satunya tempat yang dapat benar-benar menjaminnya,
 * karena dua peramban dapat menekan tombol yang sama pada saat yang sama. Layar cukup
 * menampilkan jawabannya apa adanya.
 */
export function useBandingHargaSalvageDecide() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: DecisionRequest) =>
      callAPI<DecisionResultResponse>(`${PATH}/keputusan`, {
        token,
        portal,
        metode: 'POST',
        body,
      }),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['inbox-banding-harga-salvage', 'daftar'] })
      client.invalidateQueries({ queryKey: ['inbox-banding-harga-salvage', 'ringkas'] })
      client.invalidateQueries({ queryKey: ['inbox-banding-harga-salvage', 'riwayat'] })
    },
  })
}

/**
 * Hook dialog "Lihat File" — dokumen pendukung satu banding.
 *
 * Ia hanya berjalan ketika dialognya benar-benar dibuka. Alasannya sama dengan panel
 * riwayat, dan di sini lebih kuat: kuerinya menyentuh tabel lampiran, yang barisnya membawa
 * isi berkas.
 *
 * Kedua id dikirim sebagai parameter kueri, bukan sebagai ruas jalur, karena
 * `IDDETAILSALVAGE` memuat garis miring di dalamnya — sebagai ruas jalur ia akan terpecah
 * menjadi dua segmen dan rutenya tidak akan pernah cocok.
 */
export function useBandingHargaSalvageDocuments(scope: DocumentScope | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.documents(portal, token, scope),
    queryFn: () =>
      callAPI<DocumentsResponse>(documentPath(scope!), { token, portal }),
    enabled: scope !== null && token !== null && portal !== null,

    // Dokumen pendukung jarang berubah setelah banding diajukan.
    staleTime: 60 * 1000,
  })
}

/**
 * documentDownloadURL menyusun alamat unduhan satu dokumen.
 *
 * Ia BUKAN hook: unduhan tidak melewati TanStack Query sama sekali. Yang diserahkan server
 * adalah isi berkas beserta header `Content-Disposition`, dan menariknya ke dalam cache
 * peramban lewat `fetch` berarti seluruh berkas ditahan di memori JavaScript lebih dulu —
 * tanpa satu pun manfaat, karena yang dibutuhkan hanyalah peramban menyimpannya.
 */
export function documentDownloadURL(scope: DocumentScope, documentID: string): string {
  const params = new URLSearchParams({
    detail_object: scope.detail_object,
    id_salvage: scope.id_salvage,
  })
  return `${PATH}/dokumen/${encodeURIComponent(documentID)}?${params.toString()}`
}

/** documentPath menyusun jalur daftar dokumen. */
function documentPath(scope: DocumentScope): string {
  const params = new URLSearchParams({
    detail_object: scope.detail_object,
    id_salvage: scope.id_salvage,
  })
  return `${PATH}/dokumen?${params.toString()}`
}
