import { ErrorCode } from './types'

/**
 * Galat dari API dalam bentuk yang dapat diperiksa layar.
 *
 * Yang dipakai membedakan jenis galat adalah `kode`, bukan `pesan`.
 */
export class APIError extends Error {
  readonly kode: string
  readonly status: number

  /**
   * Rincian tambahan yang dikirim backend bersama galat, bila ada.
   *
   * Sebagian galat tidak cukup dijelaskan satu pesan. Penolakan validasi klaim,
   * misalnya, menyebut SETIAP aturan yang dilanggar beserta kolomnya, supaya layar dapat
   * menandai kolom yang tepat alih-alih menampilkan satu kalimat dan membiarkan petugas
   * mencari sendiri.
   *
   * Tipenya `unknown` dengan sengaja: bentuknya berbeda per modul, dan pemanggilnyalah
   * yang tahu bentuk apa yang ia harapkan. Memaksakan satu tipe di sini berarti modul
   * yang satu ikut berubah setiap kali modul lain menambah rincian.
   */
  readonly details: unknown

  constructor(kode: string, pesan: string, status: number, details?: unknown) {
    super(pesan)
    this.name = 'GalatAPI'
    this.kode = kode
    this.status = status
    this.details = details
  }
}

/** Galat jaringan: permintaan tidak pernah sampai ke backend. */
export class NetworkError extends Error {
  constructor() {
    super('Tidak dapat menghubungi server Claim PNC.')
    this.name = 'GalatJaringan'
  }
}

type RequestOptions = {
  method?: 'GET' | 'POST'
  body?: unknown
  token?: string | null
}

/**
 * callAPI adalah satu-satunya tempat `fetch` dipanggil di seluruh aplikasi.
 *
 * Komponen tidak pernah memanggil fetch sendiri; mereka memakai hook TanStack Query
 * yang memanggil fungsi ini (docs/Steering/08-TECHNICAL-STRATEGY.md §3).
 */
export async function callAPI<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, token } = options

  const header: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) header['Content-Type'] = 'application/json'
  // Token dikirim di header, tidak pernah di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  if (token) header['Authorization'] = `Bearer ${token}`

  let response: Response
  try {
    response = await fetch(path, {
      method: method,
      headers: header,
      body: body === undefined ? null : JSON.stringify(body),
    })
  } catch {
    throw new NetworkError()
  }

  if (response.status === 204) return undefined as T

  const content = await readJSON(response)
  if (!response.ok) {
    const failure = content as { kode?: string; pesan?: string; violations?: unknown } | null
    throw new APIError(
      failure?.kode ?? ErrorCode.internalError,
      failure?.pesan ?? 'Terjadi kesalahan pada sistem.',
      response.status,
      failure?.violations,
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
