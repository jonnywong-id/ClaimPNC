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

  /**
   * Dengan APA panel rincian dibuka pada daftar ini.
   *
   * KETIGA BELAS daftar punya tombol Detail — itu keadaan di layar lama. Yang berbeda
   * adalah kuncinya: `'pengajuan'` pada tujuh daftar yang barisnya pengajuan, `'klaim'`
   * pada enam daftar yang barisnya klaim dan tidak membawa ID pengajuan sama sekali.
   */
  kunci_rincian: DetailKey

  /**
   * Tombol Detail daftar ini membuka **form sunting**, bukan panel baca.
   *
   * Satu daftar saja: "Rejected Checker" berisi pengajuan yang checker kembalikan kepada
   * PIC, dan satu-satunya tindakan yang masuk akal di sana adalah menyunting lalu
   * mengirim ulang. Membukanya sebagai panel baca menjadikan daftar itu jalan buntu.
   */
  membuka_form_sunting: boolean
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

  /**
   * Pilihan **Mata Uang** pada form Tambah.
   *
   * Dibaca server dari `POOLDATA.CURRENCY` — tabel yang sama yang dibaca Report
   * Definition `SelectCurrency_RD`, sumber dropdown ini di layar lama.
   *
   * KOSONG bila tabelnya tidak terbaca. Itu bukan galat yang menutup layar: kolomnya
   * tetap digambar, hanya tanpa isi dan tanpa tanda wajib.
   */
  pilihan_mata_uang: StatusOption[]
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

/**
 * Satu baris grid "Detail History Salvage".
 *
 * Judul kolomnya di layar lama: Tanggal Input · Nomor Klaim · PIC · Nilai Minimum ·
 * Posisi Salvage.
 */
export type HistoryRow = {
  id_salvage: string
  tanggal_input: string
  no_klaim: string
  pic: string

  /** Berjudul "Nilai Minimum" di grid ini, "Estimasi" di panel rincian. Kolomnya sama. */
  nilai_minimum: string

  /**
   * Sudah berupa kalimat — "Sudah Aksep Checker", "Salvage Waive", dan seterusnya.
   *
   * Pemetaannya BERBEDA dari "Posisi Salvage" pada panel rincian, meski keduanya berasal
   * dari kolom yang sama.
   */
  posisi_salvage: string
}

/**
 * Satu dokumen yang BERHASIL tersimpan lewat modal "UploadDocument_Salvage".
 *
 * Isi berkas dan keterangannya hidup di dua tabel berbeda dengan `data_id` yang
 * berbeda pula; yang menautkan keduanya `image_id`. Keduanya dikirim supaya penelusuran
 * ke basis data dapat dimulai dari salah satunya.
 */
export type DokumenTersimpan = {
  data_id: string
  image_id: string

  /** Nama berkas yang BENAR-BENAR tersimpan — bukan nama aslinya. */
  nama_tersimpan: string

  /**
   * Baris penaut ke pengajuan salvage ikut ditulis.
   *
   * SALAH pada form pengajuan baru: di sana berkas diunggah sebelum Submit, sehingga
   * pengajuannya belum punya nomor. Dokumennya tetap menempel pada klaimnya.
   */
  tertaut_salvage: boolean
}

export type AttachDocumentsResponse = {
  dokumen: DokumenTersimpan[]
  pesan: string
  portal: string
}

/** Kunci yang dipakai membuka panel rincian. */
export type DetailKey = 'pengajuan' | 'klaim'

/**
 * Satu pilihan pada autocomplete **"Nama Object"** di form Tambah.
 *
 * Di layar lama ia satu baris `TempObjectData.pxResults`: yang dibaca pengguna hanya
 * namanya, dan `id`-nya ikut tersimpan diam-diam saat barisnya dipilih — disalin ke
 * isian tersembunyi `TempInsert.NewNoKTP`.
 */
export type ObjekPilihan = {
  /** `T_CLAIM_OBJECTLIST.OBJECTID` — berakhir di `PNC_SALVAGE.IDOBJECT`. */
  id: string
  nama: string
}

/**
 * Satu pilihan pada autocomplete **"Nama Coverage"**.
 *
 * Bentuknya sama persis dengan `ObjekPilihan`, dan di layar lama pun kedua kuerinya
 * kembar.
 *
 * # Daftar ini TIDAK terikat objek yang dipilih
 *
 * Kueri pemasoknya tidak mengambil `OBJECTID` sama sekali, sehingga seluruh coverage
 * milik klaim ditawarkan siapa pun objeknya. Akibatnya pasangan objek dan coverage yang
 * tersimpan dapat berasal dari objek yang berbeda — keadaan sistem lama yang dibawa apa
 * adanya (`P-5`), bukan sesuatu yang modul ini perkenalkan.
 */
