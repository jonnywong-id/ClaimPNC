/**
 * Bentuk data layar Inbox Claim Treaty Prop — menu `MENU_ID 54`, pengganti harness
 * `InboxClaimTreaty_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxclaimtreatyprop/inboxclaimtreatyprop.go`: di layar
 * lama SELURUH kolom bernama `CARI` ditambah nomor urut, sehingga tidak satu pun namanya
 * menyatakan isinya — dan nomornya pun tidak konsisten antar kueri.
 */

/** Satu baris pekerjaan klaim treaty proporsional. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol rincian klaim.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  claim_id: string
  id_master: string
  no_polis: string

  /**
   * Tanggal Kejadian, dikirim APA ADANYA dari `DATA_JSONBLOB`.
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

  /** Hanya terisi pada tab Work Teknik Treatyin — hanya kueri itu yang membawanya. */
  subjectivity: string
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

  /** Barisnya hanya milik pemanggil — hanya tab pertama begitu. */
  hanya_milik_saya: boolean

  /** Checkbox "See All Claim" berlaku pada tab ini. */
  pakai_lihat_semua: boolean

  /**
   * Tab digambar tetapi belum dapat diisi.
   *
   * Ia BUKAN tab yang disembunyikan: keputusan Work Owner 2026-09-21 menetapkan tabnya
   * tetap tampak supaya pengguna tahu fiturnya ada dan terhalang, bukan mengira ia hilang.
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
}

/** Jawaban GET /api/inbox-claim-treaty-prop/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string

  /** Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel. */
  selisih_terencana: string[]

  portal: string
}

/** Jawaban GET /api/inbox-claim-treaty-prop. */
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
}

/** EMPTY_FILTER adalah keadaan awal penyaring, dipakai pula saat berpindah tab. */
export const EMPTY_FILTER: FilterForm = { tab: '', lihatSemua: false }
