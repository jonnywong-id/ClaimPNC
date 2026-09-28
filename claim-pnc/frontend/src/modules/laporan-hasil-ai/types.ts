/**
 * Bentuk data layar Laporan Hasil AI (`MENU_ID 82`).
 *
 * Nama field mengikuti kontrak API yang berbahasa Indonesia (`D-80`) — ia kontrak, bukan
 * nama internal. Nama TIPE-nya berbahasa Inggris, seperti modul lain.
 */

/** Satu baris grid rincian — satu penilaian AI pada satu objek pertanggungan. */
export type ReportRow = {
  /** Kunci baris tabel. TIDAK digambar sebagai kolom. */
  id: string

  /**
   * Kolom "No Klaim".
   *
   * SENGAJA kosong pada jenjang komite kedua ke atas — padanan
   * `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`, yang Work Owner putuskan
   * untuk ditiru pada 2026-09-26.
   */
  no_klaim: string

  /** Kolom "Object Name". SELALU kosong; kuerinya tidak memilih kolomnya. */
  nama_object: string

  /** Kolom "Komite Status" — DITERIMA, DITOLAK, MENUNGGU, atau kosong. */
  komite_status: string

  /**
   * Nilai mentah `STATUSAPPROVE`. Dikirim tetapi TIDAK digambar.
   *
   * Tanpanya, "MENUNGGU" dan kode yang tidak dikenal terlihat sama di layar sementara
   * keduanya berarti hal yang berbeda saat ditelusuri.
   */
  kode_komite_status: string

  /** Kolom "Tanggal Komite", `YYYY-MM-DD`. null bila kolomnya NULL. */
  tanggal_komite: string | null

  /** Kolom "AI Status" — DITERIMA, DITOLAK, atau kosong bila AI belum menilai. */
  ai_status: string

  /** Kolom "Tanggal AI", `YYYY-MM-DD`. null bila AI belum menilai. */
  tanggal_ai: string | null

  /** Kolom "Note AI Terima". SELALU kosong. */
  note_ai_terima: string

  /** Kolom "Note AI Tolak". SELALU kosong. */
  note_ai_tolak: string

  /** Kolom "Coverage Final". SELALU kosong. */
  coverage_final: string

  /** Kolom "Kategori Kronologi". SELALU kosong. */
  kategori_kronologi: string
}

/**
 * Satu baris grid ringkasan.
 *
 * `total` adalah `diterima + ditolak`, BUKAN jumlah baris — ditiru apa adanya dari
 * `Activity/SearchDataLaporanAI-Act.xml`. Kolom `menunggu` TIDAK ada di layar lama;
 * ia ditambahkan atas keputusan Work Owner 2026-09-26 supaya selisihnya terbaca.
 */
export type ReportTally = {
  /** Isi kolom "Keputusan" — "Komite" atau "AI". */
  keputusan: string
  total: number
  diterima: number
  ditolak: number
  menunggu: number
}

export type Pagination = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Penyaring yang BENAR-BENAR dipakai server menjawab, dipantulkan kembali. */
export type AppliedFilter = {
  dari: string
  sampai: string
}

export type SearchResponse = {
  /** Selalu dua baris, pada urutan yang tergambar: Komite lalu AI. */
  ringkasan: ReportTally[]
  baris: ReportRow[]
  paginasi: Pagination
  filter: AppliedFilter
  portal: string
}

/**
 * Isian penyaring di layar.
 *
 * Kedua namanya mengikuti label isian di layar lama: "Tgl Input Dari" dan "Tgl Input
 * Sampai". Label itu MENYESATKAN — yang disaring adalah `TANGGALKOMITE`, kolom yang juga
 * digambar sebagai "Tanggal Komite" — tetapi Work Owner memutuskan pada 2026-09-26 untuk
 * memakainya apa adanya (`D-13`).
 */
export type FilterInput = {
  dari: string
  sampai: string
}

export const emptyFilter: FilterInput = { dari: '', sampai: '' }

/** Kedua isian wajib. Layar memakainya untuk menahan permintaan yang pasti ditolak. */
export function isComplete(filter: FilterInput): boolean {
  return filter.dari !== '' && filter.sampai !== ''
}
