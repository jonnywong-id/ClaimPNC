import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import {
  SearchMode,
  toSaveRequest,
  type ArchiveForm,
  type ClaimSearchForm,
  type ClaimSearchResponse,
  type FillingCodeResponse,
  type OpenResponse,
  type PendingResponse,
  type SaveResponse,
  type SearchForm,
  type SearchResponse,
  type SendResponse,
} from './types'

const PATH = '/api/arsip-dokumen'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: berkas arsip satu badan hukum
 * bukan berkas badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan data entitas sebelumnya (`R-20`).
 */
const keys = {
  open: (portal: string | null, token: string | null) =>
    ['arsip-dokumen', 'buka', portal, token] as const,

  search: (portal: string | null, token: string | null, form: SearchForm, page: number) =>
    [
      'arsip-dokumen',
      'cari',
      portal,
      token,
      form.mode,
      form.kata_kunci,
      form.tanggal_dari,
      form.tanggal_sampai,
      page,
    ] as const,

  claims: (portal: string | null, token: string | null, form: ClaimSearchForm) =>
    ['arsip-dokumen', 'klaim', portal, token, form.tipe, form.nilai] as const,

  fillingCodes: (portal: string | null, token: string | null, keyword: string) =>
    ['arsip-dokumen', 'kode-filling', portal, token, keyword] as const,

  pending: (portal: string | null, token: string | null, page: number) =>
    ['arsip-dokumen', 'kirim-cabang', portal, token, page] as const,

  /** Awalan seluruh kunci modul ini, dipakai saat menyegarkan setelah menulis. */
  all: ['arsip-dokumen'] as const,
}

/**
 * Hook pembukaan layar — menyiapkan isi dropdown dan cakupan lini bisnis pemanggil.
 *
 * `staleTime` panjang karena isinya master yang jarang berubah; `gcTime` tidak dibuat tak
 * terhingga supaya perpindahan portal yang lama tidak menahan daftar entitas sebelumnya di
 * memori.
 */
export function useOpenArchive() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.open(portal, token),
    queryFn: () => callAPI<OpenResponse>(`${PATH}/buka`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 10 * 60 * 1000,
  })
}

/**
 * Hook pencarian grid ARCHIVE FILE KLAIM.
 *
 * # Kenapa penyaringan dikerjakan di SERVER
 *
 * Karena tabel arsip tumbuh terus dan tidak punya batas atas: satu berkas per klaim yang
 * ditutup, selamanya. Menyaringnya di peramban berarti mengirim seluruh isinya ke setiap
 * pencarian.
 */
export function useArchiveSearch(form: SearchForm, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.search(portal, token, form, page),
    queryFn: () => callAPI<SearchResponse>(searchPath(form, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel
    // berkedip menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Berkas arsip berubah saat petugas lain menyimpan, jadi cache-nya pendek.
    staleTime: 30 * 1000,
  })
}

/** Hook pencarian klaim pada bagian Input Data Archive. */
export function useClaimSearch(form: ClaimSearchForm, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.claims(portal, token, form),
    queryFn: () =>
      callAPI<ClaimSearchResponse>(
        `${PATH}/klaim?tipe=${encodeURIComponent(form.tipe)}` +
          `&nilai=${encodeURIComponent(form.nilai)}`,
        { token, portal },
      ),
    enabled: enabled && token !== null && portal !== null,
    staleTime: 30 * 1000,
  })
}

/** Hook isi pemilih "Pilih Kode". */
export function useFillingCodes(keyword: string, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.fillingCodes(portal, token, keyword),
    queryFn: () =>
      callAPI<FillingCodeResponse>(
        `${PATH}/kode-filling?kata_kunci=${encodeURIComponent(keyword)}`,
        { token, portal },
      ),
    enabled: enabled && token !== null && portal !== null,
    staleTime: 60 * 1000,
  })
}

/** Hook daftar berkas yang belum dikirim ke sistem Arsip. */
export function usePendingBranch(page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.pending(portal, token, page),
    queryFn: () =>
      callAPI<PendingResponse>(`${PATH}/kirim-cabang?halaman=${page}`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 30 * 1000,
  })
}

/**
 * Hook penyimpanan berkas arsip.
 *
 * Seluruh kunci modul disegarkan setelah berhasil, bukan hanya kunci pencarian yang
 * sedang tampil: berkas baru ikut mengubah daftar kirim ke cabang DAN daftar kode filling
 * — yang kedua karena kodenya disusun dari pemakaian yang sudah ada.
 */
