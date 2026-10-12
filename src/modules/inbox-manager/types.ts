/**
 * Bentuk data layar Inbox Manager — menu `MENU_ID 58`, pengganti harness
 * `UserInbox_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega — yang di layar ini memang tidak layak diikuti: kueri pemasoknya memberi alias seperti
 * `BRANCHNAME` untuk sebuah pencacah dan `NOPOLIS` untuk sebuah jumlah uang.
 */

/** Jenis tab menentukan bagaimana isinya digambar. */
export type TabKind = 'dashboard' | 'ringkasan' | 'antrean'

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: string
  judul: string

  /** Kolom nilai uang — dirata-kanankan dan diformat sebagai rupiah. */
  uang?: boolean
}

/** Bentuk satu grid dashboard, tanpa barisnya. */
export type PanelShape = {
  kunci: string
  judul: string
  kolom: TabColumn[]
}

/**
 * Apa yang boleh dilakukan pada sebuah antrean.
 *
 * Ia datang dari SERVER, tidak disimpulkan layar dari kode tab. Aturan yang hidup di dua
 * tempat akan menyimpang, dan yang menyimpang di sini adalah tombol yang tampak dapat ditekan
 * tetapi selalu ditolak.
 */
export type DecisionRule = {
  dapat_diputuskan: boolean

  /** Kosong berarti tombol Setujui boleh digambar aktif. */
  alasan_setuju_ditahan?: string

  alasan_wajib_saat_menolak: boolean
  label_alasan?: string

  /**
   * Label tombol keputusan, apa adanya dari Pega.
   *
   * Kosong berarti antrean ini memang tidak punya tombol bernama di sana — Payment Klaim
   * Akseptasi dan Penolakan Klaim memakai isian di dalam grid lalu satu tombol simpan.
   * Layar memakai kata aplikasi sendiri pada keduanya.
   */
  label_setujui?: string
  label_tolak?: string

  /**
   * Antrean ini dapat diputuskan banyak baris sekaligus.
   *
   * Hanya tiga antrean punya jalur itu di Pega — yang gridnya diberi `Select All` dan
   * `Deselect All`. Pada enam sisanya keputusan diambil satu baris setiap kali, dan batas
   * itu ditegakkan SERVER; di sini ia hanya menjaga layar tidak menawarkan jalur yang akan
   * ditolak.
   */
  dapat_massal: boolean
}

/** Satu tab beserta bentuk isinya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  jenis: TabKind

  /**
   * Kode tab induk; kosong pada keempat tab tingkat atas.
   *
   * Layar TIDAK menyimpulkannya sendiri dari `jenis`. Jenjang tab dibaca dari export Pega —
   * `CountDashbroardManager` menulis sembilan pencacah terakhir sebagai ANAK baris keempat —
   * dan tempat pembacaan itu tercatat adalah server.
   */
  induk?: string

  /**
   * Tab ini muncul di BILAH TAB induknya.
   *
   * Tidak sama dengan `induk`. Pega menyimpan dua fakta berbeda: pohon pencacah menaruh
   * sembilan antrean di bawah Approval Master, sedangkan `Section/InboxManager_Section2`
   * hanya menyertakan DELAPAN — Penolakan Klaim terhitung di sana tetapi tidak dapat dibuka
   * dari bilah tabnya.
   */
  dalam_bilah_induk?: boolean

  /**
   * Judul sub-tab di dalam tab ini.
   *
   * Hanya satu tab punya: di Pega, isi "Payment Klaim Akseptasi" bukan sebuah grid melainkan
   * bilah sub-tab, dan sub-tab yang tampil hanya satu — "Approval Payment Akseptasi".
   * Kosong berarti isi tab langsung gridnya.
   */
  judul_sub_tab?: string

  /** Terisi pada tab dashboard. */
  panel?: PanelShape[]

  /** Terisi pada tab antrean. */
  kolom?: TabColumn[]

  keputusan: DecisionRule

  /** Kosong berarti tab tidak dibatasi lini bisnis. */
  lini_bisnis?: string

  punya_penyaring_periode: boolean

  /**
   * Label tombol yang MENERAPKAN penyaring periode.
   *
   * Berbeda tiap tab di Pega — "Cari" pada Produktivitas Klaim, "Lihat Data" pada Klaim.
   * Kosong berarti tab ini tidak punya penyaring periode.
   */
  label_terapkan_periode?: string

  /**
   * Tab ini punya blok "Export Data Detail Klaim" — ekspor berentang tanggal yang berdiri
   * sendiri, terpisah dari ekspor antrean di kepala layar.
   */
  punya_ekspor_detail?: boolean
}

