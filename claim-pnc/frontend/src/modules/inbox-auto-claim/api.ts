import { useIsFetching, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import type {
  AutoClaimBatchListResponse,
  AutoClaimLineListResponse,
  AutoClaimPremiumCheckResponse,
  AutoClaimPremiumChoicesResponse,
  AutoClaimSummaryResponse,
  AutoClaimTabListResponse,
  AutoClaimUploadResponse,
  AutoClaimUploadTemplateResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/inbox-auto-claim'
const ROUTE_SUMMARY = `${ROUTE}/ringkasan`
const ROUTE_TAB = `${ROUTE}/tab`
const ROUTE_TEMPLATE = `${ROUTE}/format-unggahan`
const ROUTE_UPLOAD = `${ROUTE}/unggah`
const ROUTE_PREMIUM = `${ROUTE}/cek-premi`

/**
 * batchKey menyertakan portal DAN token, sama seperti modul master lain.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Halaman dan penyaring ikut supaya setiap kombinasi punya cache-nya sendiri; berpindah
 * bolak-balik antarhalaman karena itu tidak menembak server lagi.
 */
function batchKey(
  portal: string | null,
  token: string | null,
  source: string,
  company: string,
  page: number,
) {
  // TAB ikut menjadi kunci. Tanpa itu, berpindah tab akan menampilkan data tab
  // sebelumnya dari cache — ketiganya membaca TABEL yang berbeda, jadi yang tampil
  // adalah data yang sama sekali bukan milik tab yang disorot.
  return ['inbox-auto-claim', portal, token, source, company, page] as const
}

function lineKey(
  portal: string | null,
  token: string | null,
  source: string,
  company: string,
  batch: string,
  page: number,
) {
  return ['inbox-auto-claim-detail', portal, token, source, company, batch, page] as const
}

/**
 * Hook daftar batch.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useAutoClaimBatchList(source: string, company: string, page: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const parameter = new URLSearchParams({ halaman: String(page), sumber: source })
  if (company !== '') parameter.set('perusahaan', company)

  return useQuery({
    queryKey: batchKey(portal, token, source, company, page),
    queryFn: () =>
      callAPI<AutoClaimBatchListResponse>(`${ROUTE}?${parameter.toString()}`, { token, portal }),
    // `source` kosong berarti daftar tab BELUM tiba, bukan "semua tab". Menembak server
    // sekarang mengambil tabel bawaan lebih dulu, lalu mengambilnya sekali lagi begitu
    // tabnya diketahui — dua permintaan untuk satu layar, dan di antaranya tabel sempat
    // menampilkan data yang tabnya belum tentu tab itu.
    enabled: token !== null && portal !== null && source !== '',
    // Halaman sebelumnya tetap tampil selama halaman baru diambil, sehingga tabel tidak
    // berkedip menjadi kosong setiap kali tombol Next ditekan.
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook daftar tab.
 *
 * Tidak menuntut portal: daftar tab sama untuk setiap entitas. Dimuat sekali dan
 * disimpan lama — ketiga tab berasal dari harness Pega dan tidak berubah saat aplikasi
 * berjalan.
 */
export function useAutoClaimTabList() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: ['inbox-auto-claim-tab', token],
    queryFn: () => callAPI<AutoClaimTabListResponse>(ROUTE_TAB, { token }),
    enabled: token !== null,
    staleTime: 60 * 60 * 1000,
  })
}

/**
 * Hook ringkasan jumlah batch per perusahaan.
 *
 * Inilah isi tabel perusahaan. Ia dikirim UTUH — tanpa paginasi server — sehingga
 * pencarian dan paginasi tabel perusahaan dikerjakan di peramban atas seluruh barisnya.
 */
export function useAutoClaimSummary(source: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['inbox-auto-claim-ringkasan', portal, token, source],
    queryFn: () =>
      callAPI<AutoClaimSummaryResponse>(`${ROUTE_SUMMARY}?sumber=${encodeURIComponent(source)}`, {
        token,
        portal,
      }),
    // Sama seperti daftar batch: menunggu tabnya diketahui lebih dulu.
    enabled: token !== null && portal !== null && source !== '',
  })
}

/**
 * Hook tombol Refresh.
 *
 * Memuat ulang KETIGA lapis sekaligus — daftar perusahaan, grid batch yang terbuka, dan
 * rincian baris. Menyegarkan satu lapis saja membuat angka di baris perusahaan berselisih
 * dengan grid di dalamnya, dan selisih itu terbaca sebagai kerusakan.
 */
export function useRefreshAutoClaim() {
  const client = useQueryClient()
  const fetching = useIsFetching({ queryKey: ['inbox-auto-claim-ringkasan'] })

  return {
    isFetching: fetching > 0,
    refresh: () => {
      void client.invalidateQueries({ queryKey: ['inbox-auto-claim-ringkasan'] })
      void client.invalidateQueries({ queryKey: ['inbox-auto-claim'] })
      void client.invalidateQueries({ queryKey: ['inbox-auto-claim-detail'] })
    },
  }
}

