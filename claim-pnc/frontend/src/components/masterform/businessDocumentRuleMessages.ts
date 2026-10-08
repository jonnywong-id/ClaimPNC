import type { APIError } from '@/api/client'
import { ErrorCode } from '@/api/types'

import { saveErrorMessage, type CodeMessages, type MessageContent } from './saveErrorMessage'

/**
 * Pesan galat penyimpanan form Daftar Tipe Dokumen Bisnis (tambah dan ubah).
 *
 * Kalimatnya berbeda dari pesan baku form master lain, dan dipertahankan apa adanya;
 * keduanya dipakai bersama supaya tidak ditulis dua kali.
 */

const networkMessage: MessageContent = {
  title: 'Server Claim PNC tidak dapat dihubungi',
  description: 'Periksa sambungan jaringan, lalu simpan sekali lagi.',
  tone: 'gangguan',
}

const portalNotChosen: MessageContent = {
  title: 'Portal entitas belum dipilih',
  description: 'Pilih entitas di bagian atas layar, lalu simpan sekali lagi.',
  tone: 'penolakan',
}

/** Pesan galat portal entitas untuk form aturan dokumen bisnis. */
const businessRulePortalMessages: CodeMessages = {
  [ErrorCode.portalNotStated]: portalNotChosen,
  [ErrorCode.portalUnknown]: portalNotChosen,
  [ErrorCode.portalNotReady]: {
    title: 'Basis data entitas ini belum tersedia',
    description: 'Hubungi tim infrastruktur bila keadaan ini berlanjut.',
    tone: 'gangguan',
  },
}

function fallback(error: APIError): MessageContent {
  return { title: 'Penyimpanan gagal', description: error.message, tone: 'gangguan' }
}

/** businessRuleSaveMessage mengubah galat penyimpanan aturan dokumen bisnis menjadi pesan. */
export function businessRuleSaveMessage(error: unknown, messages: CodeMessages): MessageContent | null {
  return saveErrorMessage(error, { ...messages, ...businessRulePortalMessages }, fallback, networkMessage)
}
