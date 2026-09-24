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

// Master Status Progres 2 — tingkat kedua.
//
// Cerminan dto2 di internal/masterstatusprogres/http. Menggantikan layar Pega
// `Harness/StatusProgress2-Harness.xml` atas tabel POOLDATA.GCNM_MST_PROGRESS.
//
// PERHATIKAN NAMA TABELNYA: yang berakhiran `_KLAIM` adalah tingkat SATU.

/**
 * Satu baris master status progres tingkat 2.
 *
 * Kueri lama mengaliaskan kelima kolomnya ke nama yang tidak mencerminkan isi sama sekali
 * — dan `City`/`CityID` di sana TIDAK berpasangan seperti dugaan yang wajar: `City` adalah
 * nama induk sedangkan `CityID` adalah nama baris ini sendiri. Alias itu tidak dibawa
 * (`D-19`).
 */
export type ProgressStatus2 = {
  /** Kolom ID_MST. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom STS_PROGRESS2 — keterangan status tingkat 2 yang dibaca petugas. */
  nama: string
  /** Kolom ID_PROGRESS — Status Progres 1 yang menaungi baris ini. */
  id_induk: string
  /**
   * Kolom STS_PROGRESS1 — SALINAN nama induk pada saat baris ini disimpan.
   *
   * Ia salinan, bukan hasil join; itu perilaku sistem lama yang dijalankan as-is atas
   * keputusan Work Owner 2026-09-18. Nilainya karena itu dapat berbeda dari nama induk
   * yang berlaku sekarang bila induknya pernah diganti nama.
   */
  nama_induk: string
  /**
   * Kolom TIPE, dibaca apa adanya dan tidak pernah ditulis.
   *
   * Artinya tidak diketahui — di seluruh export ia hanya muncul pada dua SELECT, tanpa
   * satu pun INSERT, UPDATE, maupun penyaring, dan DDL tabelnya belum diterima (R-08).
   */
  tipe: string
}

/** Satu pilihan pada dropdown "Status Progres 1". */
export type ProgressStatus2Parent = {
  id: string
  nama: string
}

