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
 * formatTanggal menuliskan tanggal `YYYY-MM-DD` dari server sebagai tanggal Indonesia.
 *
 * # Konversi zona waktu terjadi DI SINI
 *
 * Di tempat waktu ditampilkan — satu-satunya tempat yang boleh melakukannya
 * (`08-TECHNICAL-STRATEGY.md` §4.4). Tidak ada penambahan 7 jam manual di mana pun, dan
 * `Asia/Jakarta` disebut namanya supaya hasilnya tidak bergantung pada zona waktu mesin
 * pengguna.
 *
 * Nilai KOSONG menghasilkan tanda hubung, bukan tanggal apa pun. Itu keadaan yang nyata
 * di layar ini: kolom tanggal advice memang kadang kosong, dan menggambar tanggal di sana
 * akan menampilkan sesuatu yang tidak pernah tercatat.
 */
export function formatTanggal(nilai: string): string {
  const bersih = nilai.trim()
  if (bersih === '') return '—'

  const saat = new Date(bersih)
  if (Number.isNaN(saat.getTime())) return bersih

  return saat.toLocaleDateString('id-ID', {
    timeZone: 'Asia/Jakarta',
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  })
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
