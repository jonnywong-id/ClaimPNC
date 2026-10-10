/**
 * Bentuk data layar Inbox Manager Receive / PUCL — menu `MENU_ID 56`, pengganti harness
 * `ReceiveDoucument_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxmanagerreceivepucl/inboxmanagerreceivepucl.go`:
 * sebagian besar alias di kueri lama menyesatkan — satu kolom yang sama dialiaskan "CABANG"
 * di satu rule dan "NewTelpTertanggung" di rule lain, padahal isinya nama tertanggung.
 */

/**
 * Satu baris pekerjaan pada layar ini.
 *
 * # Kenapa satu bentuk untuk dua antrean yang berbeda kelas
 *
 * Karena keduanya digambar grid yang sama bentuknya, dan yang membedakan hanyalah kolom
 * mana yang terlihat — itu ditetapkan `Tab.kolom`, bukan oleh bentuk barisnya.
 *
 * Sembilan dari enam belas isian di bawah hanya berlaku pada SALAH SATU tab, dan kosong di
 * tab yang lain. Itu tidak pernah terlihat pengguna: kolomnya memang tidak digambar di sana.
 */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol rincian.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  /**
   * Nomor case yang dibaca pengguna.
   *
   * Judulnya BERBEDA antar tab — "CaseID" pada kedua tab Receive dan "Nomor Case" pada tab
   * RCL/PUCL — dan perbedaan itu datang dari server lewat `kolom`, bukan ditebak layar.
   */
  no_case: string

  no_polis: string

  /** Nomor klaim PNC yang terbit dari berkas ini. Hanya tab Receive; kosong bila berkasnya
   * belum diregistrasi menjadi klaim, dan itu keadaan yang sah. */
  no_klaim_pnc: string

  nama_tertanggung: string

  /**
   * Tanggal Kejadian, dikirim APA ADANYA.
   *
   * Teks, bukan tanggal: DDL tabel Pega tidak tersedia (`R-08`), sehingga bentuk teks
   * kolomnya tidak dapat diperiksa. Layar memformatnya hanya bila bentuknya memang
   * `YYYY-MM-DD`; selain itu ditampilkan apa adanya.
   */
  tanggal_kejadian: string

  /**
   * Jenis Klaim — "PA" atau "NONMBU".
   *
   * Ia DITURUNKAN dari Group Panel, bukan dibaca dari isian aslinya: properti aslinya
   * (`.ReceiveDocument.TypeOfClaim`) tidak punya kolom basis data. Kosong pada tab
   * RCL/PUCL, yang tidak menurunkannya sama sekali.
   */
  jenis_klaim: string

  nama_pengirim: string
  tanggal_terima_dokumen: string

  /**
   * SELALU kosong.
   *
   * Tidak ada kolom basis data untuknya di seluruh export Pega. Kolomnya tetap digambar
   * supaya isian yang belum terbawa terlihat, bukan tersamar sebagai layar yang sudah
   * setara.
   */
  jumlah_lembar_dokumen: string

  /** Tanggal Masuk Inbox. Digambar tab RCL/PUCL; dipakai kedua tab untuk mengurutkan. */
  tanggal_masuk_inbox: string

  deskripsi_analyst: string

  /** Jalur penanganan — "RCL" atau "PUCL". Kosong bila kode jalurnya tidak dikenali. */
  rcl_pucl: string

  /**
   * Status jalur RCL/PUCL.
   *
   * Ia BUKAN Status Klaim ber-33 kode `1134`–`1166` (`R-06`). Keduanya kolom berbeda pada
   * tabel yang sama, dan namanya hanya berbeda satu huruf.
   */
  status_rcl_pucl: string

  tanggal_cetak_surat: string

  /**
   * Lama Klaim, dikirim sebagai TEKS.
   *
   * Satuannya tidak diketahui: tidak satu pun kueri di export menghitungnya, dan tidak ada
   * DDL yang menyatakan tipenya (`R-08`). Menambahkan kata "hari" di layar berarti
   * menetapkan satuan yang belum pernah dipastikan.
   */
  lama_klaim: string

  status_kadaluarsa: string

  /**
   * Layar kerja di balik nomor case baris ini dapat dibuka.
   *
   * Pada tab RCL/PUCL ia tidak selalu benar: daftar dibaca dari antrean Pega, sedangkan
   * layar tujuannya dibaca dari tabel datar milik aplikasi yang pengisiannya belum tentu
   * mencakup seluruh klaim di antrean itu. Nomor case yang belum punya pasangan digambar
   * sebagai teks biasa — tautan yang pasti gagal lebih buruk daripada tidak ada tautan.
   */
  layar_klaim_siap: boolean
}

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type WorkItemField = keyof WorkItem

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: WorkItemField
  judul: string
}

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  kolom: TabColumn[]

  /**
   * Barisnya diambil dari antrean BERSAMA, bukan dari penugasan per orang.
   *
   * Hanya tab RCL/PUCL begitu. Layar memakainya untuk menjelaskan antrean yang kosong
   * menurut sebabnya.
   */
  antrean_bersama: boolean

  /**
   * Nomor case pada tab ini adalah TAUTAN yang membuka layar kerja penerimaan dokumen.
   *
   * Hanya tab Receive begitu. Ia datang dari SERVER, bukan disimpulkan layar dari kode tab,
   * supaya perilaku klik ditetapkan di tempat buktinya dibaca.
   */
  buka_layar_kerja: boolean

  /**
   * Nomor case pada tab ini adalah TAUTAN yang membuka layar kerja klaim RCL/PUCL.
   *
   * Hanya tab RCL/PUCL begitu. Keduanya tidak pernah benar bersamaan: satu sel membuka tepat
   * satu layar, dan kelas objek kerjanyalah yang menentukan layar mana.
   */
  buka_layar_klaim: boolean

  /**
   * Tab digambar tetapi belum dapat diisi.
   *
   * Tidak ada yang begitu di layar ini hari ini. Isian tetap dibaca supaya penambahan tab
   * terhalang kelak tidak menuntut suntingan di sini.
   */
  terhalang: boolean
  alasan_terhalang?: string
  pemilik_penghalang?: string
}

