import { KodeGalat } from './tipe'

/**
 * Galat dari API dalam bentuk yang dapat diperiksa layar.
 *
 * Yang dipakai membedakan jenis galat adalah `kode`, bukan `pesan`.
 */
export class GalatAPI extends Error {
  readonly kode: string
  readonly status: number

  /**
   * Kesalahan per kolom pada galat validasi.
   *
   * Terisi hanya bila backend mengirimkannya. Ia ada supaya pesan validasi dapat
   * ditaruh di kolom yang benar alih-alih ditumpuk di atas formulir — satu kotak
   * merah berisi sembilan kalimat memaksa pengguna mencocokkan sendiri kalimat mana
   * milik kolom mana.
   */
  readonly field: Record<string, string>

  constructor(kode: string, pesan: string, status: number, field?: Record<string, string>) {
    super(pesan)
    this.name = 'GalatAPI'
    this.kode = kode
    this.status = status
    this.field = field ?? {}
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
}

/**
 * panggilAPI adalah satu-satunya tempat `fetch` dipanggil di seluruh aplikasi.
 *
 * Komponen tidak pernah memanggil fetch sendiri; mereka memakai hook TanStack Query
 * yang memanggil fungsi ini (docs/Steering/08-TECHNICAL-STRATEGY.md §3).
 */
export async function panggilAPI<T>(jalur: string, opsi: OpsiPermintaan = {}): Promise<T> {
  const { metode = 'GET', badan, token } = opsi

  const header: Record<string, string> = { Accept: 'application/json' }
  if (badan !== undefined) header['Content-Type'] = 'application/json'
  // Token dikirim di header, tidak pernah di URL: nilai di URL ikut tercatat di log
  // peramban, log proxy, dan header Referer.
  if (token) header['Authorization'] = `Bearer ${token}`

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
    const galat = isi as
      | { kode?: string; pesan?: string; field?: Record<string, string> }
      | null
    throw new GalatAPI(
      galat?.kode ?? KodeGalat.galatInternal,
      galat?.pesan ?? 'Terjadi kesalahan pada sistem.',
      respons.status,
      galat?.field,
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
