/**
 * Bentuk data layar Inbox Claim Treaty Non Prop — menu `MENU_ID 55`, pengganti harness
 * `InboxClaimNonProp_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxclaimtreatynonprop/inboxclaimtreatynonprop.go`: di
 * layar lama SELURUH kolom bernama `CARI` ditambah nomor urut, dan nomornya BERBEDA dari
 * layar Treaty Prop meski artinya sama — `CARI10` di sini kunci teknis, di sana Tanggal
 * Kejadian.
 */

/** Satu baris pekerjaan klaim treaty non-proporsional. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol rincian klaim.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  no_klaim: string

  /**
   * Master ID dari KOLOM objek kerja (`PC_ASM_FW_GCNMFW_WORK.MASTERID`).
   *
   * Ia BUKAN isian yang sama dengan `id_master` di bawah — lihat catatan di sana.
   */
  master_id: string

  /**
   * Master ID dari BLOB JSON klaim (`JSON_KLAIM.DATA_JSON` jalur `$.IDMaster`).
   *
   * Layar lama menampilkan keduanya berdampingan, dan keduanya diambil dari tempat yang
   * berbeda. Apakah isinya selalu sama belum pernah diperiksa (`R-08`), sehingga keduanya
   * dibawa apa adanya alih-alih dipilih salah satu.
   *
   * Hanya tab Admin yang memilikinya; kueri tab Teknik tidak mengambilnya sama sekali.
   */
  id_master: string

  no_polis: string

  /**
   * Tanggal Kejadian, dikirim APA ADANYA dari blob JSON.
   *
   * Teks, bukan tanggal: backend membacanya dengan `JSON_VALUE`, dan bentuk teks di dalam
   * blob itu tidak dapat diperiksa tanpa DDL (`R-08`). Layar memformatnya hanya bila
   * bentuknya memang `YYYY-MM-DD`; selain itu ditampilkan apa adanya.
   */
  tanggal_kejadian: string

  nama_bisnis: string
  sumber_bisnis: string
  ceding_co: string
  nama_tertanggung: string

  /**
   * Menyatakan ANTREAN, bukan status klaim.
   *
   * Ia teks tetap di dalam kueri — "Estimation" untuk seluruh baris tab Admin dan
   * "Acceptation" untuk seluruh baris tab Teknik — dan tidak membaca satu pun kolom
   * status. Tak satu pun dari kedua teks itu termasuk dalam 33 kode status klaim yang
   * sebenarnya (`R-06`).
   */
  status: string

  /**
   * Umur pekerjaan dalam HARI KALENDER sejak objek kerja dibuat.
   *
   * Angka, bukan teks: layar mengurutkannya dan menandai yang sudah lama menunggu.
   * Akhir pekan dan hari libur ikut terhitung, sehingga ia bukan TAT.
   */
  aging: number

  operator_pembuat: string
  operator_pengubah: string
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

  /** Barisnya hanya milik pemanggil — hanya tab Admin begitu. */
  hanya_milik_saya: boolean

  /** Checkbox "See All Claim" berlaku pada tab ini. */
  pakai_lihat_semua: boolean

  /**
   * Checkbox "See TBA Claim" berlaku pada tab ini.
   *
   * "TBA" berarti *to be advised*: klaim treaty yang sudah masuk tetapi nomor polisnya
   * belum terbit.
   */
  pakai_lihat_tba: boolean

  /**
   * Tab digambar tetapi belum dapat diisi.
   *
   * Ia BUKAN tab yang disembunyikan: tabnya tetap tampak supaya pengguna tahu fiturnya ada
   * dan terhalang, bukan mengira ia hilang.
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

/** Penyaring yang BENAR-BENAR dipakai server, belum tentu sama dengan yang dikirim. */
export type AppliedFilter = {
  lihat_semua: boolean
  lihat_tba: boolean
}

/** Jawaban GET /api/inbox-claim-treaty-non-prop/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string

  /** Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel. */
  selisih_terencana: string[]

  portal: string
}

/** Jawaban GET /api/inbox-claim-treaty-non-prop. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  penyaring: AppliedFilter
  portal: string
}

/** Isian penyaring yang dipegang layar. */
export type FilterForm = {
  tab: string
  lihatSemua: boolean
  lihatTBA: boolean
}

/** EMPTY_FILTER adalah keadaan awal penyaring, dipakai pula saat berpindah tab. */
export const EMPTY_FILTER: FilterForm = { tab: '', lihatSemua: false, lihatTBA: false }
