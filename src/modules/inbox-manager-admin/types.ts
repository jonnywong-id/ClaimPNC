/**
 * Bentuk data layar Inbox Manager Admin — menu `MENU_ID 57`, pengganti harness
 * `InboxManagerAdmin_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Di layar ini keduanya kebetulan dekat, berbeda dari Inbox Admin yang nama
 * propertinya berubah arti dari tab ke tab — ketiga tab di sini dilayani Report Definition
 * yang sama dengan satu-satunya perbedaan pada parameter unit organisasinya.
 */

/** Satu baris antrean. */
export type WorkItem = {
  /**
   * Kunci teknis Pega, dibutuhkan tombol "Lihat Detail".
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom. Di layar lama, nilai inilah yang
   * disusun `SetAssignmentInboxReg_act` menjadi kunci assignment yang dibuka tombolnya.
   */
  referensi: string

  id: string
  no_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  nama_sumber_bisnis: string
  tanggal_pendaftaran: string | null

  /**
   * Lama Waktu Klaim, sudah berbentuk teks siap tampil — misalnya "1 year 5 months ago".
   *
   * Ia datang sebagai teks, bukan sebagai jumlah hari, karena yang ditiru adalah format
   * bawaan Pega `DateTime-Frame` yang memang menyusun kalimat. Menyusun kalimatnya di layar
   * berarti aturan bentuknya hidup di dua tempat.
   */
  lama_waktu_klaim: string

  admin_pnc: string

  /**
   * Status Klaim — TIDAK digambar sebagai kolom, karena grid Pega pun tidak.
   *
   * Ia dikirim karena berkas ekspor memuatnya (kolom keenam, "Status Klaim"), dan supaya
   * layar dapat menampilkannya kelak tanpa perubahan kontrak.
   */
  status_klaim: string
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
   * Unit organisasi penugasan yang disaring tab ini — "AdminPNC", "AdminPA", atau
   * "AdminTRAVEL".
   *
   * Ia BUKAN kolom yang digambar. Ia dipakai dalam pesan saat antreannya kosong: antrean
   * kosong di layar ini dapat berarti unitnya memang sepi, atau kolom penyaringnya tidak
   * pernah terisi di produksi — dan yang kedua adalah kekeliruan konfigurasi yang tidak
   * menghasilkan satu pun galat.
   */
  unit_organisasi: string

  /** Lini bisnis yang membuka tab ini — "NONMBU", "PA", atau "TRAVEL". */
  lini_bisnis: string
}

/** Jawaban `GET /api/inbox-manager-admin/tab`. */
export type MetadataResponse = {
  /** Tab yang BOLEH dilihat pengguna ini — bukan ketiganya. */
  tab: Tab[]

  /** Tab pertama yang boleh dilihat. Kosong bila tidak ada satu pun. */
  tab_bawaan: string

  /**
   * Ketiga tab tanpa memandang kewenangan.
   *
   * Dipakai hanya ketika `tab` kosong: menyebutkan apa saja yang ada beserta lini bisnis
   * yang membukanya jauh lebih berguna daripada "Anda tidak melihat tab apa pun".
   */
  semua_tab: Tab[]

  /** Lini bisnis yang membuka tab — "NONMBU", "PA", "TRAVEL". */
  lini_bisnis_yang_diharapkan: string[]

  /**
   * Lini bisnis pengguna apa adanya, dibaca server dari `M_LOGIN_PNC.LINE_BUSINESS`.
   *
   * Pengguna yang tidak melihat satu tab pun perlu tahu nilai APA yang terbaca sistem:
   * itulah satu-satunya petunjuk yang dapat ia sampaikan saat melapor, dan yang membedakan
   * "kolomnya belum diisi" dari "diisi dengan nilai yang tidak dikenal".
   */
  lini_bisnis_anda: string
  portal: string
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban `GET /api/inbox-manager-admin`. */
export type ListResponse = {
  tab: Tab
  baris: WorkItem[]
  paginasi: PageInfo
  portal: string
}

/** Kode galat modul ini, sebagaimana dikirim backend. */
export const InboxManagerAdminError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
  tabNotAllowed: 'tab_bukan_hak_anda',
  noTabAllowed: 'tidak_ada_tab_untuk_lini_bisnis_anda',
} as const
