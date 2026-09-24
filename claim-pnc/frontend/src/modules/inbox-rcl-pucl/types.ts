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

/**
 * Bagian "Lampiran Surat" pada layar kerja RCL/PUCL.
 *
 * # Tiga isiannya DITURUNKAN, bukan disimpan
 *
 * `SetDataLampiranSuratRCLPUCL_Act` mengisinya dari anak-anak klaim — objek pertama dan
 * adjustment pertamanya. Klaim tanpa objek karena itu menghasilkan ketiganya KOSONG, dan
 * itu keadaan yang sah.
 */
export type LetterDraft = {
  /** Jalur penanganan — "RCL" atau "PUCL". Kosong bila kodenya tidak dikenali. */
  rcl_pucl: string

  /**
   * Kode jalur MENTAH.
   *
   * Dibaca layar untuk membedakan "kode tidak dikenali" dari "kode memang kosong" — dan
   * khususnya untuk mengenali nilai `3`, yang di Pega MENYEMBUNYIKAN seluruh layar ini.
   */
  kode_rcl_pucl: string

  deskripsi_analyst: string
  no_polis: string
  tanggal_kejadian: string

  /** Diturunkan dari nama objek pertama. */
  nama_peserta: string

  /**
   * "UP" (Uang Pertanggungan) — diturunkan dari SUMBER YANG SAMA dengan `nama_peserta`.
   *
   * Kedua penetapan di `SetDataLampiranSuratRCLPUCL_Act` menunjuk ekspresi yang sama persis
   * (`ObjectList(1).ObjectName`), sehingga isian ini berisi NAMA OBJEK — bukan angka.
   *
   * Itu terbaca seperti salin-tempel yang keliru, dan sempat "diperbaiki" menjadi nilai
   * pertanggungan pada 2026-09-24. Work Owner MERALATNYA hari itu juga: UP memang
   * ObjectName. Perbaikannya dicabut, dan `P-5` berlaku apa adanya.
   */
  up: string

  /** Diturunkan dari `PROPOSE_VALUE` adjustment pertama pada objek pertama. */
  jumlah_tagihan: string
}

/**
 * Bagian "Penerimaan Dokumen".
 *
 * Hanya satu isiannya punya kolom yang diketahui. Sisanya tidak dikirim sama sekali —
 * yang menjelaskan ketiadaannya adalah `isian_belum_terpetakan`.
 */
export type DocumentReceipt = {
  /**
   * Judulnya di layar **"Catatan untuk Analyst"**, bukan "Komentar PUCL".
   *
   * Nama isian JSON-nya tetap `komentar_pucl` karena ia KONTRAK, bukan nama yang dilihat
   * pengguna (`D-80`). Yang mengikuti section adalah judul di layar.
   */
  komentar_pucl: string
}

/** Jawaban GET /api/inbox-rcl-pucl/klaim/{referensi}. */
export type ClaimDetailResponse = {
  referensi: string
  no_case: string

  lampiran_surat: LetterDraft
  penerimaan_dokumen: DocumentReceipt

  /**
   * Isian layar lama yang tersimpan di CLIPBOARD Pega, bukan sebagai kolom tabel.
   *
   * Work Owner menjelaskan 2026-09-24: kesembilannya properti clipboard pada objek kerja
   * (`.ClaimData.PUCLStatus.NIK` dan seterusnya). Properti clipboard yang tidak dioptimasi
   * tidak punya kolom sendiri — jadi ini BUKAN daftar "kolom yang belum ditemukan".
   *
   * Datang dari server sebagai kalimat siap baca, bukan nama properti Pega — yang
   * membacanya petugas klaim.
   */
  isian_belum_terpetakan: string[]

  /** Layar ini di Pega adalah layar TULIS; di sini baca saja. */
  tindakan_masih_di_pega: boolean

  portal: string
}

/** Rentang tanggal laporan harian, berbentuk `YYYY-MM-DD`. */
export type DateRange = {
  dari: string
  sampai: string
}
