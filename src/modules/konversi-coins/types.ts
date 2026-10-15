/** Keadaan satu koneksi — tanpa kata sandi. */
export type ConnectionInfo = {
  alamat: string
  lengkap: boolean
  kurang: string[]
}

/** Jawaban GET /api/konversi-coins/keterangan. */
export type Info = {
  live: ConnectionInfo
  test: ConnectionInfo
  /** Daftar bawaan dari KONVERSI_POLIS. */
  polis: string[]
  maks_polis: number
  galat?: string
}

export type PolicyStatus = 'berhasil' | 'tidak_ditemukan' | 'gagal'

export type PolicyResult = {
  nomor_polis: string
  status: PolicyStatus
  versi_prodke: string[] | null
  /** Jumlah versi (PRODKE) yang dokumennya tidak membawa CoinsList. */
  versi_tanpa_coins: number
  baris_dihapus: number
  baris_ditulis: number
  peringatan?: string[]
  galat?: string
}

/** Jawaban POST /api/konversi-coins/jalankan. */
export type Report = {
  uji_coba: boolean
  mulai: string
  selesai: string
  polis: PolicyResult[]
  berhasil: number
  tidak_ditemukan: number
  gagal: number
  total_baris: number
}