export function useSaveArchive() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (form: ArchiveForm) =>
      callAPI<SaveResponse>(PATH, {
        metode: 'POST',
        body: toSaveRequest(form),
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Hook pengiriman satu berkas ke sistem Arsip.
 *
 * Ia TIDAK dicoba ulang otomatis. Percobaan ulang atas permintaan yang mengirim berkas ke
 * sistem milik tim lain dapat mengirimkannya dua kali — dan yang menahannya di server
 * hanyalah pemeriksaan status, yang baru berlaku setelah pengiriman pertama tercatat.
 */
export function useSendToBranch() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (id: number) =>
      callAPI<SendResponse>(`${PATH}/${id}/kirim-cabang`, {
        metode: 'POST',
        token,
        portal,
      }),
    retry: false,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}

/**
 * Hook tombol "Export To Excel".
 *
 * # Kenapa bukan useQuery
 *
 * Karena hasilnya BUKAN data yang disimpan di cache melainkan berkas yang diunduh. Ia juga
 * bukan sesuatu yang boleh diulang sendiri oleh TanStack Query: satu ekspor dapat menarik
 * puluhan ribu baris, dan percobaan ulang otomatis akan menariknya lagi tanpa satu pun
 * tindakan pengguna.
 *
 * `callAPI` tidak dipakai di sini — ia mengurai jawaban sebagai JSON, sedangkan yang
 * datang adalah CSV. Ini satu-satunya tempat di modul ini yang memanggil `fetch`
 * langsung, dan alasannya disebut supaya tidak ditiru tanpa sebab.
 */
export function useExportArchive() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (form: SearchForm) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const response = await fetch(`${PATH}/ekspor?${exportParams(form)}`, { headers: header })

      if (!response.ok) {
        // Galat dijawab sebagai JSON — header CSV baru dikirim setelah potongan pertama
        // berhasil dibaca. Pesannya dipakai apa adanya supaya penolakan portal dan
        // validasi tetap terbaca pengguna.
        const body = await response.json().catch(() => null)
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      return { blob: await response.blob(), filename: filenameOf(response) }
    },
    retry: false,
  })
}

/**
 * filenameOf membaca nama berkas dari header `Content-Disposition`.
 *
 * Bila headernya tidak terbaca — beberapa proxy membuangnya — dipakai nama cadangan,
 * bukan nama kosong yang membuat peramban menyimpannya sebagai "download".
 */
function filenameOf(response: Response): string {
  const header = response.headers.get('Content-Disposition') ?? ''
  const match = header.match(/filename="([^"]+)"/)
  return match?.[1] ?? 'archive-dokumen-klaim.csv'
}

/** exportParams menyusun penyaring ekspor — sama persis dengan daftar yang tampil. */
function exportParams(form: SearchForm): string {
  const params = new URLSearchParams()
  params.set('mode', form.mode)

  if (form.mode === SearchMode.keyword) {
    if (form.kata_kunci.trim()) params.set('kata_kunci', form.kata_kunci.trim())
  } else {
    if (form.tanggal_dari) params.set('tanggal_dari', form.tanggal_dari)
    if (form.tanggal_sampai) params.set('tanggal_sampai', form.tanggal_sampai)
  }

  return params.toString()
}

/**
 * searchPath menyusun parameter pencarian arsip.
 *
 * Isian yang tidak berlaku bagi mode terpilih TIDAK dikirim. Server memang sudah
 * membuangnya, tetapi mengirimkannya membuat kunci cache berbeda untuk pencarian yang
 * hasilnya sama — dan satu perjalanan jaringan tambahan setiap kali mode berganti.
 */
function searchPath(form: SearchForm, page: number): string {
  const params = new URLSearchParams()
  params.set('mode', form.mode)

  if (form.mode === SearchMode.keyword) {
    if (form.kata_kunci.trim()) params.set('kata_kunci', form.kata_kunci.trim())
  } else {
    if (form.tanggal_dari) params.set('tanggal_dari', form.tanggal_dari)
    if (form.tanggal_sampai) params.set('tanggal_sampai', form.tanggal_sampai)
  }

  if (page > 1) params.set('halaman', String(page))

  return `${PATH}?${params.toString()}`
}
