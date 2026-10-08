import { APIError } from '@/api/client'

/** Pesan cadangan bila galat tidak membawa pesan yang layak dibaca pengguna. */
const RETRY_LATER = 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'

/**
 * apiMessageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat.
 *
 * Hanya pesan `APIError` yang diteruskan: pesan galat lain (mis. galat jaringan bawaan
 * peramban) ditulis untuk pengembang, bukan untuk petugas klaim.
 */
export function apiMessageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return RETRY_LATER
}

/**
 * errorMessageOf sama dengan apiMessageOf, tetapi juga meneruskan pesan `Error` biasa
 * yang tidak kosong — dipakai layar yang galat sisi kliennya memang ditulis untuk pengguna.
 */
export function errorMessageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return RETRY_LATER
}

/** isISODate mengenali bentuk `YYYY-MM-DD`. */
export function isISODate(text: string): boolean {
  return /^\d{4}-\d{2}-\d{2}$/.test(text)
}

/**
 * fieldErrorsOf memetakan `detail` pada galat validasi menjadi pesan per isian.
 *
 * Nama isiannya dibaca dari `field` MAUPUN `kolom`: kontrak galat belum seragam antarmodul
 * (`TKT-F1-004`), dan membaca keduanya membuat layar tidak ikut rusak bila kontraknya kelak
 * berubah ke bentuk yang lain.
 */
export function fieldErrorsOf(error: unknown): Record<string, string> {
  if (!(error instanceof APIError)) return {}

  const result: Record<string, string> = {}
  for (const violation of error.detail) {
    const name = violation.field ?? violation.kolom
    if (name) result[name] = violation.pesan
  }
  return result
}
