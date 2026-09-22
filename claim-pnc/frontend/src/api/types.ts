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

// ── Master Dokumen Travel ────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterdokumentravel/http. Menggantikan layar Pega
// `Harness/BrowseMasterDocumentTravel_Harness-Harness.xml` atas tabel
// POOLDATA.M_DOCTRAVEL.

/**
 * Satu jenis dokumen yang dapat diminta pada klaim lini Travel.
 *
 * Hanya dua field, dan itu memang seluruh isi tabelnya — grid maupun form di layar Pega
 * pun hanya menampilkan keduanya.
 *
 * JANGAN tertukar dengan "Daftar Detail Dokumen Travel" (MENU_ID 39), master terpisah
 * atas V_LST_DOC_TRAVEL yang merujuk `id` di sini dan menambahkan aturan wajib/tidak
 * beserta jumlah unggahan minimum. Tipenya `TravelDocumentDetail` di bawah.
 */
export type TravelDocument = {
  /** Kolom DOCID. Diterbitkan server; tidak pernah diisi pengguna, tidak pernah berubah. */
  id: string
  /** Kolom NAMADOKUMEN. Di layar Pega ia berlabel "Judul Dokumen". */
  judul: string
}

export type TravelDocumentListResponse = {
  dokumen_travel: TravelDocument[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type TravelDocumentResponse = {
  dokumen_travel: TravelDocument
  portal: string
}

/** Badan permintaan penambahan dan penyuntingan. */
export type TravelDocumentInput = {
  judul: string
}

/**
 * Satu aturan kelengkapan dokumen Travel — baris V_LST_DOC_TRAVEL.
 *
 * Layar "Daftar Detail Dokumen Travel" (MENU_ID 39), pengganti
 * `Harness/ListDocumentTravel-Harness.xml`. Ia merujuk `TravelDocument.id` di atas dan
 * menambahkan aturannya: wajib atau tidak, berapa berkas paling sedikit, dan pada plan
 * serta jaminan mana aturan itu berlaku.
 *
 * Nama fieldnya mengikuti label isian di layar Pega
 * (`Section/BrowseDocumentTravel-Section.xml`), supaya satu istilah berlaku dari layar
 * sampai ke kontrak.
 */
export type TravelDocumentDetail = {
  /** Kunci baris. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom DOCID — label layar "ID Dokumen". Rujukan ke master dokumen travel. */
  id_dokumen: string
  /** Kolom DOCUMENTNAME — label layar "Nama Dokumen". */
  nama_dokumen: string
  /**
   * Kolom STSWAJIB — label layar "Status Wajib".
   *
   * Boolean, bukan angka 1/0 seperti di basis data. Angkanya bentuk penyimpanan, dan
   * layar pun sudah mengubahnya menjadi "Ya"/"Tidak" sebelum menampilkannya.
   */
  status_wajib: boolean
  /** Kolom MINUNGGAH — label layar "Minimal Unggah". Nol berarti tanpa tuntutan jumlah. */
  minimal_unggah: number
  /**
   * Pembatasan per Plan dan Jaminan — baris V_LST_DOC_TRAVEL_COVERAGE.
   *
   * SELALU kosong pada hasil daftar; hanya terisi pada pengambilan satu baris. Layar
   * karena itu WAJIB memuat ulang barisnya saat dibuka untuk disunting — memakai baris
   * dari daftar akan membuat form tampak seolah seluruh pembatasannya sudah dihapus, dan
   * menyimpannya benar-benar menghapusnya.
   */
  jaminan: TravelDocumentDetailCoverage[]
}

/** Satu pembatasan plan dan jaminan pada sebuah aturan dokumen. */
export type TravelDocumentDetailCoverage = {
  id: string
  id_plan: string
  nama_plan: string
  id_jaminan: string
  nama_jaminan: string
}

export type TravelDocumentDetailListResponse = {
  detail_dokumen_travel: TravelDocumentDetail[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type TravelDocumentDetailResponse = {
  detail_dokumen_travel: TravelDocumentDetail
  portal: string
}

/** Badan permintaan penambahan dan penyuntingan detail dokumen travel. */
export type TravelDocumentDetailInput = {
  id_dokumen: string
  nama_dokumen: string
  status_wajib: boolean
  minimal_unggah: number
  /**
   * Susunan AKHIR yang dikehendaki petugas — server menggantikan seluruh daftar lamanya,
   * bukan menambahinya.
   */
  jaminan: TravelDocumentDetailCoverageInput[]
}

/**
 * Satu baris grid plan dan jaminan yang dikirim layar.
 *
 * Nama DAN kode keduanya dikirim. Sebabnya perilaku layar lama: autocomplete-nya
 * menyalin kode ke properti tersembunyi saat sebuah pilihan dipilih, tetapi isiannya
 * tetap dapat diisi teks yang tidak ada di master — dan pada keadaan itu yang tersimpan
 * hanyalah namanya, tanpa kode.
 */
export type TravelDocumentDetailCoverageInput = {
  id_plan: string
  nama_plan: string
  id_jaminan: string
  nama_jaminan: string
}

/** Satu pilihan pada isian ID Dokumen, dibaca dari POOLDATA.M_DOCTRAVEL. */
export type TravelDocumentChoice = {
  id: string
  nama: string
}

export type TravelDocumentChoiceListResponse = {
  dokumen: TravelDocumentChoice[]
  portal: string
}

/** Satu pilihan pada isian Nama Plan, dibaca dari POOLDATA.M_PLANTRAVEL milik GISFW. */
export type TravelPlan = {
  id: string
  nama: string
}

/**
 * Satu pilihan pada isian Nama Jaminan.
 *
 * `id_plan` ikut dibawa supaya layar dapat menyaring jaminan menurut plan yang sudah
 * dipilih tanpa menembak server lagi untuk setiap baris grid — penyaringan yang di Pega
 * dikerjakan server lewat parameter `plan` pada `SearchCoverageTravel_RD`.
 */
export type TravelCoverage = {
  id: string
  nama: string
  id_plan: string
}

/**
 * Plan dan jaminan datang dalam SATU respons.
 *
 * Keduanya berasal dari tabel yang sama (POOLDATA.M_PLANTRAVEL) dan layar selalu
 * membutuhkannya bersamaan: grid pembatasan tidak dapat menampilkan satu baris pun tanpa
 * keduanya.
 */
export type TravelPlanListResponse = {
  plan: TravelPlan[]
  jaminan: TravelCoverage[]
  portal: string
}

/**
 * Satu tipe dokumen klaim — kolom V_LST_DOC_TYPE yang benar-benar tampil di layar.
 *
 * Tiga field, dan itu memang seluruh isi layarnya: baik grid maupun form di Pega hanya
 * menampilkan ketiganya. OLD_ID, USER_EDIT, dan TGL_EDIT ada di tabelnya tetapi tidak
 * pernah tampil, sehingga tidak ikut dikirim.
 *
 * JANGAN tertukar dengan dua master turunannya, keduanya belum dibangun:
 * "Daftar Detail Tipe Dokumen" (`V_LST_DET_TYPE_DOC`) dan "Detail Tipe Dokumen per Bisnis"
 * (`LST_TYPE_DOC_BUSINESS`). Keduanya merujuk `id` di sini.
 */
export type DocumentType = {
  /** Kolom ID. Diterbitkan server; tidak pernah diisi pengguna, tidak pernah berubah. */
  id: string
  /** Kolom TYPE_DOCUMENT. Di grid Pega ia berlabel "Tipe Dokumen". */
  tipe_dokumen: string
  /**
   * Kolom STS_PROSES, berlabel "Status Proses" di layar Pega.
   *
   * **Teks bebas, BUKAN penanda aktif/non-aktif.** Di Pega ia isian teks biasa tanpa
   * daftar pilihan, dan layar Arsip Dokumen membacanya sebagai catatan — bahkan
   * mengalias-namakannya "NoteKasir". Jangan memperlakukannya sebagai enum, dan jangan
   * merendernya sebagai lencana berstatus.
   */
  status_proses: string
}

export type DocumentTypeListResponse = {
  tipe_dokumen: DocumentType[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type DocumentTypeResponse = {
  tipe_dokumen: DocumentType
  portal: string
}

/** Badan permintaan penambahan dan penyuntingan. */
export type DocumentTypeInput = {
  tipe_dokumen: string
  status_proses: string
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
  //
  // Beberapa modul memakai kode "tidak ditemukan" MILIKNYA SENDIRI alih-alih `notFound`
  // yang umum. Itu keadaan yang ada hari ini, bukan rancangan — penyeragamannya masuk
  // TKT-F1-004. Sampai itu diputuskan, setiap kode yang benar-benar dikirim server harus
  // terdaftar di sini; kode yang tidak terdaftar akan jatuh ke cabang bawaan dan layar
  // menampilkan pesan umum untuk keadaan yang sebenarnya dapat dijelaskan.
  claimStatusNotFound: 'status_klaim_tidak_ditemukan',
  travelDocumentDetailNotFound: 'detail_dokumen_travel_tidak_ditemukan',
  statusLabelTaken: 'label_status_sudah_dipakai',
  statusCodeTaken: 'kode_status_sudah_dipakai',

  /**
   * Milik modul Master Dokumen Travel.
   *
   * Hanya satu, dan itu cerminan modulnya: Work Owner menetapkan layar itu meniru Pega
   * apa adanya tanpa validasi, sehingga tidak ada `validasi_gagal` maupun galat bentrok
   * yang dapat terjadi di sana.
   */
  travelDocumentNotFound: 'dokumen_travel_tidak_ditemukan',

  /**
   * Milik modul Daftar Tipe Dokumen.
   *
   * Hanya satu, dengan alasan yang sama seperti Master Dokumen Travel di atas: Work Owner
   * menetapkan layar itu meniru Pega apa adanya tanpa validasi.
   */
  documentTypeNotFound: 'tipe_dokumen_tidak_ditemukan',
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

// -----------------------------------------------------------------------------
// Master COL Simas Online
//
// Cerminan dto di internal/mastercolsimasonline/http. Menggantikan layar Pega
// `Harness/CauseOfLossInboxSimasOnline-Harness.xml` atas tabel
// POOLDATA.M_CAUSE_OF_LOSS beserta pemetaan bisnisnya.
// -----------------------------------------------------------------------------

/** Satu lini bisnis, dibaca dari POOLDATA.BUSINESS milik GISFW. Hanya dibaca. */
export type Business = {
  /**
   * Kolom BUSINESS.ID.
   *
   * BOLEH KOSONG pada pemetaan yang namanya diketik bebas dan tidak ada di master —
   * perilaku Pega yang dipertahankan (`pyAllowFreeFormInput=true`). Layar harus
   * menyiapkan keadaan itu; ia bukan tanda data rusak.
   */
  id: string
  /** Nama bisnis. Di layar Pega kolomnya berjudul "Bisnis" dan terikat ke `.Note`. */
  nama: string
}

/**
 * Satu baris Master COL Simas Online.
 *
 * COL adalah Cause of Loss — penyebab kerugian. Yang membuatnya "Simas Online" adalah
 * dua isian yang tidak ada di layar Master COL biasa: `id_master_kerugian` dan daftar
 * `bisnis` yang memakainya.
 */
export type CauseOfLoss = {
  /** Kolom M_COL_ID. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom COL_DESC — di layar Pega berlabel "Nama Cause of loss". */
  nama: string
  /**
   * Kolom MST_COL_ID — di layar Pega berlabel "ID Master Kerugian".
   *
   * Isinya `id` cause of loss LAIN yang menjadi induk, bukan kode dari sistem sebelah:
   * isiannya di Pega adalah daftar yang sumbernya `BrowseVMCauseOfLoss_RD` atas master
   * yang sama. Kosong berarti tanpa induk.
   */
  id_master_kerugian: string
  /**
   * Daftar bisnis yang memakai penyebab kerugian ini.
   *
   * Pada jawaban DAFTAR ia selalu `[]`, dan itu disengaja: grid hanya menampilkan ID dan
   * nama. Layar memuatnya saat baris dibuka untuk disunting.
   */
  bisnis: Business[]
}

export type CauseOfLossListResponse = {
  cause_of_loss: CauseOfLoss[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type CauseOfLossResponse = {
  cause_of_loss: CauseOfLoss
  portal: string
}

export type BusinessListResponse = {
  bisnis: Business[]
  portal: string
}

/**
 * Badan permintaan penambahan dan penyuntingan.
 *
 * `bisnis` berisi NAMA bisnis, bukan ID-nya — itulah yang diketik dan dilihat petugas di
 * layar Pega, dan nama yang diketik bebas memang tidak punya ID. Server yang
 * menyelesaikannya menjadi ID dengan mencocokkan ke master; nama yang tidak cocok tetap
 * diterima dan disimpan tanpa ID.
 */
export type CauseOfLossInput = {
  nama: string
  id_master_kerugian: string
  bisnis: string[]
}
