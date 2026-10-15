/**
 * Tipe kontrak API modul View History Claim.
 *
 * Nama field mengikuti kontrak JSON yang dikirim server — berbahasa Indonesia, mengikuti
 * `D-80` yang menetapkan nama field JSON sebagai pengecualian. Nama TIPE-nya berbahasa
 * Inggris, seperti seluruh nama di dalam kode.
 *
 * Tipe di sini ditulis tangan, dan itu utang teknis yang disadari: `09-API-STRATEGY.md`
 * §6 menetapkan tipe dihasilkan dari kontrak OpenAPI, dan kontrak itu belum ada di
 * repository ini. Selama belum ada, satu-satunya penjaga kesesuaiannya adalah uji.
 */

/** Satu baris hasil pencarian. */
export type ClaimHistory = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol "Lihat Detail Klaim".
   *
   * Ia TIDAK ditampilkan di kolom mana pun — layar rincian klaim adalah `MENU_ID 75`
   * "View Claim" yang belum dibangun.
   */
  referensi: string

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  tanggal_kejadian: string | null
  bisnis: string
  cabang: string
  status: string
  posisi_klaim: string
  tanggal_close: string | null
  catatan_close: string
  pic_teknis: string

  /** Keempatnya hanya terisi pada sebagian tipe pencarian. */
  nomor_akseptasi: string
  nomor_balai_lelang: string
  nama_objek: string
  tanggal_lahir: string | null
}

/**
 * Kolom tambahan yang hanya dibawa sebagian tipe pencarian.
 *
 * Nilainya sama persis dengan konstanta Column di backend. Layar memakainya untuk memutuskan
 * kolom mana yang digambar — bukan menebaknya dari isi baris, yang akan membuat kolom
 * hilang saat seluruh barisnya kebetulan kosong.
 */
export type ExtraColumn =
  | 'nomor_akseptasi'
  | 'nomor_balai_lelang'
  | 'nama_objek'
  | 'tanggal_lahir'

/** Satu pilihan pada dropdown "Tipe Pencarian". */
export type SearchType = {
  kode: string
  label: string

  /**
   * Ketiganya menentukan isian mana yang digambar layar.
   *
   * Server yang menentukannya, bukan layar, supaya bentuk formulir punya satu sumber
   * kebenaran. Tipe "No Rekening" menyalakan DUA sekaligus — dan itu memang bentuk
   * formulir lama, bukan kekeliruan.
   */
  pakai_teks: boolean
  pakai_tanggal_pencarian: boolean
  pakai_tanggal_lahir: boolean

  kolom_tambahan: ExtraColumn[]

  tersedia: boolean
  alasan_belum_tersedia?: string
}

/** Keadaan gerbang proteksi data. */
export type ProtectionAccess = {
  jatah_total: number
  jatah_terpakai: number
  jatah_sisa: number
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban saat layar dibuka. */
export type OpenResponse = {
  tipe_pencarian: SearchType[]
  proteksi: ProtectionAccess
  portal: string
}

/** Jawaban satu pencarian. */
export type SearchResponse = {
  klaim: ClaimHistory[]
  halaman: PageInfo
  proteksi: ProtectionAccess
  portal: string
}

/** Isi formulir pencarian sebagaimana dipegang layar. */
export type SearchForm = {
  tipe: string
  nilai: string

  /** Isian "Tanggal Pencarian" — tampil untuk Tgl Kejadian dan No Rekening. */
  tanggal_pencarian: string

  /** Isian "Tanggal Lahir" — tampil untuk tipe Tanggal Lahir. */
  tanggal_lahir: string
}

/** Formulir kosong. Tipe pencariannya belum dipilih. */
export const EMPTY_FORM: SearchForm = {
  tipe: '',
  nilai: '',
  tanggal_pencarian: '',
  tanggal_lahir: '',
}

/**
 * Kode galat modul ini.
 *
 * Klien membedakan jenis galat lewat kode, tidak pernah dengan mencocokkan teks pesan —
 * teks dapat berubah kapan saja tanpa mengubah artinya.
 */
export const ClaimHistoryError = {
  /** Pengguna belum terdaftar di Master Proteksi Data. */
  notRegistered: 'proteksi_belum_terdaftar',

  /** Jatah pencarian habis. */
  quotaExhausted: 'jatah_pencarian_habis',
} as const
