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
  readonly detail: DetailGalat[]

  /**
   * Pelanggaran per isian dalam bentuk peta `kolom → pesan`.
   *
   * Ia menyampaikan hal yang sama dengan `detail`, tetapi bentuknya berbeda karena
   * modul Master Rekening mengirimkannya sebagai objek, bukan senarai
   * (internal/masterrekening/http/dto.go:151). Keduanya hidup berdampingan supaya tidak
   * ada modul yang harus mengubah kontraknya lebih dulu; penyatuannya masuk TKT-F1-004.
   *
   * Kosong untuk galat yang tidak menunjuk isian tertentu.
   */
  readonly field: Record<string, string>

  constructor(
    kode: string,
    pesan: string,
    status: number,
    detail?: DetailGalat[],
    field?: Record<string, string>,
  ) {
    super(pesan)
    this.name = 'GalatAPI'
    this.kode = kode
    this.status = status
    this.detail = detail ?? []
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
  /**
   * PUT dipakai pengubahan master: seluruh isi yang boleh diubah dikirim setiap kali,
   * sehingga permintaannya menggantikan dan idempoten. DELETE sengaja TIDAK ada —
   * tidak satu pun layar menghapus data, dan metode yang tidak tersedia di sini tidak
   * dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar.
   */
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
    // detail dan field dibaca sebagai unknown lalu diperiksa, bukan dipercaya
    // bentuknya: badan galat datang dari jaringan, dan `as` tidak memeriksa apa pun
    // saat berjalan.
    const galat = isi as
      | { kode?: string; pesan?: string; detail?: unknown; field?: unknown }
      | null
    throw new GalatAPI(
      galat?.kode ?? KodeGalat.galatInternal,
      galat?.pesan ?? 'Terjadi kesalahan pada sistem.',
      respons.status,
      Array.isArray(galat?.detail) ? (galat.detail as DetailGalat[]) : [],
      petaField(galat?.field),
    )
  }
  return isi as T
}

/**
 * petaField menyaring `field` menjadi peta teks→teks yang aman dipakai layar.
 *
 * Senarai dan `null` ikut ditolak — keduanya bertipe `object` di JavaScript, sehingga
 * pemeriksaan `typeof` saja akan meloloskannya. Pasangan yang nilainya bukan teks
 * dibuang satu per satu, bukan membuang seluruh peta: satu isian yang bentuknya aneh
 * tidak boleh menghilangkan pesan isian lain yang sudah benar.
 */
function petaField(nilai: unknown): Record<string, string> {
  if (typeof nilai !== 'object' || nilai === null || Array.isArray(nilai)) return {}

  const hasil: Record<string, string> = {}
  for (const [kolom, pesan] of Object.entries(nilai)) {
    if (typeof pesan === 'string') hasil[kolom] = pesan
  }
  return hasil
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
