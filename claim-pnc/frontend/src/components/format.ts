/**
 * Pemformatan angka dan tanggal, di satu tempat.
 *
 * # Kenapa ini bukan sekadar kerapian
 *
 * Kriteria penerimaan `TKT-U4-001` menuntut angka uang dan tanggal di layar transaksi
 * TAMPIL SAMA dengan yang muncul di laporan. Selama pemformatan ditulis ulang di setiap
 * layar, "sama" adalah harapan, bukan sifat. Modul baku `TKT-U2-004` yang akan memiliki
 * fungsi-fungsi ini belum ada; sampai ia ada, tempatnya di sini — satu berkas yang
 * dipakai bersama, bukan satu salinan per layar.
 */

/**
 * Uang dikirim backend sebagai bilangan bulat SEN, tidak pernah sebagai pecahan.
 *
 * `ADR-0016` menuntut nilai uang presisi penuh dan pembulatan hanya saat ditampilkan.
 * Pecahan biner tidak dapat mewakili rupiah dengan tepat — `0.1 + 0.2 !== 0.3` berlaku
 * di JavaScript persis seperti di Go.
 */
export type Cents = number

/** Persentase dikirim backend sebagai persen dikali 10.000. 100% = 1000000. */
export type PercentE4 = number

const rupiahFormatter = new Intl.NumberFormat('id-ID', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

/** formatRupiah menuliskan nilai sen sebagai rupiah: `Rp 1.000.000,00`. */
export function formatRupiah(sen: Cents): string {
  return `Rp ${rupiahFormatter.format(sen / 100)}`
}

/** formatPercent menuliskan persentase empat desimal tanpa nol di belakang: `99,9999%`. */
export function formatPercent(value: PercentE4): string {
  const num = (value / 10_000).toFixed(4).replace(/\.?0+$/, '')
  return `${num.replace('.', ',')}%`
}

const monthNames = [
  'Januari', 'Februari', 'Maret', 'April', 'Mei', 'Juni',
  'Juli', 'Agustus', 'September', 'Oktober', 'November', 'Desember',
]

/**
 * formatDate mengubah `YYYY-MM-DD` menjadi `1 Juni 2026`.
 *
 * Ia sengaja TIDAK memakai `new Date(teks)`: peramban menafsirkan `YYYY-MM-DD` sebagai
 * tengah malam UTC, sehingga pengguna di WIB melihat tanggal yang mundur sehari. Kelas
 * kesalahan itu tepat yang melahirkan 101 penyesuaian tujuh jam di sistem lama, dan ia
 * tidak akan lahir kembali di sini.
 */
export function formatDate(iso: string): string {
  if (!iso) return '—'
  const part = iso.split('-')
  if (part.length !== 3) return iso

  const year = Number(part[0])
  const month = Number(part[1])
  const day = Number(part[2])
  if (!year || !month || !day || month < 1 || month > 12) return iso

  return `${day} ${monthNames[month - 1]} ${year}`
}

/** todayWIB mengembalikan tanggal hari ini menurut WIB dalam bentuk `YYYY-MM-DD`. */
export function todayWIB(): string {
  const now = new Date()
  const wib = new Date(now.getTime() + (7 * 60 + now.getTimezoneOffset()) * 60_000)
  const month = String(wib.getMonth() + 1).padStart(2, '0')
  const day = String(wib.getDate()).padStart(2, '0')
  return `${wib.getFullYear()}-${month}-${day}`
}

/**
 * rupiahToCents membaca angka yang diketik pengguna menjadi sen.
 *
 * Pemisah ribuan titik dan desimal koma diterima, karena itu yang diketik orang di
 * Indonesia. Nilai yang tidak dapat dibaca menghasilkan NaN — dan pemanggil yang
 * menolaknya lebih baik daripada nol yang tampak sah.
 */
export function rupiahToCents(text: string): number {
  const clean = text.trim().replace(/\./g, '').replace(',', '.')
  if (clean === '') return 0
  const num = Number(clean)
  if (Number.isNaN(num)) return Number.NaN
  return Math.round(num * 100)
}

/** centsToRupiah menuliskan sen menjadi teks yang dapat disunting kembali. */
export function centsToRupiah(sen: Cents): string {
  if (!sen) return ''
  return (sen / 100).toFixed(2).replace('.', ',')
}
