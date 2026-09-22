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
  /**
   * Datang dari server, bukan dihitung dari panjang senarai.
   *
   * Sejak modul ini menjadi per portal (2026-09-19), angkanya adalah isi master ENTITAS
   * YANG MENJAWAB — bukan angka yang sama untuk seluruh aplikasi.
   */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type ClaimStatusResponse = {
  status_klaim: ClaimStatus
  portal: string
}

// ── Master Dominan Factor ────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterdominanfactor/http. Menggantikan layar Pega
// `Harness/DetailDominanFactor-Harness.xml` atas tabel POOLDATA.M_DOMINAN_FACTOR.

/**
 * Satu baris master faktor dominan.
 *
 * Faktor dominan adalah daftar acuan penyebab kerugian yang paling menentukan pada sebuah
 * klaim. Faktor yang dipilih untuk satu klaim tersimpan di `T_CLAIM_DOMINANFACTOR`, dan
 * NAMANYA ikut terbaca laporan Outstanding per Cabang lewat `LISTAGG` — sehingga mengubah
 * nama di layar master langsung mengubah isi laporan itu.
 */
export type DominantFactor = {
  /** Dibuat sistem saat faktor ditambahkan; tidak pernah berubah sesudahnya. */
  id: string
  /**
   * Isi kolom NAME. Di layar ia berlabel "Keterangan", mengikuti layar Pega (`D-13`) —
   * nama field dan label layar memang berbeda peruntukan.
   */
  nama: string
}

