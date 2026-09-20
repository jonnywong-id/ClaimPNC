/**
 * Bentuk data layar Inbox Admin — menu `MENU_ID 63`, pengganti harness `PNCInboxAdmin`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxadmin/inboxadmin.go`: di layar ini satu properti
 * Pega berarti hal yang BERBEDA dari tab ke tab, karena kedelapan grid berbagi satu halaman
 * klipboard yang sama.
 */

/** Satu baris pekerjaan. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol "Lihat Detail Klaim".
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  case_id: string
  no_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  sumber_bisnis: string
  nama_cabang: string
  cabang_klaim: string
  pembuat: string
  tanggal_kejadian: string | null
  tanggal_lapor: string | null
  tanggal_input: string | null
  catatan: string
  posisi_klaim: string
  status_klaim: string
  status_lod: string

  tanggal_request: string | null
  cabang_polis: string
  cabang_survei: string
  pic_klaim: string
  surveyor: string
  no_survei: string

  tanggal_masuk_inbox: string | null
  deskripsi_analis: string
  status_rcl_pucl: string
  tanggal_cetak_surat: string | null
  lama_klaim: string
  status_kadaluarsa: string

  aging_lapor: number | null
  aging_total: number | null
  aging_lod: number | null
  aging_request: number | null
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

  /** Barisnya hanya milik pemanggil — tiga tab begitu. */
  hanya_milik_saya: boolean

  /** Kotak cari berlaku pada tab ini. */
  pakai_pencarian: boolean

  /** Dropdown lini bisnis berlaku pada tab ini. */
  pakai_lini_bisnis: boolean
}

/** Satu pilihan pada dropdown lini bisnis. */
export type BusinessLine = {
  kode: string
  label: string
}

/** Satu tab sistem lama yang sengaja tidak dibangun. */
export type DisabledTab = {
  kode: string
  nama: string
  alasan: string
}

/** Jawaban `GET /api/inbox-admin/tab`. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string
  lini_bisnis: BusinessLine[]
  tab_dinonaktifkan: DisabledTab[]
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
}

/** Penyaring yang BENAR-BENAR dipakai server. */
export type AppliedFilter = {
  bisnis: string
  cari: string
}

/** Jawaban `GET /api/inbox-admin`. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  penyaring: AppliedFilter
  portal: string
}

/** Keadaan penyaring di layar. */
export type FilterForm = {
  tab: string
  bisnis: string
  cari: string
}

/**
 * Keadaan awal penyaring.
 *
 * `tab` sengaja kosong, bukan diisi kode tab bawaan: server yang menetapkan tab mana yang
 * terbuka pertama (`tab_bawaan`), dan menuliskannya di sini berarti nilai yang sama hidup di
 * dua tempat dan dapat berselisih.
 */
export const EMPTY_FILTER: FilterForm = { tab: '', bisnis: '', cari: '' }

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const InboxAdminError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
} as const
