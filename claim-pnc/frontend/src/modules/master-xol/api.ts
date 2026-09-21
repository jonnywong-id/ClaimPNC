import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  XOLBusinessGroupResponse,
  XOLFormResponse,
  XOLInput,
  XOLListResponse,
  XOLResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTES = '/api/master/xol'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut di dalam kunci. Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat struktur treaty yang masuk akal, dan tidak ada
 * apa pun di layar yang menandakan ia milik badan hukum lain (`R-20`).
 */
const key = {
  list: (portal: string | null, token: string | null) => ['master-xol', portal, token] as const,
  detail: (portal: string | null, token: string | null, id: string) =>
    ['master-xol', portal, token, 'detail', id] as const,
  form: (portal: string | null, token: string | null) =>
    ['master-xol', portal, token, 'form'] as const,
  business: (portal: string | null, token: string | null, tipe: string) =>
    ['master-xol', portal, token, 'bisnis', tipe] as const,
}

/**
 * Hook daftar Master XOL.
 *
 * Menggantikan `RDB List/GetDataMasterXOL-SQL.xml` yang mengisi grid harness
 * `DetailMasterXOL`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server — sama seperti kueri lamanya,
 * yang memang tidak punya satu pun pembatas baris. Isinya tumbuh beberapa baris per tahun
 * treaty: delapan baris setelah sebelas tahun dipakai.
 */
export function useXOLList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.list(portal, token),
    queryFn: () => callAPI<XOLListResponse>(ROUTES, { token, portal }),
    // Tidak dijalankan sebelum portal dipilih: backend memang menolak permintaan tanpa
    // portal, tetapi menembaknya hanya untuk menerima penolakan akan menampilkan pesan
    // galat pada layar yang sebenarnya belum siap dibuka.
    enabled: token !== null && portal !== null,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook detail satu induk, lengkap dengan bisnis, layer, dan reas-nya.
 *
 * Menggantikan `Activity/UpdateMasterXOL-Act.xml`, yang memuat keempat tingkat sekaligus
 * sebelum form Ubah dibuka.
 *
 * `staleTime` nol, berbeda dari daftar: yang dibuka di sini akan segera DISUNTING, dan
 * menyunting salinan basi berarti menimpa perubahan orang lain tanpa menyadarinya.
 */
export function useXOLDetail(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.detail(portal, token, id ?? ''),
    queryFn: () => callAPI<XOLResponse>(`${ROUTES}/${encodeURIComponent(id ?? '')}`, { token, portal }),
    enabled: token !== null && portal !== null && id !== null && id !== '',
    staleTime: 0,
  })
}

/** Hook bekal awal layar: pilihan Tahun dan pilihan Type XOL. */
export function useXOLForm() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.form(portal, token),
    queryFn: () => callAPI<XOLFormResponse>(`${ROUTES}/form`, { token, portal }),
    enabled: token !== null && portal !== null,
    // Daftar tahun treaty berubah paling sering setahun sekali.
    staleTime: 30 * 60 * 1000,
  })
}

/**
 * Hook pilihan grup bisnis untuk sebuah Type XOL.
 *
 * Menggantikan `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`, yang dijalankan ulang
 * setiap kali dropdown Type XOL berubah — itulah sebabnya Type ikut di dalam kunci cache.
 */
export function useXOLBusinessGroup(tipe: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: key.business(portal, token, tipe),
    queryFn: () =>
      callAPI<XOLBusinessGroupResponse>(`${ROUTES}/bisnis?tipe=${encodeURIComponent(tipe)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
    staleTime: 30 * 60 * 1000,
  })
}

type SaveFields = {
  /** Kosong berarti menambah; terisi berarti mengubah induk dengan ID itu. */
  id?: string
  input: XOLInput
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu, dan karena penyimpanannya memang satu
 * langkah di sistem lama: `InsertUpdateMasterXOL` menangani keduanya, dan yang
 * membedakannya hanyalah `TempXOL.BranchID` kosong atau tidak. Di sini pembedanya metode
 * HTTP.
 *
 * **Menyimpan sekaligus MENGAJUKAN KE KOMITE**, tanpa syarat — persis seperti layar lama.
 * Induk yang sudah disetujui akan kembali ke keadaan menunggu, karena strukturnya berubah
 * dan persetujuan atas bentuk sebelumnya tidak lagi berlaku.
 */
export function useSaveXOL() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: SaveFields) =>
      callAPI<XOLResponse>(id ? `${ROUTES}/${encodeURIComponent(id)}` : ROUTES, {
        metode: id ? 'PUT' : 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: (_result, variables) => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan,
      // nomornya hanya diketahui server — menebaknya di klien akan menampilkan nomor yang
      // salah sampai muat ulang berikutnya.
      void client.invalidateQueries({ queryKey: key.list(portal, token) })
      if (variables.id) {
        void client.invalidateQueries({ queryKey: key.detail(portal, token, variables.id) })
      }
    },
  })
}

/**
 * Hook hapus satu induk BESERTA seluruh anaknya.
 *
 * Kaskadenya adalah selisih terencana terhadap Pega, yang menghapus satu tabel saja dan
 * meninggalkan baris yatim di produksi (keputusan Work Owner 2026-09-20).
 */
export function useDeleteXOL() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: string) =>
      callAPI<void>(`${ROUTES}/${encodeURIComponent(id)}`, { metode: 'DELETE', token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: key.list(portal, token) })
    },
  })
}

type DeleteChildFields =
  | { jenis: 'bisnis'; masterID: string; id: string }
  | { jenis: 'layer'; masterID: string; id: string }
  | { jenis: 'reas'; masterID: string; layerID: string; id: string }

/**
 * Hook hapus satu baris anak: grup bisnis, lapisan, atau reas.
 *
 * Ketiganya menghapus SEKETIKA, tidak menunggu tombol Simpan — persis seperti tombol
 * Hapus per baris di layar lama, yang memanggil `DeleteFromTabelMst` langsung. Itulah
 * sebabnya Simpan bersifat upsert dan tidak pernah menghapus baris yang tidak disebut.
 */
export function useDeleteXOLChild() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (fields: DeleteChildFields) => {
      const base = `${ROUTES}/${encodeURIComponent(fields.masterID)}`
      const path =
        fields.jenis === 'reas'
          ? `${base}/layer/${encodeURIComponent(fields.layerID)}/reas/${encodeURIComponent(fields.id)}`
          : `${base}/${fields.jenis}/${encodeURIComponent(fields.id)}`
      return callAPI<void>(path, { metode: 'DELETE', token, portal })
    },
    onSuccess: (_result, fields) => {
      void client.invalidateQueries({ queryKey: key.detail(portal, token, fields.masterID) })
    },
  })
}
