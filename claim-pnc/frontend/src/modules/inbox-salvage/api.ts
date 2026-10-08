import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { downloadExport, queryPath, useQueueList, useScreenMetadata } from '@/api/inboxShared'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  CountsResponse,
  CreateRequest,
  CreateResponse,
  DetailKey,
  DetailResponse,
  ListResponse,
  MetadataResponse,
  UploadResponse,
} from './types'

const PATH = '/api/inbox-salvage'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: pengajuan salvage satu badan
 * hukum bukan milik badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan pengajuan entitas sebelumnya (`R-20`).
 *
 * Kata kunci pencarian ikut pula. Ia menyaring DI SERVER — pada tiga daftar bahkan dengan
 * aturan "cocok persis" — sehingga hasil pencarian dan hasil tanpa pencarian tidak boleh
 * berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-salvage', 'daftar', portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    tab: string,
    page: number,
    search: string,
  ) => ['inbox-salvage', 'baris', portal, token, tab, page, search] as const,

  counts: (portal: string | null, token: string | null) =>
    ['inbox-salvage', 'ringkas', portal, token] as const,

  detail: (
    portal: string | null,
    token: string | null,
    key: DetailKey,
    reference: string,
  ) => ['inbox-salvage', 'rincian', portal, token, key, reference] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, pilihan status, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap daftar adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxsalvage/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini hal itu lebih dari sekadar kerapian: TIGA BELAS daftar dengan empat susunan
 * kolom yang berbeda-beda, dan dua di antaranya berbeda hanya satu kolom. Menyalinnya
 * dengan tangan berarti menunggu salah satunya bergeser.
 */
export function useSalvageMetadata() {
  return useScreenMetadata<MetadataResponse>(keys.metadata, `${PATH}/daftar`)
}

/**
 * Hook isi satu daftar.
 *
 * # Kenapa paginasi DAN pencarian dikerjakan SERVER
 *
 * Karena yang dibaca adalah tabel klaim berisi puluhan juta baris (`D-10`). Menyaringnya di
 * peramban berarti menarik seluruh daftar lebih dulu.
 *
 * Dan pada layar ini ada alasan kedua yang lebih menentukan: tiga daftar mencari dengan
 * aturan "COCOK PERSIS", dan satu di antaranya mencari DUA kolom sekaligus. Menyaring di
 * peramban berarti menyalin ketiga aturan itu ke sini — dan begitu keduanya bergeser, layar
 * dan server akan menjawab pertanyaan yang sama dengan hasil yang berbeda.
 */
export function useSalvageList(
  tab: string,
  page: number,
  search: string,
  enabled: boolean,
) {
  return useQueueList<ListResponse>(
    (portal, token) => keys.list(portal, token, tab, page, search),
    buildPath(tab, page, search),
    enabled,
  )
}

/**
 * Hook tabel ringkas "Status Salvage / Jumlah".
 *
 * # Kenapa ia permintaan TERSENDIRI
 *
 * Karena isinya TIDAK berubah saat pengguna berpindah daftar. Menggabungkannya ke dalam
 * jawaban daftar akan menjalankan keempat belas hitungannya setiap kali tab dibuka — dan
 * sebagian hitungan itu menyapu tabel klaim berisi puluhan juta baris.
 *
 * Cache-nya lebih panjang daripada daftar dengan alasan yang sama: angka ringkas yang
 * tertinggal beberapa puluh detik masih berguna, sementara baris yang tertinggal tidak.
 */
export function useSalvageCounts() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.counts(portal, token),
    queryFn: () => callAPI<CountsResponse>(`${PATH}/ringkas`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 1000,
  })
}

/**
 * Hook tombol "Submit" pada form Tambah.
 *
 * # Apa yang dibatalkan setelah berhasil
 *
 * SELURUH cache modul ini, dan itu disengaja: pengajuan baru langsung masuk antrean
 * "Checker", sehingga daftar yang sedang terbuka DAN tabel ringkasnya keduanya berubah.
 * Membatalkan daftarnya saja akan meninggalkan angka ringkas yang tertinggal satu — dan
 * angka yang tidak cocok dengan tabel di bawahnya adalah hal pertama yang dilaporkan
 * pengguna sebagai kerusakan.
 */
