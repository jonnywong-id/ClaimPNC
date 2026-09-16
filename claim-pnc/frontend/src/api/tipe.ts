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
} as const

export type KodeGalat = (typeof KodeGalat)[keyof typeof KodeGalat]
