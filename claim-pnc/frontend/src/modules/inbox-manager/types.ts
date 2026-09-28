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
}

/** Satu tab beserta bentuk isinya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  jenis: TabKind

  /** Terisi pada tab dashboard. */
  panel?: PanelShape[]

  /** Terisi pada tab antrean. */
  kolom?: TabColumn[]

  keputusan: DecisionRule

  /** Kosong berarti tab tidak dibatasi lini bisnis. */
  lini_bisnis?: string

  punya_penyaring_periode: boolean
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

  selisih_terencana: string[]
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
