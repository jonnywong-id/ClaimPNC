import type { Cents, PercentE4 } from '@/components/format'

/**
 * Bentuk kiriman API modul **Case Study Claim** (`MENU_ID 74`).
 *
 * Nama field mengikuti KONTRAK dari server dan karena itu berbahasa Indonesia — salah satu
 * dari lima pengecualian `D-80`, bersama komentar, nama kolom basis data, teks layar, dan
 * variabel lingkungan. Yang berbahasa Inggris hanyalah nama tipenya.
 */

/** Satu pilihan dropdown: kode yang dikirim, label yang dibaca. */
export type Option = {
  kode: string
  label: string
}

/**
 * Jenis kolom, menentukan cara sel digambar.
 *
 * Datang dari server supaya daftar "kolom mana yang berisi uang" hidup di satu tempat —
 * `internal/casestudyclaim/columns.go`. Menyalinnya ke sini berarti kolom yang ditambahkan
 * kelak akan digambar sebagai teks polos tanpa satu pun galat.
 */
export type ColumnKind = 'teks' | 'tanggal' | 'uang' | 'persen' | 'catatan'

/** Satu kolom grid. */
export type Column = {
  kunci: string
  judul: string
  jenis: ColumnKind
}

/** Keterangan layar: isi kedua dropdown, susunan kolom, dan catatan yang berlaku. */
export type MetadataResponse = {
  bisnis: Option[]
  status: Option[]
  kolom: Column[]

  /** Ambang nilai klaim yang membuat sebuah klaim masuk layar ini, dalam SEN. */
  ambang_nilai_klaim: Cents

  /** Penjelasan bahwa hanya TAHUN dari kedua tanggal yang dipakai menyaring. */
  catatan_periode: string

  /** Penjelasan kenapa "Nature of Loss" dan "Cause of Loss" selalu sama. */
  catatan_kolom_kembar: string
}

/**
 * Satu baris grid.
 *
 * # Nilai uang bertipe `Cents`, dan boleh `null`
 *
 * `null` BERBEDA dari nol: `SUM` atas himpunan kosong mengembalikan NULL, dan "belum ada
 * nilainya" bukan "nilainya nol". Layar menggambar yang `null` sebagai tanda hubung.
 */
export type CaseStudyRow = {
  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  cob: string
  periode_polis: string
  bulan_klaim: string
  tanggal_kejadian: string
  sob: string
  posisi_reasuransi: string

  /**
   * Keduanya SELALU berisi teks yang sama.
   *
   * Di layar lama pun kedua kolomnya terikat pada satu properti yang sama, dan kuerinya
   * hanya menyediakan satu nilai. Dipertahankan apa adanya (`P-5`).
   */
  nature_of_loss: string
  cause_of_loss: string

  tsi_100: Cents | null
  share_asm: PercentE4 | null
  deductible: Cents | null
  nilai_share_asm: Cents | null
  nilai_klaim_100: Cents | null
  adjuster_fee_100: Cents | null
  nilai_klaim_net_100: Cents | null
  nilai_klaim_net_share_asm: Cents | null
  lack_of_doc: Cents | null

  cabang: string
  status_klaim: string
  kronologi: string

  /** Satu-satunya isian yang dapat disunting di seluruh layar. */
  remark: string
}

/** Rentang TAHUN yang benar-benar dipakai menyaring. */
export type Period = {
  tahun_awal: string
  tahun_akhir: string
}

/** Badan respons daftar. */
export type ListResponse = {
  baris: CaseStudyRow[]
  total: number

  /**
   * Periode yang benar-benar dipakai.
   *
   * Layar menyatakannya kepada pengguna: ia memilih dua TANGGAL, tetapi yang dipakai hanya
   * tahunnya. Tanpa disebutkan, daftar yang memuat klaim di luar bulan yang dipilih akan
   * terbaca sebagai penyaring yang rusak.
   */
  periode: Period
}

/** Penyaring yang diisi pengguna di formulir. */
export type FilterInput = {
  /** Tanggal `YYYY-MM-DD`. Hanya tahunnya yang dipakai menyaring. */
  dari: string
  sampai: string
  bisnis: string
  status: string
}

/** Badan permintaan penyimpanan catatan telaah. */
export type SaveRemarkRequest = {
  nomor_klaim: string
  catatan: string
}

/** Badan respons penyimpanan catatan telaah. */
export type SaveRemarkResponse = {
  nomor_klaim: string
  catatan: string
}
