/**
 * Tipe kontrak API modul Archive Dokumen Klaim.
 *
 * Nama field mengikuti kontrak JSON yang dikirim server — berbahasa Indonesia, mengikuti
 * `D-80` yang menetapkan nama field JSON sebagai pengecualian. Nama TIPE-nya berbahasa
 * Inggris, seperti seluruh nama di dalam kode.
 *
 * Tipe di sini ditulis tangan, dan itu utang teknis yang disadari: `09-API-STRATEGY.md`
 * §6 menetapkan tipe dihasilkan dari kontrak OpenAPI, dan kontrak itu belum ada di
 * repository ini. Selama belum ada, satu-satunya penjaga kesesuaiannya adalah uji.
 */

/** Satu baris grid ARCHIVE FILE KLAIM. */
export type ArchiveFile = {
  id: number

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  tanggal_kejadian: string | null
  pic_teknis: string

  tanggal_terima_dokumen: string | null
  tanggal_input: string | null
  jumlah_lembar: number

  kode_tipe_dokumen: string
  tipe_dokumen: string
  kode_jenis_dokumen: string
  jenis_dokumen: string

  nama_box: string
  kode_filling: string
  user_input: string

  tanggal_kirim_dokumen: string | null

  /** Keempatnya dipakai bagian Kirim ke Cabang. */
  group_panel: string
  sudah_dikirim: boolean
  kode_layanan: string
  catatan_layanan: string
}

/** Satu baris grid Input Data Archive — klaim yang berkasnya hendak diarsipkan. */
export type ClaimCandidate = {
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

  /**
   * Tidak ditampilkan di grid, tetapi IKUT dikirim balik saat menyimpan.
   *
   * Kolom GROUPPANEL itulah yang menentukan siapa melihat berkasnya pada daftar kirim ke
   * cabang — membuangnya di layar berarti berkasnya hilang dari daftar orang yang
   * seharusnya mengirimnya.
   */
  group_panel: string
}

/** Satu pilihan dropdown berkode dan berlabel. */
export type Option = {
  kode: string
  label: string
}

/** Satu pilihan Jenis Dokumen beserta tipe pemiliknya. */
export type DocumentKind = Option & {
  kode_tipe_dokumen: string
}

/** Satu baris pemilih "Pilih Kode". */
export type FillingCode = {
  kode: string
  nama_box: string
  jumlah_pemakaian: number
}

/** Lini bisnis yang disembunyikan dari daftar kirim ke cabang. */
export type BranchScope = {
  lini_disembunyikan: string[]
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

export type OpenResponse = {
  tipe_input: Option[]
  tipe_dokumen: Option[]
  jenis_dokumen: DocumentKind[]
  cakupan_cabang: BranchScope
  portal: string
}

export type SearchResponse = {
  berkas: ArchiveFile[]
  halaman: PageInfo
  portal: string
}

export type ClaimSearchResponse = {
  klaim: ClaimCandidate[]
  portal: string
}

export type FillingCodeResponse = {
  kode: FillingCode[]

  /**
   * Selalu true, dan layar MENAMPILKANNYA sebagai keterangan.
   *
   * Daftar kode filling disusun dari kode yang sudah pernah dipakai, bukan dari master —
   * masternya tidak ada di export (`R-16`). Tanpa keterangan itu, pengguna akan menduga
   * kode yang belum pernah dipakai memang tidak boleh dipakai.
   */
  dari_pemakaian: boolean

  portal: string
}

export type PendingResponse = {
  berkas: ArchiveFile[]
  halaman: PageInfo
  cakupan_cabang: BranchScope
  portal: string
}

export type SaveResponse = {
  id: number
  baru: boolean
  pesan: string

  /**
   * Keempatnya melaporkan PENGIRIMAN yang menyertai penyimpanan.
   *
   * Menyimpan berkas langsung mengirimkannya ke sistem Arsip — perilaku sistem lama yang
   * direplikasi. Pengiriman yang gagal TIDAK menggagalkan penyimpanan, sehingga jawaban
   * ini tetap berhasil dengan `terkirim` bernilai false dan `galat_kirim` terisi.
   */
  terkirim: boolean
  kode_layanan?: string
  catatan_layanan?: string
  galat_kirim?: string

  portal: string
}

export type SendResponse = {
  id: number
  kode_layanan: string
  catatan_layanan: string
  pesan: string
  portal: string
}

/** Badan permintaan penyimpanan berkas arsip. */
export type SaveRequest = {
  /** Nol berarti baris baru. */
  id: number

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  tanggal_kejadian: string | null
  pic_teknis: string
  group_panel: string

  tanggal_terima_dokumen: string | null
  jumlah_lembar: number
  kode_tipe_dokumen: string
  kode_jenis_dokumen: string
  nama_box: string
  kode_filling: string
}

/** Kode galat yang dikenali layar. Nilainya sama dengan konstanta di backend. */
export const ArchiveError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
  notFound: 'berkas_tidak_ditemukan',
  alreadySent: 'berkas_sudah_dikirim',
  serviceFailed: 'layanan_arsip_gagal',
  serviceAddress: 'alamat_layanan_arsip_kosong',
} as const

