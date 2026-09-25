import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  AdjusterListResponse,
  AdminDetailResponse,
  AdminFilterInput,
  DetailResponse,
  FilterInput,
  MetadataResponse,
  PICFilterInput,
  PICTeknikResponse,
  ScorecardResponse,
  SummaryResponse,
} from './types'

const PATH = '/api/report-kpi'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: penilaian kinerja adjuster
 * yang bekerja untuk satu badan hukum bukan milik badan hukum lain, dan menyimpan keduanya
 * di bawah satu kunci akan membuat perpindahan portal menampilkan penilaian entitas
 * sebelumnya (`R-20`).
 *
 * SELURUH penyaring ikut pula. Ketiganya dikirim ke server dan mengubah hasilnya; kunci
 * yang tidak memuatnya akan menyajikan hasil penyaring lama untuk penyaring baru.
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['report-kpi', 'tab', portal, token] as const,

  summary: (portal: string | null, token: string | null, filter: FilterInput) =>
    ['report-kpi', 'ringkasan', portal, token, filter] as const,

  detail: (
    portal: string | null,
    token: string | null,
    filter: FilterInput,
    page: number,
  ) => ['report-kpi', 'rincian', portal, token, filter, page] as const,

  adjusters: (portal: string | null, token: string | null, filter: FilterInput) =>
    [
      'report-kpi',
      'pilihan-adjuster',
      portal,
      token,
      // Adjuster yang sedang dipilih SENGAJA tidak ikut kunci: daftar pilihan tidak
      // disaring olehnya, sehingga memasukkannya hanya membuat cache terpecah menjadi
      // sekian salinan yang isinya sama persis.
      filter.tipeReport,
      filter.dari,
      filter.sampai,
    ] as const,
}

/**
 * Hook keterangan layar — daftar tab, grid, komponen, tipe report, dan selisih terencana.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom dan komponennya adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan
 * itu tercatat adalah backend (`internal/reportkpi/component.go`, `screen.go`).
 * Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Di layar ini akibatnya lebih tajam daripada biasa: kesembilan komponen digambar sebagai
 * kesembilan kolom yang isinya sama-sama angka 1–5. Satu urutan yang bergeser tidak
 * terlihat sebagai kerusakan — ia hanya terlihat sebagai nilai yang berbeda.
 */
export function useReportKPIMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang setiap kali penyaring berubah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook grid Summary — satu baris per adjuster, nilainya rata-rata.
 *
 * # Kenapa TIDAK dipaginasi
 *
 * Karena barisnya satu per adjuster, dan jumlah adjuster eksternal terhitung puluhan.
 * Layar lama pun memuatnya sekaligus. Yang dipaginasi adalah grid Detail, yang barisnya
 * satu per kasus survei.
 */
export function useReportKPISummary(filter: FilterInput, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.summary(portal, token, filter),
    queryFn: () =>
      callAPI<SummaryResponse>(`${PATH}/adjuster/ringkasan?${params(filter)}`, {
        token,
        portal,
      }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,

    // Isinya baru berubah bila seseorang menjalankan perhitungan ulang di Pega, dan itu
    // tidak terjadi setiap menit. Cache-nya karena itu lebih panjang daripada inbox —
    // tetapi tidak tak terbatas, karena perhitungan itu memang bisa terjadi kapan saja.
    staleTime: 60 * 1000,
  })
}

/**
 * Hook grid Detail — satu baris per kasus survei, dipaginasi server.
 *
 * Paginasinya dikerjakan basis data dengan `OFFSET … FETCH NEXT`, sehingga jumlah "total"
 * tetap tepat tanpa menarik seluruh baris ke peramban (`D-10`).
 */
export function useReportKPIDetail(filter: FilterInput, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, filter, page),
    queryFn: () =>
      callAPI<DetailResponse>(
        `${PATH}/adjuster?${params(filter)}${page > 1 ? `&halaman=${page}` : ''}`,
        { token, portal },
      ),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,
    staleTime: 60 * 1000,
  })
}

