import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import type { ErrorTone } from '@/components/ErrorMessage'

/**
 * Kode galat khusus modul ini.
 *
 * Ia TIDAK ditambahkan ke `api/types.ts`: berkas itu memuat kode yang dipakai bersama,
 * dan kode yang hanya berarti di satu layar akan menumpuk di sana tanpa ada yang
 * membacanya. Nilainya harus sama persis dengan konstanta di
 * `internal/inboxpladlapredla/http/errors.go`.
 */
export const KodeGalat = {
  /** Daftar Pre DLA tidak punya grid rincian — di Pega pun tidak. */
  tanpaGridRincian: 'rincian_tidak_ada_di_daftar_ini',

  /** Klaimnya tidak ada pada entitas yang sedang dipilih. */
  klaimTidakAda: 'klaim_tidak_ditemukan',

  /** Tombol yang belum dibangun. */
  belumTersedia: 'belum_tersedia',
} as const

/**
 * formatTanggal menuliskan tanggal dari server sebagai `dd/MM/yyyy`, dan
 * `dd/MM/yyyy H:mm` bila sumbernya membawa jam.
 *
 * # Bentuknya MENGIKUTI grid layar lama (`D-13`)
 *
 * Begitulah ketiga grid Pega menuliskannya — `24/01/2021` pada kolom REGISTER DATE, dan
 * `25/01/2021 3:02` pada kolom DATE OF LOSS yang sumbernya memang bertanda waktu. Sampai
 * 2026-10-10 fungsi ini menulis `24 Jan 2021`, bentuk yang dikarang.
 *
 * Jam TANPA nol di depan (`3:02`, bukan `03:02`), menitnya tetap ber-nol — itu bentuk jam
 * Pega, dan `formatPegaDateTime` di `components/format.ts` memegang aturan yang sama.
 * Yang membedakan keduanya hanyalah lebar tahun: grid ini menulis empat digit, grid yang
 * dibaca `formatPegaDateTime` menulis dua. Keduanya dibiarkan berbeda karena layarnya
 * memang berbeda — menyatukannya akan membuat salah satu menyimpang dari acuannya.
 *
 * # Konversi zona waktu terjadi DI SINI
 *
 * Di tempat waktu ditampilkan — satu-satunya tempat yang boleh melakukannya
 * (`08-TECHNICAL-STRATEGY.md` §4.4). Tidak ada penambahan 7 jam manual di mana pun, dan
 * pergeserannya hanya dilakukan pada nilai yang MENYEBUTKAN zonanya.
 *
 * Nilai KOSONG menghasilkan tanda hubung, bukan tanggal apa pun. Itu keadaan yang nyata
 * di layar ini: kolom tanggal advice memang kadang kosong, dan menggambar tanggal di sana
 * akan menampilkan sesuatu yang tidak pernah tercatat.
 */
export function formatTanggal(nilai: string): string {
  const bersih = nilai.trim()
  if (bersih === '') return '—'

  // Teksnya dipecah sendiri, bukan diserahkan ke `new Date(...)`.
  //
  // Nilai dari basis data dapat datang DENGAN zona (`2026-01-05T08:00:00+07:00`) atau
  // TANPA (`2026-01-05`, `2026-01-05 08:00`). Yang pertama adalah titik waktu dan harus
  // diterjemahkan ke WIB; yang kedua sudah waktu dinding, dan menggesernya tujuh jam akan
  // menggeser angkanya menjadi salah — kelas kesalahan yang persis melahirkan ratusan
  // penyesuaian tujuh jam di sistem lama. Yang menentukan digeser atau tidak adalah ADA
  // TIDAKNYA zona pada teksnya, bukan tebakan.
  const bagian =
    /^(\d{4})-(\d{2})-(\d{2})(?:[T ](\d{2}):(\d{2})(?::\d{2}(?:\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/.exec(
      bersih,
    )

  // Teks yang tidak dapat dibaca dikembalikan APA ADANYA — nilai mentah yang terbaca aneh
  // masih dapat ditelusuri, sedangkan tanda hubung menghapus jejaknya.
  if (!bagian) return bersih

  const [, tahun, bulan, hari, jam, menit, zona] = bagian

  // Tanpa jam, yang ada hanyalah tanggalnya. Menambahkan "00:00" akan mengarang
  // ketelitian yang tidak ada di sumbernya.
  if (jam === undefined || menit === undefined) return `${hari}/${bulan}/${tahun}`

  if (zona === undefined) return `${hari}/${bulan}/${tahun} ${Number(jam)}:${menit}`

  const saat = new Date(bersih)
  if (Number.isNaN(saat.getTime())) return bersih

  // Konversi zona waktu terjadi DI SINI — di tempat waktu ditampilkan, satu-satunya
  // tempat yang boleh melakukannya (`08-TECHNICAL-STRATEGY.md` §4.4).
  const wib = new Date(saat.getTime() + 7 * 60 * 60_000)
  const dua = (angka: number) => String(angka).padStart(2, '0')

  return (
    `${dua(wib.getUTCDate())}/${dua(wib.getUTCMonth() + 1)}/${wib.getUTCFullYear()} ` +
    `${wib.getUTCHours()}:${dua(wib.getUTCMinutes())}`
  )
}

/** pesanGalat mengambil kalimat yang layak dibaca pengguna dari sebuah galat. */
export function pesanGalat(error: unknown): string {
  if (error instanceof NetworkError) {
    return 'Server Claim PNC tidak dapat dihubungi. Periksa koneksi jaringan.'
  }
  if (error instanceof APIError) return error.message
  return 'Terjadi kesalahan pada sistem.'
}

export type IsiPesan = { title: string; description: string; tone: ErrorTone }

/**
 * pesanMuat mengubah galat pemuatan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Ketiga galat portal ditangani tersendiri, dan itu bukan kelengkapan demi kelengkapan:
 * layar ini menampilkan data milik satu badan hukum, dan pengguna yang belum memilih
 * portal harus tahu bahwa itulah yang kurang — bukan mengira antreannya kosong.
 */
export function pesanMuat(error: unknown): IsiPesan {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Pemberitahuan reasuransi dimiliki masing-masing entitas. Pilih portal ' +
            'entitas di bagian atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }

      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }

      case ErrorCode.validationFailed:
        return {
          title: 'Penyaring belum benar',
          description: error.message,
          tone: 'penolakan',
        }

      default:
        return {
          title: 'Antrean tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }

  return {
    title: 'Terjadi kesalahan pada sistem',
    description:
      'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/**
 * galatIsian memetakan `detail` pada galat validasi menjadi pesan per isian.
 *
 * Server mengirim SELURUH pelanggaran sekaligus (`P-5`), dan panel pencarian menandai
 * masing-masing kotaknya. Tanpa pemetaan ini, ketiga pelanggaran hanya muncul sebagai satu
 * kalimat di atas formulir dan pengguna harus menebak kotak mana yang salah.
 */
export function galatIsian(error: unknown): Record<string, string> {
  if (!(error instanceof APIError)) return {}

  const hasil: Record<string, string> = {}
  for (const rincian of error.detail) {
    if (rincian.field) hasil[rincian.field] = rincian.pesan
  }
  return hasil
}
