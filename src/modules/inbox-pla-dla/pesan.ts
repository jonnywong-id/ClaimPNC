import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import type { ErrorTone } from '@/components/ErrorMessage'

/**
 * Kode galat khusus modul ini.
 *
 * Nilainya harus sama persis dengan konstanta di
 * `internal/inboxpladla/http/errors.go`.
 */
export const KodeGalat = {
  /**
   * Login pemanggil tidak terdaftar di master reasuransi.
   *
   * Ini kode yang paling sering akan muncul di layar ini, dan ia BUKAN kerusakan: setiap
   * petugas internal yang membuka menu ini menerimanya.
   */
  bukanReasuradur: 'bukan_reasuradur_terdaftar',

  /** Tombol yang belum dibangun. */
  belumTersedia: 'belum_tersedia',
} as const

/** formatTanggal menuliskan tanggal `YYYY-MM-DD` sebagai tanggal Indonesia. */
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

/** bukanReasuradur menyatakan sebuah galat berarti pemanggilnya bukan mitra terdaftar. */
export function bukanReasuradur(error: unknown): boolean {
  return (
    error instanceof APIError && error.kode === KodeGalat.bukanReasuradur
  )
}

export type IsiPesan = { title: string; description: string; tone: ErrorTone }

/**
 * pesanMuat mengubah galat pemuatan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Satu kode ditangani berbeda dari modul lain, dan itu inti layar ini: pemanggil yang
 * bukan reasuradur terdaftar BUKAN menghadapi kerusakan. Di Pega ia hanya melihat layar
 * kosong tanpa keterangan, dan menyimpulkan sistemnya rusak.
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
      case KodeGalat.bukanReasuradur:
        return {
          title: 'Layar ini untuk mitra reasuransi',
          description: error.message,
          tone: 'penolakan',
        }

      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Pendaftaran mitra reasuransi dimiliki masing-masing entitas. Pilih ' +
            'portal entitas di bagian atas halaman ini lebih dulu.',
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

      default:
        return {
          title: 'Daftar tidak dapat dimuat',
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
