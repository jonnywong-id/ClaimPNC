import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import type { ErrorTone } from '@/components/ErrorMessage'

/** Isi satu kotak pesan galat penyimpanan. */
export type MessageContent = { title: string; description: string; tone: ErrorTone }

/**
 * Pesan untuk satu kode galat: isi tetap, `null` (tidak ada kotak pesan), atau fungsi yang
 * membentuk pesannya dari galat itu sendiri.
 */
export type CodeMessage = MessageContent | null | ((error: APIError) => MessageContent | null)

/** Peta `kode galat → pesan`, pengganti `switch (error.kode)` yang dulu ditulis tiap form. */
export type CodeMessages = Readonly<Record<string, CodeMessage>>

/**
 * messageForCode memilih pesan untuk galat API menurut KODE-nya, bukan teks pesannya.
 *
 * Kode yang tidak ada di peta jatuh ke `fallback` — padanan cabang `default` pada switch.
 */
export function messageForCode(
  error: APIError,
  messages: CodeMessages,
  fallback: CodeMessage,
): MessageContent | null {
  const entry = Object.hasOwn(messages, error.kode) ? messages[error.kode] : fallback
  if (typeof entry === 'function') return entry(error)
  return entry ?? null
}

// ---------------------------------------------------------------------------------------
// Pesan baku form master bergaya "kotak" (form yang menerima `error` dari halamannya).
// ---------------------------------------------------------------------------------------

/** Server Claim PNC tidak dapat dihubungi; isian belum tersimpan. */
export const networkMessage: MessageContent = {
  title: 'Server Claim PNC tidak dapat dihubungi',
  description: 'Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.',
  tone: 'gangguan',
}

/** Galat tak terduga di server — padanan cabang `default`. */
export const systemErrorMessage: MessageContent = {
  title: 'Terjadi kesalahan pada sistem',
  description: 'Isian Anda belum tersimpan. Coba beberapa saat lagi.',
  tone: 'gangguan',
}

/** Baris yang disunting sudah tidak ada, mungkin diubah petugas lain. */
export const notFoundMessage: MessageContent = {
  title: 'Baris ini sudah tidak ada',
  description: 'Mungkin sudah diubah petugas lain. Tutup form ini dan muat ulang daftarnya.',
  tone: 'penolakan',
}

/**
 * Galat validasi tanpa kotak pesan bila detailnya ada.
 *
 * Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
 * mengulang hal yang sama. Yang ditampilkan sebagai kotak pesan hanyalah galat yang tidak
 * menunjuk isian tertentu, karena itulah yang tidak dapat diperbaiki pengguna dengan
 * mengetik.
 */
export function validationMessage(error: APIError): MessageContent | null {
  return Object.keys(error.violations()).length > 0
    ? null
    : {
        title: 'Belum dapat disimpan',
        description: error.message,
        tone: 'penolakan',
      }
}

/** Portal entitas belum dipilih atau tidak dikenal. */
export const portalNotChosenMessage: MessageContent = {
  title: 'Portal entitas belum dipilih',
  description: 'Pilih portal entitas di bagian atas halaman, lalu simpan lagi.',
  tone: 'penolakan',
}

/** Pesan galat portal entitas — belum dipilih, tidak dikenal, atau basis datanya belum siap. */
export const portalMessages: CodeMessages = {
  [ErrorCode.portalNotStated]: portalNotChosenMessage,
  [ErrorCode.portalUnknown]: portalNotChosenMessage,
  [ErrorCode.portalNotReady]: {
    title: 'Basis data entitas ini belum tersedia',
    description:
      'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.',
    tone: 'gangguan',
  },
}

/** Pesan baku: validasi, baris tidak ada, dan portal. */
export const standardMessages: CodeMessages = {
  [ErrorCode.validationFailed]: validationMessage,
  [ErrorCode.notFound]: notFoundMessage,
  ...portalMessages,
}

/**
 * saveErrorMessage mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat jaringan selalu memakai `network`; galat API dipilih dari `messages` menurut
 * kodenya, dan yang tidak dikenal jatuh ke `fallback`. Galat lain tidak ditampilkan.
 */
export function saveErrorMessage(
  error: unknown,
  messages: CodeMessages = standardMessages,
  fallback: CodeMessage = systemErrorMessage,
  network: MessageContent = networkMessage,
): MessageContent | null {
  if (error instanceof NetworkError) return network
  if (error instanceof APIError) return messageForCode(error, messages, fallback)
  return null
}

/** Mengambil pelanggaran per isian dari galat validasi server. */
export function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {}
}
