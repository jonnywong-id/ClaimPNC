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

  /** Milik Inbox Auto Claim: berkas unggahan tidak memuat satu baris data pun. */
  emptyUpload: 'unggahan_kosong',
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

export type AccountErrorCode = (typeof AccountErrorCode)[keyof typeof AccountErrorCode]

// ── Inbox Auto Claim ──────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/inboxautoclaim/http. Menggantikan layar Pega
// `Harness/InboxAutoClaim-Harness.xml` atas POOLDATA.TMP_BATCH_AUTO_CLAIM yang dijoin ke
// POOLDATA.M_AUTO_CLAIM_PNC.
//
// Nama field di sini sudah dinamai ulang mengikuti `D-19`. Grid lama mengikat kedelapan
// kolomnya ke property yang namanya tidak ada hubungannya dengan isinya — `.CaseID` untuk
// kode perusahaan, `.CauseOfLoss` untuk nomor batch, `.ClaimID` untuk jumlah gagal.

/**
 * Satu baris grid Inbox Auto Claim: satu unggahan milik satu perusahaan rekanan.
 *
 * Perusahaan mengirim klaimnya BORONGAN sebagai berkas, bukan satu per satu lewat layar
 * Register. Satu baris di sini adalah satu pasangan (kode perusahaan × nomor batch).
 */
export type AutoClaimBatch = {
  /** Kolom INISIALID — kode singkat perusahaan rekanan. */
  kode_perusahaan: string
  /** Dapat KOSONG bila kodenya tidak ada di Master Auto Claim; barisnya tetap tampil. */
  nama_perusahaan: string
  /** Kolom BATCH — nomor unggahan, unik di dalam satu perusahaan. */
  batch: string

  /**
   * Kolom TGLPROSES, dan ia bagian KUNCI PENGELOMPOKAN grid — bukan sekadar tampilan.
   *
   * Satu nomor batch yang diunggah pada dua tanggal berbeda tampil sebagai DUA baris.
   * Tanpa kolomnya, kedua baris itu tampak kembar tanpa sebab.
   */
  tanggal_proses: string

  jumlah_upload: number
  jumlah_proses: number
  jumlah_berhasil: number
  jumlah_gagal: number
  /** Dihitung server dari selisih upload dan proses; layar tidak menghitungnya sendiri. */
  jumlah_belum_proses: number

  /** Kolom USERINPUT — pengguna yang mengunggah batch ini. */
  user_upload: string
}

/** Keadaan satu baris klaim di dalam batch. */
export const AutoClaimResult = {
  succeeded: 'berhasil',
  failed: 'gagal',
  /** Belum tersentuh pemrosesan. BUKAN sama dengan gagal. */
  pending: 'belum',
} as const

export type AutoClaimResult = (typeof AutoClaimResult)[keyof typeof AutoClaimResult]

/**
 * Satu baris klaim di dalam sebuah batch — isi layar DETAIL.
 *
 * Seluruh nilainya bertipe teks, termasuk tanggal dan nilai uang. Tanggal karena kolomnya
 * memang menyimpan teks `dd/mm/yyyy` dan mengubahnya berarti mengubah isinya; nilai uang
 * karena `I-12` menuntut presisi penuh dan pembulatan hanya saat ditampilkan.
 */
export type AutoClaimLine = {
  nomor_polis: string
  prod_ke: string
  /** Terbit setelah baris berhasil diproses; kosong selama belum. */
  nomor_klaim: string
  nomor_aksep: string
  /** KODE mata uang hasil lookup ke POOLDATA.CURRENCY, bukan id yang tersimpan. */
  mata_uang: string
  nilai_klaim: string
  penyebab_kerugian: string
  tanggal_kejadian: string
  tanggal_lapor: string
  /** Kolom TGLPROSES — kolom kedua pada grid rincian Pega. */
  tanggal_proses: string
  catatan: string
  keyword: string
  /**
   * Dua kolom yang tidak tampil di grid rincian Pega dan tidak terbaca dari kueri mana
   * pun; keberadaannya baru diketahui dari DDL. Dibawa apa adanya supaya isinya dapat
   * diperiksa saat gerbang 1, bukan karena artinya sudah dipahami.
   */
  nama_objek: string
  flag_tidak_bayar: string
  /** Kolom TMP_MESSAGE apa adanya. Kosong berarti belum diproses. */
  keterangan: string
  /** Keadaan barisnya, dipakai memilih lencana — bukan mencocokkan teks keterangan. */
  hasil: AutoClaimResult
}

/** Satu pilihan pada penyaring Nama Perusahaan. */
export type AutoClaimCompany = {
  kode: string
  nama: string
}

