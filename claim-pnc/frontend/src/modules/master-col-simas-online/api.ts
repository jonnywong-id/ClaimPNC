import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  BusinessListResponse,
  SimasOnlineCauseOfLossInput,
  SimasOnlineCauseOfLossListResponse,
  SimasOnlineCauseOfLossResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/col-simas-online'
const ROUTE_BUSINESS = '/api/master/bisnis'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (`ADR-0030`). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya
 * dari cache — pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar
 * yang menandakan data itu milik badan hukum lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola usePortalList.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-col-simas-online', portal, token] as const
}

function detailKey(id: string, portal: string | null, token: string | null) {
  return ['master-col-simas-online', 'detail', id, portal, token] as const
}

function businessKey(portal: string | null, token: string | null) {
  return ['master-bisnis', portal, token] as const
}

/**
 * Hook daftar master COL Simas Online.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useCauseOfLossList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<SimasOnlineCauseOfLossListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook satu baris LENGKAP dengan pemetaan bisnisnya.
 *
 * Dimuat terpisah, bukan diambil dari hasil daftar, karena daftar memang tidak
 * membawanya: grid hanya menampilkan ID dan nama, dan menarik pemetaan seluruh baris
 * berarti satu kueri yang hasilnya tidak pernah dilihat siapa pun.
 *
 * `id` null berarti tidak ada baris yang sedang dibuka — hook-nya diam.
 */
export function useCauseOfLoss(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(id ?? '', portal, token),
    queryFn: () =>
      callAPI<SimasOnlineCauseOfLossResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, { token, portal }),
    enabled: token !== null && portal !== null && id !== null,
  })
}

/**
 * Hook daftar bisnis untuk isian Bisnis.
 *
 * MENUNTUT portal: POOLDATA.BUSINESS hidup di basis data setiap entitas, sehingga
 * "bisnis milik siapa" ditentukan portal yang aktif. Ini berbeda dari daftar posisi
 * klaim pada modul Master Status Progres, yang memang milik aplikasi.
 *
 * Daftarnya jarang berubah, sehingga tidak dimuat ulang setiap kali form dibuka.
 */
export function useBusinessList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: businessKey(portal, token),
    queryFn: () => callAPI<BusinessListResponse>(ROUTE_BUSINESS, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan cause of loss. */
export function useCreateCauseOfLoss() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: SimasOnlineCauseOfLossInput) =>
      callAPI<SimasOnlineCauseOfLossResponse>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari urutan basis data, dan petugas lain dapat menambah baris
    // pada saat yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi
    // tabel yang sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}

/** Hook penyuntingan cause of loss. */
export function useUpdateCauseOfLoss() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: SimasOnlineCauseOfLossInput }) =>
      callAPI<SimasOnlineCauseOfLossResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar DAN baris yang disunting keduanya dibuang dari cache. Tanpa yang kedua,
    // membuka kembali baris yang sama akan menampilkan pemetaan bisnis sebelum
    // perubahan — dan pengguna akan mengira penyimpanannya gagal.
    onSuccess: (_result, variables) => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: detailKey(variables.id, portal, token) })
    },
  })
}
