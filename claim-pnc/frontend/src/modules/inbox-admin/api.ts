import { useQuery } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { CountsResponse, FilterForm, ListResponse, MetadataResponse, ViewerResponse } from './types'

const PATH = '/api/inbox-admin'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kerja satu badan hukum
 * bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-admin', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterForm, page: number) =>
    ['inbox-admin', 'daftar', portal, token, filter.tab, filter.bisnis, filter.cari, filter.kanwil, page] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan isi dropdown penyaring.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxadmin/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Berbeda dengan View History Claim, memanggil ini TIDAK memakai jatah apa pun: layar ini
 * tidak punya gerbang proteksi data. Karena itu ia `useQuery` biasa tanpa pengamanan
 * tambahan terhadap pemanggilan ganda.
 */
export function useInboxAdminMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan
    // dari data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang berisi puluhan juta baris
 * (`D-10`). Menyaringnya di peramban berarti mengirim seluruh antrean ke setiap ketikan.
 *
 * Catatan yang perlu diketahui saat membaca jumlah "Total Data": paginasi layar ini
 * DIREPLIKASI apa adanya dari Pega atas keputusan Work Owner 2026-09-20 — server menarik
 * seluruh baris yang cocok lalu memotong halamannya. Angka totalnya karena itu selalu
 * tepat, tetapi tab yang sangat besar memakan waktu pada permintaan pertama.
 */
export function useInboxAdminList(filter: FilterForm, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter, page),
    queryFn: () => callAPI<ListResponse>(buildPath(filter, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean kerja berubah saat petugas lain menyelesaikan pekerjaannya, jadi cache-nya
    // pendek. Ia lebih pendek daripada layar riwayat klaim karena inilah layar yang
    // dibuka berulang kali sepanjang hari.
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
  const params = new URLSearchParams()

  if (filter.tab) params.set('tab', filter.tab)
  if (filter.bisnis) params.set('bisnis', filter.bisnis)
  if (filter.cari.trim()) params.set('cari', filter.cari.trim())
  if (filter.kanwil) params.set('kanwil', filter.kanwil)
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/** Tiga tombol ekspor CSV layar lama. */
export type ExportKind = 'lod' | 'hasil-auto-claim' | 'klaim-gagal'

/**
 * useInboxAdminExport mengunduh satu berkas ekspor lalu menyerahkannya ke peramban.
 *
 * Lewat `fetch`, bukan tautan biasa: unduhannya menuntut header `Authorization` dan
 * `X-Portal`, dan menaruh keduanya di URL akan mencatat token dan entitas di log proxy.
 *
 * Penyaring yang sedang dipakai ikut dikirim untuk Export LOD, sehingga berkasnya berisi
 * baris yang sedang dilihat petugas — seluruh halamannya.
 */
export function useInboxAdminExport() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return async (kind: ExportKind, filter: FilterForm): Promise<void> => {
    const params = new URLSearchParams()
    if (kind === 'lod') {
      if (filter.bisnis) params.set('bisnis', filter.bisnis)
      if (filter.cari.trim()) params.set('cari', filter.cari.trim())
      if (filter.kanwil) params.set('kanwil', filter.kanwil)
    }
    const query = params.toString()
    const path = `${PATH}/ekspor/${kind}${query ? `?${query}` : ''}`
    simpanBerkas(await unduhBerkas(path, { token, portal }))
  }
}

/**
 * useInboxAdminCounts mengambil jumlah baris setiap tab untuk daftar Status Register.
 *
 * Hanya penyaring lini bisnis yang ikut: angka di daftar adalah ukuran antrean, bukan hasil
 * pencarian.
 */
export function useInboxAdminCounts(business: string, region: string, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const params = new URLSearchParams()
  if (business) params.set('bisnis', business)
  if (region) params.set('kanwil', region)
  const query = params.toString() ? `?${params.toString()}` : ''

  return useQuery({
    queryKey: ['inbox-admin', 'jumlah', portal, token, business, region] as const,
    queryFn: () => callAPI<CountsResponse>(`${PATH}/jumlah${query}`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 15 * 1000,
  })
}

/**
 * useInboxAdminViewer membaca batas data pemanggil: apakah ia manajer (boleh memilih
 * kanwil), cabang yang membatasinya, dan isi dropdown kanwil.
 */
export function useInboxAdminViewer(enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['inbox-admin', 'batas', portal, token] as const,
    queryFn: () => callAPI<ViewerResponse>(`${PATH}/batas`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    // Peran dan cabang petugas tidak berubah selama sesi berjalan.
    staleTime: 5 * 60 * 1000,
  })
}
