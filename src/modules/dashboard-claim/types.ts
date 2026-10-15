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

  /**
   * Kolom "Posisi Klaim" dan "Progress Klaim".
   *
   * Keduanya **gabungan dipisah koma** — satu klaim dapat memegang beberapa posisi
   * berjalan sekaligus, dan sistem lama menggabungkannya di dalam
   * `GET_POSISI_PROGRESS_PNC`.
   */

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
  /**
   * Isi kedua dropdown panel penyaring, **datang dari server**.
   *
   * Nilainya dicocokkan dengan konstanta di dalam SQL, sehingga salah ketik satu huruf di
   * sini akan menghasilkan penyaring yang tidak menyaring apa pun — tanpa galat. Karena itu
   * daftarnya tidak ditulis ulang di layar.
   */
  status_transfer: Pilihan[]
  status_bayar: Pilihan[]
  /**
   * Isi dropdown "Pilih Type User" pada Transfer All Case.
   *
   * Nilainya dibandingkan sebagai **teks** oleh `GCNMTransferDataKlaim_act` — `"Admin"`,
   * `"PIC Teknik"`, `"Other"` — sehingga daftarnya datang dari server, bukan ditulis ulang
   * di sini.
   */
  tipe_pengguna: Pilihan[]
  tile: KeteranganTile[]
  portal: string
  /**
   * Apakah pengguna termasuk grup yang berwenang membuka case.
   *
   * # TIDAK dipakai layar sejak 2026-10-12
   *
   * Sistem lama memasang DUA aksi pada sel nomor klaim dan memilih lewat When rule
   * `IsManagerPNC` (`Section/DashboardClaimShow_Sec1`), dan layar ini sempat menirunya.
   * Work Owner memutuskan **satu tujuan untuk semua**: halaman klaim yang sama dengan My
   * Inbox, apa pun grup penggunanya.
   *
   * Field-nya **tetap dikirim server dan tetap dibaca di sini**, karena jawabannya dihitung
   * dari `POOLDATA.M_LOGIN_GROUP_PNC` dan merupakan satu-satunya tempat kewenangan itu
   * terbaca hari ini. Menghapusnya berarti membuang pengetahuan yang sudah terukur, dan
   * `F-3` akan membangunnya lagi dari nol.
   *
   * Menyembunyikan link **bukan** keamanan. Pemeriksaan di server pada setiap alamat adalah
   * lingkup `F-3` dan belum ada — jadi jangan membaca `false` sebagai "tidak bisa menyunting".
   */
  boleh_buka_case: boolean
}