export type CoveragePilihan = {
  /**
   * `T_CLAIM_OBJECTCOVERAGE.COVERAGEID` — berakhir di `PNC_SALVAGE.IDCOVERAGE`.
   *
   * Ia KODE JENIS jaminan, bukan penunjuk satu baris: kode yang sama dapat muncul dua
   * kali pada satu klaim bila dua objeknya punya jaminan yang sama.
   */
  id: string
  nama: string
}

/** Satu baris grid "Detail Pengajuan Salvage" pada panel Detail. */
export type DetailBarang = {
  nama_barang: string

  /**
   * Jumlah dan satuan datang TERPISAH.
   *
   * Kueri lama merangkainya di dalam SQL (`count(namabarang) || ' ' || satuan`), sehingga
   * jumlahnya berhenti menjadi angka dan tidak lagi dapat diratakan ke kanan. Di sini
   * keduanya dikirim sendiri-sendiri, dan layar yang merangkainya.
   */
  jumlah: number
  satuan: string

  total_nilai: string

  /** Sudah kalimat, bukan kode — "Terjual" / "Tidak terjual" / "Waiting approval". */
  status_terjual: string

  nama_pemenang: string
  no_akseptasi: string
  nilai_akseptasi: string
  remark: string
}

/**
 * Isi panel "Detail Salvage".
 *
 * Nama isian mengikuti judul di layar lama (`Section/DataDetail_Salvage-Section.xml`),
 * supaya petugas yang membandingkannya dengan Pega berdampingan membaca kata yang sama
 * (`D-13`).
 */
export type DetailResponse = {
  id_salvage: string
  no_klaim: string
  nama_bisnis: string

  /**
   * Klaim ini benar-benar punya pengajuan salvage.
   *
   * Selalu benar bila panel dibuka dari baris pengajuan. Dapat SALAH bila dibuka dari
   * baris klaim — dan pada daftar Salvage Outstanding ia justru yang lazim, sebab daftar
   * itu berisi klaim yang salvage-nya belum ditandai sama sekali.
   */
  ada_pengajuan: boolean

  /** Berasal dari KLAIM, bukan dari pengajuan. Terisi pada jalur kunci `'klaim'`. */
  pic: string
  tanggal_kejadian: string

  tanggal_input_salvage: string
  jenis_salvage: string
  quantity_salvage: string
  estimasi: string
  lokasi_salvage: string

  tanggal_transfer_ga: string

  /** Kode `STSTRANSFER` apa adanya — untuk penelusuran, bukan untuk dibaca. */
  kode_posisi_salvage: string
  /** Label posisinya — inilah yang digambar. */
  posisi_salvage: string

  tanggal_akseptasi: string
  no_akseptasi: string
  remark: string
  mata_uang: string

  nama_object: string
  nama_coverage: string

  /**
   * Penunjuk objek dan coverage pengajuan ini.
   *
   * Tidak digambar sebagai isian. Yang membacanya adalah form Tambah, untuk memilih ulang
   * baris yang sama pada kedua autocomplete.
   */
  id_object: string
  id_coverage: string

  nilai_salvage: string

  email: string
  nilai_penawaran: string
  nama_pemenang: string
  tanggal_lelang: string

  nama_pic_survey: string
  no_telp_pic_survey: string
  email_pic_survey: string

  lokasi_salvage_di_jabodetabek: boolean

  /**
   * Penanda pengajuan sebelum 17 Juli 2023.
   *
   * ARTINYA tidak diketahui — tidak ada satu pun rule di export yang memakainya selain
   * menggambarnya. Layar menyebutnya apa adanya tanpa menafsirkannya.
   */
  pengajuan_sebelum_juli_2023: boolean

  barang: DetailBarang[]

  /**
   * SELURUH pengajuan salvage milik klaim ini, terbaru lebih dulu.
   *
   * Digambar pada form "Menambahkan Data Salvage" sebagai grid "Detail History Salvage".
   * Kosong berarti klaim ini belum pernah diajukan salvage sama sekali.
   */
  riwayat: HistoryRow[]

  /**
   * Isi kedua autocomplete pada form "Menambahkan Data Salvage".
   *
   * Keduanya milik KLAIM, bukan milik pengajuan — sama seperti di layar lama, tempat
   * mengetik Nomor Klaim memuat keduanya sekaligus. Pada jalur `/pengajuan/{id}` keduanya
   * kosong, dan itu benar: panel rincian tidak menggambar form.
   */
  pilihan_objek: ObjekPilihan[]
  pilihan_coverage: CoveragePilihan[]

  portal: string
}
