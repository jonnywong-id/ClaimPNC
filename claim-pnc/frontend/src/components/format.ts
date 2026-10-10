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
 * formatDateTimeWIB menulis waktu RFC3339 sebagai `dd/MM/yyyy H:mm` menurut WIB — bentuk
 * DateTime Pega pada isian baca saja, mis. Tanggal Cetak LOD "30/09/2026 7:59".
 */
export function formatDateTimeWIB(iso: string | undefined): string {
  if (!iso) return ''
  const t = new Date(iso)
  if (Number.isNaN(t.getTime())) return iso
  const wib = new Date(t.getTime() + 7 * 60 * 60_000)
  const day = String(wib.getUTCDate()).padStart(2, '0')
  const month = String(wib.getUTCMonth() + 1).padStart(2, '0')
  const minute = String(wib.getUTCMinutes()).padStart(2, '0')
  return `${day}/${month}/${wib.getUTCFullYear()} ${wib.getUTCHours()}:${minute}`
}

/**
 * formatPegaDateTime menulis waktu menjadi `dd/MM/yy H:mm` — bentuk kolom tanggal pada
 * GRID Pega, mis. `28/09/26 16:58` dan `28/09/26 9:58`.
 *
 * # Jamnya TIDAK diberi nol di depan
 *
 * Grid Pega menulis `9:58`, bukan `09:58` — terbaca pada grid Detail Komunikasi Cabang yang
 * memuat jam satu digit. Sel grid pada layar daftar tidak dapat membuktikannya sendiri
 * karena jamnya kebetulan dua digit.
 *
 * Menitnya TETAP ber-nol (`2:46`, bukan `2:4`); itu bukan ketidakkonsistenan melainkan
 * bentuk jam yang lazim.
 *
 * # Kenapa bentuknya berbeda dari formatDateTimeWIB di atas
 *
 * Keduanya memang berbeda di layar lama: isian baca saja pada form menulis tahun EMPAT
 * digit (`30/09/2026 7:59`), sedangkan sel grid menulis DUA digit. Menyatukannya akan
 * membuat salah satu layar berbeda dari acuannya, dan `D-13` menuntut keduanya mengikuti
 * layarnya masing-masing.
 *
 * # Zona waktu: hanya digeser bila sumbernya MENYEBUTKAN zona
 *
 * Nilai dari basis data datang sebagai RFC3339 lengkap dengan offset
 * (`2026-09-28T16:58:56+07:00`), dan itu sebuah TITIK WAKTU yang harus diterjemahkan ke
 * WIB. Nilai tanpa zona (`2026-09-28 16:58`) BUKAN titik waktu — ia sudah waktu dinding,
 * dan menggesernya tujuh jam akan menggeser angkanya menjadi salah.
 *
 * Kelas kesalahan itu persis yang melahirkan ratusan penyesuaian tujuh jam di sistem lama.
 * Ia tidak akan lahir kembali di sini: yang menentukan digeser atau tidak adalah ADA
 * TIDAKNYA zona pada teksnya, bukan tebakan.
 *
 * Teks yang tidak dapat dibaca dikembalikan APA ADANYA, bukan menjadi tanda pisah — nilai
 * mentah yang terbaca aneh masih dapat ditelusuri, sedangkan `—` menghapus jejaknya.
 */
