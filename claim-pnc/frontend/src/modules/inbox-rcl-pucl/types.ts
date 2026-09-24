/**
 * Bentuk data layar Inbox RCL/PUCL — menu `MENU_ID 61`, pengganti harness
 * `RCLPUCL_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxrclpucl/inboxrclpucl.go`: hampir seluruh alias di
 * kueri lama menyesatkan — satu kolom yang sama dialiaskan "CABANG" di satu rule,
 * "NewTelpTertanggung" di rule lain, dan "NamaSurveyor" di rule ketiga, padahal isinya nama
 * tertanggung.
 */

/**
 * Satu baris pada grid.
 *
 * # Kenapa satu bentuk untuk ketiga tab
 *
 * Karena ketiganya memang menggambar kolom yang IDENTIK. Berbeda dari layar Inbox Manager
 * Receive / PUCL, tempat sembilan dari enam belas isian hanya berlaku pada salah satu tab,
 * di sini seluruh isian berlaku di ketiga tab.
 *
 * Yang berbeda antartab hanyalah JUDUL satu kolom, dan itu datang dari `Tab.kolom` —
 * bukan dari bentuk barisnya.
 */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol rincian.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  /** Kolom "Nomor Case". Judulnya memang begitu di layar lama, bukan "Nomor Klaim". */
  no_case: string

  no_polis: string
  nama_tertanggung: string

  /**
   * Kolom "Tanggal Masuk Inbox" — tanggal klaimnya DIKIRIM ke jalur RCL/PUCL.
   *
   * Ia BUKAN tanggal objek kerjanya dibuat, meski judulnya terbaca seperti itu. Keduanya
   * kolom yang berbeda dan dapat terpaut berbulan-bulan, karena sebuah klaim lahir jauh
   * sebelum ia masuk antrean ini.
   *
   * Yang dipakai MENGURUTKAN justru kolom yang satunya, yang tidak dikirim ke sini sama
   * sekali — sehingga tabel dapat terbaca tidak urut. Itu perilaku layar lama, dan
   * dinyatakan lewat `selisih_terencana`.
   */
  tanggal_masuk_inbox: string

  deskripsi_analyst: string

  /**
   * Jalur penanganan — "RCL" atau "PUCL". Kosong bila kode jalurnya tidak dikenali.
   *
   * # Perangkap penamaan
   *
   * Nama fieldnya `status_rcl_pucl` karena itulah judul kolomnya di layar ini. Di layar
   * Inbox Manager Receive / PUCL, judul yang SAMA menunjuk kolom basis data yang BERBEDA.
   * Jangan menyalin pemetaan satu layar ke layar lain.
   */
  status_rcl_pucl: string

  /**
   * Kolom "Tanggal Cetak Surat".
   *
   * SELALU kosong pada tab "Cetak Surat" — penyaring tab itu justru `belum dicetak`.
   * Kolomnya tetap digambar karena layar lama menggambarnya, dan tabnya membawa
   * `catatan` yang menjelaskannya.
   */
  tanggal_cetak_surat: string

  /**
   * Kolom "Lama Klaim", dikirim sebagai TEKS.
   *
   * Satuannya tidak diketahui: tidak satu pun kueri di export menghitungnya, dan tidak ada
   * DDL yang menyatakan tipenya (`R-08`). Menambahkan kata "hari" di layar berarti
   * menetapkan satuan yang belum pernah dipastikan.
   */
  lama_klaim: string

  /** Kolom "Status Kadaluarsa". Sumbernya BUKAN kolom yang dipakai menyaring tab pertama. */
  status_kadaluarsa: string
}

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type WorkItemField = keyof WorkItem

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: WorkItemField
  judul: string
}

/** Satu kolom berkas laporan harian. */
export type ReportColumn = {
  kunci: string
  judul: string
}

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  kolom: TabColumn[]

  /**
   * Tombol ekspor tab ini menghasilkan LAPORAN HARIAN berbasis rentang tanggal, bukan
   * salinan tabel.
   *
   * Hanya tab "Cetak Surat" begitu. Layar memakainya untuk menampilkan kedua isian tanggal
   * — dan untuk menjelaskan bahwa isian itu TIDAK menyaring tabel di bawahnya.
   */
  punya_laporan_rentang_tanggal: boolean

  /**
   * Keterangan yang berlaku pada tab ini saja, digambar di atas grid.
   *
   * Dua tab memilikinya: "Cetak Surat" menjelaskan kolom yang selalu kosong, dan
   * "Klaim MSIG" menjelaskan mengapa ia kemungkinan kosong.
   */
  catatan?: string

  /**
   * Tab digambar tetapi belum dapat diisi.
   *
   * Tidak ada yang begitu di layar ini. Tab "Klaim MSIG" kemungkinan KOSONG, tetapi itu
   * bukan hal yang sama — kosongnya adalah jawaban, bukan ketidakmampuan menjawab.
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

/** Jawaban GET /api/inbox-rcl-pucl/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string

  /**
   * Kolom berkas laporan harian.
   *
   * Dibaca layar untuk menyebutkan isi berkasnya SEBELUM diunduh — isinya berbeda dari
   * tabel yang sedang dilihat, dan perbedaan itu tidak boleh baru ketahuan setelah
   * berkasnya dibuka.
   */
  kolom_laporan: ReportColumn[]

  /** Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel. */
  selisih_terencana: string[]

  portal: string
}

/** Jawaban GET /api/inbox-rcl-pucl. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  portal: string
}

/** Rentang tanggal laporan harian, berbentuk `YYYY-MM-DD`. */
export type DateRange = {
  dari: string
  sampai: string
}
