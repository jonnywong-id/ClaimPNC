/**
 * Bentuk data layar Inbox OS Claim per Cabang — menu `MENU_ID 69`, pengganti harness
 * `OutstandingKlaimperCabang_Harness`.
 *
 * Nama field mengikuti KONTRAK API yang berbahasa Indonesia (`D-80`), bukan alias Pega.
 * Alasannya ada di `internal/inboxosclaimpercabang/inboxosclaimpercabang.go`: sejumlah alias
 * kueri lama menyesatkan sepenuhnya — `TanggalTerlambat` sebenarnya tanggal catatan progres
 * terakhir, `PICRekanan` sebenarnya PIC Teknik, dan `Medicare` tidak ada hubungannya dengan
 * pengobatan.
 */

/** Satu baris klaim outstanding milik satu cabang. */
export type WorkItem = {
  cabang: string
  sumbis: string

  /** Class of Business — hasil terjemahan `grouppanel`, bukan kolom tersendiri. */
  cob: string

  no_polis: string
  no_klaim: string

  /** Tanggal ISO `YYYY-MM-DD`, atau teks kosong bila memang tidak ada. */
  tanggal_registrasi: string
  tanggal_kejadian: string

  /**
   * Kolom berjudul "Reserve Claim ASM Share", dikirim sebagai TEKS desimal kanonik.
   *
   * Dibaca dengan `formatRupiah` dari `@/lib/money`, bukan `formatMoney` — yang kedua
   * menerima angka. Memakai yang salah tidak menghasilkan galat tipe, hanya tampilan yang
   * keliru.
   *
   * Nilainya BUKAN porsi ASM meski judulnya menjanjikan begitu; lihat `selisih_terencana`.
   */
  nilai_estimasi: string

  /** Kapan catatan progres TERAKHIR dibuat. Kosong berarti belum ada satu pun. */
  tanggal_update_progres: string

  status_progres_1: string
  status_progres_2: string

  /** PIC Teknik yang memegang klaim. */
  pic: string

  adjuster: string

  /** Cause of Loss — penyebab kerugian pada coverage terakhir. */
  col: string

  /** Umur klaim dalam hari kalender sejak registrasi. */
  aging_hari: number

  /**
   * Baris ini digambar MERAH.
   *
   * Dihitung SERVER, bukan di sini: aturannya `aging > 180 || progres mandek`, dan aturan
   * yang hidup di dua tempat akan berbeda antara grid dan berkas ekspor.
   */
  perlu_perhatian: boolean

  /**
   * Progres klaim ini mandek — tiga catatan progres terakhirnya bernilai sama.
   *
   * Dikirim terpisah dari `perlu_perhatian` supaya layar dapat menjelaskan MENGAPA sebuah
   * baris merah. Merah tanpa sebab yang dapat dibaca hanya memindahkan pertanyaannya.
   */
  progres_mandek: boolean

  /** Keterangan pada catatan progres terakhir. Tidak digambar sebagai kolom. */
  catatan_progres: string
}

/** Cabang yang barisnya sedang ditampilkan, dipakai judul layar. */
export type Branch = {
  kode: string
  nama: string
}

/** Keterangan halaman. */
export type PageInfo = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban `GET /api/inbox-os-claim-per-cabang`. */
export type ListResponse = {
  data: WorkItem[]
  cabang: Branch
  paginasi: PageInfo

  /** Ambang umur yang membuat baris merah, dipakai menjelaskan pewarnaannya. */
  ambang_aging: number

  /** Selisih terhadap layar Pega yang sudah diputuskan (`D-54`). */
  selisih_terencana: string[]

  portal: string
}
