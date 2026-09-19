// Bentuk data yang dipertukarkan dengan backend.
//
// Tipe di sini adalah cerminan dto Go di internal/*/http. Bila salah satu berubah, yang
// lain wajib ikut berubah — pemeriksaan tipe TypeScript adalah jaring pengaman antara
// layar dan API.
//
// NAMA TIPE berbahasa Inggris (`D-80`); NAMA FIELD tetap Indonesia karena ia kontrak
// API, bukan nama internal.

/** Dua populasi pengguna, diverifikasi lewat dua jalur berbeda. */
export const UserKind = {
  /** Diverifikasi ke API HCC/HCQ. Identitasnya NIK. */
  karyawan: 'KARYAWAN',
  /** Broker dan surveyor independen, diverifikasi ke POOLDATA.M_LOGIN_PNC. */
  nonKaryawan: 'NON_KARYAWAN',
} as const

export type UserKind = (typeof UserKind)[keyof typeof UserKind]

export type User = {
  /** NIK untuk karyawan, LOGIN_ID untuk non-karyawan. */
  identitas: string
  nama: string
  jenis: string
  login: string
  /** Dapat kosong: POOLDATA.M_LOGIN_PNC tidak memuat surel. */
  email: string
  /** Dapat kosong: hanya diisi HCC/HCQ. */
  perusahaan: string
}

export type LoginResponse = {
  token: string
  tipe_token: string
  berlaku_sampai: string
  pengguna: User
}

export type MeResponse = {
  pengguna: User
  berlaku_sampai: string
}

export type ExtendResponse = {
  berlaku_sampai: string
}

/** Satu entitas yang dilayani aplikasi ini (ADR-0030). */
export type Portal = {
  id: string
  nama: string
  alias: string
  /**
   * Koneksi basis data portal ini sudah dapat dipakai.
   *
   * Portal yang belum siap tetap ditampilkan — ditandai, bukan disembunyikan — supaya
   * pengguna melihat seluruh entitas yang direncanakan.
   */
  siap: boolean
}

export type PortalListResponse = {
  portal: Portal[]
  /** Alias portal yang basis datanya melayani sesi dan lookup pra-login. */
  utama: string
}

// ── Menu aplikasi ────────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/menu/http. Sumbernya POOLDATA.M_MENU_APLIKASI_PNC, disaring
// kewenangan pemanggil lewat M_OTORISASI_PNC.

/**
 * Satu butir menu beserta submenunya.
 *
 * Server hanya mengirim butir yang boleh DILIHAT pemanggil. Yang menentukan butir itu
 * dapat diklik atau belum adalah layar, lewat peta rute di `app/menu/registry.ts` —
 * hanya frontend yang tahu modul mana yang sudah dibangun.
 */
export type MenuItem = {
  /** MENU_ID. Kunci yang stabil: nama menu dapat berubah, nomornya tidak. */
  id: number
  nama: string
  /**
   * MENU_PROGRAM — nama harness Pega yang dituju butir ini.
   *
   * Kosong berarti butir ini tidak menuju layar mana pun: kelompok tingkat atas, atau
   * daun yang memang tidak punya tujuan di master (MENU_ID 83).
   */
  program: string
  /** Selalu ada sebagai senarai, tidak pernah null. */
  submenu: MenuItem[]
}

export type MenuListResponse = {
  menu: MenuItem[]
}

// ── Master Status Progres 1 ──────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterstatusprogres/http. Menggantikan layar Pega
// `Harness/StatusProgress-Harness.xml` atas tabel POOLDATA.GCNM_MST_PROGRESS_KLAIM.

/**
 * Satu baris master status progres tingkat 1.
 *
 * Nama field di sini sudah dinamai ulang mengikuti `D-19`; kueri Pega mengaliaskan
 * ketiga kolomnya ke nama yang tidak mencerminkan isi (`CaseID`, `City`, `CityID`).
 */
