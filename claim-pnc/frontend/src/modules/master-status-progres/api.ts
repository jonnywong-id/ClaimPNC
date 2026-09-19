import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  ClaimPositionListResponse,
  ProgressStatusInput,
  ProgressStatusListResponse,
  ProgressStatusResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/status-progres-1'
const ROUTE_POSITION = '/api/master/posisi-klaim'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang
 * menjawab (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat angka yang masuk akal, dan tidak ada apa pun
 * di layar yang menandakan data itu milik badan hukum lain (R-20).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola usePortalList.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-status-progres-1', portal, token] as const
}

/**
 * Hook daftar master status progres 1.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
 * portal (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan
 * akan menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar
 * yang menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useProgressStatusList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<ProgressStatusListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar posisi klaim untuk dropdown.
 *
 * Tidak menuntut portal: keempat posisi adalah daftar milik aplikasi, bukan isi basis
 * data entitas mana pun (lihat internal/masterstatusprogres/position.go).
 *
 * Daftarnya diambil dari server, tidak disalin ke sini. Menyalinnya berarti keempat
 * nilai itu hidup di dua tempat, dan tempat kedua akan terlupa ketika daftarnya kelak
 * pindah menjadi master data `F-4`.
 */
export function useClaimPositionList() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['posisi-klaim', token],
    queryFn: () => callAPI<ClaimPositionListResponse>(ROUTE_POSITION, { token }),
    enabled: token !== null,
    // Daftarnya tetap selama aplikasi berjalan; memuatnya ulang setiap kali komponen
    // dipasang hanya menambah permintaan tanpa menambah apa pun.
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan status progres. */
export function useCreateProgressStatus() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: ProgressStatusInput) =>
      callAPI<ProgressStatusResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari isi tabel, dan petugas lain dapat menambah baris pada
    // saat yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi
    // tabel yang sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan status progres. */
export function useUpdateProgressStatus() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ProgressStatusInput }) =>
      callAPI<ProgressStatusResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}