/** Dua mode pencarian arsip. Nilainya sama dengan SearchMode di backend. */
export const SearchMode = {
  keyword: 'kata_kunci',
  inputDate: 'tanggal_input',
} as const

export type SearchModeValue = (typeof SearchMode)[keyof typeof SearchMode]

/** Isi formulir pencarian arsip. */
export type SearchForm = {
  mode: SearchModeValue
  kata_kunci: string
  tanggal_dari: string
  tanggal_sampai: string
}

export const EMPTY_SEARCH: SearchForm = {
  mode: SearchMode.keyword,
  kata_kunci: '',
  tanggal_dari: '',
  tanggal_sampai: '',
}

/** Isi formulir pencarian klaim pada bagian Input Data Archive. */
export type ClaimSearchForm = {
  tipe: string
  nilai: string
}

export const EMPTY_CLAIM_SEARCH: ClaimSearchForm = {
  tipe: 'no_klaim',
  nilai: '',
}

/** Isi formulir berkas arsip. */
export type ArchiveForm = {
  id: number

  /** Keenam isian berikut ikut dari klaim yang dipilih; tidak diketik pengguna. */
  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  tanggal_kejadian: string | null
  pic_teknis: string
  group_panel: string

  /** Keenam isian berikut diketik pengguna. */
  tanggal_terima_dokumen: string
  jumlah_lembar: string
  kode_tipe_dokumen: string
  kode_jenis_dokumen: string
  nama_box: string
  kode_filling: string
}

export const EMPTY_ARCHIVE_FORM: ArchiveForm = {
  id: 0,
  nomor_klaim: '',
  nomor_polis: '',
  nama_tertanggung: '',
  tanggal_kejadian: null,
  pic_teknis: '',
  group_panel: '',
  tanggal_terima_dokumen: '',
  jumlah_lembar: '',
  kode_tipe_dokumen: '',
  kode_jenis_dokumen: '',
  nama_box: '',
  kode_filling: '',
}

/** formFromClaim menyiapkan formulir dari klaim yang dipilih di grid. */
export function formFromClaim(claim: ClaimCandidate): ArchiveForm {
  return {
    ...EMPTY_ARCHIVE_FORM,
    nomor_klaim: claim.nomor_klaim,
    nomor_polis: claim.nomor_polis,
    nama_tertanggung: claim.nama_tertanggung,
    tanggal_kejadian: claim.tanggal_kejadian,
    pic_teknis: claim.pic_teknis,
    group_panel: claim.group_panel,
  }
}

/*
 * TIDAK ADA pembentuk formulir dari baris arsip yang sudah ada, dan itu disengaja.
 *
 * Grid ARCHIVE FILE KLAIM BACA-SAJA, sama seperti sistem lama: di seluruh export, penanda
 * `flags` pada prosedur simpan hanya pernah disetel `"insert"` — cabang `update`-nya tidak
 * pernah dipanggil dari layar ini. Keputusan Work Owner 2026-09-25.
 *
 * Backend masih menerima `id` bukan nol pada endpoint simpan, karena prosedurnya memang
 * punya cabang itu. Tidak ada satu pun jalur layar yang memakainya sekarang.
 */

/** toSaveRequest mengubah isi formulir menjadi badan permintaan. */
export function toSaveRequest(form: ArchiveForm): SaveRequest {
  return {
    id: form.id,
    nomor_klaim: form.nomor_klaim,
    nomor_polis: form.nomor_polis,
    nama_tertanggung: form.nama_tertanggung,
    tanggal_kejadian: form.tanggal_kejadian,
    pic_teknis: form.pic_teknis,
    group_panel: form.group_panel,
    tanggal_terima_dokumen: form.tanggal_terima_dokumen || null,

    // Isian kosong dikirim sebagai nol, bukan NaN. Server menolaknya dengan pesan yang
    // menunjuk isiannya; NaN akan menjadi `null` di JSON dan ditolak sebagai bentuk yang
    // salah — galat yang benar tetapi pesannya tidak menunjuk apa pun.
    jumlah_lembar: Number(form.jumlah_lembar) || 0,

    kode_tipe_dokumen: form.kode_tipe_dokumen,
    kode_jenis_dokumen: form.kode_jenis_dokumen,
    nama_box: form.nama_box,
    kode_filling: form.kode_filling,
  }
}
