import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  RejectionInput,
  RejectionListResponse,
  RejectionParentListResponse,
  RejectionResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/penolakan-klaim'
const ROUTE_PARENT = '/api/master/penolakan-klaim/status-1'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-penolakan-klaim', portal, token] as const
}

/** Kunci daftar Status Penolakan 1; ikut per portal karena isinya milik entitas. */
function parentKey(portal: string | null, token: string | null) {
  return ['master-penolakan-klaim-status-1', portal, token] as const
}

/**
 * Hook daftar Status Penolakan 2.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useRejectionList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<RejectionListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan "Status Penolakan 1".
 *
 * Ia MENUNTUT portal: isinya dibaca dari tabel milik entitas yang bersangkutan, bukan
 * daftar tetap milik aplikasi. Dua entitas punya alasan penolakan yang berbeda, dan
 * menyajikan daftar satu entitas kepada entitas lain adalah kebocoran yang justru dicegah
 * R-20.
 *
 * Ia juga TIDAK diberi staleTime panjang: isinya bertambah setiap kali seseorang menyimpan
 * dengan induk baru, termasuk dari layar ini sendiri.
 */
export function useRejectionParentList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: parentKey(portal, token),
    queryFn: () => callAPI<RejectionParentListResponse>(ROUTE_PARENT, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook penambahan Status Penolakan 2.
 *
 * Daftar INDUK ikut dimuat ulang, bukan hanya daftar barisnya: penambahan dengan induk
 * baru menerbitkan baris tingkat 1 yang harus langsung muncul sebagai pilihan berikutnya.
 * Tanpa itu, petugas yang baru saja membuat induk tidak menemukannya di daftar dan
 * membuatnya lagi — persis duplikasi yang modul ini perbaiki.
 */
export function useCreateRejection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: RejectionInput) =>
      callAPI<RejectionResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: parentKey(portal, token) })
    },
  })
}

/**
 * Hook pengubahan Status Penolakan 2.
 *
 * # Yang perlu diketahui pemakainya
 *
 * Pengubahan MENGEMBALIKAN baris ke antrean persetujuan: server menyetel statusnya menjadi
 * menunggu dan memperbarui waktu pengajuan
 * (`Database/MASTERPENOLAKANKLAIM2.prc:14`, dipertahankan atas keputusan Work Owner
 * 2026-09-19). Layar menyatakannya sebelum pengguna menyimpan, supaya akibatnya tidak
 * ditemukan setelah terjadi.
 */
export function useUpdateRejection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: RejectionInput }) =>
      callAPI<RejectionResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN disunting di sisi klien. Status dan waktu
    // pengajuan keduanya diterbitkan server, dan petugas lain dapat mengubah baris yang
    // sama pada saat bersamaan.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
      void client.invalidateQueries({ queryKey: parentKey(portal, token) })
    },
  })
}
