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

  /**
   * Batas layer terendah — `MIN(LIMIT)` atas `MST_XOL_LAYER`, dalam mata uang perjanjian,
   * beserta nilai rupiahnya. Keduanya kolom "Min Limit" dan "Min Limit IDR" pada grid
   * layar rincian.
   */
  min_limit: number
  min_limit_idr: number

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

/**
 * Satu baris grid "DATA XOL BASED ON DOL AND COL".
 *
 * Grid ini menggabungkan hasil SELURUH perjanjian XOL, sehingga tiap baris membawa
 * perjanjian asalnya sendiri. `id_master` itulah yang dipakai membuka rincian di
 * baliknya — rincian butuh tahun, kurs, dan group business perjanjian tersebut.
 */
export type ClaimSummary = {
  id_master: string
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
 * Isian modal "INSERT DOL DAN COL".
 *
 * Ketiganya persis isian modal lama: dua kolom kunci baris yang hendak ditulis ke
 * `POOLDATA.XOL_TABLE_ALL_KLAIM`, ditambah perjanjian yang dipilih dari grid
 * "PILIH MASTER XOL" di dalam modal yang sama.
 */
export type InsertDolColForm = {
  id_master: string
  tanggal_kejadian: string
  sebab_kerugian: string
}

/**
 * Jawaban simpan INSERT DOL DAN COL.
 *
 * Ia membawa JUMLAH BARIS karena satu simpan menuliskan satu baris per Group Business
 * perjanjian — bukan satu baris. Tanpa angka itu, pengguna tidak punya cara mengetahui
 * berapa banyak yang ditulis atas namanya.
 */
export type InsertDolColResult = {
  jumlah_baris: number
}

/** Isian kosong, dipakai sebagai keadaan awal modal. */
export const EMPTY_INSERT_FORM: InsertDolColForm = {
  id_master: '',
  tanggal_kejadian: '',
  sebab_kerugian: '',
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

/**
 * Satu baris grid "Summary Data XOL" pada layar rincian.
 *
 * Hanya dua kolom, dan itu memang isi kuerinya: `GetBusinessnameXOLForSummerry` hanya
 * menyebutkan group business MANA yang menanggung klaim pada tanggal kejadian dan
 * penyebab kerugian itu — tanpa nilai uang.
 */
export type SummaryBusiness = {
  kode_group_business: string
  group_business: string
}

/**
 * Satu baris grid "No Klaim" pada layar rincian.
 *
 * Untuk baris treaty inward, `no_klaim` berisi NAMA PERUSAHAAN — klaim inward tidak punya
 * nomor klaim ASM, dan kueri lama memang mengisinya dengan `COMPANYNAME`.
 */
export type ClaimListItem = {
  no_klaim: string
  mata_uang: string
  sumber: BreakdownSource
  nilai_outstanding: number
  nilai_akseptasi: number
  kurs_tidak_tersedia: boolean
}

/**
 * Hasil unggahan "Upload MBU Salvage".
 *
 * Baris yang ditolak dilaporkan satu per satu, tidak diringkas menjadi jumlah: yang
 * diperbaiki pengguna adalah BERKASNYA, dan pada berkas ratusan baris "ada yang gagal"
 * saja tidak dapat ditindaklanjuti.
 */
export type UploadResult = {
  jumlah_baris: number
  jumlah_tersimpan: number
  ditolak: UploadRejected[]
}

/** Satu baris berkas yang tidak tersimpan, beserta nomor barisnya di berkas. */
export type UploadRejected = {
  baris: number
  no_klaim: string
  alasan: string
}
