import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import type { ErrorTone } from '@/components/ErrorMessage'

/**
 * Pesan galat pemuatan bersama layar master.
 *
 * Seluruh layar master membedakan gagal memuat dari gagal menyimpan dengan cara yang sama:
 * jaringan putus, portal belum dipilih, basis data entitas belum tersedia, lalu galat API
 * lain. Yang berbeda hanya teksnya — dan teks itu ditiru apa adanya dari layar masing-masing,
 * sehingga di sini ia menjadi parameter, bukan disamakan.
 *
 * Tiga gaya teks yang hidup berdampingan di layar-layar itu dinamai menurut saran yang
 * diberikannya kepada pengguna: tekan Refresh, coba lagi, atau muat ulang halaman.
 */
export type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Letak pemilih portal yang disebut kalimatnya — bilah atas atau bagian atas halaman. */
export type PortalPlace = 'bilah' | 'bagian'

/**
 * Portal belum dipilih — satu-satunya galat pemuatan yang dapat diperbaiki pengguna
 * sendiri, karena itu bernada penolakan, bukan gangguan.
 */
export function portalNotSelected(
  owner = 'Data master',
  place: PortalPlace = 'bagian',
): MessageContent {
  return {
    title: 'Portal entitas belum dipilih',
    description: `${owner} dimiliki masing-masing entitas. Pilih portal entitas di ${place} atas halaman ini lebih dulu.`,
    tone: 'penolakan',
  }
}

const NOT_READY_TITLE = 'Basis data entitas ini belum tersedia'
const NOT_READY_PLANNED =
  'Entitasnya sudah direncanakan, tetapi kredensial basis datanya belum diisi. Hubungi administrator Claim PNC.'
const NOT_READY_NO_RETRY =
  'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.'

type Texts = {
  network: MessageContent
  portal: MessageContent
  notReady: string
  apiDefault: (error: APIError) => MessageContent
  fallback: MessageContent
  extra?: ((error: APIError) => MessageContent | undefined) | undefined
}

function classify(error: unknown, texts: Texts): MessageContent {
  if (error instanceof NetworkError) return texts.network

  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return texts.portal
      case ErrorCode.portalNotReady:
        return { title: NOT_READY_TITLE, description: texts.notReady, tone: 'gangguan' }
      default:
        return texts.extra?.(error) ?? texts.apiDefault(error)
    }
  }

  return texts.fallback
}

/**
 * Gaya "tekan Refresh": galat jaringan menyebut apa yang belum termuat dan tombol yang
 * harus ditekan; galat API menampilkan pesan server apa adanya.
 */
export function refreshLoadMessage(
  error: unknown,
  options: Readonly<{
    /** Apa yang belum dapat dimuat, mis. "Daftar tipe surveyor". */
    subject: string
    failedTitle: string
    owner?: string | undefined
    refreshLabel?: string | undefined
    /** Galat API khusus layar yang diperiksa sebelum pesan bawaan. */
    extra?: ((error: APIError) => MessageContent | undefined) | undefined
  }>,
): MessageContent {
  const { subject, failedTitle, owner, refreshLabel = 'Refresh', extra } = options
  return classify(error, {
    network: {
      title: 'Tidak dapat menghubungi server',
      description: `${subject} belum dapat dimuat. Periksa koneksi lalu tekan ${refreshLabel}.`,
      tone: 'gangguan',
    },
    portal: portalNotSelected(owner, 'bilah'),
    notReady: NOT_READY_PLANNED,
    apiDefault: (e) => ({ title: failedTitle, description: e.message, tone: 'gangguan' }),
    fallback: {
      title: failedTitle,
      description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
      tone: 'gangguan',
    },
    extra,
  })
}

/** Gaya "coba beberapa saat lagi": pesan server tidak ditampilkan. */
export function retryLoadMessage(
  error: unknown,
  options: Readonly<{ failedTitle?: string | undefined; owner?: string | undefined }> = {},
): MessageContent {
  const { failedTitle = 'Daftar tidak dapat dimuat', owner } = options
  return classify(error, {
    network: {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan Anda, lalu muat ulang.',
      tone: 'gangguan',
    },
    portal: portalNotSelected(owner, 'bagian'),
    notReady: NOT_READY_PLANNED,
    apiDefault: () => ({
      title: failedTitle,
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
      tone: 'gangguan',
    }),
    fallback: { title: failedTitle, description: 'Coba beberapa saat lagi.', tone: 'gangguan' },
  })
}

/**
 * Gaya "muat ulang halaman ini": pesan server ditampilkan apa adanya, dan galat yang bukan
 * dari server disebut sebagai kesalahan sistem.
 */
export function reloadLoadMessage(
  error: unknown,
  options: Readonly<{ failedTitle: string; owner?: string | undefined }>,
): MessageContent {
  const { failedTitle, owner } = options
  return classify(error, {
    network: {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    },
    portal: portalNotSelected(owner, 'bagian'),
    notReady: NOT_READY_NO_RETRY,
    apiDefault: (e) => ({ title: failedTitle, description: e.message, tone: 'gangguan' }),
    fallback: {
      title: 'Terjadi kesalahan pada sistem',
      description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
      tone: 'gangguan',
    },
  })
}