/** Keterangan halaman dari server. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban GET /api/inbox-manager-receive-pucl/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string
  portal: string
}

/** Jawaban GET /api/inbox-manager-receive-pucl. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  portal: string
}

/**
 * Satu isian pada layar kerja penerimaan dokumen — flow action `InputReceiveDocument`.
 *
 * Judulnya datang dari SERVER, bukan ditulis di sini, karena ia hasil pembacaan
 * `Section/InputReceiveDocument-Section.xml` dan tempat pembacaan itu tercatat adalah di
 * backend. Menyalin 36 judul ke sini berarti daftar yang sama hidup di dua tempat.
 */
export type DocumentField = {
  kunci: string
  judul: string

  /** Di Pega digambar sebagai kotak teks bertingkat, bukan satu baris. */
  bertingkat?: boolean

  /**
   * Isian digambar tetapi belum dapat diisi.
   *
   * Enam belas dari 36 isian begitu. Ia TIDAK disembunyikan: menyembunyikannya membuat
   * pengguna yang membandingkan layar ini dengan Pega mengira isiannya hilang.
   */
  terhalang?: boolean
  alasan_terhalang?: string
  pemilik_penghalang?: string
}

/** Satu kelompok isian, digambar sebagai satu panel. */
export type DocumentFieldGroup = {
  judul: string
  isian: DocumentField[]
}

/**
 * Satu tombol yang di layar lama MENGUBAH data.
 *
 * Belum satu pun dapat dihidupkan — lihat `WriteAction` di backend. `pemilik` menyebut modul
 * yang kelak memilikinya, supaya layar dapat menjawab lebih dari "belum tersedia".
 */
export type DocumentAction = {
  kode: string
  label: string
  activity_pega: string
  pemilik: string

  /**
   * Tombol ini digambar DI ATAS isian, seperti di layar lama.
   *
   * Datang dari server karena letaknya dibaca dari section Pega — di sanalah buktinya.
   */
  di_atas: boolean
}

/** Jawaban GET /api/inbox-manager-receive-pucl/dokumen/{referensi}. */
export type DocumentResponse = {
  referensi: string
  no_case: string
  no_klaim_pnc: string
  jenis_klaim: string
  status_kerja: string

  kelompok: DocumentFieldGroup[]

  /**
   * Isi tiap isian, berkunci `DocumentField.kunci`.
   *
   * Isian bertanda `terhalang` sengaja TIDAK ada di sini. Mengisinya dengan teks kosong akan
   * membuat layar tidak dapat membedakan "belum ada sumbernya" dari "sumbernya ada tetapi
   * kosong" — dua hal yang tindak lanjutnya berbeda.
   */
  nilai: Record<string, string>

  tindakan: DocumentAction[]

  portal: string
}