/** Penyaring yang dikirim layar. */
export type PenyaringDashboard = {
  lini_bisnis?: string
  cari?: string
  halaman?: number

  /**
   * Ketiga isian panel penyaring `Section/FilterDashboardClaim_sec-Section.xml`.
   *
   * Dinamai menurut ISINYA. Di layar lama namanya menyesatkan: Nopolis disimpan pada
   * `TempInputFilter.CaseID`, dan PIC pada `TempInputFilter.ClaimID`.
   */
  nomor_polis?: string
  nomor_klaim?: string
  pic?: string

  /**
   * Kedua dropdown pada panel yang sama.
   *
   * Namanya pun menyesatkan di layar lama: status pembayaran disimpan pada
   * `TempInputFilter.City`, status transfer pada `TempInputFilter.CityID`.
   */
  status_transfer?: string
  status_bayar?: string
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
/**
 * Lingkup transfer.
 *
 *   baris    tombol Transfer pada satu baris
 *   massal   "Transfer All Case By UserID" — seluruh pekerjaan satu operator
 *   saring   "Select All" — seluruh klaim yang cocok penyaring, lintas halaman
 */
export type LingkupTransfer = 'baris' | 'massal' | 'saring'

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

/**
 * Jawaban permintaan transfer.
 *
 * KOREKSI (2026-10-06): sebelumnya jawaban ini membawa `pelaksana_belum_ada`, karena transfer
 * hanya MENCATAT permintaan dan pelaksanaannya diserahkan kepada Pega. Itu dicabut — tidak
 * ada job Pega yang membaca tabel permintaan itu, sehingga permintaannya tidak pernah akan
 * dijalankan. Sekarang transfer langsung memindahkan PIC Teknik klaimnya, seperti Pega.
 */
export type TransferResponse = {
  permintaan: {
    id: string
    lingkup: LingkupTransfer
    nomor_klaim?: string
    user_id_lama?: string
    user_id_baru: string
    status: string

    /**
     * Berapa klaim yang benar-benar berpindah.
     *
     * Pada lingkup `massal` angkanya tidak dapat diketahui di muka — "seluruh pekerjaan si A"
     * bisa berarti satu klaim atau empat puluh — sehingga layar menyebutkannya setelah selesai.
     */
    jumlah_pindah: number

    pemohon: string
    pada: string
  }
  portal: string
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

/**
 * Satu baris daftar **PIC Teknik** yang dapat menerima pemindahan klaim.
 *
 * Isi grid `Section/PNCTransferManagement_sec-Section.xml`, yang kolom tunggalnya berjudul
 * **"Nama"** dan tombolnya **"Assign"**.
 */
export type BarisPIC = {
  operator_id: string
  nama: string
  email: string
  tim: string
  /** COUNTER_QUOTA — pencacah beban yang dipakai pemilihan otomatis (`R-04`). */
  beban: number
}

/** Jawaban daftar PIC Teknik. */
export type PICResponse = {
  pic: BarisPIC[]
  halaman: KeteranganHalaman
  portal: string
}

/**
 * Jawaban **Ex Gratia** pada konfirmasi Copy Klaim.
 *
 * `Section/GCNMCopyClaimConfirmation-Section.xml` menanyakan *"Apakah klaim ini tipe Ex
 * Gratia?"* lewat radio terikat `TempGratia.ExGratia`. Daftar nilainya tidak ikut di
 * export, sehingga dipakai bentuk paling sedikit menduga: ya / tidak.
 */
export type JawabanExGratia = 'ya' | 'tidak'

/**
 * Rincian satu klaim — isi popup yang terbuka saat nomor klaim diklik.
 *
 * Menggantikan harness `ViewTempDetailClaim`, yang di Pega dibuka lewat `showHarness`
 * bertarget `popup` dari kolom nomor klaim pada `DashboardClaimShow_Sec`.
 */
export type RincianKlaim = {
  klaim_id: string
  nomor_klaim: string
  status_proses: string
  status_klaim: string
  pic_teknik: string
  admin_pnc: string
  didaftarkan_pada: string

  /**
   * Kode lini bisnis klaim, dipakai memilih VARIAN grid objek.
   *
   * Dulu digali dari dokumen JSON. Sejak migrasi lepas dari JSON (keputusan Work Owner
   * 2026-10-12), keduanya datang dari kepala — dan tanpa itu setiap klaim akan dinyatakan
   * "tidak termasuk yang digambar layar lama".
   */
  group_panel: string
  jenis_bisnis: string

  /**
   * "Catatan dari Inputor" pada kepala popup — `T_CLAIM_PNC.REMARK`.
   *
   * "Aging Amount" yang berdampingan dengannya TIDAK ada di kontrak: ia milik data polis, dan
   * polis bukan milik bounded context ini (`D-04`). Layar menggambarnya kosong, dan itu jujur.
   */
  catatan_inputor: string

  /**
   * Ketujuh bagian isi popup, dibaca server dari TABEL RELASIONAL.
   *
   * Dulu satu field `dokumen` berisi dokumen JSON apa adanya. Kolom itu terisi 721 dari 16.284
   * baris, sehingga popup tampil kosong bagi mayoritas klaim — di Pega pun demikian.
   *
   * Tiap bagian SELALU larik, tidak pernah `null`: dua bentuk untuk satu arti hanya menambah
   * cabang yang harus dijaga layar.
   */
  penerima: BarisRincian[]
  objek: BarisRincian[]
  coverage: BarisRincian[]
  adjustment: BarisRincian[]
  survey: BarisRincian[]
  lampiran: BarisRincian[]
  progres: BarisRincian[]

  /**
   * Tiga himpunan isian TUNGGAL — bukan grid.
   *
   * Layar lama menggambar ketiganya sebagai satu blok nilai berlabel, bukan daftar baris,
   * sehingga keduanya tidak boleh dimodelkan sama. Objek bersarang, bukan diratakan ke
   * kepala: `lokasi_kejadian` pada tab Registrasi dan `lokasi` pada permintaan survei adalah
   * nilai yang berbeda, dan diratakan keduanya akan bertabrakan.
   */
  registrasi: Record<string, string>
  survei: Record<string, string>
  pucl: Record<string, string>

  portal: string
}

/**
 * Satu baris bagian rincian.
 *
 * Nilainya SELALU teks — termasuk tanggal dan nilai uang. Nilai uang dikirim sebagai teks
 * dengan sengaja (`D-51`): angka JSON adalah `float64` di hampir semua klien, dan presisi
 * penuh hilang di sana tanpa satu pun tanda.
 */
export type BarisRincian = Record<string, string>