export type DominantFactorListResponse = {
  dominan_factor: DominantFactor[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type DominantFactorResponse = {
  dominan_factor: DominantFactor
  portal: string
}

// ── Master XOL ───────────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterxol/http. Menggantikan layar Pega
// `Harness/DetailMasterXOL-Harness.xml` atas empat tabel: POOLDATA.MST_XOL_PNC beserta
// MST_XOL_BUSINESS, MST_XOL_LAYER, dan MST_XOL_REAS.

/** Satu grup bisnis yang dicakup sebuah induk XOL. */
export type XOLBusiness = {
  /**
   * Isi kolom IDBUSINESS. BOLEH KOSONG: baris "TREATY INWARD" memang tidak punya ID di
   * POOLDATA.BUSINESSGROUP, dan dua baris produksi menyimpan NULL.
   */
  id: string
  /** Isi kolom GROUPBUSINESS. */
  nama: string
}

/** Satu reasuradur beserta bagiannya pada sebuah lapisan. */
export type XOLReinsurer = {
  /** Isi kolom IDREAS. */
  id: string
  /** Isi kolom NAMA. Di layar ia berlabel "Reasuransi", mengikuti layar Pega (`D-13`). */
  nama: string
  /** Isi kolom PERCENTSHARE, dalam persen utuh. */
  share: number
}

/** Satu lapisan treaty beserta pembagian share-nya. */
export type XOLLayer = {
  /** Isi kolom IDLAYER. Kosong pada lapisan baru — nomornya diterbitkan server. */
  id: string
  /** Isi kolom NAMA, misalnya "Layer 1" atau "Sub Layer". */
  nama: string
  /** Isi kolom LIMIT, dalam DOLAR. Layar lama melabelinya "Limit (USD)". */
  limit: number
  /** Isi kolom EXCESS, dalam DOLAR. */
  excess: number
  /**
   * Isi kolom CONVERT_LIMIT, dalam RUPIAH — hasil `limit × kurs induk`.
   *
   * HANYA DITERIMA dari server, tidak pernah dikirim. Server yang menghitungnya, supaya
   * limit rupiah dan limit dolar tidak dapat berbeda diam-diam.
   */
  limit_idr: number
  reas: XOLReinsurer[]
}

/** Satu induk XOL. */
export type XOL = {
  /** Diterbitkan server saat induk ditambahkan; tidak pernah berubah sesudahnya. */
  id: string
  nama: string
  /** Isi kolom TAHUN — tahun treaty. */
  tahun: string
  /** Isi kolom KURSVALUE — rupiah per satu dolar. */
  kurs: number
  /** Isi kolom TYPEXOL. Boleh kosong: dua dari delapan induk produksi menyimpan NULL. */
  tipe: string
  /**
   * Label Type XOL yang ditampilkan, dikirim server.
   *
   * Ia datang dari server, bukan dipetakan ulang di sini, supaya kode dan labelnya tidak
   * dapat berbeda antara kedua sisi.
   */
  tipe_label: string
  /** Isi kolom REMARKPIC — catatan PIC saat mengajukan ke komite. */
  remark_pic: string

  /** Keempat berikut HANYA DIBACA. Server yang mengisinya; form tidak mengirimkannya. */
  pic: string
  /** `''` belum pernah diajukan · `'0'` menunggu komite · `'1'` disetujui. */
  status_komite: string
  komite: string
  remark_komite: string

  bisnis: XOLBusiness[]
  layer: XOLLayer[]
}

export type XOLListResponse = {
  xol: XOL[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type XOLResponse = {
  xol: XOL
  portal: string
  /**
   * Pesan yang TIDAK menggagalkan penyimpanan — hari ini hanya satu jenis: total share
   * sebuah lapisan yang belum 100%.
   *
   * Layar lama pun menyimpan lebih dulu baru menampilkannya, dan perilaku itu ditiru
   * (`P-5`). Ia karena itu WAJIB ditampilkan berbeda dari galat: yang ini berarti
   * "tersimpan, tetapi perhatikan ini".
   */
  peringatan?: string[]
}

/** Satu pilihan Type XOL beserta labelnya. */
export type XOLTypeOption = {
  kode: string
  label: string
}

/** Bekal awal layar: pilihan Tahun dan pilihan Type XOL, dalam satu permintaan. */
export type XOLFormResponse = {
  tahun: string[]
  tipe: XOLTypeOption[]
  portal: string
}

export type XOLBusinessGroupResponse = {
  bisnis: XOLBusiness[]
  total: number
  portal: string
}

/**
 * Badan permintaan tambah dan ubah.
 *
 * `id`, `limit_idr`, dan keempat kolom komite TIDAK ada di sini dengan sengaja — server
 * menolak badan permintaan yang memuatnya. Nomor diterbitkan server, limit rupiah
 * dihitung server, dan PIC diisi dari sesi.
 */
export type XOLInput = {
  nama: string
  tahun: string
  kurs: number
  tipe: string
  remark_pic: string
  bisnis: XOLBusiness[]
  layer: Array<Omit<XOLLayer, 'limit_idr'>>
}

// ── Master Penyebab Kerugian ─────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterpenyebabkerugian/http. Menggantikan layar Pega
// `Harness/CauseOfLossInbox-Harness.xml` atas tabel POOLDATA.M_CAUSE_OF_LOSS.
//
// Ini TINGKAT GOLONGAN saja. Rinciannya — `D_CAUSE_OF_LOSS`, harness `DetailCauseOfLoss`,
// MENU_ID 38 — butir menu tersendiri dan belum dibangun.

/**
 * Satu golongan penyebab kerugian.
 *
 * Penyebab kerugian adalah daftar acuan sebab terjadinya kerugian sebuah klaim. Isinya
 * dibaca jauh di luar layar masternya: 19 rule Pega membaca `V_M_CAUSE_OF_LOSS`, dan dua
 * di antaranya — dasbor klaim per penyebab kerugian serta laporan XOL per bisnis —
 * MENGELOMPOKKAN hasilnya dengan `GROUP BY COL_DESC`. Mengubah keterangan di layar master
 * karena itu langsung mengubah pengelompokan laporan.
 */
export type CauseOfLoss = {
  /**
   * Dibuat sistem saat golongan ditambahkan; tidak pernah berubah sesudahnya.
   * Bentuknya kode situs ditambah tiga digit, mis. `1001`.
   */
  id: string
  /**
   * Isi kolom COL_DESC.
   *
   * Namanya `deskripsi`, mengikuti label yang terpasang di layar Pega: `pyCaption`
   * **"Deskripsi Kerugian"** pada kolom grid maupun isian formnya
   * (`Section/BrowseCauseOfLoss-Section.xml`). Field JSON adalah KONTRAK yang mencerminkan
   * isinya — dan di sini isinya memang itu.
   */
  deskripsi: string
  /**
   * Isi kolom OLD_M_COL_ID: penomoran sebelum sistem ini dibangun.
   *
   * Tidak pernah diisi maupun diubah aplikasi, dan kosong pada golongan yang tidak pernah
   * punya nomor lama.
   *
   * **Tidak ditampilkan di layar.** Grid Pega hanya memuat dua kolom — `ID` dan
   * `Deskripsi Kerugian` — dan layar baru mengikutinya apa adanya. Ia tetap ada di kontrak
   * karena lapisan data mengikuti Report Definition `BrowseVMCauseOfLoss_RD`, yang
   * memuatnya.
   */
  id_lama: string
}

export type CauseOfLossListResponse = {
  penyebab_kerugian: CauseOfLoss[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type CauseOfLossResponse = {
  penyebab_kerugian: CauseOfLoss
  portal: string
}

// ── Master Tipe Surveyors ────────────────────────────────────────────────────────
//
// Cerminan dto di internal/mastertipesurveyors/http. Menggantikan layar Pega
// `Harness/SurveyorsInbox-Harness.xml` atas tabel POOLDATA.M_SURVEYORS.

/**
 * Satu baris master tipe surveyor.
 *
 * Tipe surveyor adalah golongan petugas survei — Internal Surveyor, Loss Adjuster,
 * Expert, Survey Agent. Daftar ORANGNYA ada satu tingkat di bawah, di D_SURVEYORS —
 * lihat `Surveyor` pada bagian Master Surveyors.
 */
export type SurveyorType = {
  /**
   * Kolom M_SURVEY_ID. Dibuat sistem; tidak pernah berubah sesudahnya.
   *
   * Tiga di antaranya DIPATOK LANGSUNG di kueri Pega (`m_survey_id in ('1002')` dan
   * seterusnya), sehingga mengubahnya memutus pemilihan surveyor.
   */
  kode: string

  /** Kolom DESCRIPTION — nama tipe yang dibaca petugas. */
  deskripsi: string

  /**
   * Kolom OLD_M_SURVEY_ID, jejak penomoran sistem sebelumnya.
   *
   * KOSONG pada seluruh empat baris portal ASM. Layar hanya menampilkan kolomnya bila
   * ada baris yang benar-benar mengisinya — kolom yang selamanya kosong hanya menambah
   * lebar tabel tanpa memberi tahu apa pun.
   */
  kode_lama: string
}

export type SurveyorTypeListResponse = {
  tipe_surveyor: SurveyorType[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type SurveyorTypeResponse = {
  tipe_surveyor: SurveyorType
  portal: string
}

/** Badan permintaan penambahan dan penyuntingan. Kode tidak pernah dikirim klien. */
export type SurveyorTypeInput = {
  deskripsi: string
}

// ── Master Surveyors ─────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/mastersurveyors/http. Menggantikan layar Pega
// `Harness/DetailSurveyorsInbox-Harness.xml` atas tabel POOLDATA.D_SURVEYORS.
//
// Ini daftar ORANGNYA — anak dari SurveyorType di atas, yang berisi TIPE-nya. Keduanya
// hidup berdampingan dan mudah tertukar; perbedaannya selalu disebut di tiap keterangan.

/** Posisi persetujuan komite. Nilainya sama persis dengan kolom APPROVAL. */
export type SurveyorStatus = '0' | '1' | '2'

/** Satu baris master surveyor. */
export type Surveyor = {
  /**
   * Kolom D_SURVEY_ID. Dibuat sistem; tidak pernah berubah sesudahnya.
   *
   * Bentuknya kode situs ditambah ENAM digit — berbeda dari kode tipe surveyor yang
   * memakai tiga digit dan urutan yang berlainan.
   */
  id: string

  /** Kolom M_SURVEY_ID, merujuk baris pada master tipe surveyor. */
  kode_tipe: string
  /** Deskripsi tipe, ikut dikirim server supaya grid tidak memanggil master tipe per baris. */
  nama_tipe: string

  nama: string
  alamat: string
  kode_pos: string
  provinsi: string
  telepon: string
  faksimile: string
  email: string
  kontak_lain: string
  /** Kolom BRANCH, berlabel "Cabang" di layar lama. Isian teks — master cabang belum ada. */
  kode_cabang: string
  nama_cabang: string
  /**
   * Kolom LOGIN_APLIKASI.
   *
   * WAJIB bila tipenya Internal Surveyor — lihat `wajib_login_aplikasi`, yang dihitung
   * server. Surveyor eksternal memang tidak masuk ke aplikasi.
   */
  login_aplikasi: string
  /** Kolom DOCID, menunjuk lampiran. Unggah berkasnya menunggu modul S-1 Dokumen. */
  id_dokumen: string

  status: SurveyorStatus
  /** Sebutan status, datang dari server supaya empat layar tidak menerjemahkannya sendiri. */
  status_label: string

  /** Kolom KOMITE — berisi Operator ID, bukan kode bisnis, walau alias lamanya begitu. */
  komite: string
  catatan: string

  /** null selama komite belum memutuskan. */
  tanggal_keputusan: string | null

  /** Dihitung server dari kode tipenya. Aturannya tetap ditegakkan di server. */
  wajib_login_aplikasi: boolean
}

export type SurveyorListResponse = {
  surveyor: Surveyor[]
  /** Jumlah seluruh baris yang cocok SEBELUM dipotong paginasi. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type SurveyorResponse = {
  surveyor: Surveyor
  portal: string
}

/**
 * Badan permintaan pengajuan dan penyuntingan.
 *
 * Empat field milik sistem — status, komite, tanggal keputusan, catatan — SENGAJA tidak
 * ada di sini. Server bahkan menolak badan permintaan yang memuatnya, sehingga tidak ada
 * jalan bagi layar untuk menyetujui surveyornya sendiri dalam satu permintaan.
 */
export type SurveyorInput = {
  kode_tipe: string
  nama: string
  alamat: string
  kode_pos: string
  provinsi: string
  telepon: string
  faksimile: string
  email: string
  kontak_lain: string
  kode_cabang: string
  nama_cabang: string
  login_aplikasi: string
  id_dokumen: string
}

/** Badan permintaan keputusan komite. Status "0" ditolak — itu bukan keputusan. */
export type SurveyorDecisionInput = {
  status: Extract<SurveyorStatus, '1' | '2'>
  catatan: string
}

// ── Master PIC Teknik ────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterpicteknik/http. Menggantikan layar Pega
// `Harness/UserTeknisInbox-Harness.xml` atas tabel POOLDATA.MST_USER_TEKNIK.

/**
 * Satu baris master petugas teknik.
 *
 * PIC Teknik adalah petugas yang menangani klaim. Isi master inilah yang dipakai
 * penugasan klaim untuk memilih petugas berikutnya.
 */
export type Technician = {
  /** Kolom OPERATOR_ID — kunci alaminya, diisi pengguna dan tidak pernah berubah. */
  id_operator: string

  /**
   * Kolom MCL_NAME.
   *
   * DITURUNKAN dari direktori pegawai, tidak pernah diketik. Ia dikirim untuk
   * ditampilkan, tetapi tidak pernah diterima kembali oleh server.
   */
  nama: string

  email: string

  /**
   * Kolom TYPE_BUSINESS. Teks bebas, bukan pilihan tertutup: satu-satunya nilai yang
   * benar-benar muncul di export adalah "NONMBU".
   */
  lini_bisnis: string

  /** Kolom TEAM_GROUP, kelompok kerja petugas. */
  grup: string

  /** Kolom ATASAN — berisi ID operator atasannya, BUKAN namanya. */
  atasan: string

  /** Kolom COUNTER_QUOTA — banyaknya pekerjaan yang BOLEH dipikul. */
  kuota: number

  /**
   * Kolom COUNTER_QUOTA2 — beban kerja petugas yang sama di sistem lain.
   *
   * Namanya di sistem lama, alias "OLD_OPERATOR_ID", menyesatkan: isinya angka.
   */
  kuota_luar: number

  /**
   * Kolom TOTAL_JOB — beban pekerjaan yang SEBENARNYA, berbeda dari kuota yang
   * menyatakan batas.
   *
   * Hanya dibaca, dan hanya terisi pada daftar: ia kolom milik view dan tidak ada di
   * tabelnya.
   */
  beban_kerja: number

  /** Kolom GROUPPANEL, dialias "IBNR" di kueri lama. Hanya dibaca. */
  grup_panel: string

  /** Kolom STS_AKTIF. Hanya yang aktif muncul di daftar. */
  aktif: boolean
}

/** Jawaban pencarian direktori pegawai. */
export type Employee = {
  id_operator: string
  nama: string
  email: string
  /** ID atasan menurut struktur organisasi, dari blok EmpLeader respons direktori. */
  atasan: string
  /** Nama atasan — untuk DITAMPILKAN saja; yang disimpan hanyalah ID-nya. */
  nama_atasan: string
}

export type TechnicianListResponse = {
  pic_teknik: Technician[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type TechnicianResponse = {
  pic_teknik: Technician
  portal: string
}

export type EmployeeResponse = {
  pegawai: Employee
  portal: string
}

/**
 * Badan permintaan penambahan dan penyuntingan.
 *
 * Tiga nilai sengaja TIDAK ada di sini, dan server menolak badan yang memuatnya: `nama`
 * dimiliki direktori, `grup_panel` tidak pernah ditulis, dan `beban_kerja` dihitung.
 */
export type TechnicianInput = {
  id_operator: string
  email: string
  lini_bisnis: string
  grup: string
  atasan: string
  kuota: number
  kuota_luar: number
  aktif: boolean
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
  // Milik modul Master Dominan Factor.
  //
  // Tidak ada kode "nama sudah dipakai" di sini, dan itu disengaja: nama ganda DITERIMA
  // di modul itu (keputusan Work Owner 2026-09-20), meniru layar Pega apa adanya.
  dominantFactorNotFound: 'dominan_factor_tidak_ditemukan',
  /**
   * ID dibentuk `max+1`, sehingga dua penyimpanan yang benar-benar bersamaan dapat
   * memperebutkan nomor yang sama. Menyimpan sekali lagi hampir pasti berhasil.
   */
  dominantFactorIDTaken: 'id_dominan_factor_sudah_dipakai',

  // Milik modul Master Penyebab Kerugian.
  //
  // Tidak ada kode "deskripsi sudah dipakai" di sini, dan itu disengaja: deskripsi ganda
  // DITERIMA di modul itu (keputusan Work Owner 2026-09-20), meniru layar Pega apa adanya.
  causeOfLossNotFound: 'penyebab_kerugian_tidak_ditemukan',
  /**
   * ID dibentuk dari urutan basis data, sehingga bentrok seharusnya mustahil. Kodenya ada
   * supaya kemustahilan itu terlihat bila terjadi; menyimpan sekali lagi hampir pasti
   * berhasil karena urutannya sudah maju.
   */
  causeOfLossIDTaken: 'id_penyebab_kerugian_sudah_dipakai',
  /** Tabel situs pada basis data entitas itu tidak memuat baris aktif. */
  causeOfLossIDUnavailable: 'id_tidak_dapat_dibentuk',

  // Milik modul Master XOL.
  //
  // Tidak ada kode untuk "total share belum 100%", dan itu disengaja: keadaan itu BUKAN
  // galat. Layar lama menyimpan lebih dulu baru menampilkan pesannya, dan perilaku itu
  // ditiru — jawabannya 201/200 dengan senarai `peringatan`, bukan 422.
  xolNotFound: 'xol_tidak_ditemukan',
  xolLayerNotFound: 'layer_xol_tidak_ditemukan',
  /**
   * Nomor dibentuk `max+1`, sehingga dua penyimpanan yang benar-benar bersamaan dapat
   * memperebutkan nomor yang sama. Menyimpan sekali lagi hampir pasti berhasil.
   */
  xolIDTaken: 'id_xol_sudah_dipakai',

  // Milik modul Master Masking.
  maskingNotFound: 'masking_tidak_ditemukan',
  /**
   * Pasangan cabang+pengguna sudah punya baris.
   *
   * Aturannya diambil apa adanya dari procedure lama, yang menolak penyisipan dengan
   * pesan "LOGIN … SUDAH ADA". Dua baris untuk orang yang sama di satu cabang berarti dua
   * kewenangan yang keduanya berlaku, dan yang satu dapat luput saat dicabut.
   */
  maskingPairTaken: 'masking_pengguna_sudah_ada',
  maskingBranchUnknown: 'cabang_tidak_dikenal',
  /**
   * ID dibentuk `max+1` tanpa indeks unik yang menjaganya, sehingga dua penyimpanan yang
   * benar-benar bersamaan dapat memperebutkan nomor yang sama. Menyimpan sekali lagi
   * hampir pasti berhasil.
   */
  maskingIDTaken: 'id_masking_sudah_dipakai',

  // Milik modul Master Tipe Surveyors.
  surveyorTypeNotFound: 'tipe_surveyor_tidak_ditemukan',
  surveyorTypeTaken: 'tipe_surveyor_sudah_ada',
  surveyorTypeCodeTaken: 'kode_tipe_surveyor_sudah_dipakai',
  /** Tabel situs pada basis data entitas itu tidak memuat baris aktif. */
  surveyorTypeCodeUnavailable: 'kode_tidak_dapat_dibentuk',

  // Milik modul Master PIC Teknik.
  technicianNotFound: 'pic_teknik_tidak_ditemukan',
  technicianExists: 'id_operator_sudah_terdaftar',
  /** Direktori pegawai sedang tidak dapat dihubungi — mencoba ulang masuk akal. */
  directoryUnreachable: 'direktori_pegawai_tidak_terhubung',
  /**
   * Alamat layanan direktori belum terdaftar di POOLDATA.GCNM_CONNECT_REST.
   *
   * DIBEDAKAN dari directoryUnreachable dengan sengaja: yang ini tidak akan pulih
   * sendiri, dan menyuruh pengguna mencoba lagi hanya membuang waktunya.
   */
  directoryUnconfigured: 'direktori_pegawai_belum_terdaftar',

  // Milik modul Master Recovery.
  /**
   * Nomor batch baru saja dipakai petugas lain.
   *
   * Keadaan ini TIDAK PERNAH terlihat di sistem lama: procedure-nya melewati penyisipan
   * tanpa satu pun pesan, sehingga petugas kedua mengira batch-nya tersimpan.
   */
  recoveryBatchTaken: 'nomor_batch_sudah_dipakai',
  recoveryPolicyNotFound: 'polis_tidak_ditemukan',
  /** Layanan penerbit VA sedang tidak dapat dihubungi — mencoba ulang masuk akal. */
  vaIssuerUnreachable: 'penerbit_va_tidak_terhubung',
  /**
   * Alamat layanan penerbit VA belum terdaftar di POOLDATA.GCNM_CONNECT_REST.
   *
   * DIBEDAKAN dari yang di atas dengan sengaja: yang ini tidak akan pulih sendiri, dan
   * menyuruh pengguna mencoba lagi hanya membuang waktunya.
   */
  vaIssuerUnconfigured: 'penerbit_va_belum_terdaftar',
  /** Layanan MENJAWAB dan menolak. Mengulang dengan isian yang sama tidak menolong. */
  vaIssuerRejected: 'penerbit_va_menolak',
  recoveryDocumentNotSaved: 'bukti_bayar_gagal_disimpan',
  claimFileEmpty: 'berkas_klaim_kosong',
  claimFileTooBig: 'berkas_klaim_terlalu_besar',
  claimFileUnreadable: 'berkas_klaim_tidak_terbaca',
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
}

export type AccountErrorCode =
  (typeof AccountErrorCode)[keyof typeof AccountErrorCode]

// ── Master Recovery ──────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/masterrecovery/http. Menggantikan layar Pega
// `Harness/MasterRecovery-Harness.xml` atas tabel POOLDATA.MST_RECOVERY_ASM_PENJAMINAN.
//
// Nilai uang dikirim sebagai ANGKA dalam RUPIAH UTUH — bukan sen, dan bukan teks.
// Backend menolak pecahan; lihat masterrecovery.Amount untuk alasannya beserta buktinya.

/** Satu pilihan pada isian "Nama Principal", dari POOLDATA.MST_VIRTUAL_ACCOUNT_PNC. */
export type RecoveryPrincipal = {
  client_id: string
  nama_principal: string
  nomor_virtual_account: string
  /** Surel petugas yang dulu menerbitkan VA ini. Dapat kosong. */
  email_inputor_va: string
}

export type RecoveryPrincipalListResponse = {
  principal: RecoveryPrincipal[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type AutoClaimLineListResponse = {
  kode_perusahaan: string
  batch: string
  baris: AutoClaimLine[]
  paginasi: Pagination
}
/** Bekal awal layar: nomor batch perkiraan dan pilihan tahun, dalam satu permintaan. */
export type RecoveryFormResponse = {
  /**
   * PERKIRAAN, untuk ditampilkan saja.
   *
   * Dua petugas yang membuka layar bersamaan melihat angka yang sama. Yang MENGIKAT
   * adalah nomor yang dikembalikan server sesudah menyimpan.
   */
  nomor_batch_perkiraan: number
  /** Pilihan Tahun, terbaru lebih dulu. */
  tahun: string[]
  portal: string
}

/** Identitas yang menempel pada sebuah nomor polis, dari MST_DET_SALES@ASMD. */
export type RecoveryPolicyReference = {
  nomor_polis: string
  id_lini_bisnis: string
  id_cabang: string
  id_agen: string
  id_marketing: string
  portal: string
}

/** Jawaban penerbitan rekening virtual. */
export type VirtualAccountResponse = {
  nomor_virtual_account: string
  status: string
  pesan: string
  /**
   * Nomor DIAMBIL dari master, bukan baru diterbitkan.
   *
   * Penanda yang dapat dibaca program, bukan disiratkan lewat kalimat di dalam `pesan`
   * seperti sistem lama — layar perlu memutuskan nada pemberitahuannya, dan mencocokkan
   * teks pesan untuk itu adalah cara yang rapuh.
   */
  dipakai_ulang: boolean
  portal: string
}

/** Satu baris daftar klaim hasil unggahan CSV. */
export type RecoveryClaimLine = {
  nomor_polis: string
  nilai_klaim: number
}

export type RecoveryClaimLineResponse = {
  baris_klaim: RecoveryClaimLine[]
  total: number
  /** Jumlah seluruh baris, dihitung server. Sistem lama tidak punya penjumlahan ini. */
  jumlah_nilai_klaim: number
  /** Baris yang DITOLAK beserta alasannya. Dikirim bersama yang diterima. */
  baris_ditolak?: FieldViolation[]
  portal: string
}

/** Jawaban unggahan Bukti Bayar. */
export type RecoveryDocumentResponse = {
  /** DATAID pada POOLDATA.DATA_ATTACHFILE. Dikirim kembali saat menyimpan batch. */
  id_dokumen: string
  nama_berkas: string
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
 /* Badan permintaan Transfer Recovery.
 *
 * Empat hal sengaja TIDAK ada di sini, dan backend MENOLAK badan yang memuatnya:
 * `nomor_batch` (diterbitkan penyimpanan), `sisa` (dihitung server), keempat identitas
 * polis (dicari server dari nomor polis), dan `dicatat_oleh` (diambil dari sesi).
 */
}
export type RecoveryInput = {
  nama_principal: string
  client_id: string
  nomor_virtual_account: string
  tahun: string
  nilai_klaim: number
  pembayaran_sebelumnya: number
  pembayaran: number
  keterangan: string
  posisi_kasus: string
  nomor_polis: string
  id_dokumen: string
  id_log_layanan: string
  baris_klaim: RecoveryClaimLine[]
}

/** Batch yang tersimpan, dikembalikan apa adanya sesudah disimpan. */
export type Recovery = {
  nomor_batch: number
  nama_principal: string
  client_id: string
  nomor_virtual_account: string
  tahun: string
  nilai_klaim: number
  pembayaran_sebelumnya: number
  pembayaran: number
  /** Angka yang BENAR-BENAR tersimpan, hasil hitungan server. */
  sisa: number
  keterangan: string
  posisi_kasus: string
  id_dokumen: string
  dicatat_oleh: string
  nomor_polis: string
  id_lini_bisnis: string
  id_cabang: string
  id_agen: string
  id_marketing: string
  baris_klaim: RecoveryClaimLine[]
}

export type RecoverySaveResponse = {
  recovery: Recovery
  /**
   * Keempat identitas polis berhasil dilengkapi.
   *
   * Dikirim TERPISAH, bukan disimpulkan layar dari kolom yang kosong: kosong dapat
   * berarti nomor polisnya memang tidak diisi, dan itu hal yang berbeda dari nomor polis
   * yang diisi tetapi tidak ditemukan. Batch tetap tersimpan pada kedua keadaan.
   */
  identitas_polis_terisi: boolean
  portal: string
}

// ── Master Masking ───────────────────────────────────────────────────────────────
//
// Cerminan dto di internal/mastermasking/http. Menggantikan layar Pega
// `MasterProteksiVisibilityData` (butir menu `MENU_ID 17` "Master Masking").
//
// Yang dikelola di sini adalah KEWENANGAN MELIHAT DATA PRIBADI: siapa yang boleh melihat
// nomor KTP, surel, dan nomor telepon nasabah tanpa disamarkan, pada modul apa, dan
// seberapa banyak data yang boleh ia cari serta lihat.
//
// Kolom PASSWORD pada tabel lama sengaja TIDAK ada di sini. Isinya literal coba-coba yang
// tertinggal di sistem lama, tidak pernah tampil di layar Pega, dan tidak pernah
// diperiksa siapa pun — lihat konstanta legacyPassword di repo/sqlstore backend.

/** Satu baris master masking. */
export type Masking = {
  /** Kolom ID_MST. Dibuat sistem; tidak pernah berubah sesudahnya. */
  id: string

  /** Kolom CABANG — kode cabang, kunci ke POOLDATA.BRANCH.ID. */
  cabang: string

  /**
   * Nama cabang hasil penggabungan ke POOLDATA.BRANCH.
   *
   * BACA-SAJA: ia tidak pernah disimpan di tabel masking dan tidak pernah dikirim klien.
   * Server menghitungnya ulang setiap kali baris dibaca.
   */
  nama_cabang: string

  /** Kolom LOGIN — pengguna yang diberi kewenangan. */
  login: string

  /** Kolom MODUL — layar tempat kewenangan ini berlaku. */
  modul: string

  /**
   * Kolom SUBMODUL: DAFTAR bagian layar, dipisah koma, dengan koma di ujungnya.
   *
   * Bentuknya dibawa apa adanya (keputusan Work Owner 2026-09-20) karena Pega masih
   * membaca tabel yang sama selama masa paralel.
   */
  sub_modul: string

  /** Kolom LOGSEARCH, berlabel "MAX CARI DATA" di layar lama. Ia KUOTA, bukan penanda. */
  maks_cari: number

  /** Kolom LOGSEEN, berlabel "MAX LIHAT DATA". */
  maks_lihat: number

  /** Kolom STS_KTP. Di basis data nilainya teks 'Ya'/'Tidak'; server menerjemahkannya. */
  lihat_ktp: boolean
  /** Kolom STS_EMAIL. */
  lihat_email: boolean
  /** Kolom STS_NOTELP. */
  lihat_notelp: boolean

  /**
   * Kolom STS_AKTF, bernilai 'AKTIF' atau 'TIDAK AKTIF'.
   *
   * Inilah yang menjadi penghapusan: tombol DELETE layar lama tidak pernah membuang baris,
   * ia hanya mengubah kolom ini. Sejalan dengan `D-66`.
   */
  aktif: boolean

  /** Kolom USERINPUT — siapa yang terakhir menyimpan baris ini. Diisi dari sesi. */
  dicatat_oleh: string

  /** Kolom TANGGALINPUT dalam RFC 3339 UTC. Kosong bila belum pernah tercatat. */
  dicatat_pada: string
}

/** Satu pilihan cabang. POOLDATA.BRANCH hanya DIBACA. */
export type MaskingBranch = {
  kode: string
  nama: string
}

export type MaskingListResponse = {
  masking: Masking[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type MaskingResponse = {
  masking: Masking
  portal: string
}

export type MaskingBranchListResponse = {
  cabang: MaskingBranch[]
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
 /* Badan permintaan penambahan dan penyuntingan.
 *
 * Empat hal sengaja TIDAK ada: `id`, `nama_cabang`, `aktif`, dan `dicatat_oleh`. Server
 * menolak badan permintaan yang memuat salah satunya. Yang paling penting di antaranya
 * adalah `aktif` — tanpa pemisahan itu, menyimpan form yang dibuka dari baris nonaktif
 * akan diam-diam mengembalikan kewenangan membuka data pribadi.
 */
}
export type MaskingInput = {
  cabang: string
  login: string
  modul: string
  sub_modul: string
  maks_cari: number
  maks_lihat: number
  lihat_ktp: boolean
  lihat_email: boolean
  lihat_notelp: boolean
}

/** Badan permintaan pengaktifan dan penonaktifan. */
export type MaskingStatusInput = {
  aktif: boolean
}
