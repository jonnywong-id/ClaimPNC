import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  PanelDecisionInput,
  PanelDecisionResponse,
  PanelInput,
  PanelListResponse,
  PanelOptionsResponse,
  PanelResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/panel'
const ROUTE_DECISION = '/api/master/panel/keputusan'
const ROUTE_OPTIONS = '/api/master/panel/pilihan'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Penyaring statusnya ikut pula: ketiga tab layar adalah tiga kombinasi penyaring atas
 * satu endpoint, dan tanpa status di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string) {
  return ['master-panel', portal, token, status] as const
}

/**
 * optionsKey TIDAK menyertakan portal.
 *
 * Isinya konstanta yang ditanam di `Activity/SetLokasiSisiPanel-Act.xml`, bukan bacaan
 * basis data — sehingga jawabannya sama untuk keempat entitas. Menyertakan portal di
 * kuncinya berarti memuat ulang daftar yang sama setiap kali pengguna berpindah entitas.
 */
function optionsKey(token: string | null) {
  return ['master-panel-pilihan', token] as const
}

/**
 * Hook daftar Master Panel.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti `pyMaxRecords=500` yang
 * memotong daftar Pega tanpa satu pun cara mempersempitnya. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian
 * kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function usePanelList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<PanelListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan Lokasi dan Sisi.
 *
 * Ia TIDAK menunggu portal dipilih, karena endpoint-nya pun tidak menuntutnya: isinya
 * konstanta, bukan data entitas. Form karena itu dapat menggambar dropdown-nya bahkan
 * sebelum pengguna memilih entitas.
 */
export function usePanelOptions() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: optionsKey(token),
    queryFn: () => callAPI<PanelOptionsResponse>(ROUTE_OPTIONS, { token }),
    enabled: token !== null,
  })
}

/**
 * Hook penambahan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil, bukan hanya daftar tab yang sedang
 * dibuka: baris baru lahir berstatus menunggu, sehingga yang berubah adalah tab Waiting
 * Approval — tab yang justru TIDAK sedang dilihat pengguna saat ia menambah dari tab
 * Approve.
 */
export function useCreatePanel() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PanelInput) =>
      callAPI<PanelResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/CNMUpdatePanelHE_act` menetapkan APPROVAL := "0" tanpa
 * syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSavePanel() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PanelInput }) =>
      callAPI<PanelResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}

/**
 * Hook keputusan borongan.
 *
 * SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris. Bentuknya
 * mengikuti `Activity/SetApprovalAllMaster` beserta layarnya
 * `Section/ApprovalMasterPanelHE-Section.xml`, yang menyediakan satu isian Catatan untuk
 * seluruh pilihan — memecahnya menjadi sederet permintaan akan mengubah operasi yang di
 * Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
 */
export function useDecidePanel() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PanelDecisionInput) =>
      callAPI<PanelDecisionResponse>(ROUTE_DECISION, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}
