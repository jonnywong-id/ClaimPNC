import { useQuery } from '@tanstack/react-query'

import { APIError, callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, FilterForm, ListResponse, MetadataResponse } from './types'

const PATH = '/api/inbox-service-center'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: klaim portal rekanan satu badan
 * hukum bukan klaim badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan klaim entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-service-center', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, filter: FilterForm, page: number) =>
    [
      'inbox-service-center',
      'daftar',
      portal,
      token,
      filter.tab,
      filter.cari,
      page,
    ] as const,
}

/**
 * Hook keterangan layar — daftar tab, kolomnya, dan keterbatasan yang berlaku.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu tercatat
 * adalah backend (`internal/inboxservicecenter/tab.go`). Menyalinnya ke layar berarti daftar
 * yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
 */
export function useInboxServiceCenterMetadata() {
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
 * # Kenapa penyaringan dan paginasi dikerjakan di SERVER
 *
 * Karena `POOLDATA.T_KLAIM_PORTAL_REKANAN` tumbuh tanpa batas dan barisnya memuat nama
 * nasabah. Menyaringnya di peramban berarti mengirim seluruh antrean ke setiap ketikan —
 * termasuk baris yang seharusnya tidak pernah sampai ke sana.
 *
 * Satu hal yang perlu diketahui saat membaca jumlah "Total": begitu kotak cari terisi,
 * paginasi MATI dan seluruh baris yang cocok dikirim sekaligus. Itu ditiru apa adanya dari
 * Pega (`P-5`), dan server menyatakannya lewat `paginasi.aktif`.
 */
export function useInboxServiceCenterList(
  filter: FilterForm,
  page: number,
  enabled: boolean,
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter, page),
    queryFn: () => callAPI<ListResponse>(buildPath(filter, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean berubah saat petugas lain memutuskan komite, jadi cache-nya pendek.
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
  if (filter.cari.trim()) params.set('cari', filter.cari.trim())
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/**
 * Hook rincian satu klaim.
 *
 * # Kenapa tidak memakai ulang baris yang sudah ada di daftar
 *
 * Karena daftar hanya membawa enam kolom; rincian membawa 83. Menyusun rincian dari baris
 * daftar berarti menampilkan layar yang 77 isiannya kosong tanpa alasan yang terbaca.
 *
 * Portal ikut menjadi bagian kunci cache, sama seperti daftar: rincian satu badan hukum bukan
 * rincian badan hukum lain (`R-20`).
 */
export function useInboxServiceCenterDetail(id: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['inbox-service-center', 'rincian', portal, token, id] as const,
    queryFn: () => callAPI<DetailResponse>(`${PATH}/${encodeURIComponent(id)}`, { token, portal }),
    enabled: id !== '' && token !== null && portal !== null,

    // Rincian berubah saat petugas lain menyuntingnya di Pega — selama masa paralel, itu
    // memang masih mungkin. Cache-nya karena itu sependek daftar.
    staleTime: 15 * 1000,

    // Klaim yang tidak ditemukan TIDAK dicoba ulang: jawabannya tidak akan berubah, dan
    // mengulangnya hanya menunda pesan yang perlu dibaca pengguna.
    retry: (count, error) => !(error instanceof APIError && error.status === 404) && count < 2,
  })
}
