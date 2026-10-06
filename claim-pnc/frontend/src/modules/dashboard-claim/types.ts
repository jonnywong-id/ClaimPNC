/**
 * Bentuk data layar Dashboard Claim.
 *
 * Menggantikan `Harness/DashboardClaim_Harness-Harness.xml` beserta ketiga section-nya.
 *
 * Nama field mengikuti kunci JSON backend apa adanya — bukan diterjemahkan di sini. Kunci
 * itu KONTRAK (`D-80`), dan menerjemahkannya di satu sisi saja membuat kedua sisi menyimpang
 * tanpa ada yang menandainya.
 */

/** Keempat kartu penghitung, dikenali lewat nilai ini pada jalur telusurnya. */
export type Tile = 'outstanding' | 'close-claim' | 'loss-adjuster' | 'internal-surveyor'

/**
 * Bentuk baris telusur sebuah tile.
 *
 * Ia datang dari SERVER, bukan disimpulkan dari nama tile. Menebaknya dari nama membuat
 * penambahan tile kelak diam-diam salah gambar.
 */
export type BentukBaris = 'klaim' | 'survei'

/** Satu kartu penghitung beserta angkanya. */
export type Kartu = {
  tile: Tile
  judul: string
  jumlah: number
  bentuk: BentukBaris
}

/** Jawaban ringkasan — keempat angka kartu. */
export type RingkasanResponse = {
  kartu: Kartu[]
  lini_bisnis: string
  portal: string

  /**
   * Selisih yang DISENGAJA terhadap sistem lama.
   *
   * Ditampilkan di layar, bukan disembunyikan sebagai detail teknis: penguji gerbang 1
   * membacanya saat membandingkan angka, dan selisih yang ditemukan tanpa dinyatakan lebih
   * dulu akan dilaporkan sebagai cacat (`D-54`).
   */
  selisih_terencana: string[]

  /**
   * Perilaku yang SAMA dengan sistem lama tetapi mudah dibaca sebagai cacat.
   *
   * Terpisah dari `selisih_terencana` dengan sengaja: yang di sana berbeda dari Pega dan
   * menuntut persetujuan, yang di sini sama dengan Pega dan hanya menuntut penjelasan.
   *
   * Butir terpentingnya: angka pada kartu tidak selalu sama dengan jumlah baris telusurnya.
   */
  catatan_warisan: string[]
}

/** Satu baris telusur bertipe klaim — tile Outstanding dan Close Claim. */
export type BarisKlaim = {
  /** Kunci teknis Pega. Tidak ditampilkan sebagai teks (utang teknis §4.1). */
  klaim_id: string

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  sumber_bisnis: string
  nama_cabang: string
  pic_teknik: string
  admin_pnc: string

  /** Kolom "Tanggal Pendaftaran". */
  tanggal_pendaftaran: string

  /** Kolom "Report Date" — hanya terisi pada tile Outstanding. */
  tanggal_lapor: string

  tanggal_kejadian: string

  /** Kolom "Lama Waktu Klaim", dalam HARI. Pemformatannya urusan layar. */
  lama_hari: number

  /** KODE status klaim, bukan artinya — pelabelannya milik master status klaim (`R-06`). */
  status_klaim_kode: string

  /** Kolom "Claim status" — artinya, dari `V_STS_CLAIM.LSC_NOTE`. */
  status_klaim_label: string

  status_proses: string
}

/** Satu baris telusur bertipe survei — tile Loss Adjuster dan Internal Surveyor. */
export type BarisSurvei = {
  survei_id: string

  nomor_survei: string
  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  nomor_referensi: string
  nama_surveyor: string
  pic_teknik: string
  pic_adjuster: string
  lokasi_survei: string

  /** Hanya terisi pada tile Internal Surveyor — kueri loss adjuster tidak mengambilnya. */
  tanggal_survei: string
  tanggal_tugas: string

  /** Kolom "Aging", dalam HARI. */
  lama_hari: number

  status_survei: string
  status_proses: string
}

