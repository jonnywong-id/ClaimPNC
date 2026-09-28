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

  tanggal_register: string
  tanggal_kejadian: string

  /** KODE status klaim, bukan artinya — pelabelannya milik master status klaim (`R-06`). */
  status_klaim_kode: string
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
