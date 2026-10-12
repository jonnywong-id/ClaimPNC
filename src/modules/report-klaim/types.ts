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
 * Dua di antaranya menentukan letak isiannya, bukan sekadar aktif-tidaknya:
 *
 *	bisnis   → autocomplete yang di layar lama berada DI BARIS panel Klaim Per Bisnis
 *	rincian  → kotak centang berlabel "Treaty", DI BARIS panel Akseptasi
 *
 * Keduanya memang bukan penyaring bersama. Menaruhnya di bilah atas membuat layar
 * menjanjikan isian yang 27 dari 28 panel abaikan.
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

/** Satu baris laporan. */
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

/** Satu pilihan dropdown "Bisnis" — lini bisnis. */
export type BusinessLine = {
  nilai: string
  nama: string
}

/** Satu pilihan radio "Status Compliance". */
export type ComplianceStatus = {
  nilai: string
  nama: string
}

/**
 * Isi layar sebelum satu tombol pun ditekan.
 *
 * `laporan` DATAR dan sudah berurutan — urutan layar Pega. Layar tidak mengurutkannya
 * ulang dan tidak mengelompokkannya: susunan itulah yang dihafal pengguna.
 */
export type CatalogResponse = {
  judul: string
  laporan: Report[]
  lini_bisnis: BusinessLine[]
  status_compliance: ComplianceStatus[]
}

/** Satu pilihan autocomplete "Bisnis" pada panel Klaim Per Bisnis. */
export type BusinessOption = {
  kode: string
  nama: string
}

export type BusinessOptionResponse = {
  bisnis: BusinessOption[]
}

/**
 * Isian penyaring.
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
