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

/**
 * Satu rincian dokumen klaim — baris V_LST_DET_TYPE_DOC.
 *
 * Layar "Daftar Detail Tipe Dokumen" (MENU_ID 41), pengganti
 * `Harness/ListDetTypeDocument-Harness.xml`.
 *
 * JANGAN tertukar dengan dua saudaranya:
 *
 *   MENU_ID 40  `DocumentType`              induk — hanya ID dan nama tipe dokumen
 *   MENU_ID 41  `DetailDocumentType`        ini   — rinciannya
 *   MENU_ID 42  `BusinessDocumentRule`      aturan per lini bisnis, TABEL BERBEDA
 *
 * Yang ketiga bernama mirip tetapi tabelnya sama sekali berbeda
 * (`LST_TYPE_DOC_BUSINESS`), dan ia justru MEMBACA `id` di sini lewat `DOC_TYPE_DT_ID`.
 *
 * Nama fieldnya mengikuti label isian di layar Pega
 * (`Section/BrowseListDetailTypeDocument-Section.xml`), supaya satu istilah berlaku dari
 * layar sampai ke kontrak.
 */
export type DetailDocumentType = {
  /** Kunci baris. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom DOC_TYPE_ID — label layar "ID Tipe Dokumen". Rujukan ke master MENU_ID 40. */
  id_tipe_dokumen: string
  /**
   * Kolom TYPE_DOCUMENT — nama tipe dokumen menurut masternya.
   *
   * HANYA DIBACA: ia datang dari join view, bukan dari baris ini. KOSONG berarti
   * rujukannya sudah tidak ada di master — dan barisnya tetap dikirim, bukan
   * disembunyikan, supaya petugas dapat memperbaikinya.
   */
  nama_tipe_dokumen: string
  /** Kolom DETAIL_DOCUMENT — label layar "Detail Dokumen". Nama yang dibaca petugas. */
  detail_dokumen: string
  /**
   * Kolom STS_INSURED — label layar "Status Tertanggung".
   *
   * TEKS BEBAS, bukan penanda. Di form Pega ia isian biasa tanpa satu pun daftar pilihan
   * — sama seperti STS_PROSES pada master MENU_ID 40, yang namanya juga menyiratkan
   * status padahal bukan.
   */
  status_tertanggung: string
  /**
   * Kolom DOC_COL_ID — kode penyebab kerugian, rujukan ke Master Penyebab Kerugian.
   *
   * Ia TARGET TERSEMBUNYI, bukan isian yang dilihat petugas: di layar Pega ia hanya diisi
   * autocomplete saat sebuah pilihan dipilih (`pyPropertyTarget`). KOSONG bila
   * keterangannya diketik bebas.
   */
  id_penyebab_kerugian: string
  /**
   * Kolom DOC_COL_INFO — label layar **"Dokumen kolom ID"**.
   *
   * **TERSIMPAN, bukan hasil join.** Inilah isian yang benar-benar dilihat dan diketik
   * petugas. Karena `pyAllowFreeFormInput=true`, keterangan di luar master tetap sah — dan
   * pada keadaan itu `id_penyebab_kerugian` kosong.
   */
  keterangan_penyebab_kerugian: string
  /**
   * Kolom OBJ_DOC — kode objek dokumen, rujukan ke master MENU_ID 43.
   *
   * Target tersembunyi, sama seperti `id_penyebab_kerugian`.
   */
  id_objek_dokumen: string
  /**
   * Kolom OBJ_DOC_DESC — label layar **"Objek Dokumen"**.
   *
   * **TERSIMPAN**, dengan alasan yang sama seperti keterangan di atas.
   */
  keterangan_objek_dokumen: string
  /**
   * Kolom RISK — label layar "Resiko".
   *
   * TEKS, bukan angka, karena itulah bentuk kolomnya dan karena baris warisan dapat
   * memuat isi yang bukan angka. Isinya dibaca sebagai bilangan oleh
   * `Activity/SetTypePDFAdjustment-Act.xml` (`@toDecimal(.RISK)<=0`), dan kosong di sana
   * berarti nol.
   */
  resiko: string
  /**
   * Aturan per lini bisnis — baris V_LST_DET_TYPE_DOC_BISNIS.
   *
   * SELALU kosong pada hasil daftar; hanya terisi pada pengambilan satu baris. Layar
   * karena itu WAJIB memuat ulang barisnya saat dibuka untuk disunting — memakai baris
   * dari daftar akan membuat form tampak seolah seluruh lini bisnisnya sudah dihapus, dan
   * menyimpannya benar-benar menghapusnya.
   */
  bisnis: DetailDocumentTypeBusiness[]
}