/**
 * Hook isi dropdown "Pilih Adjuster".
 *
 * # Kenapa ia menuntut tipe report dan periode
 *
 * Karena daftarnya diambil dari tabel penilaian itu sendiri, sehingga ikut menyempit
 * mengikuti keduanya. Itu yang menjaga janji pada selisih terencana: setiap pilihan yang
 * muncul pasti menghasilkan baris.
 *
 * Sebaliknya ia TIDAK disaring oleh adjuster yang sedang dipilih — kalau disaring,
 * dropdown akan menyisakan satu pilihan saja dan pengguna tidak dapat berpindah lagi.
 */
export function useReportKPIAdjusters(filter: FilterInput, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.adjusters(portal, token, filter),
    queryFn: () =>
      callAPI<AdjusterListResponse>(
        `${PATH}/adjuster/pilihan?${params({ ...filter, adjuster: '' })}`,
        { token, portal },
      ),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 60 * 1000,
  })
}

/** Bahan permintaan ekspor: penyaring yang sedang berlaku, dan grid mana yang diunduh. */
type ExportInput = {
  filter: FilterInput
  grid: string
}

/**
 * Hook tombol ekspor.
 *
 * # Kenapa berkasnya diambil dengan fetch, bukan dengan tautan unduh biasa
 *
 * Karena `<a href>` dan `window.open` TIDAK membawa header — dan endpoint ini menuntut
 * dua: `Authorization` dan `X-Portal`. Satu-satunya cara memakai tautan biasa adalah
 * menaruh token di dalam alamat, dan itu ditolak dengan alasan yang sudah dicatat di
 * `api/client.ts`: nilai di URL ikut tercatat di riwayat peramban, log proxy, dan header
 * Referer.
 *
 * # Biaya yang disadari
 *
 * Peladen MENGALIRKAN berkasnya potong demi potong, tetapi peramban menampungnya utuh
 * sebagai Blob sebelum menyimpannya. Manfaat pengaliran karena itu tinggal di sisi
 * peladen — memorinya tetap datar — sedangkan memori peramban tumbuh sebesar berkasnya.
 */
export function useExportReportKPI() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (input: ExportInput) => {
      const address = `${PATH}/adjuster/ekspor?${params(input.filter)}&grid=${encodeURIComponent(input.grid)}`
      simpanBerkas(await unduhBerkas(address, { token, portal }))
    },
  })
}

/**
 * params menyusun parameter penyaring.
 *
 * Ia dipakai KEEMPAT permintaan — ringkasan, rincian, daftar pilihan, dan ekspor —
 * sehingga tidak satu pun dapat menyaring dengan cara yang berbeda dari yang lain. Berkas
 * ekspor yang penyaringnya berbeda dari tabel di layar tidak dapat dicocokkan dengan
 * apa pun.
 *
 * Isian kosong TETAP DIKIRIM sebagai parameter kosong, bukan dihilangkan. Server
 * membedakan "tidak diisi" dari "tidak dikirim" hanya pada adjuster — dan pada kedua
 * tanggal, yang kosong memang harus sampai supaya penolakannya menyebut isian mana yang
 * kurang, bukan menghasilkan permintaan yang tampak lengkap.
 */
/* ─────────────────────────── Tab KPI Admin ─────────────────────────── */

const adminKeys = {
  scorecard: (portal: string | null, token: string | null, filter: AdminFilterInput) =>
    ['report-kpi', 'kartu-skor', portal, token, filter] as const,

  detail: (
    portal: string | null,
    token: string | null,
    filter: AdminFilterInput,
    page: number,
  ) => ['report-kpi', 'rincian-admin', portal, token, filter, page] as const,
}

