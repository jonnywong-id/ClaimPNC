import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, unduhBerkas, uploadAPI, type DownloadedFile } from '@/api/client'
import type {
  WorkshopBankListResponse,
  WorkshopBranchListResponse,
  WorkshopCityListResponse,
  WorkshopDecisionInput,
  WorkshopDecisionResponse,
  WorkshopDocumentResponse,
  WorkshopInput,
  WorkshopListResponse,
  WorkshopResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/bengkel'
const ROUTE_BRANCH = '/api/master/bengkel/cabang'
const ROUTE_CITY = '/api/master/bengkel/kota'
const ROUTE_BANK = '/api/master/bengkel/bank'
const ROUTE_DECISION = '/api/master/bengkel/keputusan'

/**
 * Panjang minimum kata kunci lookup Kota.
 *
 * Harus sama dengan `masterbengkel.MinLookupKeyword` di backend. Server tetap yang
 * berwenang — ia menjawab daftar kosong untuk kata kunci yang lebih pendek — dan angka di
 * sini hanya mencegah perjalanan jaringan yang sudah pasti kosong.
 */
export const MIN_LOOKUP_KEYWORD = 2

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
  return ['master-bengkel', portal, token, status] as const
}

function branchKey(portal: string | null, token: string | null) {
  return ['master-bengkel-cabang', portal, token] as const
}

function cityKey(portal: string | null, token: string | null, keyword: string) {
  return ['master-bengkel-kota', portal, token, keyword] as const
}

function bankKey(portal: string | null, token: string | null) {
  return ['master-bengkel-bank', portal, token] as const
}

/**
 * Hook daftar Master Bengkel.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti `pyMaxRecords=500`
 * yang memotong daftar Pega tanpa satu pun cara mempersempitnya. Yang dipakai layar
 * sekarang adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain —
 * pencarian kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian: keduanya mengubah hal yang sama dan memecahnya berarti
 * DataTable menyaring sebagian data yang sudah tersaring server.
 */
export function useWorkshopList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<WorkshopListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar cabang untuk dropdown Cabang.
 *
 * Isinya jarang berubah, tetapi ia TETAP per portal: daftar cabang dibaca dari basis data
 * entitas yang sedang dipilih, bukan dari satu tempat bersama.
 */
export function useWorkshopBranchList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: branchKey(portal, token),
    queryFn: () => callAPI<WorkshopBranchListResponse>(ROUTE_BRANCH, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Hook daftar bank untuk dropdown Bank. */
export function useWorkshopBankList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: bankKey(portal, token),
    queryFn: () => callAPI<WorkshopBankListResponse>(ROUTE_BANK, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook lookup Kota.
 *
 * Berbeda dari Cabang dan Bank yang dimuat penuh, Kota MEMAKAI kata kunci: tabelnya
 * berbaris ribuan, dan memuatnya penuh ke peramban untuk daftar yang hanya akan dipilih
 * satu adalah biaya yang tidak perlu dibayar berulang kali.
 *
 * Tidak dijalankan untuk kata kunci yang terlalu pendek. Server pun menjawabnya dengan
 * daftar kosong, tetapi menembaknya tetap berarti satu perjalanan jaringan untuk setiap
 * huruf yang diketik pada dua huruf pertama.
 */
export function useWorkshopCityLookup(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: cityKey(portal, token, clean),
    queryFn: () =>
      callAPI<WorkshopCityListResponse>(`${ROUTE_CITY}?cari=${encodeURIComponent(clean)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && clean.length >= MIN_LOOKUP_KEYWORD,
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
export function useCreateWorkshop() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: WorkshopInput) =>
      callAPI<WorkshopResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-bengkel'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/UpdateBengkelHE_act` step 7 menetapkan APPROVAL := "0"
 * tanpa syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSaveWorkshop() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: WorkshopInput }) =>
      callAPI<WorkshopResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-bengkel'] })
    },
  })
}

/**
 * Hook keputusan borongan.
 *
 * SATU permintaan untuk seluruh baris yang dicentang, bukan satu per baris. Bentuknya
 * mengikuti `Activity/SetApprovalAllMaster`, yang menelusuri baris bercentang lalu
 * menetapkan status yang sama pada seluruhnya — memecahnya menjadi sederet permintaan
 * akan mengubah operasi yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
 */
export function useDecideWorkshop() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: WorkshopDecisionInput) =>
      callAPI<WorkshopDecisionResponse>(ROUTE_DECISION, {
        metode: 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-bengkel'] })
    },
  })
}

/** Kunci cache dokumen satu bengkel. */
const documentKey = (portal: string | null, token: string | null, id: string) =>
  ['master-bengkel', 'dokumen', portal, token, id] as const

/**
 * Hook metadata dokumen yang tertaut pada satu bengkel.
 *
 * Padanan `Activity/GetDetailDocument`, yang membaca `DOKUMENID` dari barisnya lalu
 * mengambil lampirannya.
 *
 * 404 adalah jawaban yang WAJAR di sini — bengkel yang belum pernah dilampiri. Karena itu
 * percobaan ulang dimatikan: mengulang permintaan yang sudah pasti 404 hanya menunda
 * layarnya tanpa mengubah hasilnya.
 */
export function useWorkshopDocument(id: string, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: documentKey(portal, token, id),
    queryFn: () =>
      callAPI<WorkshopDocumentResponse>(`${ROUTE}/${encodeURIComponent(id)}/dokumen`, {
        token,
        portal,
      }),
    enabled: enabled && id !== '' && token !== null && portal !== null,
    retry: false,
  })
}

/**
 * Hook unggah dokumen.
 *
 * Berkasnya dikirim sebagai `multipart/form-data` lewat uploadAPI — bukan `fetch` sendiri,
 * sesuai aturan "tidak ada fetch di dalam komponen".
 *
 * `BENGKEL_HE` hanya punya SATU kolom `DOKUMENID`, sehingga unggahan berikutnya
 * MENGGANTIKAN tautan yang sebelumnya. Itu perilaku Pega, bukan penyederhanaan.
 */
export function useUploadWorkshopDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, berkas }: { id: string; berkas: File }) =>
      uploadAPI<WorkshopDocumentResponse>(`${ROUTE}/${encodeURIComponent(id)}/dokumen`, {
        berkas,
        token,
        portal,
      }),
    onSuccess: () => {
      // Daftar ikut disegarkan: DOKUMENID adalah kolom barisnya, sehingga baris yang
      // tampil di grid pun berubah — bukan hanya isi dialognya.
      void client.invalidateQueries({ queryKey: ['master-bengkel'] })
    },
  })
}

/**
 * unduhDokumenBengkel mengambil isi berkasnya, bukan metadatanya.
 *
 * Ia BUKAN hook: unduhan adalah tindakan sesaat yang dipicu tombol, dan menaruhnya di
 * cache TanStack Query berarti menyimpan berkas beberapa megabyte di memori tanpa satu pun
 * yang membacanya lagi.
 */
export async function unduhDokumenBengkel(
  id: string,
  token: string | null,
  portal: string | null,
): Promise<DownloadedFile> {
  return unduhBerkas(`${ROUTE}/${encodeURIComponent(id)}/dokumen/berkas`, { token, portal })
}
