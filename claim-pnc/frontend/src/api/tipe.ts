// Bentuk data yang dipertukarkan dengan backend.
//
// Tipe di sini adalah cerminan dto Go di internal/*/http. Bila salah satu berubah, yang
// lain wajib ikut berubah — pemeriksaan tipe TypeScript adalah jaring pengaman antara
// layar dan API.

/** Dua populasi pengguna, diverifikasi lewat dua jalur berbeda. */
export const JenisPengguna = {
  /** Diverifikasi ke API HCC/HCQ. Identitasnya NIK. */
  karyawan: 'KARYAWAN',
  /** Broker dan surveyor independen, diverifikasi ke POOLDATA.M_LOGIN_PNC. */
  nonKaryawan: 'NON_KARYAWAN',
} as const

export type JenisPengguna = (typeof JenisPengguna)[keyof typeof JenisPengguna]

export type Pengguna = {
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

export type ResponsMasuk = {
  token: string
  tipe_token: string
  berlaku_sampai: string
  pengguna: Pengguna
}

export type ResponsSaya = {
  pengguna: Pengguna
  berlaku_sampai: string
}

export type ResponsPerpanjang = {
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

export type ResponsDaftarPortal = {
  portal: Portal[]
  /** Alias portal yang basis datanya melayani sesi dan lookup pra-login. */
  utama: string
}

// ── Master Status Progres 1 ──────────────────────────────────────────────────────
//
// Cerminan dto di internal/statusprogres/http. Menggantikan layar Pega
// `Harness/StatusProgress-Harness.xml` atas tabel POOLDATA.GCNM_MST_PROGRESS_KLAIM.

/**
 * Satu baris master status progres tingkat 1.
 *
 * Nama field di sini sudah dinamai ulang mengikuti `D-19`; kueri Pega mengaliaskan
 * ketiga kolomnya ke nama yang tidak mencerminkan isi (`CaseID`, `City`, `CityID`).
 */
export type StatusProgres = {
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
export type PosisiKlaim = {
  kode: string
  nama: string
}

export type ResponsDaftarStatusProgres = {
  status_progres: StatusProgres[]
  /** Entitas yang benar-benar menjawab permintaan ini. */
  portal: string
}

export type ResponsSatuStatusProgres = {
  status_progres: StatusProgres
  portal: string
}

export type ResponsDaftarPosisiKlaim = {
  posisi: PosisiKlaim[]
}

/** Badan permintaan penambahan dan penyuntingan. */
export type IsianStatusProgres = {
  nama: string
  kode_posisi: string
}

/**
 * Satu isian yang ditolak server.
 *
 * Backend mengirim SELURUH pelanggaran sekaligus, bukan yang pertama saja — meniru
 * perilaku Pega yang menampilkan semua pesan bersamaan (P-5). `kolom` memakai nama
 * isian, sehingga layar dapat menyorot isian yang salah.
 */
export type DetailGalat = {
  kolom: string
  pesan: string
}

/**
 * Kode galat yang dikirim backend.
 *
 * Layar membedakan jenis galat lewat kode ini, TIDAK PERNAH dengan mencocokkan teks
 * pesan — teks bisa berubah kapan saja tanpa mengubah artinya.
 */
export const KodeGalat = {
  kredensialSalah: 'kredensial_salah',
  penggunaTidakAktif: 'pengguna_tidak_aktif',
  identitasPutus: 'sistem_identitas_tidak_terhubung',
  sesiTidakSah: 'sesi_tidak_sah',
  sesiKedaluwarsa: 'sesi_kedaluwarsa',
  permintaanCacat: 'permintaan_cacat',
  galatInternal: 'galat_internal',

  // Galat modul bisnis dan portal.
  validasiGagal: 'validasi_gagal',
  tidakDitemukan: 'tidak_ditemukan',
  /** Permintaan tidak menyebut portal — pengguna belum memilih entitas. */
  portalTidakDisebut: 'portal_tidak_disebut',
  portalTidakDikenal: 'portal_tidak_dikenal',
  /** Entitasnya ada, tetapi kredensial basis datanya belum diisi tim infrastruktur. */
  portalBelumSiap: 'portal_belum_siap',
} as const

export type KodeGalat = (typeof KodeGalat)[keyof typeof KodeGalat]
