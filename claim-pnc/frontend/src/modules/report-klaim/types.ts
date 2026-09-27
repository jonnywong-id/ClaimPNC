/**
 * Bentuk data layar Report Klaim.
 *
 * Nama field mengikuti kontrak API yang berbahasa Indonesia (`D-80`) — ia kontrak, bukan
 * nama internal. Nama TIPE-nya berbahasa Inggris, seperti modul lain.
 */

/** Satu tombol Export pada sebuah kartu. */
export type ReportAction = {
  /** Kosong pada kartu bertombol tunggal. */
  kode: string
  label: string
}

/**
 * Penyaring mana yang berlaku pada satu laporan.
 *
 * Layar memakainya untuk MENONAKTIFKAN isian yang tidak berpengaruh. Menampilkan seluruh
 * isian pada seluruh kartu membuat pengguna mengisi rentang tanggal untuk laporan yang
 * kuerinya tidak menerima tanggal sama sekali, lalu menyimpulkan hasilnya salah.
 */
export type FilterUsage = {
  rentang_tanggal: boolean
  lini_bisnis: boolean
  status_compliance: boolean
  bisnis: boolean
  rincian: boolean
}

/** Jejak ke rule Pega asal sebuah laporan. */
export type ReportSource = {
  activity: string
  rule_sql?: string[]
}

/** Satu kartu laporan. */
export type Report = {
  kode: string
  judul: string
  tombol: ReportAction[]
  penyaring: FilterUsage
  tersedia: boolean
  /** Hanya terisi bila `tersedia` bernilai false. */
  alasan?: string
  penghalang?: string
  sumber: ReportSource
}

/** Satu kelompok kartu. */
export type ReportGroup = {
  kode: string
  judul: string
  laporan: Report[]
}

/** Satu pilihan dropdown yang di layar berlabel "Treaty". */
export type BusinessLine = {
  nilai: string
  nama: string
}

/** Isi layar sebelum satu tombol pun ditekan. */
export type CatalogResponse = {
  judul: string
  kelompok: ReportGroup[]
  lini_bisnis: BusinessLine[]
}

/** Satu pilihan autocomplete "Bisnis". */
export type BusinessOption = {
  kode: string
  nama: string
}

export type BusinessOptionResponse = {
  bisnis: BusinessOption[]
}

/**
 * Isian penyaring di atas layar.
 *
 * Tanggalnya berbentuk ISO `YYYY-MM-DD` — bentuk yang dihasilkan `<input type="date">`
 * dan yang tidak dapat dibaca dua arti, berbeda dari `dd/mm/yyyy` di dalam berkas.
 */
export type ReportFilter = {
  dari: string
  sampai: string
  lini: string
  status_compliance: string
  bisnis: string
  rincian: boolean
}

export const EMPTY_FILTER: ReportFilter = {
  dari: '',
  sampai: '',
  lini: '',
  status_compliance: '',
  bisnis: '',
  rincian: false,
}

/** Permintaan unduh satu berkas. */
export type ExportRequest = {
  kode: string
  /** Kosong pada kartu bertombol tunggal. */
  aksi: string
  filter: ReportFilter
  /** Penyaring mana yang benar-benar berlaku pada laporan itu. */
  penyaring: FilterUsage
}
