import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  ProgressStatus2Input,
  ProgressStatus2ListResponse,
  ProgressStatus2ParentListResponse,
  ProgressStatus2Response,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/status-progres-2'
const ROUTE_PARENT = '/api/master/status-progres-2/induk'

/**
 * listKey2 menyertakan portal DAN token, alasannya sama persis dengan tingkat 1.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Kuncinya BERBEDA dari tingkat 1 (`master-status-progres-2`), sehingga kedua layar tidak
 * pernah saling menimpa cache meski hidup di modul yang sama.
 */
function listKey2(portal: string | null, token: string | null) {
  return ['master-status-progres-2', portal, token] as const
}

/** Kunci daftar induk; ikut per portal karena isinya milik entitas, bukan aplikasi. */
function parentKey2(portal: string | null, token: string | null) {
  return ['master-status-progres-2-induk', portal, token] as const
}

/**
 * Hook daftar master status progres 2.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useProgressStatus2List() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey2(portal, token),
    queryFn: () => callAPI<ProgressStatus2ListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar induk untuk dropdown "Status Progres 1".
 *
 * BERBEDA dari useClaimPositionList pada tingkat 1, ia MENUNTUT portal: isinya dibaca dari
 * tabel tingkat 1 milik entitas yang bersangkutan, bukan daftar tetap milik aplikasi. Dua
 * entitas punya Status Progres 1 yang berbeda, dan menyajikan daftar satu entitas kepada
 * entitas lain adalah kebocoran yang justru dicegah R-20.
 *
 * Karena itu pula ia TIDAK diberi staleTime panjang seperti daftar posisi klaim — isinya
 * dapat berubah kapan saja lewat layar Master Status Progres 1.
 */
export function useProgressStatus2ParentList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: parentKey2(portal, token),
    queryFn: () => callAPI<ProgressStatus2ParentListResponse>(ROUTE_PARENT, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook penambahan status progres 2.
 *
 * Tidak ada hook penyuntingan, dan itu bukan pekerjaan yang belum selesai: sistem lama
 * tidak memiliki satu pun pernyataan yang mengubah isi POOLDATA.GCNM_MST_PROGRESS setelah
 * barisnya tersimpan. Alasan lengkapnya ada pada doc comment `masterstatusprogres.Repo2`
 * di backend.
 */
export function useCreateProgressStatus2() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: ProgressStatus2Input) =>
      callAPI<ProgressStatus2Response>(ROUTE, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari isi tabel, dan petugas lain dapat menambah baris pada saat
    // yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi tabel yang
    // sebenarnya.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: listKey2(portal, token) })
    },
  })
}
