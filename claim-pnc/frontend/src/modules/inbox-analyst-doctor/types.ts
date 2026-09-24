/**
 * Bentuk data layar Inbox Analyst Doctor.
 *
 * Nama fieldnya berbahasa Indonesia karena ia KONTRAK yang dikirim server apa adanya
 * (`D-80`); yang berbahasa Inggris hanyalah nama tipenya.
 */

/** Satu baris antrean — satu tugas penilaian medis yang menunggu. */
export type TugasAnalystDoctor = {
  /**
   * Kunci teknis `PZINSKEY`. TIDAK ditampilkan sebagai kolom — isinya memuat nama kelas
   * internal Pega — tetapi dipakai tautan baris untuk membuka klaimnya.
   */
  klaim_id: string

  /** Judul kolomnya memang "Nomor Case", bukan "Nomor Klaim" (`D-13`). */
  nomor_case: string

  nomor_polis: string
  nama_tertanggung: string
  nama_cabang: string
  nama_admin: string

  /**
   * Kolom "Komentar dari PIC Teknis".
   *
   * Masih KOSONG terhadap Oracle: properti Pega-nya (`ClaimData.AnalystDoctorRemaks`)
   * ditandai tidak terekspos, sehingga ia tidak punya kolom SQL yang dapat dibaca. Kolomnya
   * tetap digambar supaya isian yang belum terbawa terlihat, bukan tersamar.
   */
  komentar_pic_teknis: string

  /**
   * `ClaimData.UserTeknis`. Diambil Report Definition tetapi TIDAK digambar sebagai kolom —
   * harness tidak punya judul untuknya. Dipakai sebagai keterangan pada kolom komentar.
   */
  pic_teknis: string

  /** YYYY-MM-DD, sudah dalam WIB. Dikonversi server, bukan di peramban (`R-12`). */
  tanggal_pendaftaran: string

  /** Kolom "Lama Waktu Klaim" — umur tugas dalam hari, dihitung server. */
  lama_hari: number

  /** `pyStatusWork`. Tidak digambar; ia yang menentukan baris ini ada di antrean. */
  status_proses: string

  /** Pemilik penugasan. Selalu sama dengan pemanggil pada keadaan biasa. */
  operator_penerima: string
}

/** Satu judul kolom, datang dari server. */
export type KolomLayar = {
  kunci: string
  judul: string
  /** Keterangan yang ditempelkan pada judul; kosong bila tidak ada. */
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
  data: TugasAnalystDoctor[]

  /** Jumlah SELURUH baris yang cocok, bukan jumlah baris di halaman ini. */
  total: number

  /**
   * Paginasi yang BENAR-BENAR dipakai server, bukan yang diminta.
   *
   * Permintaan `batas=5000` dipangkas menjadi 100, dan bilah halaman menghitung dari angka
   * ini — bukan dari angka yang dikirim layar.
   */
  lewati: number
  batas: number

  cari: string
}