export function formatPegaDateTime(value: string): string {
  const text = value.trim()
  if (text === '') return ''

  // Dipecah sendiri, bukan lewat `new Date(...)`, supaya nilai tanpa zona tidak ikut
  // ditafsirkan peramban sebagai waktu lokal atau UTC — dua tafsir yang berbeda tujuh jam.
  const parts =
    /^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2})(?::\d{2}(?:\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/.exec(
      text,
    )
  if (!parts) return text

  const [, year, month, day, hour, minute, zone] = parts

  // Tanpa jam, yang ada hanyalah tanggalnya. Menambahkan "00:00" akan mengarang ketelitian
  // yang tidak ada di sumbernya.
  if (hour === undefined || minute === undefined) {
    return `${day}/${month}/${year?.slice(2)}`
  }

  if (zone === undefined) {
    // Nol di depan DIBUANG dari jamnya, termasuk pada nilai yang sumbernya sudah ber-nol:
    // yang menentukan bentuknya adalah layar, bukan cara basis data menuliskannya.
    return `${day}/${month}/${year?.slice(2)} ${Number(hour)}:${minute}`
  }

  const instant = new Date(text)
  if (Number.isNaN(instant.getTime())) return text

  const wib = new Date(instant.getTime() + 7 * 60 * 60_000)
  const pad = (n: number) => String(n).padStart(2, '0')

  return (
    `${pad(wib.getUTCDate())}/${pad(wib.getUTCMonth() + 1)}/` +
    `${String(wib.getUTCFullYear()).slice(2)} ` +
    `${wib.getUTCHours()}:${pad(wib.getUTCMinutes())}`
  )
}

/**
 * formatPegaElapsed menuliskan sebuah titik waktu sebagai LAMANYA sampai sekarang —
 * "21 hours ago", "1 day 1 hour ago", "1 year 5 months ago".
 *
 * # Apa yang ditiru
 *
 * `pyDateTimeFormat = DateTime-Frame`, format waktu relatif **bawaan Pega**. Ia bukan rule
 * buatan sendiri, sehingga penyusunnya tidak ada di export mana pun — kodenya ada di
 * platform, yang tidak ikut dalam export rule aplikasi. Yang dapat dibaca dari export
 * hanyalah KOLOM MANA yang memakainya; bunyinya harus diambil dari layar yang berjalan.
 *
 * Di Inbox RCL/PUCL ia dipakai tepat pada satu kolom: "Lama Klaim"
 * (`Section/InboxCetakSuratPUCLRCL_Section-Section.xml` — satu-satunya `DateTime-Frame` di
 * seluruh berkas itu). Belasan section Inbox lain memakai format yang sama.
 *
 * # Bentuknya, dan mana yang benar-benar TERAMATI
 *
 * Tangkapan layar Pega yang berjalan pada 2026-10-10 memperlihatkan dua baris sekaligus,
 * dan keduanya mengikat bentuk di bawah:
 *
 *	09/10/26 17:05  →  "21 hours ago"        selisih 21 jam 15 menit
 *	09/10/26 12:45  →  "1 day 1 hour ago"    selisih 1 hari 1 jam 35 menit
 *
 * Dari keduanya tiga hal dapat dipastikan:
 *
 *   - Cabang HARI menuliskan DUA satuan — hari beserta sisa jamnya. Bukan "1 day ago".
 *   - Cabang JAM menuliskan SATU satuan saja; sisa menitnya dibuang, tidak dibulatkan.
 *   - Bentuk jamaknya BENAR per satuan: "1 day 1 hour" bersisian dengan "21 hours".
 *
 * Cabang tahun mengikuti pengamatan terpisah di layar Post Audit — "1 year 5 months ago"
 * (lihat `inboxcompliance.FormatElapsed` di backend, yang menulis ulang format yang sama
 * untuk kolom OutStanding). Cabang bulan dan menit masih REKONSTRUKSI: belum ada baris
 * yang cukup baru di layar Pega untuk membandingkannya.
 *
 * # Kenapa di sini, bukan di backend seperti kolom Aging Compliance
 *
 * Karena nilai yang dikirim modul ini memang sebuah TANGGAL, bukan durasi — kolom
 * `LAMAKLAIM_1` bertipe `TIMESTAMP(6)`, dan judul "Lama Klaim" menyesatkan sejak di Pega.
 * Membiarkannya tanggal pada kontrak API menjaga dua hal: berkas ekspornya tetap memuat
 * tanggal yang dapat diurutkan Excel, dan teks "21 hours ago" tidak membeku pada saat
 * jawabannya dibuat — ia dihitung ulang setiap kali layar digambar, persis seperti di Pega.
 *
 * # Zona waktu
 *
 * Nilai tanpa zona diperlakukan sebagai jam dinding WIB, sama seperti `formatPegaDateTime`
 * — backend sudah menggesernya ke WIB sebelum mengirim. Nilai BER-zona dibaca apa adanya
 * sebagai titik waktu. Yang menentukan digeser atau tidak adalah ada tidaknya zona pada
 * teksnya, bukan tebakan: kelas kesalahan itulah yang melahirkan ratusan penyesuaian tujuh
 * jam di sistem lama.
 *
 * Teks yang tidak dapat dibaca dikembalikan APA ADANYA — nilai mentah yang terbaca aneh
 * masih dapat ditelusuri, sedangkan teks kosong menghapus jejaknya. Waktu di MASA DEPAN
 * menghasilkan teks kosong, bukan "0 minutes ago" yang menyatakan hal yang tidak benar.
 */
export function formatPegaElapsed(value: string, now: Date = new Date()): string {
  const text = value.trim()
  if (text === '') return ''

  const parts =
    /^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2})(?::(\d{2})(?:\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/.exec(
      text,
    )
  if (!parts) return text

  const [, year, month, day, hour = '00', minute = '00', second = '00', zone] = parts

  // Titik waktu sumbernya. Tanpa zona, angkanya adalah jam dinding WIB — maka instannya
  // tujuh jam lebih awal daripada angka yang sama dibaca sebagai UTC.
  const start =
    zone === undefined
      ? Date.UTC(
          Number(year),
          Number(month) - 1,
          Number(day),
          Number(hour),
          Number(minute),
          Number(second),
        ) -
        7 * 60 * 60_000
      : new Date(text).getTime()

  if (Number.isNaN(start)) return text

  const elapsed = now.getTime() - start
  if (elapsed < 0) return ''

  // Tahun dan bulan dihitung secara KALENDER, bukan dengan membagi selisih milidetik.
  // Membaginya akan memakai "bulan" sepanjang 30 hari yang tidak ada di kalender mana pun,
  // dan hasilnya meleset makin jauh seiring rentangnya memanjang.
  const a = wibParts(start)
  const b = wibParts(now.getTime())

  let months = (b.year - a.year) * 12 + (b.month - a.month)
  if (b.day < a.day || (b.day === a.day && b.timeOfDay < a.timeOfDay)) months--

  if (months >= 12) {
    const years = Math.floor(months / 12)
    const restMonth = months % 12

    // Sisa nol tidak ditulis. "1 year 0 months ago" bukan bentuk yang ditulis pemformat
    // waktu relatif mana pun, dan ia hanya muncul tepat pada hari ulang tahun sebuah baris.
    return restMonth === 0
      ? `${plural(years, 'year')} ago`
      : `${plural(years, 'year')} ${plural(restMonth, 'month')} ago`
  }

  if (months >= 1) return `${plural(months, 'month')} ago`

  const hours = Math.floor(elapsed / 3_600_000)

  if (hours >= 24) {
    const days = Math.floor(hours / 24)
    const restHour = hours % 24

    return restHour === 0
      ? `${plural(days, 'day')} ago`
      : `${plural(days, 'day')} ${plural(restHour, 'hour')} ago`
  }

  if (hours >= 1) return `${plural(hours, 'hour')} ago`

  return `${plural(Math.floor(elapsed / 60_000), 'minute')} ago`
}

/**
 * wibParts memecah sebuah instan menjadi bagian kalender WIB.
 *
 * Pergeserannya dilakukan di satu tempat ini, bukan di setiap pemanggil — sama alasannya
 * dengan seam Clock di backend (`F-5`).
 */
function wibParts(epochMs: number): {
  year: number
  month: number
  day: number
  timeOfDay: number
} {
  const wib = new Date(epochMs + 7 * 60 * 60_000)
  return {
    year: wib.getUTCFullYear(),
    month: wib.getUTCMonth(),
    day: wib.getUTCDate(),
    timeOfDay: wib.getUTCHours() * 60 + wib.getUTCMinutes(),
  }
}

/**
 * plural menuliskan sebuah jumlah beserta satuannya dalam bentuk Inggris yang benar.
 *
 * Kelima satuan yang dipakai — minute, hour, day, month, year — seluruhnya beraturan,
 * sehingga menambahkan "s" sudah cukup. Tidak ada gunanya menarik pustaka pluralisasi
 * untuk lima kata.
 *
 * Teksnya berbahasa Inggris karena itulah yang selama ini dibaca petugas di kolom ini
 * (`D-13`), bukan karena bahasa Indonesia dihindari.
 */
function plural(count: number, unit: string): string {
  return count === 1 ? `1 ${unit}` : `${count} ${unit}s`
}

/**
 * rupiahToCents membaca angka yang diketik pengguna menjadi sen.
 *
 * Pemisah ribuan titik dan desimal koma diterima, karena itu yang diketik orang di
 * Indonesia. Nilai yang tidak dapat dibaca menghasilkan NaN — dan pemanggil yang
 * menolaknya lebih baik daripada nol yang tampak sah.
 */
export function rupiahToCents(text: string): number {
  const clean = text.trim().replaceAll('.', '').replace(',', '.')
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
