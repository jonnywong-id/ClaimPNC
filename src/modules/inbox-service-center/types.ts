/**
 * Bentuk data layar Inbox Service Center — menu `MENU_ID 46`, pengganti harness
 * `InboxServiceCenter`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxservicecenter/inboxservicecenter.go`: di layar ini
 * alias Pega nyaris seluruhnya menyesatkan — `DateOfLoss` berisi tanggal input, `UserName`
 * berisi nama nasabah, dan `NoKTP` pada satu kueri justru berisi IMEI.
 */

/** Satu baris klaim portal rekanan. */
export type ServiceClaim = {
  id: string
  tanggal_input: string | null
  no_polis: string
  nasabah: string
  tipe: string
  pic: string

  /**
   * Ketiganya TIDAK digambar sebagai kolom — grid Pega hanya punya enam.
   *
   * `repair_id` dibawa untuk membuka rincian kelak; `no_klaim` dan `imei` dibawa karena
   * keduanya ikut dicari kotak "Cari", sehingga pengguna dapat melihat mengapa sebuah
   * baris cocok.
   */
  repair_id: string
  no_klaim: string
  imei: string

  /** Kode dan labelnya dikirim bersamaan; labelnya disusun server dari rule Pega. */
  status_perbaikan: string
  status_perbaikan_label: string
  status_persetujuan: string
  status_persetujuan_label: string
}

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type ServiceClaimField = keyof ServiceClaim

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: ServiceClaimField
  judul: string
}

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string

  /**
   * Nilai `stsapprove` sistem lama untuk tab ini.
   *
   * Ia dikirim demi ketelusuran dan TIDAK digambar. Tab pertama bernilai teks kosong —
   * itulah sebab kode tab pada kontrak API tidak memakai nilai Pega apa adanya.
   */
  parameter_pega: string

  kolom: TabColumn[]
}

/** Jawaban `GET /api/inbox-service-center/tab`. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string
  portal: string

  /**
   * Hal yang belum berjalan penuh beserta alasannya, siap ditampilkan apa adanya.
   *
   * Ia datang dari server supaya hilang dengan sendirinya begitu penghalangnya hilang —
   * tanpa menyunting layar.
   */
  keterbatasan: string[]
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number

  /**
   * Hasilnya memang dipotong per halaman.
   *
   * Bernilai `false` saat mencari: sistem lama mematikan paginasinya sendiri begitu kotak
   * cari terisi, dan perilaku itu ditiru apa adanya (`P-5`). Layar memakainya untuk
   * menyembunyikan bilah halaman alih-alih menggambar tombol yang tidak melakukan apa pun.
   */
  aktif: boolean
}

/** Penyaring yang BENAR-BENAR dipakai server. */
export type AppliedFilter = {
  cari: string
}

/** Jawaban `GET /api/inbox-service-center`. */
export type ListResponse = {
  tab: Tab
  baris: ServiceClaim[]
  paginasi: PageInfo
  penyaring: AppliedFilter
  portal: string
}

/** Keadaan penyaring di layar. */
export type FilterForm = {
  tab: string
  cari: string
}

/**
 * Keadaan awal penyaring.
 *
 * `tab` sengaja kosong, bukan diisi kode tab bawaan: server yang menetapkan tab mana yang
 * terbuka pertama (`tab_bawaan`), dan menuliskannya di sini berarti nilai yang sama hidup di
 * dua tempat dan dapat berselisih.
 */
export const EMPTY_FILTER: FilterForm = { tab: '', cari: '' }

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const InboxServiceCenterError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
} as const

// ─────────────────────────────────────────────────────────────────────────────
// Layar rincian
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Seluruh isian layar rincian, 83 buah.
 *
 * # Kenapa Record, bukan tipe dengan 83 field yang disebut satu per satu
 *
 * Karena layar ini TIDAK menggambar isian yang disebutnya sendiri — ia menggambar apa yang
 * disebut server pada `kelompok`. Menyebut ke-83 nama di sini berarti daftar yang sama hidup
 * di dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Bentuknya akurat: setiap isian pada kontrak API modul ini bertipe teks, dan yang boleh
 * kosong bertipe `null` — termasuk seluruh kolom tanggal, yang dikirim sebagai `YYYY-MM-DD`.
 */
export type ClaimDetail = Record<string, string | null>

/**
 * Satu isian pada layar rincian.
 *
 * Bentuknya sama dengan `TabColumn` — kunci dan judul — tetapi tipenya TERPISAH, dan itu
 * disengaja: `TabColumn.kunci` dibatasi ke isian yang benar-benar ada pada satu baris grid,
 * sedangkan kunci di sini menunjuk salah satu dari 83 isian rincian. Menyatukan keduanya
 * berarti melonggarkan pengetikan grid yang justru menangkap salah ketik nama kolom.
 */
export type FieldRef = {
  kunci: string
  judul: string
}

/** Satu kelompok isian pada layar rincian. */
export type FieldGroup = {
  kode: string
  judul: string
  isian: FieldRef[]
}

/** Satu catatan progres pada riwayat klaim. */
export type ProgressNote = {
  tanggal: string | null
  catatan: string
  oleh: string
}

/** Jawaban `GET /api/inbox-service-center/{id}`. */
export type DetailResponse = {
  klaim: ClaimDetail
  riwayat_progres: ProgressNote[]
  kelompok: FieldGroup[]
  portal: string
}
