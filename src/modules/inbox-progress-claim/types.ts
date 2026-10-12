/**
 * Bentuk data layar Inbox Progress Claim — menu `MENU_ID 65`, pengganti harness
 * `ProgressClaim_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan alias Pega.
 * Alasannya ada di `internal/inboxprogressclaim/inboxprogressclaim.go`: di layar ini nomor
 * klaim dan nomor polis bahkan TERTUKAR aliasnya — `CaseID` berisi nomor klaim sementara
 * `ClaimNo` berisi nomor polis.
 *
 * **Judul yang dibaca pengguna tetap memakai alias Pega**, dan itu keputusan terpisah
 * (Work Owner 2026-09-21). Judul datang dari server lewat `kolom[].judul`; arti sebenarnya
 * ikut dikirim lewat `kolom[].keterangan`.
 */

/** Satu baris klaim pada bagian Outstanding dan Next Follow Up. */
export type ClaimRow = {
  no_klaim: string
  no_polis: string
  nama_tertanggung: string
  tanggal_registrasi: string | null
  tanggal_kejadian: string | null
  catatan_lgb: string
  pic_klaim: string

  /**
   * Keempat isian berikut digabung dari SELURUH posisi yang sedang berjalan, dipisah koma.
   *
   * Padanannya positional: nilai ke-n pada `posisi` sepadan dengan nilai ke-n pada
   * `status_progres_1`, `status_progres_2`, dan `next_follow_up`. Itu bentuk sistem lama —
   * satu klaim dapat berada di beberapa posisi sekaligus.
   */
  posisi: string
  status_progres_1: string
  status_progres_2: string
  next_follow_up: string

  /** Banyaknya posisi yang sedang berjalan. */
  jumlah_posisi: number

  follow_up_terawal: string | null
  tanggal_proses: string | null
  prod_ke: string
}

/** Satu baris rekap pada bagian Progress Klaim per PIC. */
export type PICRow = {
  pic: string
  jumlah_klaim: number
  jumlah_pembaruan: number
  jatuh_tempo_hari_ini: number
  tepat_waktu: number
  terlambat: number
}

/**
 * Satu baris apa pun bentuknya.
 *
 * Kedua bentuk dilayani satu endpoint karena layar tidak tahu bagian mana yang tersedia
 * sampai ia membaca metadata dari server. Yang membedakan cara menggambarnya adalah
 * `Section.bentuk`.
 */
export type Row = ClaimRow | PICRow

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type RowField = keyof ClaimRow | keyof PICRow

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type Column = {
  /**
   * Identitas kolom, unik di dalam satu bagian.
   *
   * Ia terpisah dari `isian` karena bagian Outstanding menggambar SATU isian di DUA
   * kolom — `DateForAging` memang muncul dua kali di sistem lama.
   */
  kunci: string

  /** Isian mana pada baris yang digambar. */
  isian: RowField

  /** Judul yang dibaca pengguna — alias Pega apa adanya. */
  judul: string

  /**
   * Isi kolom yang sebenarnya.
   *
   * Ia ada KARENA judulnya memakai alias yang menyesatkan: kolom berjudul `District`
   * berisi nama tertanggung.
   */
  keterangan: string
}

/** Bentuk baris sebuah bagian. */
export type SectionKind = 'klaim' | 'pic' | 'kosong'

/** Satu kontrol yang tampil di sistem lama tetapi tidak menyaring apa pun. */
export type DeadControl = {
  nama: string
  alasan: string
}

/** Satu bagian layar beserta bentuk gridnya. */
export type Section = {
  kode: string
  nama: string
  keterangan: string
  bentuk: SectionKind
  kolom: Column[]

  pakai_paginasi: boolean
  pakai_pencarian: boolean
  pakai_rentang_tanggal: boolean
  pakai_lini_bisnis: boolean
  hanya_milik_saya: boolean

  kontrol_mati: DeadControl[]
}

/** Satu pilihan pada dropdown lini bisnis. */
export type BusinessLine = {
  kode: string
  label: string
}

/** Jawaban `GET /api/inbox-progress-claim/bagian`. */
export type MetadataResponse = {
  bagian: Section[]
  bagian_bawaan: string
  lini_bisnis: BusinessLine[]
  ukuran_halaman: number
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
  cari: string
  bisnis: string
  dari: string | null
  sampai: string | null
}

/** Jawaban `GET /api/inbox-progress-claim`. */
export type ListResponse = {
  bagian: Section
  baris: Row[]
  paginasi: PageInfo
  penyaring: AppliedFilter
  portal: string
}

/** Keadaan penyaring satu bagian di layar. */
export type FilterForm = {
  cari: string
  bisnis: string
  dari: string
  sampai: string
}

/** Keadaan awal penyaring. */
export const EMPTY_FILTER: FilterForm = { cari: '', bisnis: '', dari: '', sampai: '' }

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const ProgressClaimError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
} as const
