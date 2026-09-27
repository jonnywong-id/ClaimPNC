/**
 * Bentuk data layar Inbox RCL (`MENU_ID 62`).
 *
 * Nama fieldnya berbahasa Indonesia karena ia KONTRAK yang dikirim server apa adanya
 * (`D-80`); yang berbahasa Inggris hanyalah nama tipenya.
 */

/** Satu baris antrean — satu tugas penolakan medis yang menunggu dokter RCL. */
export type TugasRCL = {
  /** Kunci teknis `PZINSKEY`. Tidak ditampilkan; dipakai sebagai kunci baris. */
  klaim_id: string

  /** Judul kolomnya memang "Nomor Case" (`D-13`). */
  nomor_case: string

  nomor_polis: string
  nama_tertanggung: string

  /**
   * Kolom "Tanggal Masuk Inbox" — waktu analis mengirim klaim ke dokter RCL.
   * Bentuk "YYYY-MM-DD HH:mm", sudah dalam WIB; dikonversi server (`R-12`).
   */
  tanggal_masuk_inbox: string

  /** Kolom "Deskripsi Analyst" — `ClaimData.PUCLStatus.KomentarAnalisator`. */
  deskripsi_analyst: string

  /** Tidak digambar; dikirim untuk penelusuran. */
  dokter_rcl: string
  status_proses: string
  operator_penerima: string
}

/** Satu judul kolom, datang dari server. */
export type KolomLayar = {
  kunci: string
  judul: string
  keterangan?: string
}

/** Keterangan layar — judul kolom, selisih terencana, dan keterbatasan. */
export type KeteranganResponse = {
  portal: string
  kolom: KolomLayar[]
  selisih_terencana: string[]
  keterbatasan: string[]
  ukuran_halaman: number
}

/** Satu halaman antrean. */
export type DaftarResponse = {
  portal: string
  data: TugasRCL[]

  /** Jumlah SELURUH baris yang cocok, bukan jumlah baris di halaman ini. */
  total: number

  /** Paginasi yang BENAR-BENAR dipakai server, bukan yang diminta. */
  lewati: number
  batas: number
  cari: string

  /**
   * `false` berarti identitas lama Anda (`TempOperator.City` di sistem lama) tidak
   * ditemukan — antreannya kosong karena belum diketahui pekerjaan siapa, bukan karena
   * tidak ada pekerjaan.
   */
  identitas_lama_ditemukan: boolean
}
