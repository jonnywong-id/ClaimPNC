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
   * Pelanggaran per isian dalam bentuk SENARAI, hanya terisi pada galat validasi (`422`).
   *
   * Backend mengirim SELURUH pelanggaran sekaligus, bukan yang pertama saja — meniru
   * perilaku Pega yang menampilkan semua pesan bersamaan (P-5). Ia dibawa sampai ke
   * layar supaya isian yang salah dapat ditandai di tempatnya, bukan sekadar satu pesan
   * di atas form — pada form panjang pengguna harus menebak kolom mana yang dimaksud.
   *
   * Kosong untuk galat yang tidak menunjuk isian tertentu.
   *
   * CATATAN KONTRAK — TIGA BENTUK YANG BELUM SERAGAM. Ketiga modul master mengirim
   * pelanggaran validasi dengan bentuk yang berbeda, dan itu keadaan nyata hari ini:
   *
   *   masterstatus         detail: [{ field, pesan }]     internal/masterstatus/http/dto.go:55
   *   masterstatusprogres  detail: [{ kolom, pesan }]     internal/masterstatusprogres/http/dto.go:82
   *   masterrekening       field:  { kolom: pesan }       internal/masterrekening/http/dto.go:151
   *
   * Karena itu `detail` DAN `field` hidup berdampingan di sini. Menyeragamkannya
   * menuntut mengubah kontrak tiga modul sekaligus, dan kontrak galat yang mengikat
   * seluruh aplikasi adalah TKT-F1-004 yang masih terhalang. Yang dikerjakan sekarang
   * adalah menampung ketiganya tanpa membuat satu layar pun menebak bentuknya.
   */
  readonly detail: FieldViolation[]

  /**
   * Pelanggaran per isian dalam bentuk PETA `kolom → pesan`.
   *
   * Dipakai modul Master Rekening. Kosong untuk galat yang tidak menunjuk isian
   * tertentu, dan kosong pula untuk modul yang memakai `detail`.
   */
  readonly field: Record<string, string>

  constructor(
    kode: string,
    pesan: string,
    status: number,
    detail?: FieldViolation[],
    field?: Record<string, string>,
  ) {
    super(pesan)
    this.name = 'APIError'
    this.kode = kode
    this.status = status
    this.detail = detail ?? []
    this.field = field ?? {}
  }

  /**
   * violations menyatukan kedua bentuk menjadi satu peta `kolom → pesan`.
   *
   * Layar memakai ini alih-alih memilih sendiri antara `detail` dan `field`. Dengan
   * begitu, satu layar tidak perlu tahu modul mana yang memakai bentuk yang mana — dan
   * ketika TKT-F1-004 menyeragamkannya kelak, yang berubah hanya berkas ini.
   *
   * Nama kunci dibaca dari `field` maupun `kolom`, karena kedua modul yang memakai
   * `detail` pun belum sepakat menamainya.
   */
  violations(): Record<string, string> {
    const result: Record<string, string> = { ...this.field }
    for (const item of this.detail) {
      const column = item.field ?? item.kolom
      if (column) result[column] = item.pesan
    }
    return result
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
   * sehingga permintaannya menggantikan dan idempoten.
   *
   * DELETE semula sengaja TIDAK ada, dengan alasan bahwa tidak satu pun layar menghapus
   * data dan metode yang tidak tersedia tidak dapat dipakai kode yang ditulis kemudian
   * tanpa keputusan sadar. **Keputusan sadar itu diambil pada 2026-09-20**: Master XOL
   * adalah layar pertama yang menghapus, dan Work Owner memilih hapus fisik BERKASKADE —
   * menghapus satu induk XOL ikut membuang grup bisnis, layer, dan reas di bawahnya.
   *
   * Layar lama pun punya tombol Hapus pada ketiga gridnya
   * (`Activity/DeleteFromTabelMst-Act.xml.xml`), dan tombol itu menghapus SEKETIKA tanpa
   * menunggu Simpan. Yang diperbaiki hanyalah kaskadenya: sistem lama menghapus satu
   * tabel saja dan meninggalkan baris yatim yang masih ada di produksi hari ini.
   */
  metode?: 'GET' | 'POST' | 'PUT' | 'DELETE'
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
 * callAPI adalah satu-satunya tempat `fetch` dipanggil di seluruh aplikasi.
 *
 * Komponen tidak pernah memanggil fetch sendiri; mereka memakai hook TanStack Query
 * yang memanggil fungsi ini (docs/Steering/08-TECHNICAL-STRATEGY.md §3).
 */
export async function callAPI<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { metode = 'GET', body, token, portal } = options

  const header: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) header['Content-Type'] = 'application/json'
  // Token dikirim di header, tidak pernah di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  if (token) header['Authorization'] = `Bearer ${token}`
  if (portal) header[HEADER_PORTAL] = portal

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
    // detail dan field dibaca sebagai unknown lalu diperiksa, bukan dipercaya
    // bentuknya: badan galat datang dari jaringan, dan `as` tidak memeriksa apa pun
    // saat berjalan.
    const error = content as
      | { kode?: string; pesan?: string; detail?: unknown; field?: unknown }
      | null
    throw new APIError(
      error?.kode ?? ErrorCode.internalError,
      error?.pesan ?? 'Terjadi kesalahan pada sistem.',
      response.status,
      Array.isArray(error?.detail) ? (error.detail as FieldViolation[]) : [],
      fieldMap(error?.field),
    )
  }
  return content as T
}

