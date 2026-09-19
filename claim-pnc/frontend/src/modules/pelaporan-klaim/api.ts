import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type { ClaimReportListResponse, ClaimReportResponse } from '@/api/types'
import { useSession } from '@/app/session'

const PATH = '/api/pelaporan-klaim'

/** Penyaring daftar. Kosong berarti seluruh tahap. */
export type ReportFilter = {
  stage?: string
  search?: string
  offset?: number
}

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Penyaring ikut menjadi bagian kunci: tab "belum ditransfer" dan tab "ditolak" adalah dua
 * hasil berbeda, dan menyimpannya di bawah satu kunci akan membuat perpindahan tab
 * menampilkan isi tab sebelumnya.
 */
const keys = {
  all: (token: string | null) => ['pelaporan-klaim', token] as const,
  list: (token: string | null, f: ReportFilter) =>
    ['pelaporan-klaim', token, f.stage ?? '', f.search ?? '', f.offset ?? 0] as const,
}

function buildPath(f: ReportFilter): string {
  const params = new URLSearchParams()
  if (f.stage) params.set('tahap', f.stage)
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.offset) params.set('lewati', String(f.offset))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/**
 * Hook daftar laporan klaim.
 *
 * Menggantikan ketiga kueri inbox sistem lama sekaligus — `ViewTableBrowseRCVInProcess`,
 * `ViewTableBrowseRCVAcc`, dan `ViewTableBrowseRCVReject` — yang di sana menjadi tiga rule
 * terpisah karena penyaringnya dirangkai ke dalam teks SQL.
 *
 * # Kenapa penyaringan dikerjakan di SERVER, berbeda dari layar master
 *
 * Master status berisi 33 baris yang bertambah beberapa per tahun, sehingga menyaringnya
 * di peramban tidak merugikan siapa pun. Laporan klaim bertambah RIBUAN PER BULAN
 * (`D-10`) dan tidak pernah berkurang; menyaringnya di peramban berarti mengirim seluruh
 * riwayat laporan ke setiap layar yang dibuka.
 */
export function useReportList(filter: ReportFilter) {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: keys.list(token, filter),
    queryFn: () => callAPI<ClaimReportListResponse>(buildPath(filter), { token }),
    enabled: token !== null,

    // Hasil tab sebelumnya ditahan selama tab baru dimuat, alih-alih layar berkedip
    // menjadi kosong lalu terisi lagi. Lencana jumlah tetap terbaca selama perpindahan.
    placeholderData: (previous) => previous,

    // Laporan baru dapat masuk kapan saja dari petugas lain, jadi cache-nya pendek —
    // berbeda dari master yang nyaris tidak berubah dalam satu sesi kerja.
    staleTime: 30 * 1000,
  })
}

/** Isian form catat dan ubah. Bentuknya sama persis dengan SaveRequest di backend. */
export type ReportFormValues = {
  nama_pelapor: string
  email_pengirim: string
  telepon_pengirim: string
  nama_kurir: string
  subjek_email: string

  nomor_polis: string
  nama_tertanggung: string
  email_tertanggung: string
  kode_bisnis: string
  group_panel: string
  nomor_referensi: string

  tanggal_kejadian: string
  lokasi_kejadian: string
  kronologi: string
  rincian_kerusakan: string
  sim_pengendara: string
  nilai_estimasi: string
  tipe_klaim: string

  jumlah_dokumen: number
  tanggal_terima_dokumen: string

  alasan_belum_transfer: string
  catatan_belum_registrasi: string
}

/**
 * Hook simpan — mencatat maupun mengubah.
 *
 * Keduanya disatukan karena formnya memang satu. Perbedaannya hanya pada metode dan
 * jalur, dan itu satu baris.
 *
 * Nomor TIDAK pernah dikirim di badan permintaan. Pada pencatatan ia dibuat server; pada
 * pengubahan ia berada di jalur URL.
 */
export function useSaveReport() {
  const token = useSession((state) => state.token)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ number, values }: { number?: string; values: ReportFormValues }) =>
      callAPI<ClaimReportResponse>(
        number ? `${PATH}/${encodeURIComponent(number)}` : PATH,
        {
          metode: number ? 'PUT' : 'POST',
          body: values,
          token,
        },
      ),
    onSuccess: () => {
      // SELURUH daftar dimuat ulang, bukan hanya tab yang sedang terbuka: laporan baru
      // mengubah lencana setiap tab, dan menyunting cache satu tab akan membuat angkanya
      // tidak cocok dengan isi tab lain.
      void client.invalidateQueries({ queryKey: keys.all(token) })
    },
  })
}

/**
 * Hook transfer laporan ke ASM pusat.
 *
 * Menggantikan pengisian `ReceiveDocument.StatusLock` dan `DateOfSendASM`, yang di sistem
 * lama terjadi sebagai efek samping penyimpanan layar — tanpa langkah tersendiri dan tanpa
 * apa pun yang mencegah laporan ditransfer dua kali.
 */
export function useTransferReport() {
  const token = useSession((state) => state.token)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (number: string) =>
      callAPI<ClaimReportResponse>(`${PATH}/${encodeURIComponent(number)}/transfer`, {
        metode: 'POST',
        token,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all(token) })
    },
  })
}

/**
 * Hook penautan laporan ke klaim.
 *
 * Menggantikan `Activity/UpdateRCVCase-Act.xml`, yang di sistem lama dijalankan dari sisi
 * KLAIM. Modul `B-2` Input Register kelak memanggil jalur yang sama; sampai ia ada, aksi
 * ini yang menandai laporan yang klaimnya sudah dibuat di Pega supaya tidak tertinggal di
 * tab "belum diregistrasi" selamanya.
 */
export function useLinkClaim() {
  const token = useSession((state) => state.token)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ number, claimNumber }: { number: string; claimNumber: string }) =>
      callAPI<ClaimReportResponse>(`${PATH}/${encodeURIComponent(number)}/klaim`, {
        metode: 'POST',
        body: { nomor_klaim: claimNumber },
        token,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all(token) })
    },
  })
}
