import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  ClaimReportListResponse,
  ClaimReportOptionResponse,
  ClaimReportQuery,
  ClaimReportResponse,
} from './types'

const ROUTE = '/api/inbox/laporan-klaim'

/**
 * listKey menyertakan portal DAN token, mengikuti pola modul master.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan berkas entitas sebelumnya dari cache —
 * pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan berkas itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama. Itu lebih penting di sini daripada di layar master: ketiga tab
 * komunikasi menampilkan percakapan milik pemanggil.
 *
 * Penyaring ikut seluruhnya, termasuk nomor halaman: dua halaman adalah dua hasil yang
 * berbeda, dan menyatukannya dalam satu kunci membuat halaman kedua menimpa halaman
 * pertama di cache.
 */
function listKey(portal: string | null, token: string | null, query: ClaimReportQuery) {
  return ['inbox-laporan-klaim', portal, token, query] as const
}

function optionKey(portal: string | null, token: string | null) {
  return ['inbox-laporan-klaim-pilihan', portal, token] as const
}

/** Menyusun parameter kueri; yang kosong tidak ikut dikirim. */
function searchParams(query: ClaimReportQuery): string {
  const param = new URLSearchParams()
  param.set('kategori', query.kategori)
  if (query.kanwil !== '') param.set('kanwil', query.kanwil)
  if (query.bisnis !== '') param.set('bisnis', query.bisnis)
  if (query.cari.trim() !== '') param.set('cari', query.cari.trim())
  if (query.halaman > 1) param.set('halaman', String(query.halaman))
  return param.toString()
}

/**
 * Hook daftar berkas laporan beserta lencana kesembilan tab.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
 * portal (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan
 * akan menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useClaimReportList(query: ClaimReportQuery) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, query),
    queryFn: () =>
      callAPI<ClaimReportListResponse>(`${ROUTE}?${searchParams(query)}`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Hasil halaman sebelumnya tetap digambar selama halaman berikutnya dimuat, sehingga
    // tabel tidak berkedip menjadi kosong setiap kali pengguna menekan Berikutnya.
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook isi ketiga dropdown: tab, bisnis, dan kanwil.
 *
 * Ketiganya datang dalam satu permintaan karena layar membutuhkan ketiganya sebelum
 * dapat menggambar satu baris pun.
 *
 * Daftarnya diambil dari server, tidak disalin ke sini. Menyalinnya berarti aturan tab
 * hidup di dua tempat, dan tempat kedua akan terlupa saat yang pertama berubah.
 */
export function useClaimReportOptions() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: optionKey(portal, token),
    queryFn: () => callAPI<ClaimReportOptionResponse>(`${ROUTE}/pilihan`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Isinya nyaris tetap: daftar kanwil berubah ketika cabang dibuka atau ditutup,
    // yang terjadi beberapa kali setahun. Memuatnya ulang setiap kali layar dibuka hanya
    // menambah permintaan.
    staleTime: 60 * 60 * 1000,
  })
}

/**
 * Hook tombol "Buat Baru".
 *
 * Tidak mengirim badan permintaan sama sekali, dan itu bukan kelalaian: tombolnya di
 * sistem lama tidak meminta satu pun isian — berkas lahir kosong lalu dilengkapi di
 * layar berikutnya (`B-14`). Lihat catatan pada usecase.Service.Create di backend.
 */
export function useCreateClaimReport() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: () =>
      callAPI<ClaimReportResponse>(ROUTE, { metode: 'POST', token, portal }),

    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. Nomornya
    // diterbitkan server, petugas lain dapat menambah berkas pada saat yang sama, dan
    // lencana kesembilan tab ikut berubah — daftar yang disusun sendiri di peramban akan
    // berbeda dari isi tabel yang sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['inbox-laporan-klaim'] })
    },
  })
}

/**
 * Hook tombol "Export Data".
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut
 * dua: `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah
 * menaruh token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer. Berkas ini memuat data nasabah; jejaknya tidak boleh tertinggal di sana.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi
 * peladen — memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya.
 * Pada batas 50.000 baris itu beberapa megabita, dan dapat diterima.
 *
 * Bila kelak ekspor harus jauh lebih besar — pertanyaan terbuka `ADR-0011` yang belum
 * dijawab — yang harus berubah adalah caranya, bukan angkanya.
 */
export function useExportClaimReport() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (query: ClaimReportQuery) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const response = await fetch(`${ROUTE}/ekspor?${searchParams(query)}`, { headers: header })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak
        // bisa lagi. Yang dibaca di sini adalah kasus pertama.
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'laporan-klaim.csv')
    },
  })
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
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban
 * sampai tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor
 * menumpuk memori yang tidak pernah dilepas.
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