/** Satu pilihan penyaring dashboard. Nilai kosong berarti "seluruhnya". */
export type FilterOption = {
  nilai: string
  label: string
}

/** Satu penyaring dashboard beserta pilihannya. */
export type Filter = {
  kunci: string
  label: string
  pilihan: FilterOption[]
}

/** Jawaban keterangan layar. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string

  /**
   * Lini bisnis pemanggil yang benar-benar terbaca dari basis data.
   *
   * Ia ditampilkan supaya penyelia melihat DASAR penyaringan angkanya. Tanpa itu, dua
   * penyelia yang membandingkan layar masing-masing akan melihat angka berbeda tanpa satu
   * pun petunjuk kenapa.
   */
  lini_bisnis_anda: string
}

/** Satu pencacah di kepala layar. */
export type Counter = {
  tab: string
  label: string
  jumlah: number

  /** Kosong pada pencacah tingkat atas. */
  induk?: string

  /**
   * Terisi bila sumbernya sedang tidak dapat dibaca.
   *
   * Saat terisi, `jumlah` TIDAK bermakna dan layar tidak boleh menggambarnya sebagai angka.
   * Angka nol yang sesungguhnya berarti "tidak terbaca" membuat penyelia mengira antreannya
   * kosong.
   */
  tidak_tersedia?: string
}

export type CountersResponse = {
  pencacah: Counter[]
}

/**
 * Satu sel grid dashboard.
 *
 * Ketiganya saling meniadakan: yang terisi hanya satu, sesuai jenis kolomnya. Nilai uang
 * datang sebagai TEKS presisi penuh — tidak pernah sebagai angka JSON, yang akan
 * melewatkannya lewat bilangan pecahan biner (`I-12`).
 */
export type Cell = {
  jumlah?: number
  nilai?: string
  teks?: string
}

/** Satu grid dashboard beserta isinya. */
export type Panel = {
  kunci: string
  judul: string
  kolom: TabColumn[]
  baris: Record<string, Cell>[]
}

/** Satu baris antrean. */
export type QueueRow = {
  /**
   * Kunci baris, dikirim kembali APA ADANYA saat keputusan dikirim.
   *
   * Layar TIDAK pernah menyusunnya sendiri: bentuknya berbeda tiap antrean, dan satu di
   * antaranya gabungan empat kolom.
   */
  kunci: string
  sel: Record<string, string>
}

export type Pagination = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Periode yang BENAR-BENAR dipakai, termasuk bila ia nilai bawaan. */
export type Period = {
  dari: string
  sampai: string
}

/** Jawaban isi satu tab. */
export type ListResponse = {
  tab: Tab

  /** Terisi pada tab dashboard. */
  panel?: Panel[]

  /**
   * Waktu cuplikan sumbernya terakhir disegarkan, pada dashboard yang sumbernya tabel
   * cuplikan. Ia BUKAN jam aplikasi.
   */
  disegarkan_pada?: string

  /**
   * Penyaring di bawah kedua grid pertama. Ia datang bersama DATA karena salah satu
   * pilihannya dibaca dari master yang dapat berubah tanpa deploy.
   */
  penyaring?: Filter[]

  /** Terisi pada tab antrean. */
  baris?: QueueRow[]
  paginasi?: Pagination

  /** Terisi pada tab yang punya penyaring periode. */
  periode?: Period
}

/** Keputusan yang dikirim penyelia. */
export type Verdict = 'setujui' | 'tolak'

export type DecisionRequest = {
  tab: string
  keputusan: Verdict
  kunci: string[]
  alasan: string
}

export type DecisionResponse = {
  diminta: number
  berubah: number

  /**
   * Jumlah baris yang TIDAK berubah karena sudah diputuskan lebih dulu.
   *
   * Ia selalu dikirim, termasuk saat nol. Layar tidak boleh menghitungnya sendiri: ia
   * menyangkut keputusan orang lain, dan itu hal terakhir yang boleh ditebak layar.
   */
  tidak_berubah: number

  pesan: string
}

/** Bentuk penyaring periode yang dipilih pengguna. */
export type PeriodMode = 'bulan' | 'rentang'

/** Isian penyaring periode di layar. */
export type PeriodInput = {
  bentuk: PeriodMode
  bulan: string
  dari: string
  sampai: string
}

/**
 * Nilai kedua penyaring dashboard Outstanding.
 *
 * Kosong berarti "All" — sama seperti pilihan bawaan di layar lama.
 */
export type DashboardFilterInput = {
  reinsurer: string
  kategori_os: string
}

/** Rentang tanggal blok "Export Data Detail Klaim". */
export type DetailExportInput = {
  dari: string
  sampai: string
}
