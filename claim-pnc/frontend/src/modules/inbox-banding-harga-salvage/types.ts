/**
 * Bentuk data layar Inbox Banding Harga Salvage — menu `MENU_ID 72`, pengganti harness
 * `InboxRequestSalvage`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxbandinghargasalvage/inboxbandinghargasalvage.go`: di
 * layar ini alias Pega nyaris seluruhnya menyesatkan — `.Email` berisi HARGA, `.AgentID`
 * berisi harga lain pada satu grid dan LOKASI pada grid lain, dan kolom berjudul "Object
 * Name" sebenarnya berisi jenis salvage.
 */

/**
 * Satu baris pada layar.
 *
 * Kedua grid membaca tabel yang BERBEDA, sehingga isian di luar tab yang sedang terbuka memang
 * kosong. Itu tidak pernah terlihat pengguna: layar hanya menggambar kolom yang disebut
 * `tab.kolom`.
 */
export type AppealRow = {
  /** Digambar kedua tab. */
  no_klaim: string

  /** Digambar tab "Request Banding Harga". */
  tanggal_request: string | null
  detail_object: string
  nama_barang: string
  harga_barang: string
  harga_request: string
  note_request: string
  note_checker: string

  /**
   * Umur banding dalam bentuk yang dibaca pengguna: `<n> days`.
   *
   * Teksnya mengikuti layar lama. Yang TIDAK diikuti adalah pengurutannya — di sini ia
   * diurutkan sebagai angka di server, sehingga banding berumur 30 hari berada di ATAS yang
   * berumur 9 hari. Kosong bila barisnya tidak punya tanggal request.
   */
  aging: string

  /** Digambar tab "History Cheker". Judul kolomnya "Object Name" — lihat catatan di atas. */
  object_name: string
  lokasi_salvage: string
  pic: string

  /**
   * Keduanya TIDAK digambar sebagai kolom.
   *
   * Dibawa karena tombol Approve/Reject kelak membutuhkannya, dan supaya pengguna yang
   * melihat antrean komite lain dapat memastikan barisnya memang milik siapa.
   */
  id_salvage: string
  nama_komite: string
}

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type AppealRowField = keyof AppealRow

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: AppealRowField
  judul: string

  /** Kolom angka: diratakan kanan dan diberi pemisah ribuan. */
  angka: boolean
}

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string

  /**
   * Nilai `tipe` sistem lama untuk tab ini — `1` atau `2`.
   *
   * Ia dikirim demi ketelusuran dan TIDAK digambar: saat uji kesetaraan gerbang 1 menemukan
   * selisih, inilah yang menghubungkan tab di layar dengan langkah activity di export.
   */
  parameter_pega: string

  kolom: TabColumn[]
}

/** Jawaban `GET /api/inbox-banding-harga-salvage/tab`. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string
  label_cari: string
  petunjuk_cari: string

  /**
   * Hal yang SENGAJA berbeda dari layar lama.
   *
   * Dipisah dari `keterbatasan` karena keduanya menjawab pertanyaan berbeda: yang ini
   * "mengapa angkanya tidak sama dengan Pega", yang itu "mengapa tombolnya tidak ada".
   */
  selisih_terencana: string[]

  /** Hal yang belum berjalan penuh beserta alasannya, siap ditampilkan apa adanya. */
  keterbatasan: string[]

  /** Kolom panel rincian pada grid History Cheker. */
  kolom_rincian: DecisionColumn[]

  portal: string
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
}

/**
 * Antrean siapa yang sedang dibaca.
 *
 * Di layar ini ia dapat BERBEDA dari pemanggilnya: satu Operator ID melihat antrean komite
 * lain, aturan bernama orang yang ditiru dari Pega atas keputusan Work Owner. Tanpa
 * keterangan ini, petugas itu akan menyimpulkan antreannya sendiri kosong.
 */
export type QueueInfo = {
  milik: string
  diwakilkan: boolean

  /** Kalimat siap tampil, kosong bila tidak berlaku. Disusun server. */
  catatan_perwakilan: string
  catatan_giliran: string
}

/** Jawaban `GET /api/inbox-banding-harga-salvage`. */
export type ListResponse = {
  tab: Tab
  baris: AppealRow[]
  paginasi: PageInfo
  penyaring: AppliedFilter
  antrean: QueueInfo
  portal: string
}

/** Satu baris tabel ringkas "Status Salvage / Jumlah". */
export type SummaryRow = {
  status_salvage: string
  tab: string
  jumlah: number
}

