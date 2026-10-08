/**
 * Bentuk data layar Acceptation Claim — akseptasi klaim treaty NON-proporsional.
 *
 * Di Pega layar ini BUKAN butir menu melainkan Flow Action `InputAcceptation` pada kelas
 * `ASM-FW-GCNMFW-Work-ClaimTreatyNonProp`, yang dijalankan ketika pengguna mengklik nomor
 * klaim di Inbox Claim Treaty Non Prop. Sectionnya `Section/InputAcceptation-Section.xml`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega — yang di layar ini berbentuk `.ClaimData.*`, `.TreatyInMaster.*`, dan `.CNP*`.
 *
 * # Bentuk layar dan ISINYA datang bersama
 *
 * Berbeda dari modul Outstanding Claim, yang memisahkannya menjadi `/tata-letak` dan
 * `/{no_klaim}`. Layar ini selalu dibuka untuk satu klaim tertentu — tidak ada keadaan di mana
 * bentuknya dibutuhkan tanpa isinya — sehingga memisahkannya hanya menambah satu perjalanan
 * dan membuka kemungkinan keduanya menjawab keadaan yang berbeda.
 */

/** Satu isian skalar beserta nilainya. */
export type Field = {
  kunci: string
  judul: string

  /** Isi yang digambar. Kosong bila isiannya tidak ada di dokumen klaim. */
  nilai: string

  /** Isian dapat diubah petugas dan boleh ikut dikirim saat Submit. */
  dapat_diubah: boolean

  /**
   * Isian digambar tetapi belum dapat diisi.
   *
   * Ia BUKAN isian yang disembunyikan. Sembilan isian pada blok Treaty Information terikat ke
   * halaman `TreatyInMaster` dan `OfferFacIn`, yang tidak punya rule pemuat di export —
   * menyembunyikannya membuat pengguna mengira isiannya hilang, menggambarnya dengan alasan
   * membuat ia tahu isiannya ada dan kenapa kosong.
   */
  terhalang: boolean
  alasan_terhalang?: string
  pemilik_penghalang?: string
}

/** Satu kolom grid. */
export type GridColumn = {
  kunci: string
  judul: string
  dapat_diubah: boolean
}

/** Satu baris grid — peta dari kunci kolom ke teksnya. */
export type GridRow = Record<string, string>

/** Satu tabel beserta barisnya. */
export type Grid = {
  kode: string
  judul: string
  kolom: GridColumn[]
  baris: GridRow[]

  terhalang: boolean
  alasan_terhalang?: string
  pemilik_penghalang?: string
}

/** Satu kelompok isian beserta grid yang digambar sesudahnya. */
export type Group = {
  kode: string
  judul: string
  isian: Field[]
  tabel: Grid[]
}

/** Jawaban GET /api/input-acceptation/{no_klaim}. */
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

  kelompok: Group[]
  portal: string
}

/** Muatan POST /api/input-acceptation/{no_klaim}. */
export type SubmitRequest = {
  /** Isian skalar yang diubah, dikunci nama isian. */
  isian: Record<string, string>

  /**
   * Baris grid yang diubah, dikunci kode grid.
   *
   * Baris dikirim UTUH per grid — bukan sebagai selisih — karena urutan baris bermakna di
   * layar ini, dan selisih tanpa urutan tidak dapat diterapkan kembali dengan pasti.
   */
  tabel: Record<string, GridRow[]>
}

/** Jawaban Submit yang berhasil. */
export type SubmitResponse = {
  no_klaim: string
  pesan: string
}