/**
 * Hook kartu skor tab KPI Admin.
 *
 * Ia TIDAK dipaginasi dan tidak akan pernah: kuerinya mengembalikan tepat satu baris
 * dengan belasan kolom, dan layar menggambarnya sebagai kartu, bukan tabel.
 */
export function useAdminScorecard(filter: AdminFilterInput, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: adminKeys.scorecard(portal, token, filter),
    queryFn: () =>
      callAPI<ScorecardResponse>(`${PATH}/admin/kartu-skor?${adminParams(filter)}`, {
        token,
        portal,
      }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 60 * 1000,
  })
}

/** Hook grid rincian tab KPI Admin, dipaginasi server. */
export function useAdminDetail(
  filter: AdminFilterInput,
  page: number,
  enabled: boolean,
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: adminKeys.detail(portal, token, filter, page),
    queryFn: () =>
      callAPI<AdminDetailResponse>(
        `${PATH}/admin?${adminParams(filter)}${page > 1 ? `&halaman=${page}` : ''}`,
        { token, portal },
      ),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 60 * 1000,
  })
}

/** Hook tombol ekspor tab KPI Admin — mengunduh grid rinciannya. */
export function useExportAdmin() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (filter: AdminFilterInput) => {
      const address = `${PATH}/admin/ekspor?${adminParams(filter)}`
      simpanBerkas(await unduhBerkas(address, { token, portal }))
    },
  })
}

/**
 * adminParams menyusun parameter penyaring tab KPI Admin.
 *
 * Dipakai KETIGA permintaan — kartu skor, rincian, dan ekspor — sehingga tidak satu pun
 * dapat menyaring dengan cara yang berbeda dari yang lain.
 */
function adminParams(filter: AdminFilterInput): string {
  const query = new URLSearchParams()
  query.set('kelompok', filter.kelompok)
  query.set('dari', filter.dari)
  query.set('sampai', filter.sampai)
  return query.toString()
}

function params(filter: FilterInput): string {
  const query = new URLSearchParams()
  query.set('tipe_report', filter.tipeReport)
  query.set('dari', filter.dari)
  query.set('sampai', filter.sampai)
  if (filter.adjuster) query.set('adjuster', filter.adjuster)
  return query.toString()
}

/* ─────────────────────────── Tab KPI PIC Teknik ─────────────────────────── */

const picKeys = {
  scorecards: (portal: string | null, token: string | null, filter: PICFilterInput) =>
    ['report-kpi', 'kartu-skor-pic', portal, token, filter] as const,
}

/**
 * Hook kartu skor tab KPI PIC Teknik.
 *
 * SATU permintaan untuk seluruh tab — bukan satu per PIC. Kartunya dirakit di server dari
 * empat kelompok kueri sekaligus, dan memecahnya per petugas di sisi layar akan
 * menghasilkan puluhan permintaan untuk satu tampilan.
 */
export function usePICTeknik(filter: PICFilterInput, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: picKeys.scorecards(portal, token, filter),
    queryFn: () =>
      callAPI<PICTeknikResponse>(`${PATH}/pic-teknik?${picParams(filter)}`, {
        token,
        portal,
      }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 60 * 1000,
  })
}

/** Hook tombol ekspor tab KPI PIC Teknik. */
export function useExportPICTeknik() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (filter: PICFilterInput) => {
      const address = `${PATH}/pic-teknik/ekspor?${picParams(filter)}`
      simpanBerkas(await unduhBerkas(address, { token, portal }))
    },
  })
}

/**
 * picParams menyusun parameter penyaring tab KPI PIC Teknik.
 *
 * Dipakai KEDUA permintaan — kartu skor dan ekspor — sehingga berkas yang diunduh tidak
 * dapat menyaring dengan cara yang berbeda dari yang sedang terlihat di layar.
 */
function picParams(filter: PICFilterInput): string {
  const query = new URLSearchParams()
  query.set('lini_bisnis', filter.lini)
  query.set('dari', filter.dari)
  query.set('sampai', filter.sampai)
  return query.toString()
}