/**
 * downloadAPI mengambil satu berkas dari endpoint yang terlindungi sesi dan portal.
 *
 * # Kenapa tidak cukup `<a href>`
 *
 * Rute yang dilindungi menuntut header Authorization dan X-Portal, dan peramban TIDAK
 * mengirim keduanya pada navigasi biasa. Tautan polos karena itu selalu dijawab 401.
 *
 * Berkasnya diambil sebagai blob lalu diserahkan ke peramban sebagai unduhan. Alamat
 * objeknya dilepas setelah dipakai — tanpa itu, isi berkas tetap dipegang peramban selama
 * halaman terbuka.
 */
export async function downloadAPI(
  path: string,
  filename: string,
  options: { token?: string | null; portal?: string | null } = {},
): Promise<void> {
  const header: Record<string, string> = {}
  if (options.token) header['Authorization'] = `Bearer ${options.token}`
  if (options.portal) header[HEADER_PORTAL] = options.portal

  let response: Response
  try {
    response = await fetch(path, { headers: header })
  } catch {
    throw new NetworkError()
  }

  if (!response.ok) {
    const error = (await readJSON(response)) as { kode?: string; pesan?: string } | null
    throw new APIError(
      error?.kode ?? ErrorCode.internalError,
      error?.pesan ?? 'Berkas tidak dapat diunduh.',
      response.status,
    )
  }

  const url = URL.createObjectURL(await response.blob())
  try {
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    link.click()
  } finally {
    URL.revokeObjectURL(url)
  }
}

type UploadOptions = {
  /** Berkas yang diunggah. Dikirim pada bagian bernama `berkas`, sama dengan backend. */
  berkas: File
  /** Keterangan singkat, hanya dipakai sebagian endpoint. */
  keterangan?: string
  token?: string | null
  portal?: string | null
}

/**
 * uploadAPI mengirim satu berkas sebagai `multipart/form-data`.
 *
 * # Kenapa ia ada di sini, bukan di modul yang memakainya
 *
 * Aturan yang mengikat seluruh frontend adalah "tidak ada `fetch` di dalam komponen"
 * (`docs/Steering/08-TECHNICAL-STRATEGY.md` §3). Tanpa fungsi ini, modul pertama yang
 * perlu mengunggah berkas akan memanggil `fetch` sendiri — dan modul berikutnya akan
 * menirunya, masing-masing dengan cara menangani galat yang sedikit berbeda.
 *
 * Ia berbagi seluruh penanganan galat dengan callAPI, sehingga layar menghadapi APIError
 * dan NetworkError yang sama persis, apa pun bentuk permintaannya.
 *
 * `Content-Type` sengaja TIDAK disetel: peramban menyusunnya sendiri lengkap dengan
 * `boundary`, dan menuliskannya sendiri justru merusak permintaannya.
 */
export async function uploadAPI<T>(path: string, options: UploadOptions): Promise<T> {
  const { berkas, keterangan, token, portal } = options

  const form = new FormData()
  form.append('berkas', berkas)
  if (keterangan) form.append('keterangan', keterangan)

  const header: Record<string, string> = { Accept: 'application/json' }
  if (token) header['Authorization'] = `Bearer ${token}`
  if (portal) header[HEADER_PORTAL] = portal

  let response: Response
  try {
    response = await fetch(path, { method: 'POST', headers: header, body: form })
  } catch {
    throw new NetworkError()
  }

  const content = await readJSON(response)
  if (!response.ok) {
    const error = content as
      | { kode?: string; pesan?: string; detail?: unknown; field?: unknown }
      | null
    throw new APIError(
      error?.kode ?? ErrorCode.internalError,
      error?.pesan ?? 'Terjadi kesalahan pada sistem.',
      response.status,
      Array.isArray(error?.detail) ? (error.detail as FieldViolation[]) : [],
      fieldMap(error?.field),
    )
  }
  return content as T
}

/**
 * fieldMap menyaring `field` menjadi peta teks→teks yang aman dipakai layar.
 *
 * Senarai dan `null` ikut ditolak — keduanya bertipe `object` di JavaScript, sehingga
 * pemeriksaan `typeof` saja akan meloloskannya. Pasangan yang nilainya bukan teks
 * dibuang satu per satu, bukan membuang seluruh peta: satu isian yang bentuknya aneh
 * tidak boleh menghilangkan pesan isian lain yang sudah benar.
 */
function fieldMap(value: unknown): Record<string, string> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return {}

  const result: Record<string, string> = {}
  for (const [column, message] of Object.entries(value)) {
    if (typeof message === 'string') result[column] = message
  }
  return result
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
