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

  /** `RCL_PUCL`: "1" RCL, "3" MSIG. Tidak digambar. */
  mode: string
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
   * `false` berarti login Anda tidak ada atau tidak aktif di POOLDATA.M_LOGIN_PNC, sehingga
   * antreannya tidak dicari. Tidak digambar layar — sama seperti Pega, hanya grid kosong.
   */
  pengguna_ditemukan: boolean
}

/**
 * Isi layar kerja `RCLDokter` satu klaim — seluruhnya dari POOLDATA.TC_PNC_PUCL.
 *
 * `mode` menentukan isi layar (`Section/RCLDokter-Section.xml`):
 *   "1" RCL  — Catatan dari Analyst · Alasan Klaim Ditolak/RCL · Setuju / Tidak Setuju
 *   "3" MSIG — Catatan dari Analyst · Alasan Klaim MSIG        · Back / Submit
 */
export type DetailRCL = {
  portal: string
  nomor_case: string
  nomor_polis: string
  nama_tertanggung: string
  mode: string
  /** "Catatan dari Analyst" — KOMENTAR_ANALISATOR. */
  catatan_analyst: string
  /** "Alasan Klaim Ditolak/RCL" atau "Alasan Klaim MSIG" — KETERANGAN2. */
  alasan: string
  /** "Alasan Dokter" — ALASAN_DOKTER_REJECT_RCL. */
  alasan_dokter: string
  status_klaim: string
  status_proses: string
  operator_penerima: string
  tanggal_masuk_inbox: string
}

/**
 * Nilai `StatusRCL` yang dikirim tombol, apa adanya dari kedua section Pega:
 *   SETUJU      tombol Setuju (mode RCL)
 *   MSIG        tombol Submit (mode MSIG)
 *   TidakSetuju Kirim pada layar Alasan Dokter, dibuka dari Tidak Setuju (mode RCL)
 *   BackMSIG    Kirim pada layar Alasan Dokter, dibuka dari Back (mode MSIG)
 */
export type KeputusanRCL = 'SETUJU' | 'MSIG' | 'TidakSetuju' | 'BackMSIG'

export type KeputusanResponse = {
  portal: string
  nomor_case: string
  status_klaim: string
  /** Nama tahap tujuan — "RCL/PUCL" atau "Send To Analis". */
  tahap_berikutnya: string
}
