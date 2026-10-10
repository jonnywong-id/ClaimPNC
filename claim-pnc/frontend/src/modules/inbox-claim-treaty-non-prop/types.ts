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
   * Grid **Teknik** yang menggambarnya, di bawah judul "ID Master".
   */
  master_id: string

  /**
   * Master ID dari BLOB JSON klaim (`JSON_KLAIM.DATA_JSON` jalur `$.IDMaster`).
   *
   * Grid **Admin** yang menggambarnya, di bawah judul "MasterID" — satu kata tanpa spasi,
   * berbeda dari judul grid Teknik untuk hal yang terdengar sama.
   *
   * Kedua grid menggambar SATU kolom master id, dan masing-masing mengambilnya dari sumber
   * yang berbeda. Apakah isi keduanya selalu sama belum pernah diperiksa (`R-08`), sehingga
   * keduanya dibawa apa adanya alih-alih dipilih salah satu. Kueri tab Teknik bahkan tidak
   * mengambil yang dari blob JSON sama sekali.
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
   * Waktu objek kerja dibuat, digambar di bawah judul kolom **"Status"**.
   *
   * Judul itu menyesatkan sejak di Pega dan dipertahankan apa adanya (`D-13`): sel ke-12
   * grid Teknik berjudul "Status" tetapi terikat `CARI21`, yang di `GetInboxListCNP_SQL`
   * adalah `b.PXCREATEDATETIME`. Ia TIDAK ada hubungannya dengan 33 kode status klaim
   * (`R-06`).
   *
   * Bentuknya notasi internal Pega — `20240201T095612.955 GMT` — karena itulah yang selama
   * ini terbaca pengguna. Ia dikirim sebagai teks jadi; layar tidak memformatnya ulang.
   *
   * Hanya grid Teknik yang menggambarnya.
   */
  dibuat_pada: string

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

/** Satu antrean beserta bentuk gridnya. */
export type Tab = {
  kode: string

  /**
   * Teks PILIHAN pada dropdown pemilih antrean — "Treaty-In Admin", "Treaty-In Teknik".
   *
   * Jangan tertukar dengan `judul_grid`: keduanya tampil bersamaan dan berbunyi berbeda.
   */
  nama: string

  /**
   * Judul KONTAINER grid di bawah dropdown — "Work Treatyin Non Propotional Teknik".
   *
   * Kosong pada antrean yang terhalang: tidak ada grid yang digambar di sana.
   */
  judul_grid?: string

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
