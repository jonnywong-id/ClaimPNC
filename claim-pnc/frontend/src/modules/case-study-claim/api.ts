import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  FilterInput,
  ListResponse,
  MetadataResponse,
  SaveRemarkRequest,
  SaveRemarkResponse,
} from './types'

const PATH = '/api/case-study-claim'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: klaim satu badan hukum bukan
 * klaim badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan klaim entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['case-study-claim', 'penyaring', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterInput, page: number) =>
    ['case-study-claim', 'daftar', portal, token, filter, page] as const,
}

/** Ukuran halaman, mengikuti `pyPageSize` pada section Pega. */
export const UKURAN_HALAMAN = 20

/**
 * Hook keterangan layar — isi kedua dropdown, susunan kolom, dan catatan yang berlaku.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena ketiga daftarnya adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend — `filter.go` untuk kedua dropdown, `columns.go` untuk kolom.
 * Menyalinnya ke sini berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Hal yang sama berlaku untuk ambang Rp 5 miliar: begitu master ambang (`F-4`) tiba dan
 * angkanya dapat diubah tanpa deploy, layar ikut berubah tanpa satu baris pun disunting.
 */
export function useCaseStudyMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/penyaring`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali penyaring berubah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi grid.
 *
 * # Kenapa `enabled` bergantung pada tombol "Lihat Data"
 *
 * Karena layar lama pun begitu: gridnya KOSONG sampai tombol ditekan, dan baru saat itu
 * `StudyClaim_act` menjalankan kuerinya. Memuat otomatis saat layar dibuka akan menembak
 * kueri berisi delapan subkueri agregat tanpa ada yang memintanya.
 *
 * # Kenapa paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah gabungan tiga tabel klaim berisi puluhan juta baris (`D-10`),
 * dan menyaring di peramban hanya menyentuh halaman yang sedang terbuka — hasilnya BOHONG:
 * pengguna mencari klaim yang ada di halaman tiga dan diberi tahu bahwa ia tidak ada.
 */
export function useCaseStudyList(filter: FilterInput, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter, page),
    queryFn: () => callAPI<ListResponse>(buildPath(filter, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Layar telaah, bukan antrean kerja: isinya berubah jauh lebih jarang daripada inbox.
    // Cache-nya karena itu lebih panjang daripada layar inbox mana pun.
    staleTime: 60 * 1000,
  })
}

/**
 * buildPath menyusun parameter permintaan daftar.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: pada `bisnis` dan
 * `status`, keduanya kebetulan berarti sama — "semua" — tetapi tidak mengirimnya membuat
 * alamatnya lebih pendek dan kuncinya lebih mudah dibaca saat menelusuri masalah.
 */
function buildPath(filter: FilterInput, page: number): string {
  const params = new URLSearchParams()

  if (filter.dari) params.set('dari', filter.dari)
  if (filter.sampai) params.set('sampai', filter.sampai)
  if (filter.bisnis) params.set('bisnis', filter.bisnis)
  if (filter.status) params.set('status', filter.status)
  params.set('batas', String(UKURAN_HALAMAN))
  if (page > 1) params.set('lewati', String((page - 1) * UKURAN_HALAMAN))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook tombol **Export Data**.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut
 * dua: `Authorization` dan `X-Portal`. Menaruh token di dalam alamat sebagai gantinya
 * ditolak: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header Referer.
 *
 * # Penyaringnya SAMA PERSIS dengan grid
 *
 * Di Pega pun begitu: `ExportDataCaseStudyClaim` membaca halaman hasil yang sudah diisi
 * `StudyClaim_act`, bukan menjalankan kueri lain. Yang berbeda hanyalah batas halaman —
 * unduhan mengalir sampai habis.
 */
export function useExportCaseStudy() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (filter: FilterInput) => {
      const params = new URLSearchParams()
      if (filter.dari) params.set('dari', filter.dari)
      if (filter.sampai) params.set('sampai', filter.sampai)
      if (filter.bisnis) params.set('bisnis', filter.bisnis)
      if (filter.status) params.set('status', filter.status)

      simpanBerkas(await unduhBerkas(`${PATH}/unduh?${params.toString()}`, { token, portal }))
    },
  })
}

/**
 * Hook tombol **Save** pada satu baris.
 *
 * # Kenapa seluruh daftar disegarkan, bukan satu baris ditambal
 *
 * Karena yang tersimpan belum tentu sama dengan yang diketik: server memangkas spasi di
 * ujung, dan kelak dapat menolak isi tertentu. Menambal barisnya di peramban dengan teks
 * yang DIKETIK akan menampilkan sesuatu yang tidak ada di basis data — dan pengguna baru
 * mengetahuinya saat memuat ulang.
 *
 * Kuncinya disegarkan menurut awalannya, bukan satu per satu, supaya halaman keberapa pun
 * yang sedang terbuka ikut termuat ulang.
 */
export function useSaveRemark() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: SaveRemarkRequest) =>
      callAPI<SaveRemarkResponse>(
        `${PATH}/${encodeURIComponent(body.nomor_klaim)}/catatan`,
        { token, portal, metode: 'PUT', body: { catatan: body.catatan } },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['case-study-claim', 'daftar'] })
    },
  })
}
