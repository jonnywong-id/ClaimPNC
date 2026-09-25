/**
 * Bentuk data layar Inbox Komunikasi Cabang — menu `MENU_ID 70`, pengganti harness
 * `InboxKomunikasiCabang`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan nama properti
 * Pega. Alasannya ada di `internal/inboxkomunikasicabang/inboxkomunikasicabang.go`: hampir
 * seluruh alias di kueri lama menyesatkan — `ClaimNo` berisi nomor percakapan dan bukan
 * nomor klaim, `Email` berisi isi pesan dan bukan alamat surel, dan `pzInsKey` berisi
 * penanda kanal dan bukan kunci objek kerja Pega.
 */

/**
 * Satu baris pada grid.
 *
 * # Kenapa satu bentuk untuk kedua tab, padahal kolomnya berbeda
 *
 * Karena barisnya memang sama — yang berbeda hanyalah kolom mana yang DIGAMBAR. Tab "Belum
 * Dijawab" menggambar tiga kolom, tab "Sudah Dijawab" lima, dan keduanya datang dari
 * `Tab.kolom`, bukan dari bentuk barisnya.
 *
 * Seluruh isian selalu dikirim server, termasuk yang kosong. Isian yang kebetulan kosong
 * pada seluruh baris halaman tidak boleh membuat kolomnya menghilang.
 */
export type Conversation = {
  /**
   * Nomor percakapan, dibutuhkan tombol "Detail Komunikasi".
   *
   * Ia dikirim tetapi TIDAK pernah digambar sebagai kolom. Namanya `komunikasi`, bukan
   * `no_klaim`: tabel ini tidak memuat nomor klaim sama sekali.
   */
  komunikasi: string

  /** Kolom "Tanggal" — tanggal pesannya dikirim. */
  tanggal: string

  /**
   * Kolom "Pengirim(Dari)", SUDAH dirakit server menjadi `asal (operator)`.
   *
   * Bentuknya ditetapkan `Section/PengirimKomunikasi-Section.xml`, dan perakitannya ada di
   * server supaya polanya tidak hidup di dua tempat.
   */
  pengirim: string

  /** Asal pesan — "PUSAT" atau kode cabangnya. */
  asal: string

  /** Operator ID pengirimnya. */
  operator_pengirim: string

  /** Kolom "Pesan". */
  pesan: string

  /**
   * Kolom "Jawaban Terakhir".
   *
   * SELALU kosong pada tab "Belum Dijawab" — penyaring tab itu justru `IS NULL`. Kolomnya
   * memang tidak didaftarkan pada tab tersebut.
   */
  jawaban_terakhir: string

  /**
   * Kolom "Penjawab(Dari)", SUDAH dirakit menjadi `nama (tujuan)`.
   *
   * Perhatikan susunannya TERBALIK dari kolom pengirim — yang satu menaruh asal di depan,
   * yang satu menaruh nama di depan. Itu bentuk kedua section-nya apa adanya (`D-13`).
   */
  penjawab: string

  /**
   * Tujuan pesan — "PUSAT" atau "CABANG".
   *
   * Ia tidak pernah menyebut cabang MANA, berbeda dari `asal` yang menyebut kodenya.
   * Asimetri itu ada di layar lama dan direplikasi (`P-5`).
   */
  tujuan: string

  /**
   * Status percakapan, MENTAH.
   *
   * Artinya tidak diketahui — tidak ada master yang menerjemahkannya di export mana pun —
   * sehingga ia tidak digambar sebagai kolom. Ia ikut ke berkas unduhan, tempat nilai
   * mentah masih berguna bagi yang menelusuri.
   */
  status_register: string

  /**
   * Tanggal balasan.
   *
   * Ia DASAR PENGURUTAN tab "Sudah Dijawab" tetapi tidak digambar sebagai kolom.
   */
  tanggal_jawaban: string
}

/** Nama isian pada satu baris — dipakai memilih sel yang digambar sebuah kolom. */
export type ConversationField = keyof Conversation

