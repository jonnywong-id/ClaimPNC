/**
 * Pemformatan dan pembacaan nilai uang di sisi layar.
 *
 * # Pembagian tugas dengan server
 *
 * Server mengirim nilai uang sebagai **teks desimal kanonik** — `"50000001.00"` — bukan
 * angka JSON. Angka JSON adalah floating point ganda di hampir seluruh peramban, dan
 * mengirim nilai uang lewatnya berarti menyerahkan ketepatannya kepada pembulatan biner.
 *
 * Berkas ini mengubah bentuk kanonik itu menjadi bentuk yang enak dibaca
 * (`Rp 50.000.001`) dan sebaliknya. Ia **tidak pernah menghitung**: seluruh perbandingan
 * dan akumulasi terjadi di server, tempat nilainya tersimpan sebagai bilangan bulat.
 *
 * # Kenapa tidak memakai Number
 *
 * `Number('100000000000.00')` masih tepat, tetapi kebiasaan mengubah uang menjadi
 * `number` adalah awal dari kesalahan yang sulit dilihat — satu penjumlahan yang
 * terselip di layar sudah cukup membuat nilai bergeser sen demi sen. Pemformatan di sini
 * karena itu bekerja pada **teksnya**, bukan pada angkanya.
 */

/** Pemisah ribuan dan desimal mengikuti kebiasaan Indonesia: titik dan koma. */
const THOUSANDS_SEPARATOR = '.'
const DECIMAL_SEPARATOR = ','

/**
 * Mengubah bentuk kanonik dari server menjadi teks yang enak dibaca.
 *
 * `"50000001.00"` menjadi `"Rp 50.000.001"`, dan `"12.50"` menjadi `"Rp 12,50"`.
 *
 * Sen yang bernilai nol **dibuang**, karena seluruh isi master ambang komite berupa
 * rupiah bulat dan menampilkan `,00` di setiap baris hanya menambah keramaian tanpa
 * menambah keterangan. Sen yang tidak nol tetap ditampilkan — membuangnya akan
 * menyembunyikan selisih yang justru menentukan.
 */
export function formatRupiah(canonical: string, options?: { withoutSymbol?: boolean }): string {
  const trimmed = (canonical ?? '').trim()
  if (trimmed === '') return '—'

  const negative = trimmed.startsWith('-')
  const unsigned = negative ? trimmed.slice(1) : trimmed

  const [whole = '0', fraction = ''] = unsigned.split('.')
  if (!/^\d+$/.test(whole)) return canonical

  let formatted = groupThousands(whole)
  if (fraction !== '' && /[1-9]/.test(fraction)) {
    formatted += DECIMAL_SEPARATOR + fraction
  }
  if (!options?.withoutSymbol) formatted = `Rp ${formatted}`

  // Tanda minus di depan lambang, bukan di antara lambang dan angka: "-Rp 7,25" terbaca
  // sebagai satu nilai negatif, sedangkan "Rp -7,25" sekilas terbaca seperti salah ketik.
  return negative ? `-${formatted}` : formatted
}

/**
 * Mengubah apa yang diketik pengguna menjadi bentuk kanonik yang dipahami server.
 *
 * Pengguna boleh mengetik `50.000.000`, `50000000`, atau `50.000.000,50` — ketiganya
 * diterima, karena itulah cara orang menuliskan rupiah. Server sendiri **menolak**
 * pemisah ribuan, dan justru itu alasan pengubahan ini dikerjakan di sini: artinya
 * berbeda antar bahasa, sehingga penafsirannya harus terjadi di tempat yang tahu
 * bahasanya — layar — bukan di API yang melayani siapa saja.
 *
 * Mengembalikan `null` bila yang diketik bukan nilai uang yang sah.
 */
export function parseRupiah(input: string): string | null {
  let cleaned = (input ?? '').trim()
  if (cleaned === '') return null

  cleaned = cleaned.replace(/^Rp\s*/i, '')
  // Spasi dan spasi-tak-terputus kadang ikut saat pengguna menyalin dari layar lain.
  cleaned = cleaned.replace(/[\s ]/g, '')

  const negative = cleaned.startsWith('-')
  if (negative || cleaned.startsWith('+')) cleaned = cleaned.slice(1)

  // Pemisah ribuan dibuang; koma menjadi titik desimal.
  cleaned = cleaned.split(THOUSANDS_SEPARATOR).join('')
  cleaned = cleaned.replace(DECIMAL_SEPARATOR, '.')

  if (!/^\d+(\.\d{1,2})?$/.test(cleaned)) return null

  const [whole, fraction = ''] = cleaned.split('.')
  const canonical = `${whole}.${(fraction + '00').slice(0, 2)}`
  return negative ? `-${canonical}` : canonical
}

function groupThousands(digits: string): string {
  // Dikerjakan dari belakang supaya kelompok terakhir yang boleh kurang dari tiga digit,
  // bukan yang pertama.
  let result = ''
  for (let i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 === 0) result += THOUSANDS_SEPARATOR
    result += digits[i]
  }
  return result
}
