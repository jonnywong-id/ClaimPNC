/**
 * Pemformatan dan pembacaan nilai uang di sisi layar.
 *
 * # Berkas ini memuat DUA perwakilan nilai uang, dan itu disengaja
 *
 * Keduanya lahir dari dua modul yang menerima bentuk data berbeda dari server, dan
 * keduanya masih dipakai:
 *
 *	formatMoney / parseMoney / remainder    ANGKA rupiah utuh    — Master Recovery
 *	formatRupiah / parseRupiah              TEKS desimal kanonik — Ambang Komite, Inbox Komite
 *
 * Menyatukannya menuntut salah satu modul mengubah kontrak API-nya, dan itu keputusan
 * tersendiri — bukan efek samping penggabungan cabang. Sampai keputusan itu diambil,
 * PILIH menurut bentuk yang dikirim endpoint yang Anda panggil:
 *
 *	angka JSON     -> formatMoney
 *	teks "123.00"  -> formatRupiah
 *
 * Memakai yang salah TIDAK menghasilkan galat tipe pada nilai tertentu — `formatMoney`
 * menerima number dan `formatRupiah` menerima string — tetapi menghasilkan tampilan yang
 * keliru begitu bentuknya tidak cocok.
 *
 * # Kenapa berkas ini ada sama sekali
 *
 * `docs/Steering/08-TECHNICAL-STRATEGY.md` §3 aturan 4 menetapkan seluruh pemformatan angka
 * dan mata uang melewati satu berkas bersama. Di sistem lama pemformatan dikerjakan
 * `TO_CHAR` yang tersebar di 411 tempat, dan akibatnya angka tampil berbeda-beda antarlayar
 * tanpa ada yang berwenang menyeragamkannya.
 */


/**
 * formatMoney menampilkan nilai uang dengan pemisah ribuan Indonesia.
 *
 * Lambang "Rp" TIDAK disertakan: label kolom sudah menyebutkannya, dan mengulanginya di
 * setiap sel membuat tabel bernilai banyak menjadi sulit dibaca. Layar yang membutuhkannya
 * menuliskannya sendiri di label.
 *
 * Nilai negatif ditampilkan apa adanya dengan tanda minus. Itu disengaja: sisa klaim
 * memang dapat negatif ketika pembayaran melampaui nilai klaim, dan menyembunyikannya
 * akan menyembunyikan kelebihan bayar dari orang yang membaca layar.
 */
export function formatMoney(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(value)
}

/**
 * parseMoney membaca nilai uang dari isian yang diketik orang.
 *
 * Ia lapang terhadap BENTUK dan ketat terhadap ISI, sama seperti pembacanya di server:
 * pemisah ribuan berupa titik maupun koma dibuang — keduanya muncul di layar lama karena
 * formatnya tidak pernah diseragamkan — sedangkan yang bukan bilangan bulat ditolak.
 *
 * Mengembalikan `null` bila isiannya tidak dapat dibaca sebagai rupiah utuh. Pemanggil
 * yang memutuskan apa artinya: pada isian wajib ia pelanggaran, pada isian opsional ia
 * nol.
 */
export function parseMoney(text: string): number | null {
  const clean = text.trim().replaceAll('.', '').replaceAll(',', '')
  if (clean === '') return null
  // Regex, bukan Number(): Number("") bernilai 0, Number("1e3") bernilai 1000, dan
  // Number(" 12 ") bernilai 12 — ketiganya menerima bentuk yang tidak pernah diketik
  // orang sebagai nilai uang, dan server akan menolaknya.
  if (!/^-?\d+$/.test(clean)) return null

  const value = Number(clean)
  return Number.isSafeInteger(value) ? value : null
}

/**
 * remainder menghitung Sisa Klaim, meniru `masterrecovery.Remainder` di server PERSIS.
 *
 * # Kenapa aturannya ada di DUA tempat
 *
 * Duplikasi yang DISENGAJA, bukan kelalaian. Yang di sini memperlihatkan angkanya seketika
 * saat petugas mengetik — layar lama pun demikian, lewat aksi `HitungSisaKlaimRecovery`
 * pada setiap perubahan isian. Yang di server adalah yang MENGIKAT, dan nilainya tidak
 * pernah diterima dari badan permintaan.
 *
 * Menghapus yang di sini berarti setiap ketikan menunggu perjalanan jaringan; menghapus
 * yang di server berarti mempercayai angka yang dikirim klien untuk nilai uang.
 *
 * # Cabang kedua yang tampak keliru, dan kenapa ia dipertahankan
 *
 * Ketika ada pembayaran sebelumnya, pembayaran batch berjalan TIDAK mengurangi sisa.
 * Dibaca sekilas itu tampak cacat — dan ketiga baris produksi membuktikan itu memang
 * perilakunya. `P-5` menetapkan perilaku dipertahankan lebih dulu, dan aturan ini tidak
 * ada di daftar 13 perbaikan eksplisit `D-49`.
 *
 * Bila rumus ini berubah, `masterrecovery.Remainder` di backend WAJIB ikut berubah.
 */
export function remainder(claimAmount: number, previousPayment: number, payment: number): number {
  if (previousPayment === 0) return claimAmount - payment
  return claimAmount - previousPayment
}

/* ========================================================================== */
/* Perwakilan kedua: TEKS desimal kanonik — dipakai Ambang Komite dan Inbox   */
/* Komite. Server mengirimnya sebagai teks, bukan angka JSON, karena angka    */
/* JSON adalah floating point ganda di hampir seluruh peramban.               */
/* ========================================================================== */


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
