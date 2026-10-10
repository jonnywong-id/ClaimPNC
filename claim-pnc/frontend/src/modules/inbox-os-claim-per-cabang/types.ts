/**
 * Bentuk data layar Inbox OS Claim per Cabang — menu `MENU_ID 69`, pengganti harness
 * `OutstandingKlaimperCabang_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan alias Pega.
 * Alasannya ada di `internal/inboxosclaimpercabang/inboxosclaimpercabang.go`: sejumlah alias
 * kueri lama menyesatkan sepenuhnya — `TanggalTerlambat` sebenarnya tanggal catatan progres
 * terakhir, `PICRekanan` sebenarnya PIC Teknik, dan `Medicare` tidak ada hubungannya dengan
 * pengobatan.
 */

/** Satu baris klaim outstanding milik satu cabang. */
export type WorkItem = {
  cabang: string
  sumbis: string

  /** Class of Business — hasil terjemahan `grouppanel`, bukan kolom tersendiri. */
  cob: string

  no_polis: string

  /**
   * Nama tertanggung.
   *
   * Kolom ini digambar layar lama tetapi SELALU kosong di sana — tidak satu pun rule
   * mengisinya. Di sini terisi; lihat `selisih_terencana`.
   */
  nama_insured: string

  no_klaim: string

  /** Tanggal ISO `YYYY-MM-DD`, atau teks kosong bila memang tidak ada. */
  tanggal_registrasi: string
  tanggal_kejadian: string

  /**
   * Kolom berjudul "Reserve Claim ASM Share", dikirim sebagai TEKS desimal kanonik.
   *
   * Dibaca dengan `formatRupiah` dari `@/lib/money`, bukan `formatMoney` — yang kedua
   * menerima angka. Memakai yang salah tidak menghasilkan galat tipe, hanya tampilan yang
   * keliru.
   *
   * Nilainya BUKAN porsi ASM meski judulnya menjanjikan begitu; lihat `selisih_terencana`.
   */
  nilai_estimasi: string

  /** Kapan catatan progres TERAKHIR dibuat. Kosong berarti belum ada satu pun. */
  tanggal_update_progres: string

  status_progres_1: string
  status_progres_2: string

  /** PIC Teknik yang memegang klaim. */
  pic: string

  adjuster: string

  /** Cause of Loss — penyebab kerugian pada coverage terakhir. */
  col: string

  /** Umur klaim dalam hari kalender sejak registrasi. */
  aging_hari: number

  /**
   * Baris ini digambar MERAH.
   *
   * Dihitung SERVER, bukan di sini: aturannya `aging > 180 || progres mandek`, dan aturan
   * yang hidup di dua tempat akan berbeda antara grid dan berkas ekspor.
   */
  perlu_perhatian: boolean

  /**
   * Progres klaim ini mandek — tiga catatan progres terakhirnya bernilai sama.
   *
   * Dikirim terpisah dari `perlu_perhatian` supaya layar dapat menjelaskan MENGAPA sebuah
   * baris merah. Merah tanpa sebab yang dapat dibaca hanya memindahkan pertanyaannya.
   */
  progres_mandek: boolean

  /** Keterangan pada catatan progres terakhir. Tidak digambar sebagai kolom. */
  catatan_progres: string
}

/** Cabang yang barisnya sedang ditampilkan, dipakai judul layar. */
export type Branch = {
  kode: string
  nama: string
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban `GET /api/inbox-os-claim-per-cabang`. */
export type ListResponse = {
  data: WorkItem[]
  cabang: Branch
  paginasi: PageInfo

  /** Ambang umur yang membuat baris merah, dipakai menjelaskan pewarnaannya. */
  ambang_aging: number
  portal: string
}

/**
 * Bentuk isi popup Detail — pengganti harness `View_DetailKlaimCabang_Harness`.
 *
 * Empat bagian, sesuai layar lamanya: ringkasan, grid objek, riwayat progres, dan komunikasi
 * dengan loss adjuster.
 */

/** Delapan nilai ringkasan pada kepala popup. */
export type DetailHeader = {
  no_klaim: string

  /** Class of Business. Ia yang MENENTUKAN varian kolom grid objek. */
  cob: string

  occupation: string

  /**
   * Berjudul "Total Sum Insured :", dikirim sebagai TEKS desimal kanonik.
   *
   * Namanya menyesatkan dan itu dibawa apa adanya: angkanya TSI satu objek, bukan jumlah
   * seluruh objek. Lihat `selisih_terencana`.
   */
  total_sum_insured: string

  kronologi: string

  /** Berjudul "Total Reserve :", teks desimal kanonik. */
  total_reserve: string

  aging_hari: number

  /** Umurnya melewati ambang, sehingga popup menandainya seperti barisnya di grid. */
  perlu_perhatian: boolean

  /** Sudah dirangkai server, dipisah `", "`, atau `"-"` bila tidak ada. */
  dominant_factor: string

  claim_recommendation: string

  /** Berjudul "Note dari PIC :" — keterangan pada catatan progres terakhir. */
  note_pic: string
}

/**
 * Satu baris grid objek pertanggungan.
 *
 * SELURUH kolom ketiga varian dikirim server. Yang memilih varian di Pega adalah empat when
 * rule yang tidak ada di export, sehingga pemilihannya dikerjakan di sini dari `cob` — dan
 * itu dapat diperbaiki tanpa menyentuh backend begitu keempat rule itu tiba.
 */
export type DetailObject = {
  nama: string
  lokasi: string
  pekerjaan: string
  tanggal_lahir: string
  ktp_paspor: string
  status_peserta: string
}

/** Satu baris grid riwayat progres. */
export type DetailProgress = {
  /** Cap waktu WIB lengkap dengan jam — jam itu yang membedakan dua catatan sehari. */
  tanggal_input: string
  no_klaim: string
  status_progres_1: string
  status_progres_2: string
  user_input: string

  /** Tanggal saja, tanpa jam, seperti di layar lama. */
  tanggal_next_followup: string

  status: string
  keterangan: string
}

/** Satu baris grid komunikasi dengan loss adjuster. */
export type DetailMessage = {
  nama_user: string
  tanggal_proses: string
  pesan: string
  tanggal_balas: string
  jawaban: string

  /** Pengirimnya petugas teknis, bukan adjuster luar. Menentukan sisi tampilannya. */
  internal: boolean
}

/** Jawaban `GET /api/inbox-os-claim-per-cabang/{nomor}`. */
export type DetailResponse = {
  ringkasan: DetailHeader
  objek: DetailObject[]
  riwayat_progres: DetailProgress[]
  komunikasi_adjuster: DetailMessage[]
  cabang: Branch
  portal: string
}
