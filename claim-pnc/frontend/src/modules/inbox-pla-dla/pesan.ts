import { APIError } from '@/api/client'
import {
  formatAdviceDate,
  loadFailureMessage,
  networkAwareMessage,
  type LoadMessage,
} from '@/api/inboxMessages'

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
export const formatTanggal = formatAdviceDate

/** pesanGalat mengambil kalimat yang layak dibaca pengguna dari sebuah galat. */
export const pesanGalat = networkAwareMessage

/** bukanReasuradur menyatakan sebuah galat berarti pemanggilnya bukan mitra terdaftar. */
export function bukanReasuradur(error: unknown): boolean {
  return (
    error instanceof APIError && error.kode === KodeGalat.bukanReasuradur
  )
}

export type IsiPesan = LoadMessage

/**
 * pesanMuat mengubah galat pemuatan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Satu kode ditangani berbeda dari modul lain, dan itu inti layar ini: pemanggil yang
 * bukan reasuradur terdaftar BUKAN menghadapi kerusakan. Di Pega ia hanya melihat layar
 * kosong tanpa keterangan, dan menyimpulkan sistemnya rusak.
 */
export function pesanMuat(error: unknown): IsiPesan {
  return loadFailureMessage(error, {
    portalDescription:
      'Pendaftaran mitra reasuransi dimiliki masing-masing entitas. Pilih ' +
      'portal entitas di bagian atas halaman ini lebih dulu.',
    defaultTitle: 'Daftar tidak dapat dimuat',
    special: (failure) =>
      failure.kode === KodeGalat.bukanReasuradur
        ? {
            title: 'Layar ini untuk mitra reasuransi',
            description: failure.message,
            tone: 'penolakan',
          }
        : null,
  })
}
