/**
 * Bentuk data layar Inbox Close Claim.
 *
 * Nama fieldnya berbahasa Indonesia karena ia KONTRAK yang dikirim server apa adanya
 * (`D-80`); yang berbahasa Inggris hanyalah nama tipenya.
 */

/** Satu permintaan yang menunggu dijalankan atas sebuah klaim. */
export type PermintaanTertunda = {
  id: string
  /** "reopen" atau "salin". */
  jenis: string
  /** Selalu "menunggu" pada daftar ini. */
  status: string
  alasan: string
  pemohon: string
  pemohon_nama: string
  /** Tanggal dan jam WIB, sampai menit. */
  pada: string

  /** Niat yang tercatat — apa yang akan berubah saat permintaan dijalankan. */
  efek_status_kerja: string
  efek_status_klaim: string
  lingkup_salin: string
}

/** Satu baris pada layar — sebuah klaim yang sudah tutup. */
export type KlaimTutup = {
  /**
   * Kunci teknis `PZINSKEY`. TIDAK ditampilkan sebagai kolom — isinya memuat nama kelas
   * internal Pega — tetapi kedua tombol aksi mengirimnya kembali ke server.
   */
  klaim_id: string

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  sumber_bisnis: string
  nama_cabang: string
  pic_teknik: string
  admin_pnc: string

  /** YYYY-MM-DD, sudah dalam WIB. Dikonversi server, bukan di peramban (`R-12`). */
  tanggal_pendaftaran: string
  tanggal_kejadian: string
  tanggal_tutup: string

  /** Kolom "Lama Waktu Klaim" — umur klaim dalam hari, dihitung server. */
  lama_hari: number

  status_proses: string
  /** "Close" atau "Reject". Teksnya mengikuti layar Pega apa adanya (`D-13`). */
  status_tampil: string
  status_klaim_kode: string
  status_klaim_label: string

  sudah_transfer: boolean

  permintaan_tertunda: PermintaanTertunda[]
}

/** Badan respons daftar. */
export type DaftarResponse = {
  klaim: KlaimTutup[]
  total: number

  /**
   * false berarti permintaan tertunda TIDAK dapat dibaca — tabelnya belum dibuat DBA.
   *
   * Layar wajib menyatakannya. Bila tidak, penanda "permintaan terkirim" yang tidak muncul
   * akan terbaca sebagai "belum pernah diminta", dan pengguna mengajukannya lagi.
   */
  permintaan_terbaca: boolean

  /**
   * false berarti pengajuan ReOpen dan Copy Klaim TIDAK terbuka untuk pemanggil.
   *
   * Hari ini ia **selalu false**: penjaganya When rule `IsGCNMUser`, yang isinya `1 = 2`.
   * Layar memakainya untuk menonaktifkan kedua tombol BESERTA ALASANNYA — tombol yang tampak
   * dapat ditekan lalu selalu dijawab 403 terbaca sebagai gangguan sistem, bukan sebagai
   * kewenangan yang memang tidak ada.
   */
  boleh_mengajukan: boolean

  /** Terisi hanya saat `boleh_mengajukan` false. Datang dari domain, bukan ditulis di layar. */
  alasan_tidak_boleh?: string

  /**
   * true berarti permintaan yang tercatat BELUM ADA yang menjalankannya.
   *
   * Dinyatakan supaya pengguna tidak menunggu perubahan yang belum akan datang.
   */
  pelaksana_belum_ada: boolean

  /** Perbedaan yang DISENGAJA terhadap layar Pega, dinyatakan kepada pengguna (`D-54`). */
  selisih_terencana: string[]
}

/** Satu butir dropdown penyaring. */
export type Pilihan = {
  nilai: string
  label: string
}

/** Bentuk layar: isi ketiga dropdown penyaring, dibaca dari server. */
export type PenyaringResponse = {
  lini_bisnis: Pilihan[]
  status_transfer: Pilihan[]
  status_bayar: Pilihan[]
}

/** Penyaring yang sedang berlaku di layar. */
export type PenyaringKlaimTutup = {
  cari?: string
  no_polis?: string
  no_klaim?: string
  pic?: string
  lini?: string
  status_transfer?: string
  status_bayar?: string
  lewati?: number
}

/** Jenis permintaan yang dapat diajukan. */
export type JenisPermintaan = 'reopen' | 'salin'

/** Jawaban atas permintaan yang berhasil dicatat. */
export type PermintaanResponse = {
  permintaan: PermintaanTertunda
  /** Kalimat yang menjelaskan apa yang SUDAH dan BELUM terjadi. */
  pesan: string
}
