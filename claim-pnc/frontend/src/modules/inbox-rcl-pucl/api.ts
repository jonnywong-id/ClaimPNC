import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  ClaimDetailResponse,
  DateRange,
  ListResponse,
  MetadataResponse,
} from './types'

const PATH = '/api/inbox-rcl-pucl'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kerja satu badan hukum
 * bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 *
 * Kode tab ikut pula. Ketiga tab dilayani KUERI yang berbeda di server — bukan satu kueri
 * yang hasilnya disaring — sehingga hasilnya tidak boleh berbagi satu entri cache.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-rcl-pucl', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, tab: string, page: number) =>
    ['inbox-rcl-pucl', 'daftar', portal, token, tab, page] as const,

  detail: (portal: string | null, token: string | null, reference: string) =>
    ['inbox-rcl-pucl', 'klaim', portal, token, reference] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, kolom laporan, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxrclpucl/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Di layar ini hal itu lebih dari sekadar kerapian: KETIGA tab punya kolom yang identik
 * kecuali satu judul, dan satu perbedaan kecil di antara ketiganya akan berarti salah
 * satunya tidak lagi mencerminkan grid aslinya — tanpa ada apa pun yang terlihat rusak.
 */
export function useRCLPUCLMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan jaringan
    // tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa paginasinya dikerjakan SERVER
 *
 * Karena yang dibaca adalah tabel objek kerja dan tabel penugasan Pega yang berisi puluhan
 * juta baris (`D-10`). Menyaringnya di peramban berarti menarik seluruh antrean lebih dulu.
 *
 * Halaman di sini benar-benar dipotong basis data dengan `OFFSET … FETCH NEXT`, sehingga
 * jumlah "total" tetap tepat tanpa menarik seluruh baris.
 *
 * # Kenapa rentang tanggal TIDAK ikut dikirim
 *
 * Karena ia tidak menyaring grid sama sekali — itu perilaku layar lama, dan Work Owner
 * memutuskan mereplikasinya (`P-5`). Mengirimnya ke sini akan membuat orang mengira
 * tabelnya ikut tersaring.
 */
export function useRCLPUCLList(tab: string, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, tab, page),
    queryFn: () => callAPI<ListResponse>(buildPath(tab, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean berubah saat petugas lain mengerjakan pekerjaannya, jadi cache-nya pendek —
    // sama dengan modul inbox lain, yang dibuka berulang kali sepanjang hari.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook layar kerja satu klaim — yang di Pega terbuka lewat Open Assignment saat nomor klaim
 * diklik.
 *
 * # Kenapa ia mengambil data sendiri, bukan memakai baris yang sudah di tangan
 *
 * Karena isinya BERBEDA dari baris grid. Tiga isian surat — Nama Peserta, UP, dan Jumlah
 * Tagihan — diturunkan dari objek dan adjustment klaim, dan tidak satu pun ada di daftar.
 * Menggambarnya dari baris grid berarti menampilkan layar yang kolomnya benar tetapi isinya
 * tidak lengkap.
 *
 * # Kenapa cache-nya per KUNCI, bukan per nomor case
 *
 * Karena kuncinya yang dikirim ke server, dan nomor case tidak dijamin unik antar entitas.
 * Portal ikut menjadi bagian kunci dengan alasan yang sama seperti hook lain (`R-20`).
 */
export function useRCLPUCLClaim(reference: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, reference ?? ''),
    queryFn: () =>
      callAPI<ClaimDetailResponse>(
        `${PATH}/klaim/${encodeURIComponent(reference ?? '')}`,
        { token, portal },
      ),
    enabled: reference !== null && reference !== '' && token !== null && portal !== null,

    // Isi layar kerja berubah saat petugas lain mengerjakannya di Pega — dan selama masa
    // paralel itulah satu-satunya tempat ia berubah. Cache-nya sependek daftar.
    staleTime: 15 * 1000,
  })
}

/**
 * Bahan permintaan ekspor: tab yang sedang terbuka, dan rentang bila tabnya memintanya.
 *
 * `range` ditulis `DateRange | undefined`, bukan `range?:`, karena `tsconfig` menyalakan
 * `exactOptionalPropertyTypes`. Dengan setelan itu, `range?: DateRange` berarti "boleh
 * tidak ada" tetapi BUKAN "boleh bernilai undefined" — sementara pemanggilnya memang
 * mengirimkan `undefined` pada tab yang mengekspor grid.
 */
type ExportInput = {
  tab: string
  range: DateRange | undefined
}

/**
 * Hook tombol ekspor.
 *
 * # SATU tombol, DUA isi berkas
 *
 * Tab "Cetak Surat" mengunduh LAPORAN HARIAN berbasis rentang tanggal; dua tab lain
 * mengunduh salinan tabelnya. Percabangannya ada di server dan ditentukan oleh sifat
 * tabnya, bukan oleh alamat yang berbeda — di Pega pun tombolnya satu dan sama, hanya
 * activity di baliknya yang berbeda.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut dua:
 * `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah menaruh
 * token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer. Berkas ini memuat nomor polis dan nama tertanggung; jejaknya tidak boleh
 * tertinggal di sana.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi peladen
 * — memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya. Pada batas
 * 50.000 baris itu beberapa megabita, dan dapat diterima.
 */
export function useExportRCLPUCL() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (input: ExportInput) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const params = exportParams(input).toString()
      const address = params ? `${PATH}/ekspor?${params}` : `${PATH}/ekspor`

      const response = await fetch(address, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa
        // lagi. Yang dibaca di sini adalah kasus pertama — termasuk rentang tanggal yang
        // belum diisi, yang dijawab 422 beserta isian mana yang kurang.
        const body = (await response.json().catch(() => null)) as {
          pesan?: string
          detail?: { pesan?: string }[]
        } | null

        // Pesan per isian lebih berguna daripada pesan umum: ia menyebut isian MANA yang
        // harus diperbaiki, dan di layar ini isiannya ada dua.
        const detail = body?.detail?.map((item) => item.pesan).filter(Boolean)
        if (detail && detail.length > 0) throw new Error(detail.join(' '))

        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'inbox-rcl-pucl.csv')
    },
  })
}

/**
 * exportParams menyusun parameter permintaan ekspor.
 *
 * Rentang tanggal HANYA dikirim bila tabnya memang memintanya. Mengirimnya pada tab yang
 * mengekspor grid akan diabaikan server, tetapi ia tetap tidak dikirim — parameter yang
 * tidak berarti apa-apa di alamat unduhan membuat orang menduga ia berpengaruh.
 */
function exportParams(input: ExportInput): URLSearchParams {
  const params = new URLSearchParams()
  if (input.tab) params.set('tab', input.tab)

  if (input.range) {
    if (input.range.dari) params.set('dari', input.range.dari)
    if (input.range.sampai) params.set('sampai', input.range.sampai)
  }
  return params
}

/** buildPath menyusun alamat permintaan daftar beserta halamannya. */
function buildPath(tab: string, page: number): string {
  const params = new URLSearchParams()
  if (tab) params.set('tab', tab)
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/** filenameOf membaca nama berkas dari header Content-Disposition. */
function filenameOf(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  const found = /filename="([^"]+)"/.exec(disposition)
  return found?.[1] ?? null
}

/**
 * downloadBlob menyimpan berkas lewat tautan sementara.
 *
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban sampai
 * tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor menumpuk memori
 * yang tidak pernah dilepas.
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
