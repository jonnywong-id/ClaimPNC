import { APIError } from '@/api/client'
import {
  formatAdviceDate,
  loadFailureMessage,
  networkAwareMessage,
  type LoadMessage,
} from '@/api/inboxMessages'
import { ErrorCode } from '@/api/types'

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
 * Konversi zona waktunya terjadi di tempat waktu ditampilkan — lihat `formatAdviceDate`.
 */
export const formatTanggal = formatAdviceDate

/** pesanGalat mengambil kalimat yang layak dibaca pengguna dari sebuah galat. */
export const pesanGalat = networkAwareMessage

export type IsiPesan = LoadMessage

/**
 * pesanMuat mengubah galat pemuatan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Ketiga galat portal ditangani tersendiri, dan itu bukan kelengkapan demi kelengkapan:
 * layar ini menampilkan data milik satu badan hukum, dan pengguna yang belum memilih
 * portal harus tahu bahwa itulah yang kurang — bukan mengira antreannya kosong.
 */
export function pesanMuat(error: unknown): IsiPesan {
  return loadFailureMessage(error, {
    portalDescription:
      'Pemberitahuan reasuransi dimiliki masing-masing entitas. Pilih portal ' +
      'entitas di bagian atas halaman ini lebih dulu.',
    defaultTitle: 'Antrean tidak dapat dimuat',
    special: (failure) =>
      failure.kode === ErrorCode.validationFailed
        ? { title: 'Penyaring belum benar', description: failure.message, tone: 'penolakan' }
        : null,
  })
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