/**
 * Nama kolom TOMBOL.
 *
 * Keduanya BUKAN isian pada baris: yang digambar adalah tombolnya, dan yang dibawanya adalah
 * `komunikasi` yang sudah ada di barisnya.
 *
 * Keduanya kolom sungguhan di layar lama — berada DI DALAM kedua grid, bukan di bilah aksi
 * di atas tabel. Versi pertama modul ini melewatkannya dan menggantinya dengan tautan pada
 * sel "Pesan", yang di Pega tidak ada sama sekali.
 */
export type ActionField = 'aksi_detail' | 'aksi_selesai'

/** Satu kolom grid, sebagaimana ditetapkan server. */
export type TabColumn = {
  kunci: ConversationField | ActionField
  judul: string
}

/** isActionField menyatakan sebuah kolom menggambar tombol, bukan isian baris. */
export function isActionField(kunci: string): kunci is ActionField {
  return kunci === 'aksi_detail' || kunci === 'aksi_selesai'
}

/** Satu kolom berkas unduhan. */
export type ExportColumn = {
  kunci: string
  judul: string
}

/** Satu tab beserta bentuk gridnya. */
export type Tab = {
  kode: string
  nama: string
  keterangan: string
  kolom: TabColumn[]

  /**
   * Keterangan yang berlaku pada tab ini saja, digambar di atas grid.
   *
   * Satu tab memilikinya: "Sudah Dijawab" menjelaskan bahwa yang tampil adalah balasan
   * TERAKHIR, bukan seluruh utasnya.
   */
  catatan?: string
}

/** Keterangan halaman dari server. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/**
 * Kedua pencacah di atas grid.
 *
 * Menggantikan diagram lingkaran layar lama, yang memasok kedua angka yang sama dengan
 * label "Answered" dan "Not Answered".
 */
export type Summary = {
  belum_dijawab: number
  sudah_dijawab: number

  /**
   * Jumlah keduanya, DIKIRIM server dan bukan dihitung di sini.
   *
   * Keduanya BUKAN saling melengkapi: percakapan yang salah satu kolom balasannya terisi
   * sendirian tidak terhitung di mana pun. Menghitungnya di layar akan membuat orang
   * menduga angka itu jumlah seluruh percakapan — dan ia bukan.
   */
  total: number
}

/**
 * Batas data yang sedang berlaku.
 *
 * Ia dikirim ke layar karena petugas yang tidak tahu daftarnya sedang disaring akan
 * menyimpulkan tidak ada percakapan, padahal yang benar adalah tidak ada percakapan DI
 * CABANGNYA.
 */
export type BranchScope = {
  /** Kode yang dipakai menyaring. */
  kode: string

  /** Penyaringnya jatuh ke jalur kantor pusat. */
  kantor_pusat: boolean

  /**
   * Kode cabang pemanggil benar-benar terbaca.
   *
   * `kantor_pusat` true dengan `terbaca` false berarti pemanggilnya TIDAK terdaftar di HRD
   * dan sedang dilayani sebagai kantor pusat karena itu (`P-5`). Layar menyatakan keadaan
   * itu apa adanya — menyembunyikannya berarti pelebaran batas data yang tidak diketahui
   * siapa pun yang terkena.
   */
  terbaca: boolean

  /** Kalimat siap baca dari server yang menjelaskan batasnya. */
  keterangan: string
}

/** Jawaban GET /api/inbox-komunikasi-cabang/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string

  /**
   * Kolom berkas unduhan.
   *
   * Dibaca layar untuk menyebutkan isi berkasnya SEBELUM diunduh — ia memuat tiga kolom
   * yang tidak ada di tabel mana pun.
   */
  kolom_ekspor: ExportColumn[]

  /** Selisih terhadap Pega yang sudah diputuskan, ditampilkan di bawah tabel. */
  selisih_terencana: string[]

  portal: string
}