export type ProgressStatus = {
  /** Kolom ID_PROGRESS. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom STS_PROGRESS1 — keterangan status yang dibaca petugas. */
  nama: string
  /** Kolom STATUS — kode posisi klaim tempat status ini berlaku. */
  kode_posisi: string
  /** Label posisi, dikirim server supaya layar tidak menyimpan salinan daftarnya. */
  nama_posisi: string
}

/** Satu pilihan pada dropdown Posisi. */
export type ClaimPosition = {
  kode: string
  nama: string
}

export type ProgressStatusListResponse = {
  status_progres: ProgressStatus[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type ProgressStatusResponse = {
  status_progres: ProgressStatus
  portal: string
}

export type ClaimPositionListResponse = {
  posisi: ClaimPosition[]
}

/** Badan permintaan penambahan dan penyuntingan. */
export type ProgressStatusInput = {
  nama: string
  kode_posisi: string
}

/**
 * Satu baris Master Status Klaim.
 *
 * Status Klaim adalah salah satu dari empat konsep status yang `D-18` tetapkan memang
 * berbeda. Ia menjawab "klaim ini berada di keadaan bisnis apa" — Register, Claim
 * Committee, Paid, dan seterusnya. Domainnya 33 kode `1134`–`1166`.
 */
export type ClaimStatus = {
  /** Dibuat sistem saat status ditambahkan; tidak pernah berubah sesudahnya. */
  kode: string
  /** Teks yang dibaca pengguna. Di layar Pega ia berlabel "Status". */
  label: string
  /**
   * Penomoran lama `01`–`11` yang melekat pada sebelas kode pertama (`1134`–`1144`).
   * Kosong untuk 22 kode sisanya, dan tidak pernah diisi untuk status baru.
   */
  kode_lama: string
}

export type ClaimStatusListResponse = {
  status_klaim: ClaimStatus[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
}

export type ClaimStatusResponse = {
  status_klaim: ClaimStatus
}

/**
 * Posisi sebuah laporan klaim dalam perjalanannya menjadi klaim.
 *
 * Tahap DIHITUNG server dari isi laporan, tidak disimpan sebagai kolom — sama seperti
 * sistem lama menurunkannya dari kombinasi `PNCCASEID` dan `STATUSLOCK`. Layar TIDAK
 * menghitungnya sendiri: aturan yang hidup di dua tempat akan berselisih pada perubahan
 * berikutnya.
 */
export const ReportStage = {
  notTransferred: 'BELUM_TRANSFER',
  notRegistered: 'BELUM_REGISTRASI',
  registered: 'SUDAH_REGISTRASI',
  accepted: 'SUDAH_AKSEPTASI',
  rejected: 'DITOLAK',
} as const

export type ReportStage = (typeof ReportStage)[keyof typeof ReportStage]

/**
 * Satu laporan klaim — laporan kerugian yang masuk sebelum klaim diregistrasi.
 *
 * Menggantikan case `ASM-FW-GCNMFW-Work-ReceiveDocument`, yang di menu portal Pega
 * berjudul **"Inbox Laporan Klaim"**.
 */
export type ClaimReport = {
  nomor: string

  nama_pelapor: string
  email_pengirim: string
  telepon_pengirim: string
  nama_kurir: string
  subjek_email: string

  /** Polis dan tertanggung SEBAGAIMANA DISEBUT PELAPOR — bukan snapshot polis. */
  nomor_polis: string
  nama_tertanggung: string
  email_tertanggung: string
  kode_bisnis: string
  group_panel: string
  nomor_referensi: string

  /** Tanggal kalender, `YYYY-MM-DD`. Kosong berarti belum diisi. */
  tanggal_kejadian: string
  lokasi_kejadian: string
  kronologi: string
  rincian_kerusakan: string
  sim_pengendara: string
  /** Teks desimal, bukan angka — presisi penuh tanpa pembulatan floating point. */
  nilai_estimasi: string
  tipe_klaim: string

  jumlah_dokumen: number
  tanggal_terima_dokumen: string

  nomor_klaim: string
  ditransfer: boolean
  /** Waktu peristiwa, RFC 3339 UTC. Kosong berarti belum terjadi. */
  tanggal_transfer: string
  tanggal_registrasi: string
  alasan_belum_transfer: string
  catatan_belum_registrasi: string

  /** Dihitung server dari isi laporan. Layar tidak menghitungnya sendiri. */
  tahap: string
  tahap_label: string

  /** Dihitung server dari aturan yang sama yang ditegakkan usecase. */
  dapat_ditransfer: boolean
  dapat_diubah: boolean

  kode_cabang: string
  diinput_oleh: string
  diinput_pada: string
  diubah_pada: string
}

/** Jumlah laporan pada satu tahap, untuk lencana di atas tabnya. */
export type StageSummary = {
  tahap: string
  label: string
  jumlah: number
}

export type ClaimReportListResponse = {
  laporan: ClaimReport[]
  /** Banyaknya baris yang cocok SEBELUM dipotong paginasi. */
  jumlah: number
  batas: number
  lewati: number
  /** Selalu memuat kelima tahap, termasuk yang jumlahnya nol. */
  ringkasan: StageSummary[]
}

export type ClaimReportResponse = {
  laporan: ClaimReport
}

/**
 * Kode galat modul Pelaporan Klaim.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const ClaimReportErrorCode = {
  notFound: 'laporan_klaim_tidak_ditemukan',
  alreadyTransferred: 'laporan_sudah_ditransfer',
  alreadyRegistered: 'laporan_sudah_diregistrasi',
  numberTaken: 'nomor_laporan_sudah_dipakai',
} as const

export type ClaimReportErrorCode =
  (typeof ClaimReportErrorCode)[keyof typeof ClaimReportErrorCode]

/**
 * Satu aturan yang dilanggar beserta kolom yang melanggarnya.
 *
 * Kedua nama kunci ada dan keduanya opsional, karena kedua modul yang memakai bentuk
 * senarai BELUM sepakat menamainya: `masterstatus` mengirim `field`
 * (internal/masterstatus/http/dto.go:55), `masterstatusprogres` mengirim `kolom`
 * (internal/masterstatusprogres/http/dto.go:82).
 *
 * Layar tidak perlu memilih sendiri di antara keduanya — `APIError.violations()`
 * menyatukannya. Penyeragaman kontraknya masuk TKT-F1-004.
 */
export type FieldViolation = {
  field?: string
  kolom?: string
  pesan: string
}

/**
 * Kode galat yang dikirim backend.
 *
 * Layar membedakan jenis galat lewat kode ini, TIDAK PERNAH dengan mencocokkan teks
 * pesan — teks bisa berubah kapan saja tanpa mengubah artinya.
 *
 * NILAI-nya tetap berbahasa Indonesia: ia kontrak API, bukan nama internal (`D-80`).
 */
export const ErrorCode = {
  wrongCredential: 'kredensial_salah',
  userInactive: 'pengguna_tidak_aktif',
  identityDown: 'sistem_identitas_tidak_terhubung',
  invalidSession: 'sesi_tidak_sah',
  sessionExpired: 'sesi_kedaluwarsa',
  malformedRequest: 'permintaan_cacat',
  internalError: 'galat_internal',

  // Galat modul bisnis dan portal.
  //
  // `validationFailed` dipakai BERSAMA oleh modul bisnis dan modul master data — ia
  // tidak menunjuk satu modul tertentu, sehingga tempatnya di kelompok ini.
  validationFailed: 'validasi_gagal',
  notFound: 'tidak_ditemukan',
  /** Permintaan tidak menyebut portal — pengguna belum memilih entitas. */
  portalNotStated: 'portal_tidak_disebut',
  portalUnknown: 'portal_tidak_dikenal',
  /** Entitasnya ada, tetapi kredensial basis datanya belum diisi tim infrastruktur. */
  portalNotReady: 'portal_belum_siap',

  // Milik modul master data.
  claimStatusNotFound: 'status_klaim_tidak_ditemukan',
  statusLabelTaken: 'label_status_sudah_dipakai',
  statusCodeTaken: 'kode_status_sudah_dipakai',

  // Milik modul Komite.
  unknownLine: 'lini_tidak_dikenal',
  malformedClaimValue: 'nilai_klaim_cacat',
} as const

export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode]

/**
 * Posisi sebuah rekening dalam alur persetujuan komite.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.LST_ACCOUNT. Ia tidak
 * dapat dipilih bebas selama tabel yang sama masih dibaca sistem lama.
 */
export const AccountStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type AccountStatus = (typeof AccountStatus)[keyof typeof AccountStatus]

/** Satu baris master rekening tujuan pembayaran klaim. */
export type Account = {
  nomor_rekening: string
  nama_pemilik: string
  nama_bank: string
  cabang_bank: string
  alamat_bank: string
  kode_bank: string
  tipe_rekening: string
  aktif: boolean

  email: string
  telepon: string
  nik: string
  catatan: string
  id_dokumen: string
  diinput_oleh: string

  status: string
  /** Sebutan status dalam bahasa yang dibaca pengguna, dihitung server. */
  status_label: string
  komite_approval: string

  diinput_pada: string
  diputuskan_pada?: string

  status_layanan: string
  id_rekening_kasir: string
  respons_kasir: string

  /**
   * Boleh tidak rekening ini menerima pembayaran klaim.
   *
   * Dihitung server dari dua syarat — disetujui komite DAN masih aktif — supaya
   * keduanya tidak perlu diulang di setiap layar.
   */
  dapat_dipakai: boolean
}

export type Bank = {
  kode: string
  nama: string
}

export type AccountListResponse = {
  rekening: Account[]
  /** Banyaknya baris yang cocok SEBELUM dipotong paginasi. */
  jumlah: number
  batas: number
  lewati: number
}

export type BankListResponse = {
  bank: Bank[]
}

/**
 * Kode galat modul Master Rekening.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const AccountErrorCode = {
  notFound: 'rekening_tidak_ditemukan',
  alreadyExists: 'nomor_rekening_sudah_ada',
  alreadyDecided: 'keputusan_sudah_diambil',
  invalidInput: 'isian_tidak_sah',
} as const

export type AccountErrorCode =
  (typeof AccountErrorCode)[keyof typeof AccountErrorCode]

/**
 * Satu baris master ambang komite — satu jenjang persetujuan untuk satu lini bisnis.
 *
 * Nilai uang datang sebagai **teks desimal kanonik** (`"50000001.00"`), bukan angka JSON.
 * Angka JSON adalah floating point ganda di peramban, dan `I-12` menetapkan nilai uang
 * disimpan dengan presisi penuh. Pemformatannya lewat `@/lib/money`.
 */
export type KomiteThreshold = {
  id: string
  nama: string
  operator_id: string
  lini: string
  /**
   * Kolom `TYPE_KOMITE` apa adanya, dan kolom itu memikul **dua arti**: pita nilai di
   * Non-MBU, varian jalur (PA reguler versus PA TKI) di PA. Jangan menafsirkannya di
   * layar — pakai `berpita_nilai` pada respons penjenjangan.
   */
  jenis_komite: string
  batas_bawah: string
  /**
   * `LIMIT_TOP`. **Tidak pernah** dipakai memilih baris — memakainya untuk menyaring
   * akan mengembalikan tepat satu baris dan menghapus penjenjangan seluruhnya (`D-47`).
   * Ia hanya bahan pemeriksaan integritas master.
   */
  batas_atas: string
  /** `DEGREE` — penentu urutan, bukan jumlah jenjang. Boleh berulang dan boleh melompat. */
  jenjang: number
  aktif: boolean
  untuk_adjustment: boolean
  untuk_registrasi: boolean
  untuk_penolakan: boolean
  sedang_absen: boolean
  /** Kesimpulan server: baris ini benar-benar ikut menyetujui nilai klaim. */
  jenjang_persetujuan: boolean
}

/** Aturan pita nilai yang berlaku untuk sebuah lini. Hanya Non-MBU yang memakainya. */
export type KomiteBandPolicy = {
  lini: string
  batas: string
  pita_bawah: string
  pita_atas: string
}

export type KomiteThresholdListResponse = {
  ambang: KomiteThreshold[]
  total: number
  /** Hanya baris yang benar-benar ikut menyetujui. Selalu lebih kecil dari `total`. */
  total_jenjang: number
  /** Datang dari data, bukan dari daftar tetap di dalam kode (`D-15`). */
  lini: string[]
  kebijakan_pita: KomiteBandPolicy[]
  /**
   * Mode penjenjangan portal ini — `"kumulatif"` atau `"satu-penyetuju"`.
   *
   * Ditentukan per **portal**, bukan per lini. Layar memakainya untuk menentukan apakah
   * isian Operator ID pengaju perlu ditampilkan sama sekali.
   */
  mode: string
}

/** Satu hal yang ditemukan pada master ambang. */
export type KomiteFinding = {
  /** `"cacat"` atau `"peringatan"`. Dibedakan lewat nilai ini, bukan teks pesannya. */
  tingkat: string
  jenis: string
  lini: string
  pita?: string
  pesan: string
  id_ambang: string[]
}

export type KomiteIntegrityResponse = {
  temuan: KomiteFinding[]
  jumlah_cacat: number
  jumlah_peringatan: number
}

/** Satu orang yang harus menyetujui sebuah nilai klaim. */
export type KomiteApprover = {
  /** Selalu berurutan tanpa lompatan, 1 sampai jumlah penyetuju. */
  urutan: number
  /** `DEGREE` dari master. Boleh berulang — lihat `urutan_tidak_pasti`. */
  jenjang: number
  nama: string
  operator_id: string
  /** Ambang yang membuat orang ini ikut — alasannya, bukan sekadar hasilnya. */
  batas_bawah: string
  sedang_absen: boolean
  id_ambang: string
}

/** Mode penjenjangan, ditentukan per portal. */
export const KomiteMode = {
  /** Non-MBU, PA, Travel, Bonding: setiap jenjang yang terlampaui ikut menyetujui. */
  cumulative: 'kumulatif',
  /** Simasnet: dipilih tepat satu penyetuju, diacak, penginput dikecualikan. */
  singleApprover: 'satu-penyetuju',
} as const

export type KomiteMode = (typeof KomiteMode)[keyof typeof KomiteMode]

export type KomiteTieringResponse = {
  nilai: string
  lini: string
  mode: string
  /** Membedakan "lini ini tidak memakai pita" dari "pitanya gagal dihitung". */
  berpita_nilai: boolean
  pita?: string
  penyetuju: KomiteApprover[]
  /**
   * Hanya terisi pada mode satu-penyetuju: seluruh orang yang **layak** dipilih pada
   * jenjang terendah, sebelum satu di antaranya diacak.
   *
   * Yang dapat diperiksa pada mode itu bukan siapa yang terpilih — itu acak — melainkan
   * apakah kumpulan yang layak sudah benar.
   */
  kandidat?: KomiteApprover[]
  /** Operator yang **diminta** dikecualikan karena dialah yang mengajukan. */
  dikecualikan_penginput?: string
  /**
   * Orang yang **benar-benar** keluar karena pengecualian itu.
   *
   * Dibedakan dari `dikecualikan_penginput` dengan sengaja: penginput yang bukan anggota
   * komite tidak mengubah apa pun. Isinya paling penting justru saat `penyetuju` kosong —
   * ia satu-satunya keterangan yang menjelaskan kenapa.
   */
  tersingkir?: KomiteApprover[]
  jumlah_jenjang: number
  /** Keadaan, bukan galat: temuan tentang isi master yang harus dilihat Work Owner. */
  tanpa_penyetuju: boolean
  /**
   * Dua penyetuju ber-`DEGREE` sama. Kueri sistem lama mengurutkan dengan
   * `ORDER BY DEGREE` saja, sehingga saat seri urutannya ditentukan basis data.
   */
  urutan_tidak_pasti: boolean
}
