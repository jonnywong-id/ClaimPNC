import { KodeGalat, type DetailGalat } from './tipe'

/**
 * Galat dari API dalam bentuk yang dapat diperiksa layar.
 *
 * Yang dipakai membedakan jenis galat adalah `kode`, bukan `pesan`.
 */
export class GalatAPI extends Error {
  readonly kode: string
  readonly status: number

  /**
   * Pelanggaran per isian, bila galatnya berupa kegagalan validasi.
   *
   * Backend mengirim SELURUH pelanggaran sekaligus, bukan yang pertama saja — meniru
   * perilaku Pega yang menampilkan semua pesan bersamaan (P-5). Layar memakainya untuk
   * menyorot setiap isian yang salah; meringkasnya menjadi satu pesan akan membuang
   * justru bagian yang berguna.
   *
   * Kosong untuk galat yang tidak menunjuk isian tertentu.
   */
  readonly detail: DetailGalat[]

  constructor(kode: string, pesan: string, status: number, detail: DetailGalat[] = []) {
    super(pesan)
    this.name = 'GalatAPI'
    this.kode = kode
    this.status = status
    this.detail = detail
  }
}

/** Galat jaringan: permintaan tidak pernah sampai ke backend. */
export class GalatJaringan extends Error {
  constructor() {
    super('Tidak dapat menghubungi server Claim PNC.')
    this.name = 'GalatJaringan'
  }
}

type OpsiPermintaan = {
  metode?: 'GET' | 'POST' | 'PUT'
  badan?: unknown
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
export async function panggilAPI<T>(jalur: string, opsi: OpsiPermintaan = {}): Promise<T> {
  const { metode = 'GET', badan, token, portal } = opsi

  const header: Record<string, string> = { Accept: 'application/json' }
  if (badan !== undefined) header['Content-Type'] = 'application/json'
  // Token dikirim di header, tidak pernah di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  if (token) header['Authorization'] = `Bearer ${token}`
  if (portal) header[HEADER_PORTAL] = portal

  let respons: Response
  try {
    respons = await fetch(jalur, {
      method: metode,
      headers: header,
      body: badan === undefined ? null : JSON.stringify(badan),
    })
  } catch {
    throw new GalatJaringan()
  }

  if (respons.status === 204) return undefined as T

  const isi = await bacaJSON(respons)
  if (!respons.ok) {
    const galat = isi as { kode?: string; pesan?: string; detail?: unknown } | null
    throw new GalatAPI(
      galat?.kode ?? KodeGalat.galatInternal,
      galat?.pesan ?? 'Terjadi kesalahan pada sistem.',
      respons.status,
      Array.isArray(galat?.detail) ? (galat.detail as DetailGalat[]) : [],
    )
  }
  return isi as T
}

async function bacaJSON(respons: Response): Promise<unknown> {
  const teks = await respons.text()
  if (teks.trim() === '') return null
  try {
    return JSON.parse(teks)
  } catch {
    return null
  }
}