/** Keterangan halaman. Halaman dihitung mulai 1. */
export type KeteranganHalaman = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/**
 * Jawaban telusur satu tile.
 *
 * Hanya SATU dari `klaim` dan `survei` yang terisi, ditentukan `bentuk`. Keduanya selalu
 * berupa larik — tidak pernah null — sehingga layar dapat memetakannya tanpa memeriksa lebih
 * dulu.
 */
export type TelusurResponse = {
  tile: Tile
  judul: string
  bentuk: BentukBaris

  klaim: BarisKlaim[]
  survei: BarisSurvei[]

  halaman: KeteranganHalaman

  lini_bisnis: string
  portal: string
}

/** Satu pilihan pada dropdown. */
export type Pilihan = {
  nilai: string
  label: string
}

/** Keterangan satu kartu tanpa angkanya. */
export type KeteranganTile = {
  tile: Tile
  judul: string
  bentuk: BentukBaris
}

/**
 * Jawaban penyaring — bentuk layar, bukan data.
 *
 * Ia dijawab meski koneksi entitas sedang bermasalah, sehingga layar tetap dapat menggambar
 * kerangkanya dan menampilkan galatnya.
 */
export type PenyaringResponse = {
  lini_bisnis: Pilihan[]
  tile: KeteranganTile[]
  portal: string
}

/** Penyaring yang dikirim layar. */
export type PenyaringDashboard = {
  lini_bisnis?: string
  cari?: string
  halaman?: number
}

/** Satu baris tab **Inbox Tampungan PIC** — klaim yang belum punya PIC Teknik. */
export type BarisTampungan = {
  klaim_id: string

  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  sumber_bisnis: string
  nama_cabang: string
  admin_pnc: string

  tanggal_pendaftaran: string
}

/** Jawaban tab Inbox Tampungan PIC. */
export type TampunganResponse = {
  klaim: BarisTampungan[]
  halaman: KeteranganHalaman
  portal: string
}

/** Kedua tab layar ini. */
export type TabDashboard = 'dashboard' | 'tampungan'

/** Lingkup permintaan transfer — tombol per baris, atau "Transfer All Case By UserID". */
export type LingkupTransfer = 'baris' | 'massal'

/** Badan permintaan transfer. */
export type PermintaanTransfer = {
  lingkup: LingkupTransfer
  klaim_id?: string
  nomor_klaim?: string
  user_id_lama?: string
  user_id_baru: string
  tipe_pengguna?: string
  alasan?: string
}

/** Jawaban permintaan transfer. */
export type TransferResponse = {
  permintaan: {
    id: string
    lingkup: LingkupTransfer
    nomor_klaim?: string
    user_id_lama?: string
    user_id_baru: string
    status: string
    pemohon: string
    pada: string
  }
  portal: string

  /** Permintaan TERCATAT, tetapi penugasannya belum berpindah — Pega yang menjalankan. */
  pelaksana_belum_ada: boolean
}

/**
 * Jenis permintaan yang dapat diajukan dari tile Close Claim.
 *
 * Keduanya tombol nyata pada `Section/InboxManagerReopen1_Sec-Section.xml`, section yang
 * **disertakan** layar Dashboard Claim — `Section/DashboardClaim_Section1-Section.xml`
 * memuat `<pyInclude>InboxManagerReopen1_Sec</pyInclude>`.
 *
 * Nilainya mengikuti kontrak modul Inbox Close Claim yang sudah berjalan, bukan nama baru:
 * `reopen` dan `salin`.
 */
export type JenisPermintaanKlaim = 'reopen' | 'salin'

/** Izin mengajukan ReOpen dan Copy Klaim, dibaca dari modul Inbox Close Claim. */
export type IzinPermintaanKlaim = {
  boleh_mengajukan: boolean
  alasan_tidak_boleh?: string
}

/**
 * Hasil satu pengajuan di dalam satu kumpulan.
 *
 * Pengajuan massal adalah N permintaan terpisah, bukan satu permintaan berisi N klaim —
 * lihat `useAjukanPermintaanKlaim`. Karena itu sebagiannya dapat berhasil dan sebagiannya
 * gagal, dan hasil per baris harus dapat dilaporkan satu per satu.
 */
export type HasilPermintaanKlaim = {
  klaim_id: string
  nomor_klaim: string
  berhasil: boolean
  pesan?: string
}