/** Jawaban GET /api/inbox-komunikasi-cabang. */
export type ListResponse = {
  tab: Tab
  baris: Conversation[]
  paginasi: PageInfo
  ringkasan: Summary
  batas_cabang: BranchScope
  portal: string
}

/** Satu lampiran pada layar detail. */
export type Attachment = {
  /**
   * Rujukan ke penyimpanan dokumen.
   *
   * Dikirim tetapi belum dapat dipakai mengunduh apa pun: pengambilan berkasnya menempuh
   * seam DocumentStore (`D-16`) yang belum dibangun di modul ini.
   */
  id_dokumen: string

  jenis_dokumen: string
  rincian_dokumen: string
  catatan: string
  tanggal_unggah: string

  /**
   * Lampirannya sudah benar-benar diunggah.
   *
   * Dikirim sebagai penanda, bukan disimpulkan di sini dari `tanggal_unggah` yang kosong:
   * kesimpulan yang sama di dua tempat dapat menyimpang, dan yang di server-lah yang
   * mengikuti aturan sistem lama.
   */
  sudah_diunggah: boolean
}

/** Satu baris pada utas layar detail. */
export type ThreadMessage = {
  /** Kolom "Tanggal". */
  tanggal: string

  /**
   * Kolom "Pengirim" — Operator ID pengirimnya, APA ADANYA.
   *
   * Ia TIDAK dirakit menjadi `asal (operator)` seperti di grid, dan itu bukan pilihan: tabel
   * riwayat tidak memuat kolom asal sama sekali, sehingga tidak ada yang dapat dirakit.
   */
  pengirim: string

  /** Kolom "Pesan". */
  pesan: string
}

/** Jawaban GET /api/inbox-komunikasi-cabang/komunikasi/{komunikasi}. */
export type ConversationDetailResponse = {
  komunikasi: string

  /** Asal percakapan, dipakai judul layar detail. */
  asal: string

  pesan: ThreadMessage[]
  lampiran: Attachment[]

  /**
   * Kotak balasan dapat dipakai.
   *
   * Dikirim sebagai DATA, bukan ditulis tetap di layar. Nilainya berubah dari `false` menjadi
   * `true` pada 2026-09-24 — persis alasan penanda ini ada.
   *
   * Field lamanya bernama `tindakan_masih_di_pega` dan artinya KEBALIKAN dari ini. Ia diganti,
   * bukan dibalik nilainya: nama yang artinya berlawanan dengan isinya adalah cacat yang
   * menunggu giliran.
   */
  balas_tersedia: boolean

  batas_cabang: BranchScope
  portal: string
}

/**
 * Satu pilihan pada pemilih cabang tujuan.
 *
 * Alamat surelnya TIDAK ada di sini, dan itu disengaja: layar tidak membutuhkannya, dan
 * alamat surel yang dikirim ke peramban ikut tercatat di cache, log proxy, dan alat
 * pengembang. Peladen membacanya untuk keperluan notifikasi dan tidak pernah meneruskannya.
 */
export type BranchOption = {
  kode: string
  nama: string
}

/** Jawaban `GET /api/inbox-komunikasi-cabang/cabang`. */
export type BranchListResponse = {
  cabang: BranchOption[]

  /**
   * Kedua pilihan dropdown tujuan — "PUSAT" dan "CABANG".
   *
   * Datang dari peladen, bukan ditulis tetap di layar: keduanya hasil pembacaan export
   * (`CNMShowInsertKomunikasi_dt` mengisinya secara harfiah), dan tempat pembacaan itu
   * tercatat adalah backend.
   */
  tujuan: string[]

  portal: string
}

/** Badan permintaan `POST /api/inbox-komunikasi-cabang/pesan`. */
export type NewMessageRequest = {
  tujuan: string
  cabang: string
  pesan: string
}

/** Jawaban aksi tulis yang berhasil — balas, selesai, dan kirim pesan. */
export type ActionResponse = {
  komunikasi: string
  pesan: string
  portal: string
}
