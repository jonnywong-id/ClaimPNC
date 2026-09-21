import { APIError, NetworkError } from '@/api/client'

import { InboxXOLError } from './types'

/**
 * messageOf mengambil pesan yang layak dibaca pengguna dari galat apa pun.
 *
 * Ia ada supaya setiap tempat di modul ini tidak menuliskan rantai `instanceof` yang
 * sama — dan supaya galat yang tidak dikenali tidak pernah bocor apa adanya ke layar.
 * Pesan galat internal dapat memuat nama tabel dan potongan SQL
 * (`11-ERROR-HANDLING.md` §1.2 butir 5).
 */
export function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof NetworkError) return error.message
  return 'Terjadi kesalahan yang tidak dikenali. Coba lagi beberapa saat lagi.'
}

/**
 * violationsOf mengembalikan pelanggaran per isian, atau peta kosong bila galatnya bukan
 * galat validasi.
 *
 * Layar memakainya untuk menandai isian yang salah di tempatnya — bukan sekadar satu
 * pesan di atas formulir, yang pada formulir berisian banyak memaksa pengguna menebak
 * kolom mana yang dimaksud.
 */
export function violationsOf(error: unknown): Record<string, string> {
  if (error instanceof APIError && error.kode === InboxXOLError.validationFail) {
    return error.violations()
  }
  return {}
}

/** isValidationError menyatakan galat ini sudah ditandai di isiannya masing-masing. */
export function isValidationError(error: unknown): boolean {
  return error instanceof APIError && error.kode === InboxXOLError.validationFail
}
