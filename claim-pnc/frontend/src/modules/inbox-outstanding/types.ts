/**
 * Bentuk data layar Inbox Outstanding.
 *
 * Nama field mengikuti kontrak JSON backend apa adanya — berbahasa Indonesia, sesuai lima
 * pengecualian `D-80`. Nama tipe dan nama variabel di sekitarnya berbahasa Inggris.
 */

/** Satu baris pada layar: sebuah klaim yang masih berjalan. */
export type OutstandingClaim = {
  klaim_id: string

  /** Kosong untuk klaim yang belum lolos Input Register — nomornya belum terbit. */
  nomor_klaim: string

  nomor_polis: string // kolom "Policy no"
  nama_tertanggung: string // kolom "Insured name"
  nama_bisnis: string // kolom "Business Name"
  sumber_bisnis: string // kolom "Business source"
  nama_cabang: string // kolom "Branch name"
  group_panel: string

  /** YYYY-MM-DD dalam WIB; backend yang mengonversinya dari UTC. */
  tanggal_pendaftaran: string // kolom "Register Date"
  tanggal_kejadian: string // kolom "Date of loss"
  tanggal_lapor: string // tidak ditampilkan

  status_proses: string

  /** Kolom "Claim status" — "On Progress" · "Close" · "Reject", diturunkan backend. */
  status_tampil: string

  /**
   * Kolom "Status ASM" — isi `STATUSLOCK_1` apa adanya.
   *
   * Apakah ia kode atau label BELUM DIPASTIKAN; lebar kolomnya `VARCHAR2(100)` menunjuk
   * label, tetapi isinya belum pernah terlihat. Layar menampilkannya tanpa menyatakan
   * yang mana. Tercatat di `docs/keputusan-implementasi.md` §18.20.
   */
  status_klaim: string

  /** Kolom "ASM PIC". */
  pic_teknik: string

  /** Kolom "Admin name". */
  admin_pnc: string

  /** Kolom "Total Aging" — umur klaim dalam hari sejak didaftarkan. */
  umur_hari: number

  /**
   * Kolom "Aging" — dibaca apa adanya dari kolom `AGING` sistem lama.
   *
   * `null` berarti belum terisi, dan itu BERBEDA dari nol hari.
   */
  aging_hari: number | null

  // Tiga field berikut tidak ditampilkan sebagai kolom; section rujukan tidak punya
  // kolomnya. Keduanya tetap diterima karena berguna saat menelusuri satu klaim.
  posisi_klaim: string
  tahap_kini: string
  pemegang_tugas: string
}

/**
 * Batas data yang berlaku saat daftar dibaca.
 *
 * Ia dipakai layar untuk MENYATAKAN keadaannya kepada pengguna. Tanpa itu, petugas yang
 * lini bisnisnya belum diisi admin melihat klaim seluruh lini tanpa cara apa pun untuk
 * mengetahui bahwa yang dilihatnya lebih luas dari haknya.
 */
export type LineScope = {
  tanpa_batas: boolean
  group_panel: string[]
}

export type OutstandingListResponse = {
  klaim: OutstandingClaim[]
  total: number
  batas_lini: LineScope
}

/** Penyaring daftar. Seluruhnya opsional; kosong berarti tidak menyaring. */
export type OutstandingFilter = {
  search?: string
  stage?: string
  branch?: string
  offset?: number
}