/** Aturan dokumen pada satu lini bisnis. */
export type DetailDocumentTypeBusiness = {
  /** Kolom DFT_BISNIS_ID — label layar "ID Bisnis". */
  id_bisnis: string
  /** NOTE pada POOLDATA.BUSINESS. HANYA DIBACA; kosong bila bisnisnya sudah tidak ada. */
  nama_bisnis: string
  /**
   * Kolom STS_WAJIB — label layar "Status Wajib".
   *
   * Boolean di kontrak; yang tersimpan TEKS "Ya"/"Tidak" (keputusan Work Owner
   * 2026-09-23). Data lama bercampur — kolomnya memuat "Ya", "Tidak", "1", dan "0" —
   * dan server yang menjembataninya.
   */
  status_wajib: boolean
  /** Kolom MIN_DOC — label layar "Minimum Dokumen". Nol berarti tanpa tuntutan jumlah. */
  minimum_dokumen: number
}

export type DetailDocumentTypeListResponse = {
  detail_tipe_dokumen: DetailDocumentType[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type DetailDocumentTypeResponse = {
  detail_tipe_dokumen: DetailDocumentType
  portal: string
}

/**
 * Keempat daftar pilihan form Daftar Detail Tipe Dokumen.
 *
 * Dikirim dalam SATU respons meski berasal dari empat master, karena form selalu
 * membutuhkan keempatnya bersamaan.
 */
export type DetailDocumentTypeReferenceResponse = {
  tipe_dokumen: DetailDocumentTypeChoice[]
  penyebab_kerugian: DetailDocumentTypeChoice[]
  objek_dokumen: DetailDocumentTypeChoice[]
  bisnis: DetailDocumentTypeChoice[]
  /**
   * Master mana yang gagal dibaca. Kosong berarti keempatnya terbaca.
   *
   * Ia ada supaya layar dapat membedakan "masternya kosong" dari "masternya tidak dapat
   * dibaca" — dua keadaan yang tampak sama persis (daftar pilihan kosong) tetapi menuntut
   * kalimat yang berbeda kepada petugas.
   */
  tidak_tersedia: string[]
  portal: string
}

/** Satu pilihan pada isian berdaftar milik layar Daftar Detail Tipe Dokumen. */
export type DetailDocumentTypeChoice = {
  id: string
  nama: string
}

/** Badan permintaan penambahan dan penyuntingan detail tipe dokumen. */
export type DetailDocumentTypeInput = {
  id_tipe_dokumen: string
  detail_dokumen: string
  status_tertanggung: string
  /**
   * Kode DAN keterangan dikirim BERPASANGAN.
   *
   * Keterangannya yang diketik petugas; kodenya diisi layar saat keterangan itu cocok
   * dengan salah satu pilihan master. Keterangan di luar master tetap dikirim dengan kode
   * kosong — persis `pyAllowFreeFormInput=true` di layar lama. Mengirim kodenya saja akan
   * membuang isian yang sah.
   */
  id_penyebab_kerugian: string
  keterangan_penyebab_kerugian: string
  id_objek_dokumen: string
  keterangan_objek_dokumen: string
  resiko: string
  /**
   * Susunan AKHIR yang dikehendaki petugas — server menggantikan seluruh daftar lamanya,
   * bukan menambahinya.
   */
  bisnis: DetailDocumentTypeBusinessInput[]
}

/**
 * Satu baris grid bisnis yang dikirim layar.
 *
 * Tanpa nama bisnis: namanya milik master dan dibaca lewat join, tidak disimpan di baris
 * ini. Mengirimkannya hanya akan menyimpan salinan yang dapat menyimpang dari masternya.
 */
export type DetailDocumentTypeBusinessInput = {
  id_bisnis: string
  status_wajib: boolean
  minimum_dokumen: number
}

/* ── Daftar Tipe Dokumen Bisnis — MENU_ID 42 ──────────────────────────────────
 *
 * Satu baris menjawab: DOKUMEN APA yang harus diunggah, untuk LINI BISNIS mana, pada
 * TAHAP KLAIM mana. Tabelnya POOLDATA.LST_TYPE_DOC_BUSINESS, dan pembacanya bukan layar
 * ini melainkan enam layar unggah dokumen di sepanjang perjalanan klaim.
 *
 * Layarnya BERTINGKAT DUA, dan itu tercermin di dua tipe jawaban yang berbeda: daftar
 * bisnis lebih dulu, lalu aturan dokumen milik bisnis yang dipilih.
 */

/** Satu lini bisnis — POOLDATA.BUSINESS (ID, NOTE). */
export type BusinessChoice = {
  id: string
  nama_bisnis: string
  /**
   * Bisnis yang DILEWATI tombol "Pilih semua" — kelimanya lini MBU.
   *
   * Di Pega kodenya ditulis di dalam rule (`SetAllBusiness-Act.xml:984`); di sini
   * daftarnya datang dari konfigurasi. Bisnis bertanda ini **tetap dapat dipilih satu
   * per satu**: pengecualiannya hanya berlaku pada pemilihan massal.
   */
  dikecualikan_pilih_semua: boolean
}

export type BusinessChoiceListResponse = {
  bisnis: BusinessChoice[]
  total: number
  portal: string
  /**
   * Apakah tombol "Pilih semua" ditampilkan — meniru `pyVisible` layar lama, yang
   * menuntut `OperatorID.pyPosition='NONMBU'`.
   *
   * Ia penyembunyian TAMPILAN, bukan kewenangan: bisnis yang sama tetap dapat dipilih
   * satu per satu oleh siapa pun yang membuka layarnya.
   */
  boleh_pilih_semua: boolean
}

/** Satu aturan kelengkapan dokumen. */
export type BusinessDocumentRule = {
  /** LST_TYPE_DOC_BUSINESS.ID — diterbitkan penyimpanan, tidak pernah diisi pengguna. */
  id: string

  /**
   * BUSINESSID. TIDAK dapat diubah setelah baris dibuat — procedure lama pun tidak
   * mengubahnya. Aturan dokumen disalin ke lini bisnis lain, bukan dipindahkan.
   */
  id_bisnis: string
  /** BUSINESS.NOTE hasil join; tidak pernah dikirim saat menyimpan. */
  nama_bisnis: string

  /** DOCUMENT_TYPE_ID → V_LST_DOC_TYPE. */
  id_tipe_dokumen: string
  /**
   * V_LST_DOC_TYPE.TYPE_DOCUMENT hasil join — TAHAP klaim tempat dokumen ini diminta.
   *
   * Enam nilainya: REGISTER, SURVEY, COMMITEE (salah ketik ada di datanya), PAYMENT,
   * SALVAGE, COLLECTING DOCUMENT.
   */
  tipe_dokumen: string

  /** OBJECT_DOC_ID → V_LST_DOC_OBJ. Boleh kosong. */
  id_object_dokumen: string
  /** V_LST_DOC_OBJ.KET_DOC_OBJ hasil join. */
  object_dokumen: string

  /** DOC_TYPE_DT_ID → V_LST_DET_TYPE_DOC. */
  id_detail_dokumen: string
  /**
   * DETAIL_DOKUMEN — nama dokumen yang dibaca petugas di layar unggah.
   *
   * Nilai tepat "-" MENYEMBUNYIKAN baris ini dari SELURUH layar unggah: keenam kueri
   * pembacanya menyaring `AND DETAIL_DOKUMEN != '-'`. Barisnya tetap ada dan tetap
   * tampil di layar master ini.
   */
  detail_dokumen: string

  /**
   * STS_WAJIB.
   *
   * WAJIB DI SINI BELUM TENTU WAJIB DI KLAIM. Baris ber-`status_wajib` true yang daftar
   * `jenis_klaim`-nya KOSONG tidak pernah menjadi wajib pada klaim mana pun — kueri
   * klaim menghitung jaminannya lebih dulu, dan nol jaminan berarti "Tidak".
   */
  status_wajib: boolean

  /** MIN_DOC — berkas paling sedikit yang harus diunggah. */
  minimum_dokumen: number

  /**
   * Jaminan yang membuat baris ini benar-benar wajib — COVERAGE_DOC_BUSINESS.COVERAGEID.
   *
   * Hanya terisi pada pengambilan SATU baris; pada daftar ia selalu kosong. Ia
   * DITAMBAHKAN, tidak pernah diganti maupun dibuang — tidak ada satu pun jalur hapus
   * terhadap tabel itu di seluruh sistem lama.
   */
  jenis_klaim: string[]
}

export type BusinessDocumentRuleListResponse = {
  tipe_dokumen_bisnis: BusinessDocumentRule[]
  total: number
  portal: string
}

export type BusinessDocumentRuleResponse = {
  tipe_dokumen_bisnis: BusinessDocumentRule
  portal: string
}

/**
 * Jawaban penambahan, yang menghasilkan BANYAK baris sekaligus.
 *
 * Bentuknya berbeda dari `BusinessDocumentRuleResponse` karena penambahan di layar ini
 * memang perkalian bisnis kali baris dokumen.
 */
export type BusinessDocumentRuleCreateResponse = {
  tipe_dokumen_bisnis: BusinessDocumentRule[]
  total: number
  portal: string
}

/** Satu baris aturan yang dikirim layar. */
export type BusinessDocumentRuleInput = {
  id_tipe_dokumen: string
  id_object_dokumen: string
  id_detail_dokumen: string
  detail_dokumen: string
  status_wajib: boolean
  minimum_dokumen: number
}

/**
 * Badan permintaan penambahan: beberapa bisnis dikali beberapa baris dokumen.
 *
 * Bentuk jamak-kali-jamak ini bukan kenyamanan yang ditambahkan, melainkan perilaku layar
 * lama — aturan kelengkapan dokumen berulang nyaris sama di banyak lini bisnis.
 */
export type BusinessDocumentRuleBatchInput = {
  bisnis: string[]
  dokumen: BusinessDocumentRuleInput[]
}

/** Badan permintaan penambahan satu jaminan. */
export type BusinessDocumentRuleCoverageInput = {
  id_jenis_klaim: string
}

/** Satu pilihan pada isian yang merujuk master lain. */
export type MasterChoice = {
  id: string
  nama: string
  /**
   * Hanya terisi pada daftar Detail Dokumen, dan di sana ia id tahap dokumen pemiliknya.
   *
   * Layar memakainya untuk menyempitkan pilihan Detail Dokumen mengikuti Tipe Dokumen
   * yang sudah dipilih pada baris yang sama — persis parameter `idDocument` pada
   * autocomplete `BrowseVLstDetTypeDoc_RD`.
   */
  id_induk: string
}

export type MasterChoiceListResponse = {
  pilihan: MasterChoice[]
  total: number
  portal: string
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

  /**
   * Milik modul Daftar Tipe Dokumen Bisnis.
   *
   * HANYA DUA, dan itu cerminan layar lamanya:
   * `businessDocumentRuleBusinessRequired` meniru satu-satunya validasi yang benar-benar
   * ada di sana (`InsertDetailTypeDocumentBusiness_act:209`), beserta kalimatnya.
   *
   * Dua kode lain sempat ada — baris dokumen kosong dan jenis klaim kosong — lalu dicabut
   * pada 2026-09-23. Keduanya penambahan saya sendiri; di Pega kedua keadaan itu DILEWATI
   * diam-diam, bukan ditolak, dan Work Owner menetapkan layar ini mengikuti Pega apa
   * adanya.
   */
  businessDocumentRuleNotFound: 'tipe_dokumen_bisnis_tidak_ditemukan',
  businessDocumentRuleBusinessRequired: 'nama_bisnis_belum_diisi',

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
  /** Entitas yang benar-benar menjawab permintaan ini. Cerminan BatchListResponse.Portal. */
  portal: string
}

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
 * satu isian yang tidak ada di layar Master COL biasa: daftar `bisnis` yang memakainya.
 *
 * Isian `id_master_kerugian` (MST_COL_ID) DICABUT 2026-09-23 atas keputusan Work Owner —
 * kolomnya sudah tidak dipakai.
 *
 * Namanya berawalan `SimasOnline` dengan sengaja. `CauseOfLoss` tanpa awalan milik modul
 * Master Penyebab Kerugian di atas — modul LAIN, atas tabel yang sama tetapi dengan isian
 * dan kontrak yang berbeda. Sebelum keduanya dibedakan, keduanya bernama sama dan yang
 * satu diam-diam menimpa yang lain. Backend membedakannya dengan cara yang sama
 * (`simasOnlineCauseOfLossSelector`).
 */
export type SimasOnlineCauseOfLoss = {
  /** Kolom M_COL_ID. Diterbitkan server; tidak pernah diisi pengguna. */
  id: string
  /** Kolom COL_DESC — di layar Pega berlabel "Nama Cause of loss". */
  nama: string
  /**
   * Daftar bisnis yang memakai penyebab kerugian ini.
   *
   * Pada jawaban DAFTAR ia selalu `[]`, dan itu disengaja: grid hanya menampilkan ID dan
   * nama. Layar memuatnya saat baris dibuka untuk disunting.
   */
  bisnis: Business[]
}

export type SimasOnlineCauseOfLossListResponse = {
  cause_of_loss: SimasOnlineCauseOfLoss[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

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

export type SimasOnlineCauseOfLossResponse = {
  cause_of_loss: SimasOnlineCauseOfLoss
  portal: string
}

export type BusinessListResponse = {
  bisnis: Business[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

/**
 * Satu baris Daftar Objek Dokumen (MENU_ID 43).
 *
 * Objek Dokumen menyatakan sebuah dokumen melekat pada APA — objek pertanggungan mana, atau
 * pihak mana. Ia data acuan yang dirujuk master lain lewat
 * `LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID`, sehingga `id`-nya tidak pernah berubah.
 *
 * JANGAN tertukar dengan `DocumentType` (MENU_ID 40, "Daftar Tipe Dokumen"), yang menyatakan
 * dokumen itu JENISNYA apa. Keduanya dirujuk bersamaan oleh master yang sama, dan di layar
 * Pega pun keduanya berdampingan.
 */
export type DocumentObject = {
  /** Kolom ID. Diterbitkan server, tidak pernah berubah. */
  id: string
  /**
   * Kolom KET_DOC_OBJ.
   *
   * Namanya mengikuti LABEL layar — "Daftar Objek Dokumen" — bukan nama kolomnya. Kata
   * "Daftar" di depan label itu milik LAYARNYA; nilai satu barisnya adalah satu objek
   * dokumen.
   */
  objek_dokumen: string
  /**
   * Kolom OLD_ID: penomoran sebelum sistem ini dibangun.
   *
   * Dikirim server, tetapi TIDAK ditampilkan di layar — grid Pega hanya memuat dua kolom.
   * Ketiadaannya di layar dijaga uji, supaya menampilkannya kembali menjadi keputusan dan
   * bukan perbaikan yang menyelinap.
   */
  id_lama: string
  /**
   * Bisnis yang memakai objek dokumen ini.
   *
   * Pada jawaban DAFTAR ia selalu kosong — grid hanya menampilkan dua kolom, sehingga
   * server tidak menariknya untuk seluruh baris. Layar memuatnya lewat permintaan satu
   * baris saat baris itu dibuka untuk disunting.
   */
  bisnis: Business[]
}

export type DocumentObjectListResponse = {
  objek_dokumen: DocumentObject[]
  total: number
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type DocumentObjectResponse = {
  objek_dokumen: DocumentObject
  portal: string
}

/**
 * Badan permintaan penambahan dan penyuntingan Daftar Objek Dokumen.
 *
 * `bisnis` berisi NAMA bisnis, bukan ID-nya — sama seperti Master COL Simas Online, dan
 * dengan alasan yang sama: itulah yang diketik dan dilihat petugas, dan nama yang diketik
 * bebas memang tidak punya ID. Server yang menyelesaikannya menjadi ID dengan mencocokkan
 * ke master; nama yang tidak cocok tetap diterima dan disimpan tanpa ID.
 *
 * `id` dan `id_lama` sengaja tidak ada: yang pertama diterbitkan server atau diambil dari
 * jalur URL, yang kedua jejak sejarah yang bukan isian.
 */
export type DocumentObjectInput = {
  objek_dokumen: string
  bisnis: string[]
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
  total: number
  portal: string
}

/* Badan permintaan Transfer Recovery.
 *
 * Empat hal sengaja TIDAK ada di sini, dan backend MENOLAK badan yang memuatnya:
 * `nomor_batch` (diterbitkan penyimpanan), `sisa` (dihitung server), keempat identitas
 * polis (dicari server dari nomor polis), dan `dicatat_oleh` (diambil dari sesi).
 */
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
 * Badan permintaan penambahan dan penyuntingan Master COL Simas Online.
 *
 * `bisnis` berisi NAMA bisnis, bukan ID-nya — itulah yang diketik dan dilihat petugas di
 * layar Pega, dan nama yang diketik bebas memang tidak punya ID. Server yang
 * menyelesaikannya menjadi ID dengan mencocokkan ke master; nama yang tidak cocok tetap
 * diterima dan disimpan tanpa ID.
 */
export type SimasOnlineCauseOfLossInput = {
  nama: string
  bisnis: string[]
}

/**
 * Badan permintaan penambahan dan penyuntingan Master Masking.
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

// ═══════════════════════════════════════════════════════════════════════════════
// Tipe yang DIPULIHKAN, bukan ditulis baru
//
// Deklarasi di bawah ini pernah ada di berkas ini dan hilang pada commit 99273e5
// ("backup fran sebelum masuk master"), yang memangkas types.ts dari 1609 menjadi
// 1191 baris. Sembilan modul yang memakainya karena itu tidak dapat dikompilasi sejak
// saat itu — bukan sejak penggabungan cabang.
//
// Isinya diambil APA ADANYA dari commit c1e3194, bukan ditulis ulang: hanya deklarasi
// yang namanya belum ada di berkas ini yang dipulihkan, sehingga tidak ada kontrak yang
// diam-diam berubah bentuk.
// ═══════════════════════════════════════════════════════════════════════════════

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
// ── Pelaporan Klaim ──────────────────────────────────────────────────────────────
//
// Cerminan dto di `internal/pelaporanklaim/http`. Berbeda dari blok yang dipulihkan di
// atas, tipe modul ini TIDAK PERNAH ada di berkas ini — modulnya dulu menyimpan tipenya
// sendiri di `src/modules/inbox-laporan-klaim/types.ts`, lalu modulnya diganti nama
// menjadi `pelaporan-klaim` dan impornya dialihkan ke sini tanpa tipenya ikut pindah.
//
// Isinya karena itu diturunkan dari `http/dto.go` BARIS PER BARIS, bukan dari berkas lama
// yang bentuknya sudah tidak sama: `ReportStage` dan `StageSummary` bahkan tidak ada di
// sana sama sekali.

/**
 * Tahap sebuah laporan klaim.
 *
 * DIHITUNG server dari keadaan laporan, bukan disimpan dan bukan dihitung layar
 * (`http/dto.go:71`). Menghitungnya di layar berarti aturan tahap hidup di dua tempat.
 */
export const ReportStage = {
  notTransferred: 'BELUM_TRANSFER',
  notRegistered: 'BELUM_REGISTRASI',
  registered: 'SUDAH_REGISTRASI',
  accepted: 'SUDAH_AKSEPTASI',
  rejected: 'DITOLAK',
} as const

export type ReportStage = (typeof ReportStage)[keyof typeof ReportStage]

/** Satu laporan klaim. Cerminan `ReportDTO`. */
export type ClaimReport = {
  nomor: string

  nama_pelapor: string
  email_pengirim: string
  telepon_pengirim: string
  nama_kurir: string
  subjek_email: string

  nomor_polis: string
  nama_tertanggung: string
  email_tertanggung: string
  kode_bisnis: string
  group_panel: string
  nomor_referensi: string

  /** `YYYY-MM-DD`, tanpa jam dan tanpa zona. Kosong berarti belum diisi. */
  tanggal_kejadian: string
  lokasi_kejadian: string
  kronologi: string
  rincian_kerusakan: string
  sim_pengendara: string
  nilai_estimasi: string
  tipe_klaim: string

  jumlah_dokumen: number
  /** `YYYY-MM-DD`. Kosong berarti belum diisi. */
  tanggal_terima_dokumen: string

  nomor_klaim: string
  ditransfer: boolean
  /** RFC 3339 UTC. Kosong berarti belum terjadi. */
  tanggal_transfer: string
  /** RFC 3339 UTC. Kosong berarti belum terjadi. */
  tanggal_registrasi: string
  alasan_belum_transfer: string
  catatan_belum_registrasi: string

  tahap: ReportStage
  /** Label siap tampil, dikirim server supaya layar tidak memuat terjemahannya sendiri. */
  tahap_label: string

  /**
   * Dihitung server dari aturan yang sama yang ditegakkan usecase (`http/dto.go:79`).
   * Tanpa keduanya, layar harus menirukan aturannya sendiri — dan tiruan itulah yang
   * akan menyimpang.
   */
  dapat_ditransfer: boolean
  dapat_diubah: boolean

  kode_cabang: string
  diinput_oleh: string
  /** RFC 3339 UTC. */
  diinput_pada: string
  /** RFC 3339 UTC. */
  diubah_pada: string
}

/** Jumlah laporan per tahap, untuk lencana di atas tiap tab. Cerminan `SummaryDTO`. */
export type StageSummary = {
  tahap: ReportStage
  label: string
  jumlah: number
}

export type ClaimReportListResponse = {
  laporan: ClaimReport[]
  /**
   * Banyaknya baris yang cocok SEBELUM dipotong paginasi — bukan panjang senarai di atas.
   * Tanpa pembedaan ini, layar tidak dapat tahu masih ada halaman berikutnya.
   */
  jumlah: number
  batas: number
  lewati: number
  /** Memuat KELIMA tahap selalu, termasuk yang jumlahnya nol. */
  ringkasan: StageSummary[]
}

export type ClaimReportResponse = {
  laporan: ClaimReport
}

/**
 * Kode galat modul Pelaporan Klaim.
 *
 * Terpisah dari ErrorCode karena ia milik satu modul, sementara ErrorCode mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const ClaimReportErrorCode = {
  notFound: 'laporan_klaim_tidak_ditemukan',
  validationFailed: 'validasi_gagal',
  alreadyTransferred: 'laporan_sudah_ditransfer',
  alreadyRegistered: 'laporan_sudah_diregistrasi',
  numberTaken: 'nomor_laporan_sudah_dipakai',
  malformedRequest: 'permintaan_cacat',
  internalError: 'galat_internal',
} as const

export type ClaimReportErrorCode =
  (typeof ClaimReportErrorCode)[keyof typeof ClaimReportErrorCode]
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

// ---------------------------------------------------------------------------
// Inbox Komite (`TKT-B07-002`, MENU_ID 52 — harness `InboxKomite_Harness`)
// ---------------------------------------------------------------------------

/**
 * Kotak mana yang sedang dibuka.
 *
 * Ketiganya menggantikan tiga grid pada `Section/InboxKomite_section-Section.xml`:
 * "Kotak Masuk Komite Outstanding", "…Diterima", dan "…Ditolak".
 *
 * Hanya yang pertama berisi **pekerjaan** dalam arti `D-79` — barisnya hilang setelah
 * diputuskan dan punya tenggat. Dua sisanya riwayat.
 */
export const KomiteInboxKind = {
  outstanding: 'outstanding',
  accepted: 'diterima',
  rejected: 'ditolak',
} as const

export type KomiteInboxKind = (typeof KomiteInboxKind)[keyof typeof KomiteInboxKind]

/** Tiga keputusan yang dapat diberikan komite. */
export const KomiteDecisionKind = {
  approve: 'setuju',
  reject: 'tolak',
  return: 'kembalikan',
} as const

export type KomiteDecisionKind =
  (typeof KomiteDecisionKind)[keyof typeof KomiteDecisionKind]

/** Kesimpulan atas seluruh keputusan pada satu kasus. */
export const KomiteOutcome = {
  pending: 'menunggu',
  approved: 'disetujui',
  rejected: 'ditolak',
  returned: 'dikembalikan',
} as const

export type KomiteOutcome = (typeof KomiteOutcome)[keyof typeof KomiteOutcome]

/** Satu keputusan komite yang tercatat. Tidak pernah diubah dan tidak pernah dihapus. */
export type KomiteDecision = {
  id: string
  jenjang: number
  keputusan: KomiteDecisionKind
  catatan?: string
  oleh: string
  nama_pemutus?: string
  /** RFC 3339 dalam UTC. */
  pada: string
}

/** Keadaan penjenjangan satu kasus menurut sistem ini. */
export type KomiteProgress = {
  kesimpulan: KomiteOutcome
  /** Nol berarti **belum diketahui**, bukan nol jenjang — lihat penanda di bawah. */
  jumlah_jenjang: number
  jenjang_kini: number
  jenjang_disetujui: number
  /**
   * Dikirim sebagai kesimpulan, bukan dibiarkan disimpulkan layar dari angka nol.
   *
   * Jumlah jenjang yang tidak diketahui lalu digambar sebagai "1 dari 1" akan membuat
   * persetujuan pertama tampak menutup seluruh komite.
   */
  jumlah_jenjang_belum_diketahui: boolean
  selesai: boolean
  /** Layar memakainya menyembunyikan tombol; penegakannya tetap di server (409). */
  sudah_saya_putuskan: boolean
  keputusan: KomiteDecision[]
}

/**
 * Satu baris Inbox Komite.
 *
 * Nama field di sini **sudah benar**, berbeda dari property Pega yang digantikannya:
 * `.IBNR` → nilai ASM share, `.pyScore` → nilai OR ASM, `.DraftWordingID` → PIC klaim,
 * `.StatusKlaim` → tipe komite (`D-19`).
 */
export type KomiteCase = {
  nomor_case: string
  nomor_klaim: string

  nomor_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  sumber_bisnis: string
  cabang: string
  group_panel?: string
  pic_klaim?: string

  /** RFC 3339 dalam UTC; kosong berarti belum ada, bukan tahun 1. */
  tanggal_komite?: string
  tanggal_input?: string
  /** Dihitung **server**, dalam hari kalender WIB. Jam peramban tidak dipakai. */
  aging_komite: number

  status_kerja?: string
  tipe_komite?: string

  /** Teks desimal kanonik — bukan angka JSON, yang akan dibulatkan diam-diam. */
  nilai_klaim: string
  nilai_asm_share: string
  nilai_or_asm: string

  note_komite?: string

  /** Membedakan "belum dinilai AI" dari "dinilai dengan hasil kosong". */
  ada_penilaian_ai: boolean
  jawaban_ai?: string
  note_ai_diterima?: string
  note_ai_ditolak?: string
  tanggal_ai?: string

  /** Keputusan dan jenjang yang tercatat **di Pega**, bukan di sistem ini. */
  keputusan_pega?: KomiteOutcome
  jenjang_pega?: number

  penjenjangan: KomiteProgress
}

export type KomiteInboxSummary = {
  outstanding: number
  diterima: number
  ditolak: number
}

export type KomiteInboxListResponse = {
  kasus: KomiteCase[]
  /** Seluruh yang cocok, bukan isi halaman ini. */
  total: number
  lewati: number
  batas: number
  ringkasan: KomiteInboxSummary
  /** Kotak yang benar-benar dipakai server — yang tidak dikenali jatuh ke outstanding. */
  kotak: KomiteInboxKind
  /**
   * Operator yang dipakai menyaring, dipantulkan kembali.
   *
   * Dengan ini inbox yang kosong dapat dibedakan sebabnya: tidak ada pekerjaan, versus
   * identitas sesi tidak cocok dengan satu pun `OPERATOR_ID` di data warisan — keadaan
   * yang sangat mungkin selama pemetaan identitas HCC/HCQ belum ada (`ADR-0024`).
   */
  operator: string
  /** Jam server yang dipakai menghitung aging. */
  sekarang: string
}

export type KomiteCaseResponse = {
  kasus: KomiteCase
  sekarang: string
}

/** Kode galat khusus layar Inbox Komite. */
export const KomiteInboxErrorCode = {
  caseNotFound: 'kasus_tidak_ditemukan',
  decisionClosed: 'komite_sudah_selesai',
  malformedBody: 'permintaan_cacat',
} as const

export type KomiteInboxErrorCode =
  (typeof KomiteInboxErrorCode)[keyof typeof KomiteInboxErrorCode]