export function useCreateSalvage() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: CreateRequest) =>
      callAPI<CreateResponse>(PATH, {
        token,
        portal,
        metode: 'POST',
        body,
      }),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['inbox-salvage'] })
    },
  })
}

/**
 * Hook tombol "Upload Detail Salvage".
 *
 * # Ia TIDAK menyimpan apa pun
 *
 * Itu yang paling mudah disalahpahami tentang tombolnya. Di Pega ia memanggil
 * `pxUploadCSVResults` lalu menyalin isinya ke grid DI DALAM form — tidak ada satu pun
 * penyimpanan. Penyimpanan baru terjadi saat Submit ditekan.
 *
 * Karena itu hook ini TIDAK membatalkan cache apa pun: tidak ada yang berubah di server.
 *
 * # Berkasnya tetap lewat callAPI
 *
 * `callAPI` sudah mengenali `FormData` dan mengirimkannya apa adanya, termasuk membiarkan
 * peramban menyusun sendiri header `Content-Type` beserta penanda batasnya. Memanggil
 * `fetch` sendiri di sini akan menambah tempat kedua `fetch` dipanggil tanpa satu pun
 * manfaat — dan `08-TECHNICAL-STRATEGY.md` §3 menetapkan hanya ada satu.
 */
export function useUploadSalvageDetail() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('berkas', file)

      return callAPI<UploadResponse>(`${PATH}/unggah-detail`, {
        token,
        portal,
        metode: 'POST',
        body,
      })
    },
  })
}

/**
 * Hook tombol "Export Data".
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut dua:
 * `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah menaruh
 * token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer. Berkas ini memuat nomor klaim dan nilai uang; jejaknya tidak boleh tertinggal di
 * sana.
 */
export function useExportSalvage() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (input: { tab: string; search: string }) => {
      const params = new URLSearchParams()
      if (input.tab) params.set('daftar', input.tab)
      if (input.search) params.set('cari', input.search)

      const query = params.toString()
      const address = query ? `${PATH}/ekspor?${query}` : `${PATH}/ekspor`

      await downloadExport({
        address,
        token,
        portal,
        fallbackName: 'inbox-salvage.csv',
      })
    },
  })
}

/** buildPath menyusun alamat permintaan daftar beserta halaman dan pencariannya. */
function buildPath(tab: string, page: number, search: string): string {
  return queryPath(PATH, [
    ['daftar', tab],
    ['halaman', page > 1 && String(page)],
    ['cari', search],
  ])
}

/**
 * Hook panel "Detail Salvage" — satu pengajuan beserta barangnya.
 *
 * # Kenapa baru ditembak saat panelnya dibuka
 *
 * `salvageID` kosong berarti panelnya tertutup, dan permintaan tidak dijalankan. Mengambil
 * rinciannya untuk setiap baris yang tampil akan menjalankan dua kueri per baris — pada
 * halaman berisi dua puluh baris itu empat puluh kueri untuk satu panel yang mungkin tidak
 * pernah dibuka.
 *
 * # Kenapa TIDAK disimpan lama di cache
 *
 * Isinya berubah karena perbuatan orang lain — Checker menyetujui, Balai Lelang menetapkan
 * pemenang — dan panel yang menampilkan posisi lama akan membuat petugas mengambil
 * keputusan atas keadaan yang sudah lewat. Ia diambil ulang setiap kali panelnya dibuka.
 */
export function useSalvageDetail(key: DetailKey, reference: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  // Dua rute, bukan satu rute dengan penanda jenis.
  //
  // Nomor klaim dan ID pengajuan adalah dua ruang nilai yang berbeda, dan satu rute yang
  // menerima keduanya akan menjawab "tidak ditemukan" untuk nilai yang sebenarnya sah
  // pada ruang yang lain — pesan yang menyesatkan justru di layar tempat keduanya sudah
  // sering tertukar oleh alias Pega yang menyesatkan.
  const path =
    key === 'klaim'
      ? `${PATH}/klaim/${encodeURIComponent(reference)}`
      : `${PATH}/pengajuan/${encodeURIComponent(reference)}`

  return useQuery({
    queryKey: keys.detail(portal, token, key, reference),
    queryFn: () => callAPI<DetailResponse>(path, { token, portal }),
    enabled: token !== null && portal !== null && reference !== '',
    staleTime: 0,
  })
}