/** Jawaban `GET /api/inbox-banding-harga-salvage/ringkas`. */
export type SummaryResponse = {
  baris: SummaryRow[]
  antrean: QueueInfo
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
export const BandingHargaSalvageError = {
  validationFail: 'validasi_gagal',
  callerUnknown: 'profil_pemanggil_tidak_lengkap',
  writeNotBuilt: 'tindakan_belum_tersedia',
  badRequest: 'permintaan_tidak_terbaca',

  /** Dokumen tidak ada, sudah ditandai ditolak, atau bukan milik banding Anda. */
  documentGone: 'dokumen_tidak_ditemukan',

  /**
   * Banding ini sudah diputus, ATAU bukan milik komite yang menekan tombolnya.
   *
   * Keduanya sengaja tidak dibedakan backend, dan layar tidak boleh menebak yang mana:
   * membedakannya memberi tahu penanya bahwa sebuah ID nyata dan sedang ditangani orang
   * lain. Ia pula jawaban atas penekanan tombol kedua.
   */
  alreadyDecided: 'banding_sudah_diputus',
} as const

// ─────────────────────────────────────────────────────────────────────────────
// Panel rincian History Cheker
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Satu keputusan banding harga atas satu barang.
 *
 * Satu klaim dapat punya beberapa barang yang dibanding, sehingga panel ini DAFTAR — bukan
 * perluasan satu baris.
 */
export type Decision = {
  tanggal_approve: string | null
  detail_object: string
  nama_barang: string
  harga_barang: string
  harga_request_keputusan: string

  /**
   * Keputusan dikirim sebagai kode DAN label.
   *
   * Kode dipakai memilih warna lencana — "Setuju" dan "Tidak setuju" layak dibedakan
   * sekilas. Label dipakai menggambarnya, dan ia disusun server dari rule Pega supaya
   * tabel terjemahannya tidak hidup di dua tempat.
   */
  jawaban_checker_kode: string
  jawaban_checker: string

  nama_checker: string
}

/** Nama isian pada satu baris panel rincian. */
export type DecisionField = keyof Decision

/** Satu kolom panel rincian, sebagaimana ditetapkan server. */
export type DecisionColumn = {
  kunci: DecisionField
  judul: string
  angka: boolean
}

/** Jawaban `GET /api/inbox-banding-harga-salvage/riwayat/{noKlaim}`. */
export type DecisionsResponse = {
  no_klaim: string
  baris: Decision[]
  antrean: QueueInfo
  portal: string
}

/** Kode keputusan, sebagaimana dikirim backend. */
export const DecisionCode = {
  approved: '1',
  rejected: '0',
} as const

// ─────────────────────────────────────────────────────────────────────────────
// Tombol Approve dan Reject
// ─────────────────────────────────────────────────────────────────────────────

/** Badan `POST /api/inbox-banding-harga-salvage/keputusan`. */
export type DecisionRequest = {
  detail_object: string
  id_salvage: string

  /**
   * Harga tandingan yang sedang diputuskan.
   *
   * Dikirim ULANG oleh layar, bukan dibaca server dari barisnya, karena begitulah tombolnya
   * di Pega: nilainya diambil dari sel yang sedang tergambar.
   */
  harga_request: string

  catatan: string

  /** `true` menerima harga tandingan balai lelang, `false` menolaknya. */
  setujui: boolean
}

/**
 * Jawaban keputusan yang tersimpan.
 *
 * Ia menyatakan langkah mana yang BENAR-BENAR berjalan, bukan sekadar "berhasil". Di layar
 * ini "disetujui" tidak selalu berarti "harganya berubah" — penerapan harga di Pega hanya
 * dilakukan jenjang komite terakhir.
 */
export type DecisionResultResponse = {
  tersimpan: boolean
  harga_diterapkan: boolean
  dokumen_ditandai: boolean

  /** Kalimat siap tampil yang menjelaskan ketiganya. Disusun server. */
  pesan: string

  portal: string
}

/** Keadaan isian keputusan pada satu baris. */
export type DecisionForm = {
  /** Baris yang sedang diputuskan — kosong berarti tidak ada. */
  detail_object: string
  catatan: string
}

/** Keadaan awal isian keputusan. */
export const EMPTY_DECISION_FORM: DecisionForm = { detail_object: '', catatan: '' }

// ─────────────────────────────────────────────────────────────────────────────
// Dialog "Lihat File"
// ─────────────────────────────────────────────────────────────────────────────

/** Satu dokumen banding. */
export type AppealDocument = {
  id: string

  /**
   * SELALU bernilai sama — ia konstanta yang ditetapkan activity lama untuk setiap baris,
   * bukan isi kolom basis data. Tetap digambar karena layar lama menggambarnya (`D-13`).
   */
  kategori: string

  nama: string

  /** Dari `SALAVAGEDOCUMENT.TGLINS`, bukan dari tabel lampiran. */
  tanggal_unggah: string | null
}

/** Jawaban `GET /api/inbox-banding-harga-salvage/dokumen`. */
export type DocumentsResponse = {
  /** Kosong berarti banding itu tidak punya dokumen pendukung — keadaan yang sah. */
  baris: AppealDocument[]
  portal: string
}

/** Banding yang dialog dokumennya sedang terbuka. Null berarti tertutup. */
export type DocumentScope = {
  no_klaim: string
  detail_object: string
  id_salvage: string
  nama_barang: string
}
