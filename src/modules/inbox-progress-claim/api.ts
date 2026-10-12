import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { FilterForm, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-progress-claim'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: progres klaim satu badan hukum
 * bukan progres badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan klaim entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-progress-claim', 'bagian', portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    section: string,
    filter: FilterForm,
    page: number,
  ) =>
    [
      'inbox-progress-claim',
      'daftar',
      portal,
      token,
      section,
      filter.cari,
      filter.bisnis,
      filter.dari,
      filter.sampai,
      page,
    ] as const,
}

/**
 * Hook keterangan layar — daftar bagian, kolomnya, dan isi dropdown penyaring.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap bagian adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxprogressclaim/view.go`). Menyalinnya ke layar
 * berarti daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang
 * lain diperbaiki.
 *
 * Itu terasa paling nyata di layar ini: judul kolomnya memakai alias Pega yang menyesatkan,
 * dan arti sebenarnya hanya tercatat di backend.
 */
export function useProgressClaimMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/bagian`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan
    // dari data. Mengambilnya ulang tiap kali bagian dibuka hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu bagian.
 *
 * # Kenapa `enabled` ada di tangan pemanggil
 *
 * Karena bagian layar ini DIMUAT SAAT DIBUKA, bukan seluruhnya sekaligus. Itu meniru
 * `pyDeferLoadRetrievalActivity` pada kelima kontainer di
 * `Section/ProgressClaim_Section-Section.xml` — sistem lama pun tidak menjalankan ketiga
 * kuerinya bersamaan.
 *
 * Alasannya bukan kesetiaan semata: ketiga kueri menembak tabel ringkasan yang sama, dan
 * rekap per PIC menghitung lima subkueri agregat. Memuat ketiganya sekaligus saat layar
 * dibuka berarti membayar seluruh biayanya untuk bagian yang mungkin tidak dilihat.
 *
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah tabel ringkasan atas seluruh klaim berjalan. Berbeda dengan
 * Inbox Admin, layar ini memang memaginasi di basis data — begitu pula sistem lamanya,
 * lewat `ROW_NUMBER` antara `FirstRow` dan `LastRow` ditambah kueri `COUNT` terpisah.
 */
export function useProgressClaimList(
  section: string,
  filter: FilterForm,
  page: number,
  enabled: boolean,
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, section, filter, page),
    queryFn: () =>
      callAPI<ListResponse>(buildPath(section, filter, page), { token, portal }),
    enabled: enabled && section !== '' && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Progres klaim berubah saat petugas lain mencatat tindak lanjutnya, jadi cache-nya
    // pendek — sama dengan Inbox Admin, karena keduanya dibuka berulang kali sepanjang
    // hari.
    staleTime: 15 * 1000,
  })
}

/**
 * buildPath menyusun parameter permintaan.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `bagian` yang kedua akan menjadi bagian
 * bawaan sementara yang pertama memang itu yang diinginkan.
 */
function buildPath(section: string, filter: FilterForm, page: number): string {
  const params = new URLSearchParams()

  params.set('bagian', section)

  if (filter.cari.trim()) params.set('cari', filter.cari.trim())
  if (filter.bisnis) params.set('bisnis', filter.bisnis)
  if (filter.dari) params.set('dari', filter.dari)
  if (filter.sampai) params.set('sampai', filter.sampai)
  if (page > 1) params.set('halaman', String(page))

  return `${PATH}?${params.toString()}`
}
