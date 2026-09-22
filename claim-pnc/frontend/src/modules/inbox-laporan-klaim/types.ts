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

  /**
   * Cabang yang membatasi daftar ini.
   *
   * Kosong punya SATU arti: pengguna memilih kanwil, dan pilihan itu menggantikan batas
   * cabangnya. Ia tidak pernah berarti "batasnya hilang" — permintaan yang cabangnya
   * tidak dapat ditentukan dijawab `403 cabang_tidak_dikenali`, bukan dijawab daftar
   * tanpa batas.
   */
  batas_cabang: string

  portal: string
}

export type ClaimReportOptionResponse = {
  kategori: ReportCategory[]
  bisnis: ReportOption[]
  kanwil: ReportOption[]
  portal: string
}

/**
 * Isian form **Input Receive Document**.
 *
 * Asalnya `Flow/InputReceiveDocument.xml` — assignment tunggal pada alur Receive
 * Document — yang merender flow action `InputReceiveDocument`. Label setiap isian dibaca
 * apa adanya dari `Section/ViewInputReceiveDocument_sec-Section.xml`.
 *
 * Tiga blok form lama TIDAK ada di sini; alasannya di
 * `backend/internal/inboxlaporanklaim/detail.go`: blok data pelapor beserta alamatnya
 * (area Heavy Equipment yang `D-34` keluarkan dari lingkup), grid dokumen (`S-1` belum
 * ada), serta riwayat komunikasi dan progres (milik modul lain).
 */
export type ClaimReportDetail = {
  /** `YYYY-MM-DD`, atau kosong bila tanggalnya memang belum diisi. */
  tanggal_terima_dokumen: string
  tanggal_kejadian: string

  nama_pelapor: string
  email_pelapor: string
  telepon_pelapor: string
  nama_kurir: string

  nomor_polis: string
  tertanggung: string
  nama_bisnis: string
  nomor_rujukan: string

  /** Bilangan bulat SEN, tidak pernah pecahan (`ADR-0016`). Rp 1.000 = 100000. */
  estimasi_kerugian: number

  lokasi_kejadian: string
  subjek_email: string

  kronologis: string
  rincian_kerusakan: string
  alasan: string
  keterangan_belum_registrasi: string

  jumlah_dokumen: number
}

export type ClaimReportResponse = {
  laporan: ClaimReport
  isian?: ClaimReportDetail

  /**
   * Boleh tidak berkas ini disimpan dari layar ini.
   *
   * Dihitung server, BUKAN disimpulkan layar dari `asal`. Aturannya — siapa yang
   * berwenang menulis sebuah baris selama masa paralel (`ADR-0004`, `P-1`) — milik
   * server, dan menyalinnya ke sini berarti satu aturan hidup di dua tempat.
   */
  dapat_disunting: boolean

  portal: string
}

/** Isian form yang seluruhnya kosong — keadaan berkas yang baru dibuat. */
export const EMPTY_DETAIL: ClaimReportDetail = {
  tanggal_terima_dokumen: '',
  tanggal_kejadian: '',
  nama_pelapor: '',
  email_pelapor: '',
  telepon_pelapor: '',
  nama_kurir: '',
  nomor_polis: '',
  tertanggung: '',
  nama_bisnis: '',
  nomor_rujukan: '',
  estimasi_kerugian: 0,
  lokasi_kejadian: '',
  subjek_email: '',
  kronologis: '',
  rincian_kerusakan: '',
  alasan: '',
  keterangan_belum_registrasi: '',
  jumlah_dokumen: 0,
}

/**
 * Batas panjang setiap isian, mengikuti kolom yang dibuat migrasi 0004.
 *
 * Angkanya diulang dari `backend/internal/inboxlaporanklaim/detail.go`. Server tetap yang
 * berwenang; yang di sini hanya kenyamanan supaya pengguna tahu sebelum mengirim. Bila
 * salah satu berubah, KEDUA tempat harus ikut berubah — utang yang disadari dari
 * menduplikasi sebuah angka.
 */
export const FIELD_LIMIT = {
  nama: 255,
  email: 200,
  telepon: 64,
  polis: 64,
  rujukan: 64,
  lokasi: 500,
  subjek: 1000,
  catatan: 1000,
  narasi: 4000,
  jumlahDokumen: 9999,
} as const

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

  /** Berkas ada, tetapi penulisnya masih Pega selama masa paralel. */
  readOnly: 'laporan_hanya_baca',

  /** Cabang klaim petugas tidak dapat ditentukan — layar tidak dapat dibuka. */
  branchUnknown: 'cabang_tidak_dikenali',

  /** Sumber data cabang sedang tidak dapat dibaca — keadaan sementara. */
  branchUnreadable: 'sumber_cabang_tidak_terbaca',
} as const

export type ClaimReportErrorCode =
  (typeof ClaimReportErrorCode)[keyof typeof ClaimReportErrorCode]
