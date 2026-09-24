/**
 * Tipe kontrak API modul Inbox XOL.
 *
 * Nama field mengikuti kontrak JSON yang dikirim server — berbahasa Indonesia, mengikuti
 * `D-80` yang menetapkan nama field JSON sebagai pengecualian. Nama TIPE-nya berbahasa
 * Inggris, seperti seluruh nama di dalam kode.
 *
 * Tipe di sini ditulis tangan, dan itu utang teknis yang disadari: `09-API-STRATEGY.md`
 * §6 menetapkan tipe dihasilkan dari kontrak OpenAPI, dan kontrak itu belum ada di
 * repository ini. Selama belum ada, satu-satunya penjaga kesesuaiannya adalah uji.
 */

/** Tipe pemberitahuan kepada reasuradur. */
export type AdviceType = 'PLA' | 'DLA'

/** Satu perjanjian XOL — satu tahun, satu kurs, sekumpulan group business. */
export type MasterXOL = {
  id: string
  nama: string
  tahun: string

  /** Kurs perjanjian; nilai klaim rupiah dibagi angka ini sebelum ditampilkan. */
  kurs: number

  tipe: string

  /** Nama group business yang sudah dirangkai, seperti kolom "Group Business". */
  group_business: string

  /**
   * Jumlah group business.
   *
   * Dipakai membedakan "perjanjian belum diisi" dari "sudah diisi tetapi namanya
   * kosong" — dua keadaan yang tampak sama pada `group_business`, dan hanya yang
   * pertama yang berarti grid klaimnya memang kosong.
   */
  jumlah_group_business: number

  status_komite: string
  menunggu_komite: boolean
  catatan_komite: string
  pic: string
  catatan_pic: string
}

/** Satu baris grid "DATA XOL BASED ON DOL AND COL". */
export type ClaimSummary = {
  tanggal_kejadian: string
  sebab_kerugian: string
  group_business: string
  nilai_outstanding: number
  nilai_akseptasi: number
}

/** Isi grid utama beserta perjanjian yang menjadi dasarnya. */
export type ClaimSummaryResponse = {
  perjanjian: MasterXOL
  baris: ClaimSummary[]
}

/** Asal satu baris rincian. */
export type BreakdownSource = 'bisnis' | 'treaty'

/** Satu baris grid rincian, pengganti `Sec_Detail_claim_XOL`. */
export type Breakdown = {
  group_business: string
  kode_group_business: string
  jumlah_klaim: number
  nilai_outstanding: number
  nilai_akseptasi: number
  sumber: BreakdownSource

  /**
   * Kurs mata uang baris ini tidak ditemukan, sehingga nilainya TIDAK dapat dipercaya.
   *
   * Di sistem lama keadaan ini tidak pernah terlihat: fungsi kurs mengembalikan `1`,
   * dan nilai valuta asing diperlakukan satu banding satu terhadap rupiah tanpa satu
   * pun tanda (`D-49` butir 5).
   */
  kurs_tidak_tersedia: boolean
}

/** Satu baris grid PLA/DLA yang sudah diterbitkan. */
export type Advice = {
  /** Sudah dirakit beserta revisinya — "nomor / revisi". */
  nomor: string
  nomor_asli: string
  revisi: string
  nama_insurance: string
  nama_layer: string
  tahun: string
  sebab_kerugian: string
  kurs: number

  /** Angka, bukan teks bersatuan. Tanda persennya ditambahkan saat ditampilkan. */
  share_percent: number

  email: string
  remark: string
  status_persetujuan: string
  sudah_disetujui: boolean
  catatan_persetujuan: string
  catatan_pic: string
  batas_layer: string
  negara: string
  tanggal_terbit: string
  tipe: AdviceType
}

/** Satu baris antrean persetujuan pemberitahuan pada tab Komite. */
export type ApprovalItem = {
  /**
   * Tahun perjanjian.
   *
   * Kolomnya di grid lama berjudul "Date Of Loss", dan judul itu MENYESATKAN — isinya
   * tahun, bukan tanggal. Judul lama tetap dipakai di layar (`P-5`, `D-13`); nama
   * field-nya di sini dibetulkan.
   */
  tahun: string

  sebab_kerugian: string
  tipe: AdviceType
  tanggal_insert: string
}

/** Isi tab "Inbox XOL Komite": dua antrean yang berdiri sendiri. */
export type ApprovalResponse = {
  pemberitahuan: ApprovalItem[]
  perjanjian: MasterXOL[]
}

/** Satu pilihan Penyebab Kerugian. */
export type CauseOfLoss = {
  id: string

  /**
   * Yang tersimpan di kolom CAUSEOFLOSS pada tabel klaim XOL adalah DESKRIPSI-nya,
   * bukan ID-nya. Layar karena itu mengirim deskripsi saat menyaring, bukan kode.
   */
  deskripsi: string
}

/** Penyaring pencarian PLA/DLA. */
export type AdviceForm = {
  tahun: string
  sebab_kerugian: string
  tipe: AdviceType | ''
}

/** Penyaring kosong, dipakai sebagai keadaan awal formulir. */
export const EMPTY_ADVICE_FORM: AdviceForm = {
  tahun: '',
  sebab_kerugian: '',
  tipe: '',
}

/**
 * Kode galat yang dikenali layar.
 *
 * Nilainya sama persis dengan konstanta di `internal/inboxxol/http/errors.go`. Layar
 * membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan.
 */
export const InboxXOLError = {
  validationFail: 'validasi_gagal',
  masterNotFound: 'perjanjian_tidak_ditemukan',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
  writeNotAllowed: 'aksi_belum_tersedia',
} as const
