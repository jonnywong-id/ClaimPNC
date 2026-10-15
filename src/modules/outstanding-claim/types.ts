/**
 * Bentuk data layar Outstanding Claim — rincian klaim treaty proporsional.
 *
 * Di Pega layar ini BUKAN butir menu melainkan Flow Action `OutstandingClaim` pada kelas
 * `ASM-FW-GCNMFW-Work-ClaimTreaty`, yang dijalankan ketika pengguna mengklik nomor klaim di
 * Inbox Claim Treaty Prop. Sectionnya `Section/OutstandingClaim-Section.xml`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega — yang di layar ini berbentuk `.ClaimData.*` dan `.TreatyInMaster.*`.
 */

/** Satu isian skalar pada layar rincian. */
export type Field = {
  kunci: string
  judul: string

  /**
   * Isian digambar tetapi belum dapat diisi.
   *
   * Ia BUKAN isian yang disembunyikan. Delapan isian pada blok Treaty Information terikat ke
   * halaman `TreatyInMaster`, yang tidak punya rule pemuat di export — menyembunyikannya
   * membuat pengguna mengira isiannya hilang, menggambarnya dengan alasan membuat ia tahu
   * isiannya ada dan kenapa kosong.
   */
  terhalang?: boolean
}

/** Satu kolom grid. */
export type GridColumn = {
  kunci: string
  judul: string
}

/** Susunan satu tabel pada layar rincian. */
export type Grid = {
  kode: string
  judul: string
  kolom: GridColumn[]

  terhalang?: boolean
  alasan_terhalang?: string
  pemilik_penghalang?: string
}

/** Satu kelompok isian beserta grid yang digambar sesudahnya. */
export type Group = {
  kode: string
  judul: string
  isian: Field[]

  /**
   * Kode grid yang digambar sesudah isian kelompok ini.
   *
   * Susunan lengkap gridnya ada sekali saja di `LayoutResponse.grid` — menyalinnya ke tiap
   * kelompok membuat satu grid yang dipakai dua kelompok punya dua susunan yang dapat
   * berselisih.
   */
  grid: string[]
}

/** Jawaban GET /api/outstanding-claim/tata-letak. */
export type LayoutResponse = {
  kelompok: Group[]
  grid: Grid[]
  portal: string
}

/** Satu baris grid — peta dari kunci kolom ke teksnya. */
export type GridRow = Record<string, string>

/** Jawaban GET /api/outstanding-claim/{no_klaim}. */
export type DetailResponse = {
  no_klaim: string

  /**
   * Status alur kerja Pega — `PYSTATUSWORK`.
   *
   * Ia BUKAN Status Klaim berkode `1134`–`1166` milik master `V_STS_CLAIM`; keduanya konsep
   * berbeda (`D-18`), dan nilainya tidak diterjemahkan ke label master mana pun.
   */
  status_kerja: string

  operator_pengubah: string

  /**
   * Isian skalar, dikunci nama isian pada `Field.kunci`.
   *
   * Isian yang TIDAK ada kuncinya di sini berbeda artinya dari isian yang kuncinya ada
   * tetapi kosong: yang pertama berarti jalurnya tidak ditemukan di dokumen klaim, yang
   * kedua berarti datanya memang belum diisi. Layar menggambar keduanya sebagai tanda pisah
   * — pengguna tidak dapat berbuat apa pun dengan perbedaan itu — tetapi perbedaannya
   * terbawa di respons supaya dapat diperiksa.
   */
  isian: Record<string, string>

  /** Baris setiap grid, dikunci kode grid. */
  baris: Record<string, GridRow[]>

  portal: string
}
