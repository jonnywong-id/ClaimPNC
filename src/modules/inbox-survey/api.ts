import { useMutation, useQuery } from '@tanstack/react-query'

import { HEADER_PORTAL, callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  DaftarResponse,
  JumlahTabResponse,
  KPIResponse,
  KeteranganResponse,
  TahunKPIResponse,
} from './types'

const PATH = '/api/inbox-survey'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan antrean entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal, persis `R-20`.
 *
 * TOKEN ikut pula, dan di layar ini alasannya lebih tajam daripada di layar lain: antreannya
 * disaring NAMA SURVEYOR yang diturunkan dari login, sehingga cache yang bertahan melewati
 * pergantian pengguna akan menampilkan pekerjaan surveyor sebelumnya.
 */
const keys = {
  keterangan: (portal: string | null, token: string | null) =>
    ['inbox-survey', 'keterangan', portal, token] as const,

  jumlahTab: (portal: string | null, token: string | null) =>
    ['inbox-survey', 'jumlah-tab', portal, token] as const,

  daftar: (
    portal: string | null,
    token: string | null,
    tab: string,
    cari: string,
    lewati: number,
  ) => ['inbox-survey', 'daftar', portal, token, tab, cari, lewati] as const,

  kpi: (
    portal: string | null,
    token: string | null,
    status: string,
    tipe: string,
    kuartal: string,
    tahun: string,
  ) => ['inbox-survey', 'kpi', portal, token, status, tipe, kuartal, tahun] as const,

  tahunKPI: (portal: string | null, token: string | null) =>
    ['inbox-survey', 'kpi-tahun', portal, token] as const,
}

/**
 * Hook keterangan layar — judul kolom, judul tab, selisih terencana, dan keterbatasan.
 *
 * # Kenapa judul kolom dan judul tab datang dari server
 *
 * Karena keduanya HASIL PEMBACAAN `Section/InboxSurvey_section-Section.xml`, dan tempat
 * pembacaan itu tercatat adalah backend. Menyalinnya ke layar berarti daftar yang sama hidup
 * di dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Keterbatasan ikut datang dari server supaya ia HILANG DENGAN SENDIRINYA begitu
 * penghalangnya hilang — tanpa menyentuh satu baris pun di layar. Pada modul ini daftar itu
 * panjang: empat kueri tab layar lama hilang dari export, dan dua pemetaan kolom masih
 * menunggu DBA.
 */
export function useKeteranganSurvei() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.keterangan(portal, token),
    queryFn: () => callAPI<KeteranganResponse>(`${PATH}/keterangan`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali halaman berpindah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook jumlah baris ketujuh tab.
 *
 * # Kenapa TERPISAH dari daftar
 *
 * Karena bilah tab tidak berubah saat pengguna berpindah halaman. Menempelkannya pada setiap
 * permintaan daftar akan menjalankan tujuh penjumlahan setiap kali tombol halaman ditekan —
 * tujuh pekerjaan untuk satu yang diminta.
 *
 * Ia juga tidak ikut berubah saat kata kunci diketik: angka pada bilah tab menyatakan isi
 * SELURUH tab, bukan isi hasil pencarian. Itu perilaku Pega, dan menyamakan keduanya akan
 * membuat angka tab berkedip setiap huruf yang diketik.
 */
export function useJumlahTabSurvei() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.jumlahTab(portal, token),
    queryFn: () => callAPI<JumlahTabResponse>(`${PATH}/jumlah-tab`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 0,
  })
}

/**
 * Hook antrean survei milik pengguna yang sedang masuk.
 *
 * Menggantikan `Activity/SetTempLostAdjuster-Act.xml` beserta keempat kueri tabnya — dan
 * keempat kueri itu HILANG dari export, sehingga penyaringnya dipulihkan dari
 * `CountOSLostAdjuster` yang menghitung keranjang yang sama.
 *
 * # Penyaringan dan paginasi dikerjakan SERVER
 *
 * Tabel klaim berisi puluhan juta baris (`D-10`). Menyaring di peramban berarti hasil
 * pencariannya akan BOHONG, karena hanya menyentuh halaman yang sedang terbuka.
 *
 * # Antreannya milik SATU ORANG, dan itu ditentukan server
 *
 * Layar tidak mengirim identitas apa pun — server membacanya dari sesi, lalu
 * menerjemahkannya menjadi nama surveyor lewat `MST_LOGIN_SURVEYOR`. Itu disengaja:
 * identitas yang dikirim layar adalah identitas yang dapat diganti layar.
 */
