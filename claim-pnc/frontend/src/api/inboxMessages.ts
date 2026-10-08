import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import type { ErrorTone } from '@/components/ErrorMessage'

/**
 * Penerjemah galat dan tanggal yang dipakai bersama layar-layar Inbox PLA / DLA.
 */

/**
 * formatAdviceDate menuliskan tanggal `YYYY-MM-DD` dari server sebagai tanggal Indonesia.
 *
 * # Konversi zona waktu terjadi DI SINI
 *
 * Di tempat waktu ditampilkan — satu-satunya tempat yang boleh melakukannya
 * (`08-TECHNICAL-STRATEGY.md` §4.4). Tidak ada penambahan 7 jam manual di mana pun, dan
 * `Asia/Jakarta` disebut namanya supaya hasilnya tidak bergantung pada zona waktu mesin
 * pengguna.
 *
 * Nilai KOSONG menghasilkan tanda hubung, bukan tanggal apa pun: kolom tanggal advice
 * memang kadang kosong, dan menggambar tanggal di sana akan menampilkan sesuatu yang tidak
 * pernah tercatat.
 */
export function formatAdviceDate(nilai: string): string {
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

/** networkAwareMessage mengambil kalimat yang layak dibaca pengguna dari sebuah galat. */
export function networkAwareMessage(error: unknown): string {
  if (error instanceof NetworkError) {
    return 'Server Claim PNC tidak dapat dihubungi. Periksa koneksi jaringan.'
  }
  if (error instanceof APIError) return error.message
  return 'Terjadi kesalahan pada sistem.'
}

export type LoadMessage = { title: string; description: string; tone: ErrorTone }

/**
 * loadFailureMessage mengubah galat pemuatan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Ketiga galat portal ditangani tersendiri: layar-layar ini menampilkan data milik satu
 * badan hukum, dan pengguna yang belum memilih portal harus tahu bahwa itulah yang kurang —
 * bukan mengira antreannya kosong. Kode yang hanya berarti di satu layar ditangani lewat
 * `special`, yang diperiksa lebih dulu.
 */
export function loadFailureMessage(
  error: unknown,
  options: {
    /** Penjelasan saat portal belum dipilih. */
    portalDescription: string
    /** Judul untuk galat API lain. */
    defaultTitle: string
    special?: ((error: APIError) => LoadMessage | null) | undefined
  },
): LoadMessage {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    }
  }

  if (error instanceof APIError) {
    const special = options.special?.(error)
    if (special) return special

    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description: options.portalDescription,
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
          title: options.defaultTitle,
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
