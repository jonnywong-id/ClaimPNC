/**
 * Bentuk data layar Inbox Compliance — menu `MENU_ID 47`, pengganti harness
 * `inboxCompliance_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega.
 */

/** Satu baris pekerjaan. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol buka detail klaim.
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom.
   */
  referensi: string

  nomor_case: string

  /**
   * Kolom "No Klaim" pada tab Post Audit.
   *
   * Namanya menyesatkan dan itu bentuk sistem lama: isinya BUKAN nomor klaim melainkan
   * kunci teknis Pega, `ASM-FW-GCNMFW-WORK PNC-2114`. Layar Pega menampilkannya apa
   * adanya, dan `D-13` menetapkan tampilan ditiru.
   */
  no_klaim: string

  no_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  nama_cabang: string
  nama_admin: string

  /** Tanggal saja: `YYYY-MM-DD`. */
  tanggal_kirim_compliance: string | null

  /**
   * Tanggal DAN jam: `YYYY-MM-DD HH:MM`.
   *
   * Berbeda dari kolom tanggal lain, dan itu disengaja — layar Pega menampilkannya sebagai
   * `22/04/25 13:46`.
   */
  tanggal_kirim_post_audit: string | null

  catatan_compliance: string

  /**
   * Teks kolom Aging, mengikuti bentuk sistem lama — "5 hours ago",
   * "2 days 3 hours ago". Kosong bila tidak dapat dihitung.
   */
  aging: string

  /**
   * Angka mentah Aging dalam jam, sudah dipotong akhir pekan.
   *
   * `null` berarti tidak dapat dihitung — dan itu BERBEDA dari `0`, yang berarti baris
   * baru saja masuk antrean. Layar memakai angka ini untuk menandai baris yang terlalu
   * lama menunggu, tanpa harus mengurai teksnya kembali menjadi angka.
   */
  aging_jam: number | null

  /**
   * Kolom "OutStanding" pada tab Post Audit — `1 year 5 months ago`.
   *
   * BUKAN kolom Aging dengan nama lain: dasarnya waktu kalender apa adanya, sedangkan
   * Aging memotong akhir pekan.
   */
  outstanding: string
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

  /**
   * Tab ini sudah dapat menampilkan data.
   *
   * Tab yang belum dapat dilayani TETAP dikirim beserta penghalangnya — bukan
   * disembunyikan. Menyembunyikannya membuat pengguna yang mencarinya menduga modulnya
   * belum selesai.
   */
  tersedia: boolean

  /** Apa yang kurang dan siapa pemiliknya. Kosong bila tersedia. */
  penghalang?: string
}

/** Jawaban `GET /api/inbox-compliance/tab`. */
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
}

/** Jawaban `GET /api/inbox-compliance`. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  portal: string
}

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const InboxComplianceError = {
  validationFail: 'validasi_gagal',
  tabNotReady: 'tab_belum_tersedia',
} as const

/** Badan permintaan `POST /api/inbox-compliance/post-audit`. */
export type SendPostAuditRequest = {
  /** Kunci klaim yang dikirim — isian `referensi` pada baris tab Compliance. */
  referensi: string

  /** Catatan. Boleh kosong: kolomnya nullable dan tidak ada bukti bahwa ia wajib. */
  catatan: string
}

/** Jawaban pengiriman yang berhasil. */
export type SendPostAuditResponse = {
  nomor_case: string
  no_klaim: string
  nama_tertanggung: string
  no_polis: string
  catatan: string
  tanggal_kirim_post_audit: string | null
  portal: string
}
