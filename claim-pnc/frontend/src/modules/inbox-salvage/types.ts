/**
 * Bentuk data layar Inbox Salvage — menu `MENU_ID 71`, pengganti harness `InboxSalvage`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxsalvage/inboxsalvage.go`, dan di layar ini ia lebih
 * tajam daripada di modul mana pun: kueri lama mengalihnamakan SELURUH kolomnya menjadi
 * nama properti klaim yang sudah ada, dan dua di antaranya menyatakan hal yang SALAH —
 * `"ClaimNo"` yang sebenarnya ID salvage, dan `"DateOfLoss"` yang sebenarnya tanggal input.
 */

/**
 * Satu baris pada grid mana pun.
 *
 * # Kenapa satu bentuk untuk tiga belas daftar yang kolomnya berbeda
 *
 * Karena yang menentukan kolom mana yang digambar adalah `Tab.kolom`, bukan bentuk barisnya.
 * Isian yang tidak berlaku pada sebuah daftar bernilai kosong, dan itu tidak pernah terlihat
 * pengguna: kolomnya memang tidak digambar di sana.
 */
export type SalvageRow = {
  /**
   * Kunci baris, dibutuhkan tombol rincian.
   *
   * Isinya BERBEDA menurut keluarga daftarnya: nomor klaim pada enam daftar berbasis
   * klaim, ID salvage pada ketujuh daftar berbasis pengajuan. Keenam daftar pertama
   * membaca klaim yang BELUM punya baris salvage sama sekali.
   */
  referensi: string

  no_klaim: string
  id_salvage: string

  /**
   * Kolom "Tanggal Input" — tanggal PENGAJUAN salvage dibuat.
   *
   * Bukan tanggal kejadian, meski alias kueri lama menyebutnya begitu.
   */
  tanggal_input: string

  /** Kolom "Tgl Kejadian". Hanya daftar Salvage Outstanding menggambarnya. */
  tanggal_kejadian: string

  pic: string

  /**
   * Kolom "COB" pada daftar Salvage Outstanding, "Lokasi" pada kelima daftar berbasis
   * klaim lainnya.
   *
   * Kolomnya SAMA — nama lini bisnis. Judul "Lokasi" keliru di sistem lama dan dibawa apa
   * adanya (`D-13`). Jangan tertukar dengan `lokasi_salvage`, yang benar-benar lokasi.
   */
  nama_bisnis: string

  nama_objek: string
  jenis_salvage: string
  lokasi_salvage: string

  /** Kolom "Status Lelang" — "Terjual" atau "Belum Terjual". */
  status_lelang: string

  /**
   * Nilai uang datang sebagai TEKS, bukan angka.
   *
   * `D-51` menetapkan nilai uang disimpan presisi penuh dan hanya dibulatkan saat
   * ditampilkan. Mengirimnya sebagai angka JSON berarti melewatkannya lewat bilangan
   * pecahan biner JavaScript, yang membulatkannya sebelum pembulatan yang disengaja sempat
   * terjadi.
   */
  nilai_pengajuan_pic: string
  nilai_request_balai_lelang: string

  email: string
  keterangan_pic: string

  /** Kolom "Tipe Pengajuan" — "Pengajuan Baru" atau "Request Balai Lelang". */
  tipe_pengajuan: string

  /**
   * Kolom "Aging", berbentuk `"N day"`.
   *
   * Satuannya HARI KALENDER, bukan hari kerja seperti di Pega — lihat
   * `selisih_terencana`. Kosong berarti tanggal inputnya tidak terbaca, dan itu BERBEDA
   * dari nol hari.
   */
  aging: string

  /**
   * Kolom "Catatan". SELALU kosong, dan itu keadaan di Pega pula — tidak ada satu pun
   * penulisnya di jalur pemuat daftar.
   */
  catatan: string
}

/** Satu kolom grid, sebagaimana dikirim server. */
export type TabColumn = {
  kunci: string
  judul: string
  /** Ratakan ke kanan dan format sebagai angka. */
  angka: boolean
}

/** Satu daftar beserta kolom dan keterangannya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  kolom: TabColumn[]

  /** Judul kotak pencarian; kosong berarti daftar ini tidak punya. */
  label_pencarian: string

  /**
   * Pencarian daftar ini COCOK PERSIS, bukan mengandung.
   *
   * Dipakai menjelaskan ke pengguna mengapa separuh nomor klaim tidak menghasilkan
   * apa-apa — keterangan yang TIDAK ada di Pega, dan yang ketiadaannya membuat perilaku
   * ini terbaca sebagai kerusakan.
   */
  pencarian_cocok_persis: boolean

  /** Keterangan yang berlaku pada daftar ini saja. */
  catatan_daftar?: string
}

/** Satu pilihan daftar "Status Salvage" pada form Tambah. */
export type StatusOption = {
  kode: string
  label: string
}

/** Keterangan layar — bentuknya, bukan isinya. */
export type MetadataResponse = {
  daftar: Tab[]
  daftar_bawaan: string
  pilihan_status_salvage: StatusOption[]
  kolom_berkas_unggahan: string[]
  selisih_terencana: string[]
  portal: string
}

/** Keterangan paginasi yang BENAR-BENAR dipakai server. */
export type Paging = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Isi satu daftar. */
export type ListResponse = {
  daftar: Tab
  baris: SalvageRow[]
  paginasi: Paging
  /** Kata kunci yang BENAR-BENAR dipakai; tidak selalu sama dengan yang dikirim. */
  cari: string
  portal: string
}

/**
 * Satu baris tabel ringkas "Status Salvage / Jumlah".
 *
 * `daftar` kosong berarti barisnya TIDAK menuju daftar mana pun — dan satu baris memang
 * begitu, karena di Pega pun tidak ada tab yang menerima kodenya.
 */
export type StatusCount = {
  status_salvage: string
  jumlah: number
  daftar?: string
}

export type CountsResponse = {
  baris: StatusCount[]
  portal: string
}

/** Satu baris grid "Detail Item Salvage" pada form Tambah. */
export type DetailItem = {
  nama_item: string
  jumlah_item: string
  satuan: string
  remark: string
}

/**
 * Badan permintaan simpan pengajuan salvage.
 *
 * Nama isian mengikuti judul pada `Section/TambahData_Salvage-Section.xml` (`D-13`), bukan
 * nama kolom basis data — kolomnya beralias menyesatkan dan `D-19` melarang membawanya ke
 * kontrak.
 */
export type CreateRequest = {
  mode: string
  id_salvage: string

  nomor_klaim: string
  id_object: string
  nama_object: string
  id_coverage: string
  nama_coverage: string

  tanggal_input: string
  jenis_salvage: string
  status_salvage: string
  lokasi_salvage: string
  lokasi_salvage_di_jabodetabek: boolean
  mata_uang: string
  minimum_salvage: string
  quantity_salvage: string
  nilai_penawaran: string
  share_tertanggung: string
  remark: string
  email: string

  nama_pic_survey: string
  no_telp_pic_survey: string
  email_pic_survey: string

  detail_item_salvage: DetailItem[]
}

export type CreateResponse = {
  id_salvage: string
  jumlah_detail_item: number
  pesan: string
  portal: string
}

/**
 * Hasil pembacaan berkas "Upload Detail Salvage".
 *
 * Ia TIDAK menyimpan apa pun — hanya mengisi tabel di dalam form. Penyimpanan baru terjadi
 * saat pengguna menekan Submit, sama seperti di Pega.
 */
export type UploadResponse = {
  detail_item_salvage: DetailItem[]
  pesan: string
}
