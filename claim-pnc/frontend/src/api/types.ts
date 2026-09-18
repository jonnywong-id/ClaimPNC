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