/** Hook rincian satu batch. */
export function useAutoClaimLineList(source: string, company: string, batch: string, page: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const enabled =
    token !== null && portal !== null && company !== '' && batch !== '' && source !== ''

  return useQuery({
    queryKey: lineKey(portal, token, source, company, batch, page),
    queryFn: () =>
      callAPI<AutoClaimLineListResponse>(
        `${ROUTE}/${encodeURIComponent(company)}/${encodeURIComponent(batch)}?halaman=${page}&sumber=${encodeURIComponent(source)}`,
        { token, portal },
      ),
    enabled,
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook bentuk berkas unggahan.
 *
 * Tidak menuntut portal: bentuk berkas sama untuk setiap entitas. Daftarnya diambil dari
 * server, tidak disalin ke sini — bila flow action Pega yang asli akhirnya tiba dan judul
 * kolomnya berbeda, yang berubah hanya satu tempat.
 */
export function useAutoClaimUploadTemplate(source: string) {
  const token = useSession((state) => state.token)

  return useQuery({
    // Tab ikut menjadi kunci: bentuk berkas BERBEDA per bisnis — Kredit tanpa tanggal
    // kejadian/lapor, Travel tanpa tanggal lapor.
    queryKey: ['inbox-auto-claim-format', token, source],
    queryFn: () =>
      callAPI<AutoClaimUploadTemplateResponse>(
        `${ROUTE_TEMPLATE}?sumber=${encodeURIComponent(source)}`,
        { token },
      ),
    enabled: token !== null && source !== '',
    // Bentuknya tetap selama aplikasi berjalan; memuatnya ulang setiap kali form dibuka
    // hanya menambah permintaan tanpa menambah apa pun.
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook unggah berkas klaim. */
export function useUploadAutoClaim(source: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => {
      const form = new FormData()
      // Nama bagiannya harus sama persis dengan uploadFormField di
      // internal/inboxautoclaim/http/routes.go.
      form.append('berkas', file)
      return callAPI<AutoClaimUploadResponse>(
        `${ROUTE_UPLOAD}?sumber=${encodeURIComponent(source)}`,
        {
          metode: 'POST',
          body: form,
          token,
          portal,
        },
      )
    },
    // SELURUH daftar batch dimuat ulang, bukan hanya halaman yang sedang tampil: batch
    // baru dapat muncul di halaman mana pun tergantung urutannya, dan nomor batch
    // diterbitkan server sehingga layar tidak dapat menebak di mana ia akan muncul.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['inbox-auto-claim'] })
      // Ringkasan ikut disegarkan. Tanpa ini, grafik dan tabel ringkasan tetap
      // menampilkan angka sebelum unggahan — dan selisihnya dengan grid yang sudah
      // diperbarui terbaca sebagai kerusakan, bukan sebagai data basi.
      void client.invalidateQueries({ queryKey: ['inbox-auto-claim-ringkasan'] })
    },
  })
}

/**
 * Hook unduh berkas ekspor.
 *
 * Ia mutation, bukan query, walau permintaannya GET. Sebabnya bukan semantik HTTP
 * melainkan siklus hidupnya: unduhan dipicu tombol, tidak punya cache yang berguna, dan
 * tidak boleh berjalan ulang sendiri saat komponen dipasang kembali — tiga hal yang
 * justru menjadi sifat query.
 */
export function useExportAutoClaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (input: {
      source: string
      company: string
      batch: string
      hasil: 'berhasil' | 'gagal'
    }) => {
      const path =
        `${ROUTE}/${encodeURIComponent(input.company)}/${encodeURIComponent(input.batch)}` +
        `/ekspor?hasil=${input.hasil}&sumber=${encodeURIComponent(input.source)}`

      const file = await unduhBerkas(path, { token, portal })
      simpanBerkas(file)
      return file.namaBerkas
    },
  })
}

/** Hook isi kedua isian tab Cek Premi (Nama Bisnis dan Sumber Bisnis). */
export function useAutoClaimPremiumChoices() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['inbox-auto-claim-cek-premi-pilihan', portal, token],
    queryFn: () =>
      callAPI<AutoClaimPremiumChoicesResponse>(`${ROUTE_PREMIUM}/pilihan`, { token, portal }),
    enabled: token !== null && portal !== null,
    // Master bisnis dan master auto claim jarang berubah.
    staleTime: 10 * 60 * 1000,
  })
}

/**
 * Hook tombol Cek Premi.
 *
 * Ia query, bukan mutation: tombolnya hanya membaca. `pair` null berarti tombolnya belum
 * ditekan. Hasilnya TIDAK disimpan di cache lama-lama (`gcTime: 0`) — total premi berubah
 * setiap ada pembayaran, dan angka lama yang tampil lagi saat tab dibuka ulang akan
 * dikira angka terbaru.
 */
export function useAutoClaimPremiumCheck(pair: { bisnis: string; sumber: string } | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['inbox-auto-claim-cek-premi', portal, token, pair?.bisnis, pair?.sumber],
    queryFn: () =>
      callAPI<AutoClaimPremiumCheckResponse>(
        `${ROUTE_PREMIUM}?kode_bisnis=${encodeURIComponent(pair?.bisnis ?? '')}` +
          `&kode_sumber_bisnis=${encodeURIComponent(pair?.sumber ?? '')}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && pair !== null,
    gcTime: 0,
    retry: false,
  })
}
