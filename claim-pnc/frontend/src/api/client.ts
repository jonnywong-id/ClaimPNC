import { ErrorCode } from './types'
import type { FieldViolation } from './types'

/**
 * Galat dari API dalam bentuk yang dapat diperiksa layar.
 *
 * Yang dipakai membedakan jenis galat adalah `kode`, bukan `pesan`.
 */
export class APIError extends Error {
  readonly kode: string
  readonly status: number

  /**
   * Pelanggaran per isian, hanya terisi pada galat validasi (`422`).
   *
   * Backend mengirim SELURUH pelanggaran sekaligus, bukan yang pertama saja — meniru
   * perilaku Pega yang menampilkan semua pesan bersamaan (P-5). Ia dibawa sampai ke
   * layar supaya isian yang salah dapat ditandai di tempatnya, bukan sekadar satu pesan
   * di atas form — pada form panjang pengguna harus menebak kolom mana yang dimaksud.
   * Meringkasnya menjadi satu pesan membuang justru bagian yang berguna.
   *
   * Kosong untuk galat yang tidak menunjuk isian tertentu.
   *
   * CATATAN KONTRAK: kedua modul master belum sepakat nama kuncinya — `statusprogres`
   * mengirim `kolom` (internal/statusprogres/http/dto.go:82), `masterstatus` mengirim
   * `field` (internal/masterstatus/http/dto.go:55). Tipe di sini mengikuti `kolom`;
   * layar Status Klaim hanya membaca `pesan` sehingga belum terdampak. Penyeragamannya
   * masuk kontrak galat TKT-F1-004.
   */
  readonly detail: FieldViolation[]

  constructor(kode: string, pesan: string, status: number, detail?: FieldViolation[]) {
    super(pesan)
    this.name = 'APIError'
    this.kode = kode
    this.status = status
    this.detail = detail ?? []
    // this.field = field ?? {}
  }
}

/** Galat jaringan: permintaan tidak pernah sampai ke backend. */
export class NetworkError extends Error {
  constructor() {
    super('Tidak dapat menghubungi server Claim PNC.')
    this.name = 'NetworkError'
  }
}

type RequestOptions = {
  /**
   * PUT dipakai pengubahan master: seluruh isi yang boleh diubah dikirim setiap kali,
   * sehingga permintaannya menggantikan dan idempoten. DELETE sengaja TIDAK ada —
   * tidak satu pun layar menghapus data, dan metode yang tidak tersedia di sini tidak
   * dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar.
   */
  metode?: 'GET' | 'POST' | 'PUT'
  body?: unknown
  token?: string | null
  /**
   * Alias portal entitas yang melayani permintaan ini.
   *
   * Wajib untuk setiap endpoint yang menyentuh basis data entitas: satu aplikasi
   * melayani empat badan hukum dengan basis data terpisah (ADR-0030), dan backend
   * MENOLAK permintaan yang tidak menyebutkannya — ia tidak pernah jatuh ke portal
   * utama sebagai cadangan (R-20).
   *
   * Dikirim sebagai header, bukan di URL: nilai di URL ikut tercatat di log peramban,
   * log proxy, dan header Referer.
   */
  portal?: string | null
}

/** Nama header tempat portal entitas disebut. Sama dengan portalhttp.HeaderPortal. */
export const HEADER_PORTAL = 'X-Portal'

/**
 * panggilAPI adalah satu-satunya tempat `fetch` dipanggil di seluruh aplikasi.
 *
 * Komponen tidak pernah memanggil fetch sendiri; mereka memakai hook TanStack Query
 * yang memanggil fungsi ini (docs/Steering/08-TECHNICAL-STRATEGY.md §3).
 */
export async function callAPI<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { metode = 'GET', body, token } = options

  const header: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) header['Content-Type'] = 'application/json'
  // Token dikirim di header, tidak pernah di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  if (token) header['Authorization'] = `Bearer ${token}`
  // if (portal) header[HEADER_PORTAL] = portal

  let response: Response
  try {
    response = await fetch(path, {
      method: metode,
      headers: header,
      body: body === undefined ? null : JSON.stringify(body),
    })
  } catch {
    throw new NetworkError()
  }

  if (response.status === 204) return undefined as T

  const content = await readJSON(response)
  if (!response.ok) {
    const error = content as { kode?: string; pesan?: string; detail?: FieldViolation[] } | null
    throw new APIError(
      error?.kode ?? ErrorCode.internalError,
      error?.pesan ?? 'Terjadi kesalahan pada sistem.',
      response.status,
      error?.detail,
    )
  }
  return content as T
}

async function readJSON(response: Response): Promise<unknown> {
  const text = await response.text()
  if (text.trim() === '') return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}