export type ProgressStatus2ListResponse = {
  status_progres_2: ProgressStatus2[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type ProgressStatus2Response = {
  status_progres_2: ProgressStatus2
  portal: string
}

export type ProgressStatus2ParentListResponse = {
  /**
   * Daftar induk BERBEDA antarentitas — ia dibaca dari tabel tingkat 1 milik portal yang
   * bersangkutan, bukan daftar tetap milik aplikasi seperti halnya posisi klaim.
   */
  induk: ProgressStatus2Parent[]
  portal: string
}

/**
 * Badan permintaan penambahan tingkat 2.
 *
 * Dua isian saja, persis seperti layar lama. ID diturunkan server dari isi tabel,
 * `nama_induk` disalin server dari baris induk, dan `tipe` tidak pernah ditulis.
 */
export type ProgressStatus2Input = {
  nama: string
  id_induk: string
}

/**
 * Satu baris Status Penolakan 2 — POOLDATA.MST_PENOLAKAN_KLAIM_2.
 *
 * Kueri lama mengaliaskan kedelapan kolomnya ke nama yang tidak mencerminkan isi sama
 * sekali, dan `City`/`CityID` di sana TIDAK berpasangan seperti dugaan yang wajar —
 * persis seperti pada Master Status Progres 2. Alias itu tidak dibawa (`D-19`).
 */
export type Rejection = {
  /** Kolom ID_ND. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom NOTE_ND — Status Penolakan 2. */
  nama: string
  /** Kolom ID_ST — Status Penolakan 1 yang menaungi baris ini. */
  id_status_1: string
  /** Kolom NOTE_ST — SALINAN nama induk saat baris ini disimpan. */
  nama_status_1: string
  /** Kolom STATUS apa adanya: `'0'` menunggu, `'1'` disetujui, `'2'` ditolak. */
  status: string
  /**
   * Sebutan status yang dibaca pengguna: MENUNGGU, APPROVED, atau REJECTED.
   *
   * Teksnya sama persis dengan derivasi kueri lama, termasuk huruf besarnya (`D-13`).
   * Layar menandai baris berdasarkan `status`, bukan teks ini.
   */
  status_label: string
  /** Kolom USER_INPUT. */
  diajukan_oleh: string
  /** Kolom TANGGALKIRIM, RFC 3339 UTC; kosong bila tidak terisi. */
  diajukan_pada: string
  /** Kolom APPROVEBY — diisi layar Inbox Manager, baca-saja di sini. */
  disetujui_oleh: string
  /** Kolom TANGGAL_APPROVE; kosong bila belum pernah diputuskan. */
  disetujui_pada: string
  /** Kolom NOTEAPPROVED — catatan checker, baca-saja di sini. */
  catatan_persetujuan: string
}

/** Satu pilihan pada daftar "Status Penolakan 1". */
export type RejectionParent = {
  id: string
  nama: string
}

export type RejectionListResponse = {
  penolakan_klaim: Rejection[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type RejectionResponse = {
  penolakan_klaim: Rejection
  portal: string
}

export type RejectionParentListResponse = {
  /**
   * Daftar BERBEDA antarentitas — ia dibaca dari tabel milik portal yang bersangkutan,
   * bukan daftar tetap milik aplikasi seperti halnya posisi klaim.
   */
  status_1: RejectionParent[]
  portal: string
}

/**
 * Badan permintaan penambahan dan penyuntingan Status Penolakan 2.
 *
 * `id_status_1` dan `nama_status_1` adalah DUA CARA menunjuk induk, dan tepat satu yang
 * boleh terisi: yang pertama memilih induk yang sudah ada, yang kedua membuat induk baru.
 * Keduanya terisi bersamaan ditolak server — dua sumber untuk satu nilai berarti keduanya
 * dapat berbeda.
 *
 * ID, status, pelaku, dan waktunya tidak ada di sini: seluruhnya diterbitkan server.
 */
export type RejectionInput = {
  nama: string
  id_status_1: string
  nama_status_1: string
}

/** Satu baris Master Penolakan Komite — POOLDATA.MST_REJECTED_KOMITE. */
export type CommitteeRejection = {
  /** Kolom IDMASTER. Diterbitkan server; baris pertama pada tabel kosong bernomor 111. */
  id: string
  /** Kolom NOTEMASTER — di layar lama berlabel "Note Komite Reject". */
  catatan: string
}

export type CommitteeRejectionListResponse = {
  penolakan_komite: CommitteeRejection[]
  portal: string
}

export type CommitteeRejectionResponse = {
  penolakan_komite: CommitteeRejection
  portal: string
}

/** Badan permintaan penambahan dan penyuntingan Penolakan Komite — satu isian saja. */
export type CommitteeRejectionInput = {
  catatan: string
}

// ── Master Auto Claim ────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterautoclaim/http. Menggantikan layar Pega
// `Harness/AutoKlaim-Harness.xml` atas tabel POOLDATA.M_AUTO_CLAIM_PNC.
//
// Ia daftar **Sumber Bisnis yang klaimnya boleh dibuat otomatis**, beserta ke mana
// ganti ruginya dibayarkan — bukan master rekening dan bukan master klaim. Kuncinya
// dicocokkan dengan `T_GENERAL.SOURCEOFBUSINESS` milik polis saat klaim otomatis dibuat.

/**
 * Posisi sebuah baris dalam alur persetujuan komite.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.M_AUTO_CLAIM_PNC. Ia
 * tidak dapat dipilih bebas selama tabel yang sama masih dibaca sistem lama.
 *
 * Kebetulan sama persis dengan AccountStatus milik Master Rekening, dan keduanya
 * sengaja TIDAK dipakai bersama: keduanya milik tabel yang berbeda, dan menyatukannya
 * berarti perubahan pada satu master menyeret master lain.
 */
export const AutoClaimStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type AutoClaimStatus = (typeof AutoClaimStatus)[keyof typeof AutoClaimStatus]

/**
 * Satu baris Master Auto Claim.
 *
 * Nama field di sini sudah dinamai ulang mengikuti `D-19`. Kueri Pega mengaliaskan
 * sepuluh kolomnya ke nama yang tidak mencerminkan isi — dan dua di antaranya
 * menyesatkan secara aktif: `City` berarti NAMA_PENERIMA saat dibaca tetapi
 * BANK_PENERIMA saat ditulis, sedangkan `District`/`DistrictID` tidak berpasangan
 * (yang satu NO_REKENING, yang lain EMAIL_LAPOR).
 */
export type AutoClaim = {
  /** Kolom INISIALID — kode Sumber Bisnis, kunci baris ini. */
  inisial: string

  /**
   * Kolom NAMA_PENERIMA.
   *
   * TIDAK dapat diubah setelah baris dibuat: `UpdateAutoClaim-SQL.xml` tidak menyebut
   * kolomnya, dan keputusan Work Owner 2026-09-19 mempertahankannya. Layar
   * menampilkannya sebagai keterangan, bukan isian.
   */
  nama_penerima: string

  nama_bank: string
  no_rekening: string
  pct_max: string
  pic_lapor: string
  email_lapor: string
  alamat_penerima: string

  id_client: string
  nama_client: string

  /**
   * Kolom CLAIM_ALLOWED, dibaca apa adanya.
   *
   * Aplikasi ini SELALU menulis "1" (keputusan Work Owner 2026-09-19), tetapi baris
   * lama dapat bernilai lain — dan nilai itu menentukan apakah klaim otomatis
   * benar-benar terbentuk. Ia ditampilkan supaya petugas punya cara mengetahui kenapa
   * sebuah baris yang tampak disetujui tidak pernah dipakai.
   */
  claim_allowed: string

  /** Kolom KOMITE — operator yang berwenang memutuskan baris ini. */
  komite: string

  status: string
  /** Sebutan status dalam bahasa yang dibaca pengguna, dihitung server. */
  status_label: string

  /**
   * Boleh tidak baris ini dipakai pembuatan klaim otomatis.
   *
   * Dihitung server dari dua syarat — disetujui komite DAN `claim_allowed` "1" —
   * supaya keduanya tidak perlu diulang di setiap layar, dan supaya keduanya persis
   * sama dengan penyaring `GetReceiverClaimAsuransiKredit-SQL.xml`.
   */
  dapat_dipakai: boolean
}

/** Satu pilihan pada lookup Sumber Bisnis, dari POOLDATA.AGENT. */
export type BusinessSource = {
  id: string
  /**
   * Kolom POOLDATA.AGENT.CLIENTNAME.
   *
   * Namanya menyesatkan: pada tabel AGENT, `CLIENTNAME` adalah nama SUMBER BISNIS,
   * bukan nama tertanggung. Master tertanggung yang sebenarnya adalah POOLDATA.CLIENT.
   */
  nama: string
}

/** Satu pilihan pada lookup Client, dari POOLDATA.CLIENT. */
export type AutoClaimClient = {
  id: string
  nama: string
}

/** Satu pilihan pada dropdown Bank Penerima, dari GENERAL.LST_BANK_GROUP. */
export type AutoClaimBank = {
  /** LBG_ID. TIDAK pernah disimpan ke master — tabelnya tidak punya kolom kode bank. */
  kode: string
  /** BANK_GROUP. Inilah yang disimpan sebagai BANK_PENERIMA. */
  nama: string
}

export type AutoClaimListResponse = {
  auto_claim: AutoClaim[]
  /** Penyaring yang benar-benar dipakai server, bukan yang diminta. */
  status: string
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type AutoClaimResponse = {
  auto_claim: AutoClaim
  portal: string
}

export type BusinessSourceListResponse = {
  sumber_bisnis: BusinessSource[]
  portal: string
}

export type AutoClaimClientListResponse = {
  client: AutoClaimClient[]
  portal: string
}

export type AutoClaimBankListResponse = {
  bank: AutoClaimBank[]
  portal: string
}

/**
 * Badan permintaan penambahan.
 *
 * Empat nilai yang ada di tabel tidak ada di sini karena keempatnya diturunkan server:
 * status (selalu menunggu), claim_allowed (selalu "1"), komite (hasil lookup
 * EMAILKOMITE), dan operator penginput.
 */
export type AutoClaimInput = {
  inisial: string
  nama_penerima: string
  nama_bank: string
  no_rekening: string
  pct_max: string
  pic_lapor: string
  email_lapor: string
  alamat_penerima: string
  id_client: string
  nama_client: string
}

/**
 * Badan permintaan penyimpanan — sekaligus keputusan komite.
 *
 * SATU bentuk untuk TIGA tombol, karena di sistem lama memang satu:
 * `Activity/UpdateMstAutoClaim_act` melayani Update, Approve, dan Reject sekaligus,
 * dibedakan hanya oleh parameter `stsapprove`. Keputusan Work Owner 2026-09-19
 * mempertahankan bentuk itu.
 *
 * `inisial` dan `nama_penerima` tidak ada: yang pertama di jalur URL, yang kedua tidak
 * dapat diubah.
 */
export type AutoClaimSaveInput = {
  nama_bank: string
  no_rekening: string
  pct_max: string
  pic_lapor: string
  email_lapor: string
  alamat_penerima: string
  id_client: string
  nama_client: string
  /** "0" simpan (kembali menunggu) · "1" approve · "2" reject. */
  status: string
}

/**
 * Kode galat modul Master Auto Claim.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const AutoClaimErrorCode = {
  /** Sumber bisnis itu sudah punya baris — konflik keadaan, bukan isian yang cacat. */
  initialTaken: 'sumber_bisnis_sudah_ada',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type AutoClaimErrorCode =
  (typeof AutoClaimErrorCode)[keyof typeof AutoClaimErrorCode]

// ── Master Pasal Kerugian ────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterpasal/http. Menggantikan layar Pega
// `Harness/DetailMasterPasalRejected-Harness.xml` (MENU_ID 27) atas tabel
// POOLDATA.V_M_DATA_PASAL.
//
// Tabelnya hanya punya TIGA kolom — IDDATA, IDPASAL, dan JSONPASAL — dan seluruh isi
// selain No Pasal hidup di dalam dokumen JSON pada kolom ketiga. Bentuk di bawah adalah
// dokumen itu sesudah dibongkar server; layar tidak pernah melihat JSON-nya.

/**
 * Satu butir Kategori pasal.
 *
 * `kode` KOSONG untuk "Notifikasi", dan itu bukan nilai yang belum diisi: ekspresi Pega
 * yang menurunkannya memakai cabang `else`, bukan `when == 3`
 * (`Activity/CNMInsertPasalDataMaster-Act.xml` langkah 1). Daftar pilihan aslinya tidak
 * ada di export (`R-16`), sehingga mengarang kode "3" berarti menebak.
 */
export type ClauseCategory = {
  kode: string
  nama: string
}

/** Satu lini bisnis — POOLDATA.BUSINESS. */
export type ClauseBusiness = {
  /** Kolom ID. BOLEH kosong pada butir yang diketik bebas, seperti di layar lama. */
  id: string
  /** Kolom NOTE. */
  nama: string
}

/**
 * Satu baris Master Pasal Kerugian.
 *
 * Kelas Pega-nya `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS` — kelas Detail Cause of Loss, bukan
 * kelas pasal — sehingga tidak satu pun nama propertinya menyebutkan isinya. Nama itu
 * tidak dibawa (`D-19`).
 */
export type Clause = {
  /** Kolom IDDATA. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /**
   * Kolom IDPASAL — di layar berlabel "No Pasal".
   *
   * Ia TIDAK dijamin unik; kuncinya `id`. Pega pun tidak memeriksanya, dan keputusan Work
   * Owner 2026-09-19 mempertahankannya apa adanya.
   */
  no_pasal: string
  /** `$.DESCRIPTION` — di grid berlabel "ISI PASAL". */
  isi_pasal: string
  /**
   * `$.OLD_D_COL_ID` — di grid berlabel "Deskripsi".
   *
   * Pasangannya dengan `isi_pasal` memang terbalik dari dugaan yang wajar; itu bentuk
   * aslinya, bukan salah petakan.
   */
  deskripsi: string
  /** `$.pyCountry` — kode kategori apa adanya; lihat ClauseCategory. */
  kategori: string
  /** Sebutan kategori, diturunkan server dari `kategori`. */
  kategori_label: string
  /**
   * Lini bisnis tempat pasal ini berlaku.
   *
   * KOSONG pada hasil daftar, dan itu bukan kelalaian: grid layar lama pun tidak
   * menampilkannya. Ia terisi saat satu baris dibaca untuk disunting.
   */
  bisnis: ClauseBusiness[]
}

export type ClauseListResponse = {
  pasal_kerugian: Clause[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type ClauseResponse = {
  pasal_kerugian: Clause
  portal: string
}

export type ClauseBusinessListResponse = {
  bisnis: ClauseBusiness[]
  portal: string
}

/**
 * Daftar pilihan Kategori.
 *
 * Ia TIDAK menyebutkan portal: isinya milik aplikasi, bukan dibaca dari basis data entitas
 * mana pun.
 */
export type ClauseCategoryListResponse = {
  kategori: ClauseCategory[]
}

export type ClauseDeleteResponse = {
  id: string
  portal: string
}

/**
 * Badan permintaan penambahan dan penyuntingan pasal.
 *
 * `kategori_label` tidak ada di sini: ia turunan server. Mengirimkannya DITOLAK dengan 400,
 * bukan diabaikan — server menolak field yang tidak dikenal.
 */
export type ClauseInput = {
  no_pasal: string
  isi_pasal: string
  deskripsi: string
  kategori: string
  bisnis: ClauseBusiness[]
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

// ── Master Bengkel ───────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterbengkel/http. Menggantikan layar Pega
// `Harness/BengkelHE-Harness.xml` (MENU_ID 28) atas tabel POOLDATA.BENGKEL_HE.
//
// Ia master KOMERSIAL, bukan sekadar daftar alamat bengkel: diskon jasa, diskon
// sparepart, persen material, PPN, dan jenis PPh di dalamnya menentukan hasil hitungan
// uang pada klaim yang memakai bengkel itu.

/**
 * Posisi sebuah baris dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.BENGKEL_HE. Ia tidak dapat
 * dipilih bebas selama tabel yang sama masih dibaca sistem lama.
 *
 * Kebetulan sama persis dengan AutoClaimStatus dan AccountStatus, dan ketiganya sengaja
 * TIDAK dipakai bersama: ketiganya milik tabel yang berbeda, dan menyatukannya berarti
 * perubahan pada satu master menyeret master lain.
 */
export const WorkshopStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type WorkshopStatus = (typeof WorkshopStatus)[keyof typeof WorkshopStatus]

/**
 * Nilai STATUS_REKANAN yang berarti BUKAN rekanan.
 *
 * Artinya terbaca dari percabangan Pega, bukan dari label mana pun:
 * `Activity/ValidationLoginBengkel_act` melompat keluar pada prasyarat
 * `Local.STS_REKANAN=='0'`, dan langkah yang dilewatinya adalah pembuatan login aplikasi.
 *
 * Konsekuensinya dipakai layar: bengkel non-rekanan tidak wajib punya login aplikasi.
 */
export const NON_PARTNER = '0'

/**
 * Satu baris Master Bengkel.
 *
 * Nama field-nya MENCERMINKAN ISI, bukan nama kolomnya apa adanya — `NAMA_KABUPATEN`
 * menjadi `nama_kota` karena label layar Pega memang "NAMA KOTA", dan `NO_ACCOUNT`
 * menjadi `no_rekening` karena itulah yang dibacanya.
 */
export type Workshop = {
  /** Kolom ID_BENGKEL — kunci baris, diterbitkan server. */
  id_bengkel: string

  nama_bengkel: string
  alamat_bengkel: string
  telp_bengkel: string
  nohp_bengkel: string
  email: string
  /** Surel tujuan perintah kerja, terpisah dari surel umum bengkel. */
  email_wo: string

  id_cabang: string
  nama_cabang: string
  id_kota: string
  nama_kota: string

  status_rekanan: string
  status_bengkel: string
  alasan_status_bengkel: string
  tanggal_status: string

  login_aplikasi: string

  id_bank: string
  nama_bank: string
  no_rekening: string
  nama_rekening: string
  id_rekening: string

  nama_npwp: string
  no_npwp: string
  alamat_npwp: string
  jenis_pph: string

  ppn: string
  diskon_jasa: string
  diskon_sparepart: string
  persen_material: string
  pct_selisih_pl: string
  sla: string

  status_disupply_asm: string
  supplier: string
  status_eklaim: string
  status_auto_aksep: string
  status_payment: string
  status_autopayment: string
  status_tekno: string
  status_order: string

  /** Kolom DOKUMENID. Layar ini tidak mengunggah lampiran; nilainya dipertahankan. */
  id_dokumen: string

  status: string
  status_label: string

  /** Dihitung server dari status_rekanan, supaya layar tidak menyimpan salinan sandinya. */
  rekanan: boolean
}

/** Satu pilihan pada dropdown Cabang. */
export type WorkshopBranch = {
  id: string
  nama: string
}

/** Satu pilihan pada lookup Kota. */
export type WorkshopCity = {
  id: string
  nama: string
}

/**
 * Satu pilihan pada dropdown Bank.
 *
 * Berbeda dari Master Auto Claim yang hanya menyimpan nama banknya, tabel bengkel punya
 * kolom kode banknya sendiri — sehingga keduanya ikut tersimpan.
 */
export type WorkshopBank = {
  kode: string
  nama: string
}

export type WorkshopListResponse = {
  bengkel: Workshop[]
  status: string
  portal: string
}

export type WorkshopResponse = {
  bengkel: Workshop
  portal: string
}

export type WorkshopBranchListResponse = {
  cabang: WorkshopBranch[]
  portal: string
}

export type WorkshopCityListResponse = {
  kota: WorkshopCity[]
  portal: string
}

export type WorkshopBankListResponse = {
  bank: WorkshopBank[]
  portal: string
}

export type WorkshopDecisionResponse = {
  /** Jumlah baris yang BENAR-BENAR berubah — bukan jumlah yang dikirim. */
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Badan permintaan penambahan DAN penyimpanan.
 *
 * Satu bentuk untuk dua jalur, karena isiannya memang sama: layar Pega memakai form yang
 * sama untuk menambah dan mengubah.
 *
 * Yang TIDAK ada di sini: `id_bengkel` (diterbitkan server pada penambahan, diambil dari
 * jalur URL pada penyimpanan), `status` (bukan isian melainkan akibat — penambahan dan
 * penyimpanan SELALU menghasilkan status menunggu), dan `id_dokumen` (jalur unggah
 * lampiran tidak dibawa).
 */
export type WorkshopInput = Omit<
  Workshop,
  'id_bengkel' | 'status' | 'status_label' | 'rekanan' | 'id_dokumen'
>

/** Badan permintaan keputusan borongan. */
export type WorkshopDecisionInput = {
  id_bengkel: string[]
  /** "1" approve · "2" reject · "0" kembalikan ke antrean. */
  status: string
}

/**
 * Kode galat modul Master Bengkel.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const WorkshopErrorCode = {
  /** Nama bengkel itu sudah dipakai — konflik keadaan, bukan isian yang cacat. */
  nameTaken: 'nama_bengkel_sudah_ada',
  /** Login aplikasi itu sudah dipakai bengkel lain. */
  loginTaken: 'login_sudah_dipakai',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type WorkshopErrorCode =
  (typeof WorkshopErrorCode)[keyof typeof WorkshopErrorCode]

/**
 * Posisi sebuah panel dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.PANEL_HE. Ia tidak dapat
 * dipilih bebas selama tabel yang sama masih dibaca sistem lama (ADR-0004).
 */
export const PanelStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type PanelStatus = (typeof PanelStatus)[keyof typeof PanelStatus]

/**
 * Sandi kolom SISI_PANEL pada tabel anak POOLDATA.LOKASI_PANEL_HE.
 *
 * Ketiganya satu-satunya daftar pilihan modul ini yang BENAR-BENAR terbaca dari export
 * Pega — `Activity/SetLokasiSisiPanel-Act.xml` menyusunnya apa adanya. Kesembilan penanda
 * STS_* pada tabel induk tidak punya daftar seperti ini; lihat catatan pada Panel.
 */
export const PanelSide = {
  tidakAda: '-',
  kiri: '1',
  kanan: '2',
} as const

export type PanelSide = (typeof PanelSide)[keyof typeof PanelSide]

/** Satu baris lokasi pada sebuah panel — satu baris POOLDATA.LOKASI_PANEL_HE. */
export type PanelLocation = {
  /** Kolom LOKASI_PANEL — KIRI, KANAN, DEPAN, BELAKANG, atau LAIN-LAIN. */
  lokasi_panel: string
  /** Kolom SISI_PANEL: "-", "1" kiri, "2" kanan. */
  sisi_panel: string
  /** Sebutan sisi yang dibaca pengguna, dihitung server. Tidak dikirim balik saat simpan. */
  sisi_label?: string
}

/**
 * Cerminan dto di internal/masterpanel/http. Menggantikan layar Pega
 * `Harness/MasterPanel_HE-Harness.xml` atas POOLDATA.PANEL_HE (MENU_ID 30).
 *
 * # Kesembilan penanda STS_* nilainya TIDAK diketahui
 *
 * Kesembilannya dirender `pxDropdown` atau `pxRadioButtons` di Pega dengan
 * `pyListSource=associated`, artinya pilihannya datang dari rule Field Value pada
 * propertinya — dan tidak ada satu pun direktori Property maupun Field Value di export
 * (`R-16`). Karena itu tidak satu pun dibuat dropdown bernilai tetap; layar menawarkan
 * nilai yang SUDAH DIPAKAI baris lain sebagai saran.
 *
 * # Satu nama yang memikul dua arti, dan di sini ia dipisah
 *
 * `status_sisi` adalah kolom STS_SISI pada tabel INDUK, caption "STATUS SISI". Ia BUKAN
 * sisi lokasi — nama `STS_SISI` dipakai juga sebagai alias kolom SISI_PANEL pada tabel
 * anak di `RDB List/GetLokasiSisiPanel-SQL.xml`. Sisi lokasi bernama `sisi_panel`, di
 * dalam setiap baris `lokasi`.
 */
export type Panel = {
  /** Kolom ID_PANEL — kunci baris, diterbitkan server. */
  id_panel: string
  nama_panel: string

  status_repair: string
  status_edit_quantity: string
  status_premium_repair: string
  status_pecah: string
  status_sticker: string
  /** Kolom STS_SISI pada tabel INDUK. Bukan sisi lokasi. */
  status_sisi: string
  status_rusak_parah: string
  status_aktif: string
  exclusion_c: string

  /**
   * Kolom STS_APPROVAL — dan ia BUKAN kolom APPROVAL.
   *
   * Artinya tidak disebut di mana pun dalam export: tidak punya caption, tidak dipakai
   * penyaring, tidak dibandingkan di rule mana pun. Ia ditampilkan apa adanya dan tidak
   * punya isian di form.
   */
  status_approval: string

  /** Kolom ALASAN_TOLAK, diisi pada jalur keputusan. */
  alasan_tolak: string

  /** Kolom DOKUMENID. Jalur unggah lampiran tidak dibawa modul ini. */
  id_dokumen: string

  status: string
  status_label: string

  /** Baris anak. SELALU berupa senarai, tidak pernah null. */
  lokasi: PanelLocation[]
}

export type PanelListResponse = {
  panel: Panel[]
  status: string
  portal: string
}

export type PanelResponse = {
  panel: Panel
  portal: string
}

export type PanelDecisionResponse = {
  /** Jumlah baris yang BENAR-BENAR berubah — bukan jumlah yang dikirim. */
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Daftar pilihan Lokasi dan Sisi.
 *
 * Isinya KONSTANTA yang ditanam di `Activity/SetLokasiSisiPanel-Act.xml`, bukan bacaan
 * basis data — karena itu endpoint-nya tidak menuntut portal.
 */
export type PanelOptionsResponse = {
  lokasi_panel: string[]
  sisi_panel: { nilai: string; label: string }[]
}

/**
 * Badan permintaan penambahan DAN penyimpanan.
 *
 * Yang TIDAK ada di sini: `id_panel` (diterbitkan server pada penambahan, diambil dari
 * jalur URL pada penyimpanan), `status` (bukan isian melainkan akibat), `status_approval`
 * dan `alasan_tolak` (tidak punya isian di layar ini), serta `id_dokumen` (jalur unggah
 * lampiran tidak dibawa).
 */
export type PanelInput = Omit<
  Panel,
  | 'id_panel'
  | 'status'
  | 'status_label'
  | 'status_approval'
  | 'alasan_tolak'
  | 'id_dokumen'
>

/** Badan permintaan keputusan borongan. */
export type PanelDecisionInput = {
  id_panel: string[]
  /** "1" approve · "2" reject · "0" kembalikan ke antrean. */
  status: string
  /** Hanya tersimpan pada keputusan TOLAK. */
  catatan: string
}

/**
 * Kode galat modul Master Panel.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const PanelErrorCode = {
  /** Nama panel itu sudah dipakai — konflik keadaan, bukan isian yang cacat. */
  nameTaken: 'nama_panel_sudah_ada',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type PanelErrorCode = (typeof PanelErrorCode)[keyof typeof PanelErrorCode]

/**
 * Nilai JENIS_STATUS yang berarti supplier Heavy Equipment.
 *
 * Artinya terbaca dari percabangan Pega, bukan dari label mana pun:
 * `Activity/CreateNewMasterSupplier_post` step 7 menetapkan `SUPPLIER_HE := "1"` ketika
 * `JENIS_STATUS` bernilai `"1"`, dan `Activity/GetDataSupplier_pre` step 6.3 membalik
 * arahnya saat memuat. Keduanya dua wajah dari satu hal.
 */
export const SUPPLIER_HEAVY_EQUIPMENT = '1'

/**
 * Nilai STS_AKTIF dan STS_AKTIF_PROMLIST.
 *
 * Keduanya terbaca dari percabangan, bukan dari daftar nilai — daftar pilihan aslinya ada
 * di rule Field Value yang TIDAK ikut di export (R-16):
 *
 *	CreateNewMasterSupplier_post step 6   STS_AKTIF := "0" pada setiap supplier baru
 *	EditMasterSupplier_post step 12       persetujuan diminta hanya bila "1" atau kosong
 */
export const SUPPLIER_ACTIVE = '1'
export const SUPPLIER_INACTIVE = '0'

/**
 * Satu baris Master Supplier.
 *
 * Nama field-nya MENCERMINKAN ISI, bukan nama kunci JSON-nya apa adanya — `JENIS_STATUS`
 * menjadi `status_supply` karena label layar Pega memang "Status Supply", dan
 * `STS_AKTIF_PROMLIST` menjadi `status_aktif` karena itulah yang tertulis di atas isiannya.
 *
 * # Seluruh isinya tinggal di SATU kolom JSONDATA
 *
 * `M_SUPPLIER` hanya punya `ID`, `OLDID`, dan `JSONDATA` — tidak ada kembaran berkolom
 * bernama seperti POOLDATA.BENGKEL_HE. Itu tidak terlihat dari sini, dan memang tidak
 * perlu; yang penting bagi layar adalah bentuk di bawah ini.
 */
export type Supplier = {
  /** Kolom ID — kunci baris, diterbitkan server. */
  id_supplier: string

  /**
   * Kolom OLDID — nomor warisan.
   *
   * Tidak dapat disunting: tidak ada satu pun rule di export yang mengisinya. Ia
   * ditampilkan supaya baris lama dapat ditautkan dengan catatan yang menyebut nomor itu.
   */
  id_lama: string

  nama: string
  alamat: string
  kota: string
  nama_cabang: string
  kode_pos: string
  negara: string

  telepon: string
  fax: string
  email: string

  npwp: string
  contact_person: string

  status_rekanan: string
  status_supply: string

  /** Turunan status_supply, bukan isian. Server yang menurunkannya. */
  supplier_he: string

  term_of_payment: string
  term_of_delivery: string

  /** Ikut tersalin ke kolom ALASAN_REQ pada baris permintaan persetujuan. */
  keterangan: string

  bank: string
  no_account: string
  account_name: string
  bank_branch: string

  jenis_supplier: string

  /** Kunci STS_AKTIF_PROMLIST — status aktif yang DIMINTA, isian di form. */
  status_aktif: string

  /**
   * Kunci STS_AKTIF — status aktif yang SEBENARNYA berlaku.
   *
   * Bukan isian. Pada supplier yang baru ditambahkan keduanya BERBEDA: yang diminta dapat
   * "1" sementara yang berlaku selalu "0" sampai persetujuannya turun. Tanpa nilai ini,
   * layar tidak punya cara menjelaskan kenapa supplier yang baru disimpan sebagai aktif
   * belum juga aktif.
   */
  status_aktif_berlaku: string

  status_autopayment: string

  /** Kunci USERKLAIMID dan TGL_INSERT — ditulis sistem lama, tidak pernah dibacanya. */
  diubah_oleh: string
  diubah_pada: string

  /** Dihitung server, supaya layar tidak menyimpan salinan sandinya. */
  heavy_equipment: boolean

  /** Dihitung dari status_aktif_berlaku, BUKAN dari status_aktif. */
  aktif: boolean
}

/** Satu pilihan pada dropdown Cabang. */
export type SupplierBranch = {
  id: string
  nama: string
}

/** Satu pilihan pada lookup Kota. */
export type SupplierCity = {
  id: string
  nama: string
}

/** Satu pilihan pada isian Negara. */
export type SupplierCountry = {
  id: string
  nama: string
}

/**
 * Satu pilihan pada dropdown Bank.
 *
 * Kodenya dikirim meski TIDAK tersimpan — dokumen supplier tidak punya kunci BANK_ID. Ia
 * hanya pembeda bila ada dua bank bernama mirip.
 */
export type SupplierBank = {
  kode: string
  nama: string
}

/**
 * Satu pilihan pada dropdown bersandi.
 *
 * Daftar pilihan kelima isian bersandi TIDAK ADA di export: isiannya `pxDropdown`
 * bersumber `associated`, artinya daftarnya hidup di rule Field Value yang tidak ikut
 * diekspor (R-16). Yang ditawarkan server adalah gabungan sandi yang artinya terbukti di
 * activity dan sandi yang benar-benar dipakai baris yang ada.
 *
 * Label sebuah sandi yang hanya ditemukan di data berisi SANDINYA SENDIRI — bukan tebakan
 * artinya. Menampilkan tebakan yang tampak meyakinkan lebih buruk daripada menampilkan
 * sandinya apa adanya.
 */
export type SupplierCode = {
  nilai: string
  label: string
}

export type SupplierListResponse = {
  supplier: Supplier[]
  portal: string
}

export type SupplierResponse = {
  supplier: Supplier
  portal: string
}

export type SupplierBranchListResponse = {
  cabang: SupplierBranch[]
  portal: string
}

export type SupplierCityListResponse = {
  kota: SupplierCity[]
  portal: string
}

export type SupplierCountryListResponse = {
  negara: SupplierCountry[]
  portal: string
}

export type SupplierBankListResponse = {
  bank: SupplierBank[]
  portal: string
}

/**
 * Kelima daftar sandi, dikirim sekaligus.
 *
 * Satu perjalanan untuk lima daftar, bukan lima: kelimanya dibaca dari tabel yang sama
 * dalam satu kali pemindaian.
 */
export type SupplierCodeListResponse = {
  status_rekanan: SupplierCode[]
  status_supply: SupplierCode[]
  jenis_supplier: SupplierCode[]
  status_aktif: SupplierCode[]
  status_autopayment: SupplierCode[]
  portal: string
}

/**
 * Badan permintaan penambahan DAN penyimpanan.
 *
 * Satu bentuk untuk dua jalur, karena isiannya memang sama:
 * `Section/CreateMasterSupplier_Sec-Section.xml` dipakai KEDUA Flow Action, dan yang
 * berbeda hanyalah activity pre dan post-nya.
 *
 * Yang TIDAK ada di sini seluruhnya diturunkan server: `id_supplier`, `id_lama`,
 * `supplier_he`, `status_aktif_berlaku`, jejak pelaku, dan kedua nilai hitungan.
 *
 * `nama` TETAP dikirim meski tidak dapat diubah — layar mengirimkannya kembali apa adanya
 * karena isiannya memang ada di form, hanya terkunci. Server membandingkannya dengan yang
 * tersimpan dan MENOLAK bila berbeda.
 */
export type SupplierInput = Omit<
  Supplier,
  | 'id_supplier'
  | 'id_lama'
  | 'supplier_he'
  | 'status_aktif_berlaku'
  | 'diubah_oleh'
  | 'diubah_pada'
  | 'heavy_equipment'
  | 'aktif'
>

/**
 * Kode galat modul Master Supplier.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const SupplierErrorCode = {
  /** Nama supplier itu sudah dipakai — konflik keadaan, bukan isian yang cacat. */
  nameTaken: 'nama_supplier_sudah_ada',
  /**
   * Nama diubah pada baris yang sudah tersimpan.
   *
   * Layar mengunci isiannya, sehingga pengguna tidak akan menemuinya lewat jalur biasa.
   * Ia tetap ditangani: permintaan yang tidak datang dari layar tidak tersentuh
   * penguncian itu.
   */
  nameLocked: 'nama_supplier_terkunci',
} as const

export type SupplierErrorCode =
  (typeof SupplierErrorCode)[keyof typeof SupplierErrorCode]

/**
 * Posisi sebuah sparepart dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.SPAREPART_HE. Ia tidak
 * dapat dipilih bebas selama tabel yang sama masih dibaca sistem lama (ADR-0004).
 */
export const SparepartStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type SparepartStatus = (typeof SparepartStatus)[keyof typeof SparepartStatus]

/** Satu pilihan Kategori Sparepart — satu baris POOLDATA.GCNM_M_SPAREPART_CATEGORY. */
export type SparepartCategory = {
  /** Kolom PART_CATEGORY_ID; inilah yang tersimpan di SPAREPART_HE.KATEGORI_SPART. */
  kode: string
  /** Kolom PART_CATEGORY_NAME — yang dilihat pengguna. */
  nama: string
}

/** Satu pilihan Tipe Sparepart — satu baris POOLDATA.GCNM_M_SPAREPART_TYPE. */
export type SparepartType = {
  /** Kolom PART_SECTION_ID; inilah yang tersimpan di SPAREPART_HE.TIPE_SPART. */
  kode: string
  /** Kolom PART_SECTION_NAME — yang dilihat pengguna. */
  nama: string
  /**
   * Kolom PART_CATEGORY_ID — kategori induk tipe ini.
   *
   * Layar memakainya untuk mempersempit daftar Tipe begitu Kategori dipilih.
   */
  kode_kategori: string
}

/**
 * Cerminan dto di internal/mastersparepart/http. Menggantikan layar Pega
 * `Harness/SparePart_HE-Harness.xml` atas POOLDATA.SPAREPART_HE (MENU_ID 31).
 *
 * # Kedelapan isian angka bertipe TEKS
 *
 * Termasuk `harga_jual`. JSON number diurai peramban sebagai IEEE-754 double, sehingga
 * nilai uang dengan banyak digit dapat berubah hanya karena melewati jaringan. Teks
 * melewatinya tanpa disentuh (`I-12`, `D-51`).
 *
 * # Empat penanda yang nilainya TIDAK diketahui
 *
 * `jenis_sparepart`, `satuan`, `status_aktif`, dan `status_sparepart` dirender
 * `pxRadioButtons` atau `pxDropdown` di Pega dengan `pyListSource=associated`, artinya
 * pilihannya datang dari rule Field Value pada propertinya — dan tidak ada satu pun
 * direktori Property maupun Field Value di export (`R-16`). Layar menawarkan nilai yang
 * SUDAH DIPAKAI baris lain sebagai pilihan, bukan daftar yang dikarang.
 */
export type Sparepart = {
  /** Kolom ID — kunci baris, diterbitkan server dari kode situs dan sequence. */
  id_sparepart: string

  nama_sparepart: string
  nomor_sparepart: string
  kode_sparepart: string

  /** Kolom HARGA_JUAL. Teks, bukan angka; lihat catatan pada tipe ini. */
  harga_jual: string

  /** Kolom KATEGORI_SPART — berisi PART_CATEGORY_ID, bukan namanya. */
  kategori_sparepart: string
  /** Nama kategorinya, dihitung server. Kosong bila kategorinya tidak ada di acuan. */
  nama_kategori_sparepart?: string

  /** Kolom TIPE_SPART — berisi PART_SECTION_ID. */
  tipe_sparepart: string
  /** Nama tipenya, dihitung server. Kosong bila tipenya tidak ada di acuan. */
  nama_tipe_sparepart?: string

  berat: string
  panjang: string
  lebar: string
  tinggi: string
  stock_minimal: string
  stock_maximal: string
  kuantitas_pesanan: string

  /** Kolom PROD_DATE — TEKS, bukan tanggal. Pega pun merendernya sebagai isian biasa. */
  tanggal_produksi: string

  part_substitusi: string

  jenis_sparepart: string
  satuan: string
  status_aktif: string
  status_sparepart: string

  /** Kolom USER_UPDATE — login petugas yang terakhir menyimpannya. Baca-saja. */
  user_update: string

  /** Kolom TGL_UPDATE_HARGA dalam RFC 3339 UTC. Kosong berarti belum pernah terisi. */
  tanggal_update_harga?: string

  /** Kolom DOKUMENID. Layar ini tidak mengunggah lampiran, tetapi tetap menampilkannya. */
  id_dokumen: string

  status: string
  status_label: string
}

export type SparepartListResponse = {
  sparepart: Sparepart[]
  status: string
  portal: string
}

export type SparepartResponse = {
  sparepart: Sparepart
  portal: string
}

export type SparepartDecisionResponse = {
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Jawaban GET /api/master/sparepart/pilihan.
 *
 * Berbeda dari Master Panel yang daftar pilihannya konstanta di dalam kode, isi di sini
 * DIBACA DARI BASIS DATA entitas yang sedang dibuka — kategori dan tipe suku cadang
 * berbeda antarentitas. Karena itu permintaannya menuntut portal.
 */
export type SparepartOptionsResponse = {
  kategori: SparepartCategory[]
  tipe: SparepartType[]
  portal: string
}

/**
 * Isian yang dikirim saat menambah dan menyimpan.
 *
 * Lima kolom sengaja tidak ada: kunci baris, status, pelaku, stempel tanggal harga, dan
 * id dokumen — kelimanya diturunkan server.
 */
export type SparepartInput = Omit<
  Sparepart,
  | 'id_sparepart'
  | 'nama_kategori_sparepart'
  | 'nama_tipe_sparepart'
  | 'user_update'
  | 'tanggal_update_harga'
  | 'id_dokumen'
  | 'status'
  | 'status_label'
>

/**
 * Badan permintaan keputusan borongan.
 *
 * TANPA catatan, berbeda dari Master Panel: POOLDATA.SPAREPART_HE tidak punya kolom
 * penampungnya. Layar karena itu tidak menggambar isian catatan sama sekali.
 */
export type SparepartDecisionInput = {
  id_sparepart: string[]
  status: string
}

/**
 * Kode galat modul Master Sparepart.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const SparepartErrorCode = {
  /**
   * Salah satu dari ketiga kunci alami sudah dipakai — konflik keadaan, bukan isian yang
   * cacat.
   *
   * SATU kode untuk nomor, nama, dan kode sparepart. Yang membedakan ketiganya adalah
   * isian mana yang disorot, dan itu sudah disampaikan lewat `detail[].kolom`.
   */
  keyTaken: 'kunci_sparepart_sudah_ada',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type SparepartErrorCode =
  (typeof SparepartErrorCode)[keyof typeof SparepartErrorCode]

/**
 * Posisi sebuah kategori sparepart dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.GCNM_M_SPAREPART_CATEGORY.
 * Ia tidak dapat dipilih bebas: tabel yang sama dibaca sistem lama (ADR-0004) DAN dibaca
 * layar Master Sparepart sebagai daftar acuan, yang menyaring APPROVAL = '1'.
 */
export const PartCategoryStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type PartCategoryStatus =
  (typeof PartCategoryStatus)[keyof typeof PartCategoryStatus]

/**
 * Satu baris Master Kategori Sparepart — satu baris POOLDATA.GCNM_M_SPAREPART_CATEGORY.
 *
 * # Kenapa namanya PartCategory, bukan SparepartCategory
 *
 * Karena `SparepartCategory` sudah dipakai untuk bentuk yang BERBEDA: ia DTO dropdown pada
 * layar Master Sparepart, dan isinya hanya `{kode, nama}`. Yang di sini adalah barisnya
 * sendiri beserta statusnya. Dua bentuk untuk satu tabel, dan keduanya memang dibutuhkan —
 * menyamakannya akan memaksa dropdown membawa kolom yang tidak dipakainya.
 *
 * Namanya mengikuti awalan kolomnya sendiri, `PART_CATEGORY_*`.
 *
 * # Hanya empat field
 *
 * Tabelnya hanya punya tiga kolom, dan yang keempat diturunkan dari yang ketiga. Tidak ada
 * pencatat pelaku maupun stempel waktu — tabelnya tidak punya kolomnya.
 */
export type PartCategory = {
  /** Kolom PART_CATEGORY_ID. Teks, meski isinya angka berurut. */
  id_kategori_sparepart: string
  /** Kolom PART_CATEGORY_NAME — satu-satunya isian yang diketik pengguna. */
  nama_kategori_sparepart: string
  /** Kolom APPROVAL: "0", "1", atau "2". */
  status: string
  /** Sebutan status yang dibaca pengguna: "Waiting Approval", "Approve", "Reject". */
  status_label: string
}

export type PartCategoryListResponse = {
  kategori_sparepart: PartCategory[]
  status: string
  portal: string
}

export type PartCategoryResponse = {
  kategori_sparepart: PartCategory
  portal: string
}

export type PartCategoryDecisionResponse = {
  /** Jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dicentang. */
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Badan permintaan tambah dan simpan.
 *
 * SATU isian. ID tidak dikirim — pada penambahan ia diterbitkan server, dan pada
 * penyimpanan ia ada di jalur URL. Status juga tidak: menyimpan selalu mengembalikan baris
 * ke antrean persetujuan.
 */
export type PartCategoryInput = Pick<PartCategory, 'nama_kategori_sparepart'>

/**
 * Badan permintaan keputusan borongan.
 *
 * TANPA catatan: POOLDATA.GCNM_M_SPAREPART_CATEGORY tidak punya kolom penampungnya. Layar
 * karena itu tidak menggambar isian catatan sama sekali.
 */
export type PartCategoryDecisionInput = {
  id_kategori_sparepart: string[]
  status: string
}

/**
 * Kode galat modul Master Kategori Sparepart.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const PartCategoryErrorCode = {
  /**
   * Nama sudah dipakai kategori lain — konflik keadaan, bukan isian yang cacat.
   *
   * Termasuk bila yang memakainya adalah baris yang sudah DITOLAK: pemeriksaan lamanya
   * tidak menyaring APPROVAL sama sekali (P-5). Pesan pada `detail` yang menjelaskannya.
   */
  nameTaken: 'kunci_kategori_sparepart_sudah_ada',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type PartCategoryErrorCode =
  (typeof PartCategoryErrorCode)[keyof typeof PartCategoryErrorCode]

/**
 * Posisi sebuah tipe sparepart dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.GCNM_M_SPAREPART_TYPE. Ia
 * tidak dapat dipilih bebas: tabel yang sama dibaca sistem lama (ADR-0004) DAN dibaca layar
 * Master Sparepart sebagai daftar acuan, yang menyaring APPROVAL = '1'.
 */
export const PartTypeStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type PartTypeStatus = (typeof PartTypeStatus)[keyof typeof PartTypeStatus]

/**
 * Satu baris Master Tipe Sparepart — satu baris POOLDATA.GCNM_M_SPAREPART_TYPE.
 *
 * # Kenapa namanya PartType, bukan SparepartType
 *
 * Karena `SparepartType` sudah dipakai untuk bentuk yang BERBEDA: ia DTO dropdown pada
 * layar Master Sparepart, dan isinya hanya `{kode, nama, kode_kategori}`. Yang di sini
 * adalah barisnya sendiri beserta status dan nama kategori induknya. Dua bentuk untuk satu
 * tabel, dan keduanya memang dibutuhkan — menyamakannya akan memaksa dropdown membawa kolom
 * yang tidak dipakainya.
 *
 * Pasangannya `PartCategory`, yang menamai hal yang setara pada tabel kategori.
 *
 * # Enam field untuk tabel berkolom empat
 *
 * `nama_kategori_sparepart` bukan kolom tabel ini — ia milik tabel kategori dan dibaca lewat
 * JOIN. `status_label` diturunkan dari `status`. Tidak ada pencatat pelaku maupun stempel
 * waktu: tabelnya tidak punya kolomnya.
 */
export type PartType = {
  /** Kolom PART_SECTION_ID. Teks, meski isinya angka berurut. */
  id_tipe_sparepart: string
  /** Kolom PART_SECTION_NAME. */
  nama_tipe_sparepart: string
  /** Kolom PART_CATEGORY_ID — kategori induk tipe ini. */
  id_kategori_sparepart: string
  /**
   * Kolom PART_CATEGORY_NAME milik tabel kategori, dibaca lewat JOIN.
   *
   * Dapat KOSONG, dan itu bukan galat: kuerinya memakai LEFT JOIN, sehingga tipe yang
   * menunjuk kategori yang tidak ada tetap terkirim. Di Pega baris seperti itu justru
   * HILANG dari daftar. Layar menampilkannya sebagai "—" beserta keterangan.
   */
  nama_kategori_sparepart: string
  /** Kolom APPROVAL: "0", "1", atau "2". */
  status: string
  /** Sebutan status yang dibaca pengguna: "Waiting Approval", "Approve", "Reject". */
  status_label: string
}

export type PartTypeListResponse = {
  tipe_sparepart: PartType[]
  status: string
  portal: string
}

export type PartTypeResponse = {
  tipe_sparepart: PartType
  portal: string
}

export type PartTypeDecisionResponse = {
  /** Jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dicentang. */
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Satu pilihan pada dropdown Kategori di layar Master Tipe Sparepart.
 *
 * Bentuknya sama persis dengan `SparepartCategory` — keduanya membaca tabel yang sama untuk
 * keperluan yang sama. Ia tetap dinamai tersendiri supaya kedua layar dapat berubah
 * sendiri-sendiri; menyatukannya akan membuat perubahan di satu layar menyeret layar lain.
 */
export type PartTypeCategory = {
  /** Kolom PART_CATEGORY_ID; inilah yang tersimpan sebagai PART_CATEGORY_ID pada tipe. */
  kode: string
  /** Kolom PART_CATEGORY_NAME — yang dilihat pengguna. */
  nama: string
}

/**
 * Jawaban GET /api/master/tipe-sparepart/pilihan.
 *
 * Isinya DIBACA DARI BASIS DATA entitas yang sedang dibuka — kategori suku cadang berbeda
 * antarentitas. Karena itu permintaannya menuntut portal.
 */
export type PartTypeOptionsResponse = {
  kategori: PartTypeCategory[]
  /**
   * Daftarnya terpotong pada batas lookup.
   *
   * Dikirim supaya layar dapat mengatakannya. Tanpa itu, pemotongan terjadi diam-diam dan
   * terbaca pengguna sebagai "kategorinya belum dibuat".
   */
  terpotong: boolean
  portal: string
}

/**
 * Badan permintaan tambah dan simpan.
 *
 * DUA isian. ID tidak dikirim — pada penambahan ia diterbitkan server, dan pada penyimpanan
 * ia ada di jalur URL. Status juga tidak: menyimpan selalu mengembalikan baris ke antrean
 * persetujuan. Nama kategori juga tidak: ia milik tabel kategori dan tidak pernah ditulis.
 */
export type PartTypeInput = Pick<
  PartType,
  'nama_tipe_sparepart' | 'id_kategori_sparepart'
>

/**
 * Badan permintaan keputusan borongan.
 *
 * TANPA catatan: POOLDATA.GCNM_M_SPAREPART_TYPE tidak punya kolom penampungnya. Layar
 * karena itu tidak menggambar isian catatan sama sekali.
 */
export type PartTypeDecisionInput = {
  id_tipe_sparepart: string[]
  status: string
}

/**
 * Kode galat modul Master Tipe Sparepart.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const PartTypeErrorCode = {
  /**
   * Nama sudah dipakai tipe lain — konflik keadaan, bukan isian yang cacat.
   *
   * Termasuk bila yang memakainya adalah tipe di KATEGORI LAIN, dan termasuk bila yang
   * memakainya adalah baris yang sudah DITOLAK: pemeriksaan lamanya tidak menyaring
   * keduanya (P-5). Pesan pada `detail` yang menjelaskannya.
   */
  nameTaken: 'kunci_tipe_sparepart_sudah_ada',
  /**
   * Kategori yang dipilih tidak ada, atau persetujuannya dicabut sementara form terbuka.
   *
   * Perbaikannya memuat ulang daftar pilihan, bukan membetulkan ketikan — pengguna
   * memilihnya dari dropdown.
   */
  categoryNotFound: 'kategori_sparepart_tidak_ditemukan',
  unknownStatus: 'status_tidak_dikenal',
} as const

export type PartTypeErrorCode =
  (typeof PartTypeErrorCode)[keyof typeof PartTypeErrorCode]

/**
 * Posisi sebuah grouping sparepart dalam alur persetujuan.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.SPAREPART_HE_VIN_KEY. Ia
 * tidak dapat dipilih bebas selama tabel yang sama masih dibaca sistem lama (ADR-0004).
 */
export const GroupingStatus = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type GroupingStatus = (typeof GroupingStatus)[keyof typeof GroupingStatus]

/** Satu pilihan Nama Panel — satu baris POOLDATA.PANEL_HE yang sudah disetujui. */
export type GroupingPanel = {
  /** Kolom ID_PANEL; dikirim tersembunyi dan dipakai mencari daftar Sisi. */
  kode: string
  /** Kolom NAME — yang dilihat pengguna DAN yang tersimpan sebagai NAMA_PANEL. */
  nama: string
}

/** Satu pilihan Tipe Kendaraan — satu baris `branddetail` yang aktif pada lini ANEKA. */
export type GroupingVehicleType = {
  /** Kolom `id`; dibawa untuk penelusuran, TIDAK disimpan. */
  kode: string
  /** Kolom TYPENAME — yang dilihat pengguna DAN yang disimpan sebagai TIPE. */
  nama: string
}

/**
 * Satu pilihan Sisi pada sebuah panel.
 *
 * Sandinya "-", "1" KIRI, "2" KANAN — nilai kolom SISI_PANEL pada
 * POOLDATA.LOKASI_PANEL_HE. Sebutannya dihitung server, sehingga layar tidak menyimpan
 * salinan ketiga sandinya.
 */
export type GroupingSide = {
  kode: string
  nama: string
}

/**
 * Satu baris Master Grouping Sparepart.
 *
 * Cerminan dto di internal/mastergroupingsparepart/http. Menggantikan layar Pega
 * `Harness/GroupingSparePart_HE-Harness.xml` (MENU_ID 32).
 *
 * # Lima isian yang TIDAK diketik pengguna
 *
 * `nama_sparepart`, `kategori_sparepart`, `tipe_sparepart`, `kode_sparepart`, dan
 * `tanggal_produksi` seluruhnya DITURUNKAN dari Master Sparepart begitu Nomor Sparepart
 * diisi — meniru `Activity/SetDataSparepart-Act.xml`. Server yang membacanya; klien tidak
 * dapat mengirimnya.
 *
 * # Nama isiannya mengikuti ISINYA, bukan properti Pega yang dipinjam
 *
 * Layar lama menyimpan nama panel di properti bernama `PANJANG`, sisi di `LEBAR`, nomor
 * rangka di `TINGGI`, tipe kendaraan di `QTY_PESAN`, dan catatan di `MAX_STOCK` — sisa
 * salin-tempel dari layar Master Sparepart. Tidak satu pun dibawa.
 */
export type Grouping = {
  /** Kolom ID — kunci baris, diterbitkan server. */
  id_grouping: string

  /** Kolom NO_PART — "Nomor Sparepart". Satu-satunya identitas sparepart yang diketik. */
  nomor_sparepart: string

  /** Keempat berikut diturunkan dari Master Sparepart; baca-saja di layar. */
  nama_sparepart: string
  kategori_sparepart: string
  tipe_sparepart: string
  kode_sparepart: string

  /** Kolom PROD_DATE — TEKS, bukan tanggal. Diturunkan. */
  tanggal_produksi: string

  /** Kolom ID_PANEL — isian tersembunyi di balik pilihan Nama Panel. */
  id_panel: string

  /** Kolom NAMA_PANEL — "Nama Panel". Anggota kunci alami. */
  nama_panel: string

  /** Kolom SISI_PANEL — "Sisi", berisi sandi "-", "1", atau "2". Anggota kunci alami. */
  sisi: string

  /** Sebutan sandi sisi, dihitung server: "-", "KIRI", atau "KANAN". */
  sisi_label: string

  /** Kolom NO_RANGKA — "No Rangka". Anggota kunci alami. */
  no_rangka: string

  /** Kolom TIPE pada tabel pendamping — "Tipe Kendaraan". */
  tipe_kendaraan: string

  /**
   * Kolom GROUPING_DGN_RANGKA — "Grouping Dengan No Rangka".
   *
   * Isinya sebuah NOMOR RANGKA, bukan nomor grup. Kosong berarti baris ini membuka grup
   * sendiri.
   */
  grouping_dengan_no_rangka: string

  /**
   * Kolom NO_GROUP_RANGKA — nomor grup kendaraan, diterbitkan server.
   *
   * Baca-saja. Ia TIDAK digambar di layar lama sama sekali, dan tetap ditampilkan di sini
   * supaya baris satu grup dapat dikenali — itulah satu-satunya cara pengguna melihat
   * bahwa penggabungannya berhasil.
   */
  nomor_grup: string

  /** Kolom CATATAN — "Catatan". */
  catatan: string

  /** Kolom APPROVAL. */
  status: string
  status_label: string
}

export type GroupingListResponse = {
  grouping: Grouping[]
  status: string
  portal: string
}

export type GroupingResponse = {
  grouping: Grouping
  portal: string
}

export type GroupingDecisionResponse = {
  jumlah_berubah: number
  status: string
  status_label: string
  portal: string
}

/**
 * Jawaban GET /api/master/grouping-sparepart/pilihan.
 *
 * Daftar Sisi TIDAK ada di sini: ia bergantung pada panel yang dipilih, sehingga baru
 * dapat dibaca setelah pengguna memilih — persis seperti `Activity/GetSisiPanel-Act.xml`
 * yang berjalan belakangan.
 */
export type GroupingOptionsResponse = {
  panel: GroupingPanel[]
  tipe_kendaraan: GroupingVehicleType[]
  portal: string
}

/** Jawaban GET /api/master/grouping-sparepart/sisi. */
export type GroupingSideResponse = {
  sisi: GroupingSide[]
  portal: string
}

/**
 * Jawaban GET /api/master/grouping-sparepart/sparepart.
 *
 * Padanan `Activity/SetDataSparepart-Act.xml`: kelima isian turunan yang muncul begitu
 * Nomor Sparepart selesai diketik.
 */
export type GroupingPartResponse = {
  nomor_sparepart: string
  nama_sparepart: string
  kategori_sparepart: string
  tipe_sparepart: string
  kode_sparepart: string
  tanggal_produksi: string
  portal: string
}

/**
 * Isian yang dikirim saat menambah dan menyimpan.
 *
 * Sepuluh kolom sengaja tidak ada: kunci baris, nomor grup, status beserta sebutannya,
 * sebutan sisi, dan kelima isian turunan — seluruhnya diterbitkan atau dibaca server.
 *
 * Mengirim salah satunya DITOLAK sebagai permintaan cacat, bukan diabaikan: server
 * memasang `DisallowUnknownFields`, supaya cacat pada klien terlihat saat pertama dicoba.
 */
export type GroupingInput = Omit<
  Grouping,
  | 'id_grouping'
  | 'nama_sparepart'
  | 'kategori_sparepart'
  | 'tipe_sparepart'
  | 'kode_sparepart'
  | 'tanggal_produksi'
  | 'sisi_label'
  | 'nomor_grup'
  | 'status'
  | 'status_label'
>

/**
 * Badan permintaan keputusan borongan.
 *
 * TANPA catatan: kedua tabel modul ini tidak punya kolom penampung alasan penolakan, dan
 * layar persetujuan Pega pun tidak punya isian catatan.
 */
export type GroupingDecisionInput = {
  id_grouping: string[]
  status: string
}

/**
 * Kode galat modul Master Grouping Sparepart.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const GroupingErrorCode = {
  /**
   * Keempat kunci alami sudah dipakai baris lain — konflik keadaan, bukan isian yang
   * cacat.
   *
   * SATU kode untuk keempatnya: kuncinya memang satu — nomor sparepart, nama panel, no
   * rangka, dan sisi BERSAMA-SAMA — sehingga tidak ada pilihan isian mana yang harus
   * disorot sendirian.
   */
  duplicate: 'grouping_sudah_ada',

  /**
   * Nomor sparepart yang diketik tidak ada di Master Sparepart.
   *
   * Dibedakan dari `tidak_ditemukan`: yang tidak ada bukan baris yang sedang dibuka,
   * melainkan acuan yang diketik pengguna — dan layar menanganinya dengan menyorot
   * isiannya, bukan dengan menutup form.
   */
  partNotFound: 'sparepart_tidak_ditemukan',

  unknownStatus: 'status_tidak_dikenal',
} as const

export type GroupingErrorCode =
  (typeof GroupingErrorCode)[keyof typeof GroupingErrorCode]

/**
 * Satu baris Master Login — satu baris POOLDATA.MST_LOGIN_SURVEYOR.
 *
 * # Kenapa namanya SurveyorLogin, bukan Login
 *
 * Karena `Login` sudah dipakai untuk hal yang sama sekali berbeda di aplikasi ini: masuknya
 * pengguna ke sistem. Yang di sini adalah barisnya sendiri — daftar orang yang boleh bekerja
 * sebagai surveyor, beserta kontaknya.
 *
 * Menyamakan keduanya akan membuat dua hal yang tidak berhubungan terlihat berhubungan, dan
 * pada modul ini kekeliruan itu paling mudah terjadi: menyimpan baris di sini TIDAK
 * menerbitkan akun siapa pun. Lihat catatan pada `status_login`.
 *
 * # Ketujuh kolomnya, dan DUA yang tidak pernah digambar layar Pega
 *
 * Kelima yang pertama adalah isian di layar. Kedua yang terakhir — `status_login` dan
 * `login_leader` — ditulis sistem lama tetapi tidak muncul sekali pun di
 * `Section/BrowseLoginSurveyor-Section.xml`. Keduanya tetap dikirim server, dan layar baru
 * MENAMPILKANNYA sebagai keterangan baca-saja: nilainya menentukan peran dan tim seseorang,
 * dan menyembunyikan hal yang tersimpan tidak membuatnya tidak tersimpan.
 *
 * Ketiadaan kolom lain juga bermakna: tidak ada APPROVAL, tidak ada pencatat pelaku, tidak
 * ada stempel waktu, dan tidak ada penanda aktif. Itu sebabnya layar ini tidak bertab, dan
 * sebabnya tidak ada cara menyatakan sebuah login sudah tidak berlaku.
 */
export type SurveyorLogin = {
  /** Kolom NAMA — caption "Nama". Login diturunkan darinya, sehingga ia TERKUNCI saat diubah. */
  nama: string
  /**
   * Kolom LOGIN — caption "Login", dan KUNCI baris ini.
   *
   * Tidak diketik pengguna: server menurunkannya dari `nama` dengan membuang spasi, titik,
   * koma, dan tanda hubung. Ia yang dipakai pada jalur URL, bukan sebuah ID terpisah.
   */
  login: string
  /** Kolom EMAIL — caption "Email". Wajib. */
  email: string
  /** Kolom TELP — caption "Telp". Wajib. Properti klipboard Pega-nya bernama `Ekst`. */
  telp: string
  /** Kolom ALAMAT — caption "Alamat". Tidak wajib. */
  alamat: string
  /**
   * Kolom STSLOGIN. Tidak ada di layar Pega; selalu "Member" pada penambahan.
   *
   * Ia BUKAN penanda bahwa akun aplikasinya sudah diterbitkan — tidak ada akun yang
   * diterbitkan sama sekali; lihat catatan pada SurveyorLoginPage.
   */
  status_login: string
  /**
   * Kolom LOGINLEADER — login orang yang menjadi leader baris ini.
   *
   * Tidak ada di layar Pega. Server menurunkannya dari leader milik pengguna yang menyimpan
   * — bukan dari login pengguna itu sendiri. Kosong adalah nilai yang sah.
   */
  login_leader: string
}

/** Jawaban GET /api/master/login. */
export type SurveyorLoginListResponse = {
  login_surveyor: SurveyorLogin[]
  portal: string
}

/** Jawaban satu baris — dipakai Get, Create, dan Save. */
export type SurveyorLoginResponse = {
  login_surveyor: SurveyorLogin
  portal: string
}

/**
 * Badan permintaan tambah dan simpan.
 *
 * KEEMPAT isian yang diketik pengguna. Ketiga kolom turunan — `login`, `status_login`, dan
 * `login_leader` — dikirim KELUAR pada setiap jawaban tetapi tidak dapat dikirim MASUK.
 *
 * Mengirim salah satunya DITOLAK sebagai permintaan cacat, bukan diabaikan: server memasang
 * `DisallowUnknownFields`, supaya cacat pada klien terlihat saat pertama dicoba.
 */
export type SurveyorLoginInput = Pick<
  SurveyorLogin,
  'nama' | 'email' | 'telp' | 'alamat'
>

/**
 * Kode galat modul Master Login.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const SurveyorLoginErrorCode = {
  /**
   * LOGIN yang diturunkan sudah dipakai baris lain — konflik keadaan, bukan isian cacat.
   *
   * Yang bentrok bukan Nama melainkan LOGIN yang DITURUNKAN darinya, sehingga dua nama yang
   * terlihat berbeda — "Budi Hartono" dan "Budi.Hartono" — menghasilkan login yang sama.
   */
  loginTaken: 'kunci_login_surveyor_sudah_ada',

  /**
   * Nama yang dikirim berbeda dari yang tersimpan.
   *
   * Tidak seharusnya terlihat pengguna: layar mengunci isiannya. Bila ia muncul, daftarnya
   * sudah basi — atau permintaannya tidak datang dari layar ini.
   */
  nameLocked: 'nama_login_surveyor_terkunci',
} as const

export type SurveyorLoginErrorCode =
  (typeof SurveyorLoginErrorCode)[keyof typeof SurveyorLoginErrorCode]

/**
 * Satu baris Master Reas — satu baris POOLDATA.T_REINSURER.
 *
 * # Apa yang diwakilinya
 *
 * Satu **member reasuransi**: pihak yang menerima pemberitahuan PLA, Pre-DLA, dan DLA atas
 * klaim entitas ini, beserta login portal dan alamat surel tujuannya.
 *
 * # Satu perusahaan dapat punya BEBERAPA baris
 *
 * Yang membedakannya adalah `tipe` — karakter pertama nomor dokumen PLA/DLA yang dilayani
 * baris itu. Tanpa mengetahui hal ini, daftar akan terlihat memuat nama yang sama berkali-
 * kali tanpa sebab. Lihat catatan pada `tipe`.
 *
 * # Enam kolom, dan SATU kolom tabel yang sengaja tidak dikirim
 *
 * `COUNTRYID` ditulis `Database/UPDATEREAS.prc` — diterjemahkan dari COUNTRY lewat tabel
 * COUNTRY — lalu **tidak dibaca satu pun rule di seluruh export Pega**. Ia tidak dikirim:
 * membawa kolom yang tidak ada pembacanya berarti mengarang kegunaan yang tidak dapat
 * ditunjukkan.
 */
export type ReasMember = {
  /**
   * Kolom REINSURERID — kode perusahaan reasuransi.
   *
   * Ia yang menghubungkan baris ini ke dokumen PLA/DLA: `T_PLALIST.REINSCODE`,
   * `T_DLALIST.REINSCODE`, dan `IDREAS` pada kedua tabel XOL menunjuk ke sini.
   */
  kode_reas: string
  /** Kolom REINSURERNAME — nama perusahaan reasuransi. */
  nama_reas: string
  /**
   * Kolom LOGIN — identitas yang dipakai mitra reasuransi saat masuk.
   *
   * Kolom bertaruh paling tinggi di tabel ini: lima kueri inbox menyaring klaim yang boleh
   * dilihat seseorang dengan membandingkannya terhadap identitas pemanggil. Ia TIDAK pernah
   * diketik orang — alur PLA/DLA menurunkannya dari nama perusahaan.
   */
  login: string
  /** Kolom EMAIL — tujuan pemberitahuan PLA, Pre-DLA, dan DLA. */
  email: string
  /**
   * Kolom COUNTRY — negara asal perusahaan reasuransi.
   *
   * Isinya sampai ke pihak luar: `GetDataPreDLA` dan `BrowseAllDataXOL_PLA` menempatkannya
   * pada dokumen PLA/DLA. Kosong adalah nilai yang mungkin — baris yang lahir dari
   * `GetListDataLoginReas` memang disisipkan tanpa kolom ini sama sekali.
   */
  negara: string
  /**
   * Kolom TYPE — karakter pertama nomor dokumen PLA/DLA yang dilayani baris ini.
   *
   * Terbukti dari `GetDataPreDLA`: `... and substr(a.NODLA,0,1) = TYPE`.
   *
   * **Arti tiap nilainya dalam bahasa bisnis TIDAK diketahui** — tidak ada master, tidak ada
   * daftar nilai sah, dan tidak ada satu pun rule yang menerjemahkannya menjadi label
   * (`R-16`). Layar menampilkannya apa adanya alih-alih mengarang keterangan.
   */
  tipe: string
  /**
   * Penanda baris CADANGAN (`TYPE = '1'`).
   *
   * TURUNAN, bukan kolom — dihitung server supaya layar tidak perlu mengetahui bahwa `'1'`
   * punya arti khusus.
   *
   * Artinya: `BrowseEmailReas` memakai baris ini ketika tidak ada baris yang cocok dengan
   * jenis dokumen yang sedang dikirim (`... or type = '1'`). Perusahaan tanpa baris cadangan
   * karena itu hanya terlayani untuk jenis dokumen yang kebetulan sudah punya barisnya
   * sendiri.
   */
  cadangan: boolean
}

/** Jawaban GET /api/master/reas. */
export type ReasMemberListResponse = {
  member_reas: ReasMember[]
  portal: string
}

// ── Detail Penyebab Kerugian ─────────────────────────────────────────────────────
//
// Cerminan dto di internal/detailpenyebab/http. Menggantikan layar Pega
// `Harness/DetailCauseOfLoss-Harness.xml` (MENU_ID 38) atas POOLDATA.D_CAUSE_OF_LOSS.
//
// Tabelnya hanya punya DUA kolom — D_COL_ID dan JSONDATA — dan seluruh isi selain ID
// hidup di dalam dokumen JSON pada kolom kedua, dibentangkan kembali menjadi kolom oleh
// view POOLDATA.V_D_CAUSE_OF_LOSS. Bentuk di bawah adalah dokumen itu sesudah dibongkar
// server; layar tidak pernah melihat JSON-nya.

/** Satu lini bisnis tempat sebuah detail penyebab kerugian berlaku. */
export type CauseOfLossBusiness = {
  id: string
  nama: string
}

/**
 * Satu baris Detail Penyebab Kerugian.
 *
 * Ketiga field yang DIKIRIM server tetapi TIDAK dapat dikirim balik — `id`, `nama_master`,
 * dan `label_status_aktif` — sengaja tetap ada di tipe ini: layar menampilkan ketiganya.
 * Yang boleh dikirim balik dipisahkan sebagai CauseOfLossDetailInput.
 */
export type CauseOfLossDetail = {
  /** D_COL_ID. Diterbitkan server; layar tidak pernah mengetiknya. */
  id: string

  /**
   * OLD_D_COL_ID — ID baris ini pada sistem sebelum Pega.
   *
   * Ia TIDAK digambar di layar, tetapi tetap dikirim balik saat menyimpan. Tanpa itu,
   * tautan ke sistem lama lenyap pada setiap penyuntingan — server memang menjaganya, tapi
   * mengirimkannya membuat layar tidak bergantung pada penjagaan itu.
   */
  id_lama: string

  /** M_COL_ID — kunci induk di V_M_CAUSE_OF_LOSS. */
  id_master: string

  /**
   * COL_DESC — sebutan induk, hanya untuk ditampilkan.
   *
   * Kosong berarti induknya tidak ditemukan — baris yatim, yang mungkin ada karena tidak
   * ada foreign key yang diketahui (`R-08`).
   */
  nama_master: string

  /** DESCRIPTION, berlabel "Deskripsi Kerugian" di layar. */
  deskripsi_kerugian: string

  /** LOSS_CODE, berlabel "Kode Kehilangan" di layar. */
  kode_kehilangan: string

  /** STS_AKTIF — "1" aktif, "0" tidak aktif, kosong belum pernah diisi. */
  status_aktif: string

  /** Sebutan status yang siap ditampilkan; DITURUNKAN server. */
  label_status_aktif: string

  /**
   * Lini bisnis tempat detail ini berlaku.
   *
   * KOSONG pada jawaban daftar dan terisi pada jawaban satu baris — grid layar lama pun
   * tidak menampilkannya.
   */
  bisnis: CauseOfLossBusiness[]
}

/** Satu pilihan pada isian "ID Master Kerugian". */
export type CauseOfLossMasterOption = {
  id: string
  nama: string
  /** Gabungan kode dan sebutannya; disusun server supaya kedua sisi menampilkannya sama. */
  label: string
}

/** Satu pilihan pada dropdown Status Aktif. */
export type CauseOfLossActiveOption = {
  kode: string
  label: string
}

/** Jawaban GET /api/master/detail-penyebab. */
export type CauseOfLossDetailListResponse = {
  detail: CauseOfLossDetail[]
  portal: string
}

/** Jawaban GET satu baris, POST, dan PUT. */
export type CauseOfLossDetailResponse = {
  detail: CauseOfLossDetail
  portal: string
}

/** Jawaban pencarian daftar pilihan. Hanya SATU daftar terisi per permintaan. */
export type CauseOfLossLookupResponse = {
  master: CauseOfLossMasterOption[]
  bisnis: CauseOfLossBusiness[]
  portal: string
}

/** Jawaban GET /api/master/detail-penyebab/pilihan. */
export type CauseOfLossOptionsResponse = {
  status_aktif: CauseOfLossActiveOption[]
}

/**
 * Badan permintaan POST dan PUT.
 *
 * `id`, `nama_master`, dan `label_status_aktif` TIDAK boleh ikut: server menolak field yang
 * tidak dikenal dengan `permintaan_cacat`, bukan mengabaikannya diam-diam.
 */
export type CauseOfLossDetailInput = Pick<
  CauseOfLossDetail,
  | 'id_lama'
  | 'id_master'
  | 'deskripsi_kerugian'
  | 'kode_kehilangan'
  | 'status_aktif'
  | 'bisnis'
>

/**
 * Kode galat khas modul Detail Penyebab Kerugian.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const CauseOfLossDetailErrorCode = {
  /**
   * Nomor yang diterbitkan sequence sudah dipakai baris lain.
   *
   * BUKAN kesalahan pengguna: isiannya sudah benar, dan yang bermasalah adalah penomoran
   * di basis data. Layar mengatakannya demikian alih-alih menyuruh petugas membetulkan
   * isian yang tidak salah.
   */
  idTaken: 'kunci_detail_penyebab_sudah_ada',
} as const

export type CauseOfLossDetailErrorCode =
  (typeof CauseOfLossDetailErrorCode)[keyof typeof CauseOfLossDetailErrorCode]

// ── Inbox Investigator ───────────────────────────────────────────────────────────

/**
 * Satu baris Inbox Investigator — satu **pekerjaan** yang menunggu Investigator.
 *
 * # Ia pekerjaan, bukan klaim
 *
 * Yang didaftar layar ini adalah Tugas: satuan pekerjaan yang menunggu dikerjakan pada satu
 * tahap klaim. Klaim yang sama dapat muncul di beberapa inbox pada waktu yang berbeda, dan
 * yang membedakannya adalah penugasannya — bukan klaimnya (`CONTEXT.md`, `D-26`).
 *
 * Itulah yang membuat layar ini INBOX dan bukan layar daftar biasa: barisnya HILANG setelah
 * selesai dikerjakan, dan "hanya yang jadi tanggung jawab saya" adalah aturan kewenangan,
 * bukan sekadar penyaring (`D-79`).
 *
 * # Nama field mengikuti CAPTION GRID layar lama
 *
 * Bukan nama kolom basis data dan bukan nama properti Pega. `D-13` menetapkan tampilan
 * meniru Pega supaya pengguna tidak perlu belajar ulang, dan caption itulah yang selama ini
 * mereka baca di `Section/InputInvestigator_Section-Section.xml`.
 */
export type InvestigatorTask = {
  /**
   * Kunci teknis Pega (`pzInsKey`), dibutuhkan untuk MEMBUKA pekerjaannya.
   *
   * Ia dikirim tetapi **tidak ditampilkan**. Layar kerja Investigator belum dibangun; begitu
   * ia ada, inilah yang dipakai membukanya — sehingga menyalakannya kelak tidak menuntut
   * perubahan kontrak.
   */
  referensi: string
  /** Caption "Nomor Case" — `pyID`. Dua format hidup berdampingan: `PNC-xxxx` dan `PNCN.YY.xxxx` (`D-71`). */
  nomor_case: string
  /** Caption "No Polis". */
  nomor_polis: string
  /** Caption "Nama Tertanggung". */
  nama_tertanggung: string
  /**
   * Caption "Nama Peserta" — objek pertanggungan **pertama** pada klaim itu.
   *
   * Captionnya menyebut "Peserta", bukan "Objek", dan itu dipakai apa adanya (`D-13`):
   * investigasi paling sering menyangkut lini Personal Accident, tempat objek pertanggungan
   * memang seorang peserta.
   *
   * Klaim berobjek banyak tampil seolah berobjek satu — perilaku sistem lama yang
   * direplikasi apa adanya (`P-5`).
   */
  nama_peserta: string
  /** Caption "Nama Bisnis" — lini bisnis klaim. */
  nama_bisnis: string
  /** Caption "Nama Cabang". */
  nama_cabang: string
  /**
   * Caption "Nama Admin" — `pyOrigUserID`, yaitu **pembuat** kasus.
   *
   * Bukan petugas yang sedang memegangnya: pekerjaan di workbasket memang belum bertuan
   * (`D-26`) — itulah yang membuatnya antrean bersama. Kosong berarti kasusnya dibuat proses
   * terjadwal, bukan orang (`D-57`).
   */
  nama_admin: string
  /** Caption "Tanggal Pendaftaran", ISO 8601 UTC. Null bila kosong. */
  tanggal_pendaftaran: string | null
  /**
   * Kolom **kesembilan**, yang di layar lama bercaption **"Lama Masuk Inbox"** —
   * `.ClaimData.SurveyResults(1).SurveyDate`.
   *
   * # Captionnya menyebut durasi, isinya TANGGAL
   *
   * Bukan salah baca. Penelusuran sel per sel pada
   * `Section/InputInvestigator_Section-Section.xml` memasangkan kesembilan caption dengan
   * kesembilan sel datanya satu lawan satu, dan yang kesembilan berpasangan dengan properti
   * di atas.
   *
   * Sebabnya terbaca: section ini Save-As dari inbox Compliance, tempat kolom bernama sama
   * memang berisi lama menunggu. Di sini selnya diikat ulang ke tanggal survei dan
   * **captionnya tidak ikut diganti**.
   *
   * Nama field mengikuti ISI supaya kontraknya tidak berbohong tentang tipe datanya; yang
   * mengikuti Pega adalah **judul kolom di layar** (`D-13`).
   *
   * ISO 8601 UTC. Null bila klaimnya belum punya baris survei.
   */
  tanggal_survey: string | null
}

export type InvestigatorInboxResponse = {
  tugas: InvestigatorTask[]
  /**
   * Masih ada pekerjaan yang cocok tetapi TIDAK terkirim.
   *
   * Ia ada karena sistem lama memotong pada 500 baris **tanpa memberi tahu siapa pun**
   * (`pyMaxRecords`). Batas yang diketahui adalah batas; batas yang senyap adalah data yang
   * hilang. Layar WAJIB menyebutkannya saat bernilai `true`.
   */
  terpotong: boolean
  /** Batas baris yang berlaku, dikirim supaya layar tidak menuliskan angkanya sendiri. */
  batas_baris: number
  portal: string
}

// ── Inbox Receive TKA ────────────────────────────────────────────────────────────

/**
 * Satu baris Inbox Receive TKA — satu klaim TKA yang tanggal kelengkapan dokumennya belum
 * diisi.
 *
 * # Ia inbox yang DIKERJAKAN, bukan hanya dibaca
 *
 * Berbeda dari Inbox Investigator yang baca-saja, layar ini punya tombol Submit: pengguna
 * mengisi tanggal kelengkapan dokumen, dan barisnya HILANG dari daftar. Ciri kedua Inbox
 * pada `D-79` — baris hilang setelah dikerjakan — di sini benar-benar terjadi lewat layar
 * ini sendiri.
 *
 * # Nama field mengikuti CAPTION GRID layar lama, dengan satu pengecualian
 *
 * `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang, dan
 * caption itulah yang selama ini mereka baca di `Section/InboxTKA_Section-Section.xml`.
 * Pengecualiannya `tanggal_kejadian`; lihat di bawah.
 */
export type ReceiveTKATask = {
  /**
   * Kunci teknis Pega (`pzInsKey`), dan **kunci baris** layar ini.
   *
   * Dikirim tetapi tidak ditampilkan. Selalu terisi — sumbernya tabel yang menggerakkan
   * kuerinya, bukan hasil gabungan.
   */
  referensi: string
  /**
   * Klaimnya ditemukan di data klaim utama.
   *
   * # Kenapa layar perlu mengetahuinya sebelum Submit ditekan
   *
   * Pekerjaan yang ada di antrean Pega tetapi belum ada di tabel klaim bisnis **tetap
   * tampil** — gabungannya sengaja LEFT, sama seperti Pega yang juga menampilkannya.
   * Tetapi tanggalnya tidak dapat disimpan: tidak ada baris klaim yang dapat diperbarui.
   *
   * Bernilai `false`, layar mematikan isian dan tombolnya beserta alasannya — alih-alih
   * membiarkan pengguna mengetik tanggal lalu ditolak.
   */
  klaim_tersedia: boolean
  /** Caption "Nomor Klaim" — `pyID`. */
  nomor_klaim: string
  /** Caption "No Polis". */
  nomor_polis: string
  /**
   * Caption "Nama Tertanggung" — kolom `QQNAME`.
   *
   * Pemasangannya sempat diragukan: `InboxTKA_RD` melabeli properti ini dan `nama_peserta`
   * secara TERBALIK. Yang benar adalah Section, dan buktinya datang dari sumber ketiga yang
   * berdiri sendiri — `HTML/NotificationKelengkapanTKA-HTML.xml` memberi label
   * "Nama Tertanggung" pada nilai yang diisi dari `Param.QQName`.
   */
  nama_tertanggung: string
  /**
   * Caption "Nama Peserta" — `POOLDATA.T_GENERAL.THEINSURED`.
   *
   * `.Policy.TheInsured` tidak di-expose sebagai kolom pada tabel kerja Pega, sehingga
   * nilainya diambil dari tabel polis. Dapat kosong bila polisnya tidak ditemukan; layar
   * menuliskannya sebagai tanda hubung.
   */
  nama_peserta: string
  /**
   * Kolom kelima, yang di layar lama bercaption **"Date Of Loss"**.
   *
   * Nama fieldnya bahasa Indonesia meski captionnya bahasa Inggris: `D-80` menetapkan nama
   * field JSON memakai bahasa Indonesia, dan `CONTEXT.md` sudah menetapkan padanan
   * resminya — **Tanggal Kejadian**. Yang tetap mengikuti Pega adalah **judul kolom di
   * layar**.
   *
   * Tanggal ISO (`YYYY-MM-DD`) tanpa jam, karena kolomnya memang tanggal kalender.
   * Mengirimkannya sebagai stempel waktu akan mengarang bagian jam yang dapat menggeser
   * tanggalnya satu hari saat ditampilkan dalam WIB (`R-12`). Null bila kosong.
   */
  tanggal_kejadian: string | null
  /**
   * Mengisi kolom yang di layar lama bercaption **"Aging"** — `.ClaimData.RegisterDate`.
   *
   * # Yang dikirim adalah TANGGAL, bukan lama menunggu
   *
   * Pega menampilkan kolom itu sebagai waktu relatif — "2 years 6 months ago" — tetapi yang
   * disimpannya adalah tanggal registrasi. Terverifikasi: klaim ber-`REGISTERDATE_1 =
   * 20240319` ditampilkan Pega sebagai "2 years 6 months ago", tepat selisihnya terhadap
   * hari pembacaan.
   *
   * **Layar yang menyusun kalimatnya**, karena "berapa lama menunggu" bergantung pada kapan
   * ia dibaca. Menghitungnya di server berarti nilainya membeku pada saat permintaan.
   *
   * Tanggal ISO (`YYYY-MM-DD`). Null bila kosong atau bentuk sumbernya tidak dikenali —
   * dan null **bukan** nol hari: yang satu "tidak diketahui", yang lain "baru masuk hari
   * ini".
   */
  tanggal_registrasi: string | null
}

export type ReceiveTKAInboxResponse = {
  tugas: ReceiveTKATask[]
  /**
   * Masih ada pekerjaan yang cocok tetapi TIDAK terkirim.
   *
   * Ia ada karena sistem lama memotong pada 500 baris **tanpa memberi tahu siapa pun**
   * (`pyMaxRecords`). Layar WAJIB menyebutkannya saat bernilai `true`.
   */
  terpotong: boolean
  /** Batas baris yang berlaku, dikirim supaya layar tidak menuliskan angkanya sendiri. */
  batas_baris: number
  portal: string
}

/** Badan permintaan tombol Submit: satu baris, satu tanggal. */
export type ReceiveTKACompleteRequest = {
  nomor_klaim: string
  /** Tanggal kelengkapan dokumen, berformat `YYYY-MM-DD`. */
  tanggal_dokumen_lengkap: string
}

export type ReceiveTKACompleteResponse = {
  nomor_klaim: string
  tanggal_dokumen_lengkap: string
  /**
   * Bernilai `false` bila pemberitahuan memang **tidak dipasang** di lingkungan ini.
   *
   * Dipisahkan dari `pemberitahuan_terkirim` supaya layar dapat membedakan "belum dipasang"
   * dari "gagal dikirim". Menyatukan keduanya akan membuat lingkungan pengembangan yang
   * memang tidak punya server surel terus-menerus menampilkan peringatan gagal — dan
   * peringatan yang selalu muncul berhenti dibaca.
   */
  pemberitahuan_dicoba: boolean
  /**
   * Bernilai `true` hanya bila surelnya benar-benar terkirim.
   *
   * Surel yang gagal **tidak** membatalkan penyimpanan; tanggalnya tetap tersimpan. Itu
   * perbedaan yang disengaja terhadap sistem lama, yang mengirim surel sebelum `Commit`
   * sehingga kegagalannya membuang seluruh pekerjaan pengguna.
   */
  pemberitahuan_terkirim: boolean
  portal: string
}

/**
 * Kode galat modul Inbox Receive TKA.
 *
 * Ketiganya dibedakan karena TINDAKAN penggunanya berbeda, bukan demi kerapian.
 */
export const ReceiveTKAErrorCode = {
  /**
   * Barisnya sudah tidak ada di inbox — tidak pernah ada, atau sudah diisi orang lain.
   *
   * Tindakannya: segarkan daftar.
   */
  taskNotFound: 'pekerjaan_tidak_ditemukan',
  /**
   * Barisnya ada di inbox, tetapi klaimnya tidak ada di data klaim utama.
   *
   * Tindakannya BUKAN menyegarkan daftar — barisnya akan muncul lagi dan gagal lagi. Yang
   * perlu terjadi adalah perbaikan data, dan layar mengatakannya begitu.
   */
  claimMissing: 'klaim_tidak_ditemukan',
  /**
   * Satu nomor klaim menunjuk lebih dari satu baris.
   *
   * Mungkin terjadi karena tidak ada satu pun constraint keunikan pada kedua tabel
   * (`R-08`). Penulisan dihentikan, bukan diteruskan atas salah satu barisnya.
   */
  claimAmbiguous: 'nomor_klaim_ganda',
  /** Tanggal kelengkapan dokumen kosong. */
  dateRequired: 'tanggal_dokumen_wajib_diisi',
} as const

export type ReceiveTKAErrorCode =
  (typeof ReceiveTKAErrorCode)[keyof typeof ReceiveTKAErrorCode]
