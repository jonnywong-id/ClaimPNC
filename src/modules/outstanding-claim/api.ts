import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, LayoutResponse } from './types'

const PATH = '/api/outstanding-claim'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: rincian klaim satu badan hukum
 * bukan rincian badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan klaim entitas sebelumnya (`R-20`).
 */
const keys = {
  layout: (portal: string | null, token: string | null) =>
    ['outstanding-claim', 'tata-letak', portal, token] as const,

  detail: (portal: string | null, token: string | null, claimID: string) =>
    ['outstanding-claim', 'rincian', portal, token, claimID] as const,
}

/**
 * Hook susunan layar — kelompok isian, judulnya, dan susunan setiap grid.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena 97 judul isian dan 10 susunan grid adalah HASIL PEMBACAAN export Pega, dan tempat
 * pembacaan itu tercatat adalah backend (`internal/outstandingclaim/section.go`).
 * Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Keadaan "terhalang" ikut datang dari sana dengan alasan yang sama — begitu Tim Pega
 * mengirim rule pemuat halaman `TreatyInMaster`, penghalangnya hilang tanpa menyunting
 * frontend.
 */
export function useOutstandingClaimLayout() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.layout(portal, token),
    queryFn: () => callAPI<LayoutResponse>(`${PATH}/tata-letak`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang setiap kali klaim lain dibuka hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu klaim.
 *
 * # Kenapa cache-nya pendek
 *
 * Karena klaim treaty yang sedang berjalan diubah petugas lain sepanjang hari — nilai
 * estimasi bertambah, spreading dihitung ulang. Menahan hasil lama membuat dua petugas
 * membaca angka yang berbeda untuk klaim yang sama.
 */
export function useOutstandingClaimDetail(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, claimID),
    queryFn: () =>
      callAPI<DetailResponse>(`${PATH}/${encodeURIComponent(claimID)}`, { token, portal }),
    enabled: claimID !== '' && token !== null && portal !== null,
    staleTime: 15 * 1000,
  })
}