/** Keterangan halaman pada daftar yang dipaginasi server. */
export type Pagination = {
  halaman: number
  ukuran: number
  total: number
  /** Dikirim server, bukan dihitung layar — pembulatannya mudah salah pada sisa halaman. */
  total_halaman: number
}

export type AutoClaimBatchListResponse = {
  batch: AutoClaimBatch[]
  paginasi: Pagination
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type AutoClaimLineListResponse = {
  kode_perusahaan: string
  batch: string
  baris: AutoClaimLine[]
  paginasi: Pagination
  portal: string
}

/**
 * Satu tab pada layar Inbox Auto Claim.
 *
 * Ketiganya berasal dari satu harness Pega dan hanya berbeda TABEL batch-nya:
 *
 *   aneka   TMP_BATCH_AUTO_CLAIM    kolom INISIALID
 *   kredit  TMP_BATCH_CLAIM_KREDIT  kolom AGENID
 *   travel  TMP_BATCH_AUTO_TRAVEL   kolom INISIALID
 *
 * Daftarnya datang dari SERVER, tidak ditulis di sini: tab adalah pengetahuan tentang
 * tabel mana yang ada, dan itu milik backend.
 */
export type AutoClaimTab = {
  kode: string
  /** Teks tab, disalin dari pyCaption harness Pega: ANEKA, Asuransi Kredit, Travel. */
  label: string
  /** Tabel Oracle yang dibaca tab ini, ditampilkan sebagai keterangan sumber grid. */
  tabel: string
}

export type AutoClaimTabListResponse = {
  tab: AutoClaimTab[]
  bawaan: string
}

export type AutoClaimCompanyListResponse = {
  perusahaan: AutoClaimCompany[]
  portal: string
}

/**
 * Satu irisan grafik sekaligus satu baris tabel ringkasan.
 *
 * Yang dihitung adalah BATCH, bukan baris klaim — satuannya sama dengan satu baris grid.
 * Dengan begitu angka di sini dan total paginasi setelah disaring selalu cocok: pengguna
 * yang melihat "3" lalu mengeklik perusahaan itu mendapat tepat tiga baris.
 */
export type AutoClaimCompanySummary = {
  kode: string
  /** Dapat kosong bila kodenya tidak ada di Master Auto Claim; irisannya tetap ada. */
  nama: string
  jumlah_batch: number
}

export type AutoClaimSummaryResponse = {
  perusahaan: AutoClaimCompanySummary[]
  /** Baris "All". Dikirim server, bukan dijumlahkan layar. */
  total: number
  portal: string
}

/**
 * Satu batch yang terbentuk dari sebuah unggahan.
 *
 * Perusahaannya TIDAK diketik pengunggah — ia diturunkan dari nomor polis tiap baris.
 * Akibatnya satu berkas dapat menghasilkan beberapa batch sekaligus.
 */
export type AutoClaimUploadBatch = {
  kode_perusahaan: string
  nama_perusahaan: string
  batch: string
  jumlah_baris: number

  /**
   * Hasil PEMERIKSAAN SAAT UNGGAH, bukan hasil pemrosesan menjadi klaim.
   *
   * Baris "lolos" berarti lolos pemeriksaan polis dan MENUNGGU diproses — ia belum
   * menjadi klaim apa pun. Kata "berhasil" sengaja dihindari supaya tidak tertukar
   * dengan hasil akhir yang tampil di grid.
   */
  jumlah_lolos: number
  jumlah_bertanda: number
}

/** Satu baris berkas yang TIDAK disisipkan sama sekali. */
export type AutoClaimUploadRejected = {
  /** Nomor baris di dalam berkas, terhitung sejak baris judul. */
  baris: number
  nomor_polis: string
  pesan: string
}

/**
 * Hasil satu unggahan, membedakan TIGA keadaan:
 *
 * - lolos — disisipkan, menunggu diproses
 * - bertanda — disisipkan beserta pesan galatnya, persis perilaku Pega
 * - ditolak — TIDAK disisipkan karena perusahaannya tidak dapat diturunkan dari polis
 *
 * Yang bertanda masih ada di tabel dan masih terlihat di grid; yang ditolak tidak ada di
 * mana pun. Pengguna yang mengira keduanya sama akan mencari baris yang tidak pernah
 * tersimpan.
 */
export type AutoClaimUploadResponse = {
  batch: AutoClaimUploadBatch[]
  jumlah_baris: number
  ditolak: AutoClaimUploadRejected[]
  portal: string
}

/**
 * Bentuk berkas yang diterima unggahan.
 *
 * Dilayani server, tidak disalin ke sini: bila flow action Pega yang asli akhirnya tiba
 * dan judul kolomnya ternyata berbeda, yang berubah hanya satu tempat.
 */
export type AutoClaimUploadTemplateResponse = {
  kolom_wajib: string[]
  kolom_opsional: string[]
  batas_baris: number
}