export function useDaftarSurvei(tab: string, cari: string, lewati: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = new URLSearchParams()
  if (tab) params.set('tab', tab)
  if (cari.trim()) params.set('cari', cari.trim())
  if (lewati > 0) params.set('lewati', String(lewati))
  params.set('batas', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.daftar(portal, token, tab, cari.trim(), lewati),
    queryFn: () => callAPI<DaftarResponse>(`${PATH}?${params.toString()}`, { token, portal }),

    // Tidak ditembak sebelum tab diketahui. Tab bawaan datang dari keterangan layar, dan
    // menembak tanpanya berarti satu permintaan yang hasilnya langsung dibuang.
    enabled: token !== null && portal !== null && tab !== '',

    // Antrean berubah setiap kali sebuah janji survei diselesaikan — termasuk oleh Pega,
    // yang masih memiliki penugasannya selama masa paralel. Data dianggap usang seketika,
    // tetapi tidak ditembak ulang sendiri: pengguna yang menekan Muat ulang.
    staleTime: 0,

    // Halaman sebelumnya dipertahankan selama halaman berikutnya diambil, supaya tabel tidak
    // berkedip menjadi kosong lalu terisi lagi setiap kali nomor halaman atau tab berpindah.
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook ringkasan KPI adjuster.
 *
 * Menggantikan `GetSummaryKPIAdjuster`, `GetSummaryKPIAdjusterALL`, dan
 * `GetSummaryKPIAdjusterKuartal` — ketiganya membaca satu tabel datar
 * `POOLDATA.DETAIL_KPI_ADJUSTER`.
 *
 * # Ia disaring cakupan yang SAMA dengan antrean
 *
 * Tabel itu memuat penilaian SELURUH adjuster. Tanpa penyaring cakupan, tab ini berubah
 * menjadi papan peringkat yang tidak pernah diminta siapa pun.
 *
 * # `enabled` menunggu tab KPI benar-benar dibuka
 *
 * Tab KPI membaca tabel LAIN yang diisi procedure terpisah, dan sebagian lingkungan belum
 * memilikinya. Menembaknya saat pengguna masih di tab INBOX berarti satu galat yang tidak
 * pernah dilihat siapa pun — tetapi tetap tercatat di log.
 */
export function useKPISurvei(f: IsianKPI, aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.kpi(portal, token, f.status, f.tipe, f.kuartal, f.tahun.trim()),
    queryFn: () =>
      callAPI<KPIResponse>(`${PATH}/kpi?${paramKPI(f).toString()}`, { token, portal }),
    enabled: aktif && token !== null && portal !== null,
    staleTime: 0,
    placeholderData: (previous) => previous,
  })
}

/**
 * Hook isi dropdown **Tahun Kuartal**.
 *
 * # Kenapa rute tersendiri, bukan bagian keterangan layar
 *
 * Karena isinya bergantung pada CAKUPAN pemanggil, sedangkan keterangan layar tidak menyentuh
 * basis data sama sekali dan di-cache selamanya. Menyatukannya akan membuat daftar tahun milik
 * pengguna sebelumnya ikut terbawa setelah pergantian pengguna.
 *
 * Ditembak begitu tab KPI dibuka — dropdown yang baru terisi setelah pencarian pertama bukan
 * dropdown, melainkan kotak teks yang menyamar.
 */
export function useTahunKPI(aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.tahunKPI(portal, token),
    queryFn: () => callAPI<TahunKPIResponse>(`${PATH}/kpi/tahun`, { token, portal }),
    enabled: aktif && token !== null && portal !== null,
    staleTime: Infinity,
  })
}

/** IsianKPI adalah keempat kendali panel KPI, sama dengan layar lama. */
export type IsianKPI = {
  status: string
  tipe: string
  kuartal: string
  tahun: string
}

/**
 * paramKPI menyusun parameter kueri dari isian.
 *
 * Dipakai BERSAMA oleh tombol Cari dan tombol Export Data — satu tempat, karena keduanya
 * wajib menghasilkan permintaan yang sama. `ExportKPILoginAdjuster` di Pega pun menjalankan
 * kueri yang sama dengan `GetReportKPIAdjuster`; berkas yang isinya berbeda dari layar tidak
 * dapat dicocokkan siapa pun yang membukanya nanti.
 *
 * Isian kosong TIDAK dikirim. Server membacanya sebagai "belum dipilih" dan menjawab 422 —
 * jawaban yang benar, dan yang berbeda dari "dipilih, tetapi kosong".
 */
export function paramKPI(f: IsianKPI): URLSearchParams {
  const params = new URLSearchParams()
  if (f.status) params.set('status_survei', f.status)
  if (f.tipe) params.set('tipe_report', f.tipe)
  if (f.kuartal) params.set('kuartal', f.kuartal)
  if (f.tahun.trim()) params.set('tahun', f.tahun.trim())
  return params
}

/**
 * Hook tombol **Export Data** pada panel KPI.
 *
 * # Kenapa lewat `fetch` dan `blob`, bukan `<a href>` biasa
 *
 * Karena rutenya menuntut dua header — token sesi dan portal aktif — dan tautan biasa tidak
 * membawa keduanya. Tanpa header portal, permintaannya DITOLAK, bukan dilayani portal utama
 * sebagai cadangan (`R-20`).
 *
 * # Kenapa memakai isian yang sama dengan tombol Cari
 *
 * `Activity/ExportKPILoginAdjuster-Act.xml` menjalankan kueri yang sama dengan tombol Cari,
 * lalu melewatkan hasilnya ke `pxConvertResultsToCSV`. Berkasnya karena itu berisi persis
 * yang terlihat di layar — dan isian yang berbeda akan menghasilkan berkas yang tidak dapat
 * dicocokkan dengan apa pun.
 */
export function useEksporKPI() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (f: IsianKPI) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const response = await fetch(`${PATH}/kpi/ekspor?${paramKPI(f).toString()}`, {
        headers: header,
      })
      if (!response.ok) {
        // Galat dijawab sebagai JSON selama satu byte pun badan belum terkirim; sesudah itu
        // tidak bisa lagi. Yang dibaca di sini kasus pertama.
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas ekspor tidak dapat diambil.')
      }

      simpanBerkas(await response.blob(), namaBerkas(response) ?? 'kpi-adjuster.csv')
    },
  })
}

/** Nama berkas dari header `Content-Disposition`, bila ada. */
function namaBerkas(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  return /filename="([^"]+)"/.exec(disposition)?.[1] ?? null
}

/**
 * simpanBerkas menyimpan blob lewat tautan sementara.
 *
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban sampai
 * tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor menumpuk memori
 * yang tidak pernah dilepas.
 */
function simpanBerkas(blob: Blob, nama: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = nama
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
