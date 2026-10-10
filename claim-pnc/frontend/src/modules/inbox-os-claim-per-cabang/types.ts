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

  /** Selisih terhadap layar Pega yang sudah diputuskan (`D-54`). */
  selisih_terencana: string[]

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
  /** Kunci baris. Tidak digambar; ia yang menandai baris mana yang sedang terbuka. */
  id: string
  nama: string
  lokasi: string
  pekerjaan: string
  tanggal_lahir: string
  ktp_paspor: string
  status_peserta: string

  /** Isi panel yang terbuka ketika baris ini dibuka. Lihat `DetailCoverage`. */
  coverage: DetailCoverage[]
}

/**
 * Satu baris coverage di bawah sebuah objek pertanggungan.
 *
 * Ketiga kolomnya persis yang digambar layar lama. Di Pega, baris objek adalah
 * **master-detail** (`pyRowEditing = masterDetail`, `pyEditAction = ViewObjectItem`), dan
 * aksi itu merender `ViewObjectCoverage` yang menggambar `.Currency`, `.SumTSI`, dan
 * `.CoverageNote`.
 */
export type DetailCoverage = {
  id: string
  /** Isi kolom "Coverage" — NAMA jaminan, mis. "FLEXAS". */
  coverage: string
  mata_uang: string
  /** Teks desimal kanonik, bukan angka — digambar lewat `formatRupiah`. */
  tsi: string
  /** Isi grid "Object Item" yang terbuka saat baris coverage dibuka. */
  object_item: DetailItem[]
  /** Isi grid "List Spreading". */
  list_spreading: DetailSpreading[]
  /** Isi grid "CO MEMBER". */
  co_member: DetailCoMember[]
}

/**
 * Satu baris grid "List Spreading".
 *
 * Melekat pada COVERAGE, bukan pada adjustment. Kedua nilai uangnya DIHITUNG peladen —
 * kolom tersimpannya NULL pada seluruh baris tabel sumbernya.
 */
export type DetailSpreading = {
  /** `ORS`, `QS`, `FAC-OUT`, dan seterusnya — bukan kodenya. */
  tipe_treaty: string
  currency: string
  /** Teks desimal kanonik. */
  estimasi_value: string
  /** Teks berdesimal EMPAT, mengikuti layar lama yang menuliskannya `100,0000%`. */
  pembagian_persentase: string
  /** Teks desimal kanonik — `estimasi_value × persentase / 100`. */
  result_value: string
}

/** Satu baris grid "CO MEMBER". Bentuknya sama dengan Spreading. */
export type DetailCoMember = {
  asuransi: string
  currency: string
  estimasi_value: string
  pembagian_persentase: string
  result_value: string
}

/**
 * Satu baris grid "Object Item" — tingkat ketiga.
 *
 * Nama dan deskripsinya HAMPIR SELALU KOSONG, dan itu bukan cacat: sumbernya memang tidak
 * memuatnya. `T_CLAIM_OBJECTITEMLIST` hanya punya 1 baris di seluruh tabel, sedangkan
 * 22.004 klaim punya estimasi. Layar lama pun menggambar barisnya dengan sel kosong.
 */
export type DetailItem = {
  id: string
  object_item: string
  deskripsi_item: string
  /** Isi grid "Estimasi" yang terbuka saat baris item dibuka. */
  estimasi: DetailEstimation[]
}

/** Satu baris grid "Estimasi" — tingkat terdalam. */
export type DetailEstimation = {
  estimasi_ke: string
  tanggal_estimasi: string
  tipe_estimasi: string
  mata_uang: string
  /** Teks desimal kanonik. */
  nilai_kurs: string
  /** Teks desimal kanonik. DAPAT negatif — koreksi yang saling meniadakan. */
  nilai_estimasi: string
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

  /** Selisih POPUP terhadap Pega — daftar yang berbeda dari milik layar daftar. */
  selisih_terencana: string[]

  portal: string
}

/**
 * Bentuk panel ringkasan di atas grid — kartu angka, sebaran umur, dan rincian per COB
 * maupun per sumber bisnis.
 *
 * Panel ini TIDAK ada di Pega; ia diminta Work Owner (2026-10-08). Karena tidak ada
 * pembandingnya, yang dapat diuji hanyalah konsistensinya dengan grid di bawahnya.
 */

/** Satu pita pada batang sebaran umur. */
export type AgeBucket = {
  label: string
  berkas: number

  /** Teks desimal kanonik; dibaca dengan `formatRupiah`. */
  nilai: string
}

/** Satu baris rincian per COB atau per sumber bisnis. */
export type SummaryGroup = {
  nama: string
  berkas: number
  nilai: string
  umur_di_atas_2_tahun: number
}

/** Jawaban `GET /api/inbox-os-claim-per-cabang/ringkasan`. */
export type SummaryResponse = {
  /** Kapan angkanya dibaca — waktu PEMBACAAN, bukan tanggal posisi data. */
  posisi: string

  total_berkas: number

  /** Teks desimal kanonik. */
  total_estimasi: string
  total_reserve_or: string

  /**
   * Membedakan "nol" dari "tidak terbaca".
   *
   * Rp 0 adalah jawaban yang sah dan memang lazim di sini — tabel treaty hampir tidak
   * memuat klaim PNC. Yang `false` berarti DB Link sedang bermasalah, dan itu menuntut
   * tindakan yang berbeda.
   */
  total_reserve_or_terbaca: boolean

  umur_di_atas_2_tahun: number

  sebaran_umur: AgeBucket[]
  per_cob: SummaryGroup[]
  per_sumber_bisnis: SummaryGroup[]

  cabang: Branch
  portal: string
}
