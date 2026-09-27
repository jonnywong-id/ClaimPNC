import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  GroupingDecisionInput,
  GroupingDecisionResponse,
  GroupingInput,
  GroupingListResponse,
  GroupingOptionsResponse,
  GroupingPartResponse,
  GroupingResponse,
  GroupingSideResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/grouping-sparepart'
const ROUTE_DECISION = '/api/master/grouping-sparepart/keputusan'
const ROUTE_OPTIONS = '/api/master/grouping-sparepart/pilihan'
const ROUTE_SIDES = '/api/master/grouping-sparepart/sisi'
const ROUTE_PART = '/api/master/grouping-sparepart/sparepart'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030). Tanpa
 * itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache — pengguna
 * melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan data itu
 * milik badan hukum lain (R-20).
 *
 * Penyaring statusnya ikut pula: ketiga tab layar adalah tiga kombinasi penyaring atas satu
 * endpoint, dan tanpa status di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string) {
  return ['master-grouping-sparepart', portal, token, status] as const
}

/** Kunci daftar pilihan Panel dan Tipe Kendaraan. Keduanya data entitas, bukan konstanta. */
function optionsKey(portal: string | null, token: string | null) {
  return ['master-grouping-sparepart-pilihan', portal, token] as const
}

/**
 * Kunci daftar Sisi. Panel yang dipilih ikut, karena isinya memang berbeda per panel.
 *
 * Nama panelnya ikut pula: kueri aslinya menyaring `id_panel` DAN `nama` sekaligus, sehingga
 * dua panel berkode sama bernama berbeda adalah kunci cache yang berbeda.
 */
function sidesKey(
  portal: string | null,
  token: string | null,
  panelID: string,
  panelName: string,
) {
  return ['master-grouping-sparepart-sisi', portal, token, panelID, panelName] as const
}

/** Kunci pencarian sparepart. Nomornya ikut — itulah yang menentukan jawabannya. */
function partKey(portal: string | null, token: string | null, number: string) {
  return ['master-grouping-sparepart-part', portal, token, number] as const
}

/**
 * Hook daftar Master Grouping Sparepart.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan menampilkan
 * pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang penyaring sisi server. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian
 * kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function useGroupingList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<GroupingListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan Panel dan Tipe Kendaraan.
 *
 * Ia MENUNGGU portal dipilih, karena endpoint-nya pun menuntutnya: isinya data entitas, bukan
 * konstanta.
 */
export function useGroupingOptions() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: optionsKey(portal, token),
    queryFn: () => callAPI<GroupingOptionsResponse>(ROUTE_OPTIONS, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar Sisi pada sebuah panel.
 *
 * Ia BARU berjalan setelah panelnya dipilih — persis seperti `Activity/GetSisiPanel-Act.xml`
 * yang dipanggil setelah Nama Panel dipilih, bukan saat layar dibuka.
 *
 * Panel yang belum punya baris lokasi menjawab daftar KOSONG, bukan galat; form yang
 * mengatakan panel itu belum punya sisi.
 */
export function useGroupingSides(panelID: string, panelName: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const query = new URLSearchParams({ id_panel: panelID, nama_panel: panelName })

  return useQuery({
    queryKey: sidesKey(portal, token, panelID, panelName),
    queryFn: () =>
      callAPI<GroupingSideResponse>(`${ROUTE_SIDES}?${query.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null && panelID !== '' && panelName !== '',
  })
}

/**
 * Hook pencarian sparepart menurut nomornya.
 *
 * Ia padanan `Activity/SetDataSparepart-Act.xml`, yang berjalan saat isian Nomor Sparepart
 * kehilangan fokus dan mengisi kelima isian turunannya.
 *
 * # Kenapa TIDAK dijalankan setiap ketikan
 *
 * Karena di Pega pun tidak: pengisiannya dipicu saat isian selesai diketik, bukan saat sedang
 * diketik. Menjalankannya per ketikan berarti satu permintaan basis data untuk setiap huruf —
 * dan sebagian besar jawabannya "tidak ditemukan", yang akan membuat pesan galat berkedip
 * sepanjang pengguna mengetik.
 *
 * Yang dipakai: nomor yang SUDAH selesai diketik, dikirim pemanggil lewat parameter ini.
 * Galat 404 TIDAK diulang-coba — nomor yang tidak ada tidak akan menjadi ada.
 */
export function useGroupingPart(number: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: partKey(portal, token, number),
    queryFn: () =>
      callAPI<GroupingPartResponse>(`${ROUTE_PART}?nomor=${encodeURIComponent(number)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && number !== '',
    retry: false,
  })
}

/**
 * Hook penambahan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil, bukan hanya daftar tab yang sedang
 * dibuka: baris baru lahir berstatus menunggu, sehingga yang berubah adalah tab Waiting
 * Approval — tab yang justru TIDAK sedang dilihat pengguna saat ia menambah dari tab Approve.
 */
export function useCreateGrouping() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: GroupingInput) =>
      callAPI<GroupingResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-grouping-sparepart'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval, sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSaveGrouping() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: GroupingInput }) =>
      callAPI<GroupingResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-grouping-sparepart'] })
    },
  })
}

/**
 * Hook keputusan borongan.
 *
 * SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris — memecahnya
 * menjadi sederet permintaan akan mengubah operasi yang di Pega utuh menjadi sesuatu yang
 * dapat gagal separuh jalan.
 *
 * TANPA catatan: kedua tabel modul ini tidak punya kolom penampungnya.
 */
export function useDecideGrouping() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: GroupingDecisionInput) =>
      callAPI<GroupingDecisionResponse>(ROUTE_DECISION, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-grouping-sparepart'] })
    },
  })
}
