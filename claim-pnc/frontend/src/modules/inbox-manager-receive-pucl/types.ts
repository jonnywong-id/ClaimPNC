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

  /** Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel. */
  selisih_terencana: string[]

  portal: string
}

/** Jawaban GET /api/inbox-manager-receive-pucl. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  portal: string
}
