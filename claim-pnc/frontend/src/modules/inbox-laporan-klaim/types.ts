/**
 * Tipe kontrak API modul Inbox Laporan Klaim.
 *
 * # Kenapa di dalam modul, bukan di `api/types.ts`
 *
 * Berkas itu memuat tipe yang dipakai LEBIH DARI SATU tempat — sesi, portal, dan galat
 * yang mengikat seluruh aplikasi. Tipe di bawah hanya dipakai satu layar, dan
 * menaruhnya di sana membuat berkas bersama tumbuh setiap kali ada modul baru sampai
 * tidak seorang pun tahu lagi mana yang benar-benar bersama. Modul `registrasi` sudah
 * memakai pola yang sama.
 *
 * Nama field mengikuti kontrak API apa adanya — berbahasa Indonesia (`D-80`). Ia BUKAN
 * nama internal: mengubahnya adalah perubahan yang merusak klien.
 *
 * Tipe ini ditulis tangan, dan itu utang yang disadari: `09-API-STRATEGY.md` §6
 * menetapkan tipe dihasilkan dari kontrak OpenAPI, yang belum ada di aplikasi ini.
 * Seluruh modul sebelumnya punya utang yang sama.
 */

/** Satu baris pada daftar Inbox Laporan Klaim. */
export type ClaimReport = {
  id: string
  nomor_klaim: string
  nomor_polis: string
  tertanggung: string
  nama_bisnis: string
  nomor_rujukan: string

  /** `YYYY-MM-DD`, atau kosong bila tanggalnya memang belum ada. */
  tanggal_kejadian: string
  tanggal_masuk: string
  tanggal_aging: string

  /** Umur berkas dalam hari, dihitung server terhadap satu waktu acuan. */
  umur_hari: number

  pembuat: string
  kode_cabang: string
  nama_cabang: string

  alasan: string
  subjek_email: string

  /** Pesan terakhir; hanya terisi pada ketiga tab komunikasi. */
  pesan_akhir: string

  /** Outstanding · Not Registered · Not Transferred — teks layar Pega (`D-13`). */
  posisi: string

  /** `pega` atau `claimpnc` — sistem yang menerbitkan baris ini. */
  asal: string

  /** Kunci penugasan warisan; kosong untuk berkas terbitan aplikasi ini. */
  rujukan_pega: string
}

/** Satu tab beserta lencana jumlahnya. */
export type ReportCategory = {
  kode: string
  judul: string

  /**
   * `null` berarti tab itu memang TIDAK dicacah sistem lama — bukan berarti nol.
   * Lihat tab "Data rejected".
   */
  jumlah: number | null

  /** Tab yang isinya percakapan; kolom "Last message" hanya digambar untuk ini. */
  komunikasi: boolean
}

/** Satu pilihan dropdown. */
export type ReportOption = {
  kode: string
  nama: string
}

export type ReportPagination = {
  halaman: number
  ukuran: number

  /** Banyaknya baris yang cocok di SELURUH tabel, bukan di halaman ini. */
  total: number
  total_halaman: number
}

export type ClaimReportListResponse = {
  laporan: ClaimReport[]
  kategori: ReportCategory[]
  halaman: ReportPagination
  portal: string
}

export type ClaimReportOptionResponse = {
  kategori: ReportCategory[]
  bisnis: ReportOption[]
  kanwil: ReportOption[]
  portal: string
}

export type ClaimReportResponse = {
  laporan: ClaimReport
  portal: string
}

/** Penyaring yang dipilih pengguna di layar. */
export type ClaimReportQuery = {
  kategori: string
  kanwil: string
  bisnis: string
  cari: string
  halaman: number
}

/** Tab pertama layar, sama seperti membuka layarnya di Pega. */
export const DEFAULT_CATEGORY = 'outstanding'

/** Penyaring awal saat layar pertama dibuka. */
export const EMPTY_QUERY: ClaimReportQuery = {
  kategori: DEFAULT_CATEGORY,
  kanwil: '',
  bisnis: '',
  cari: '',
  halaman: 1,
}

/**
 * Kode galat modul Inbox Laporan Klaim.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const ClaimReportErrorCode = {
  notFound: 'tidak_ditemukan',
  callerIncomplete: 'profil_pemanggil_tidak_lengkap',
  validationFailed: 'validasi_gagal',
} as const

export type ClaimReportErrorCode =
  (typeof ClaimReportErrorCode)[keyof typeof ClaimReportErrorCode]
