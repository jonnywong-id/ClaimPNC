/** Keadaan satu koneksi — tanpa kata sandi. */
export type InfoKoneksi = {
  alamat: string
  lengkap: boolean
  kurang: string[]
}

/** Jawaban GET /api/konversi-coverage/keterangan. */
export type Keterangan = {
  live: InfoKoneksi
  test: InfoKoneksi
  /** Daftar bawaan dari KONVERSI_POLIS. */
  polis: string[]
  maks_polis: number
  galat?: string
}

export type StatusPolis = 'berhasil' | 'tidak_ditemukan' | 'gagal'

export type HasilPolis = {
  nomor_polis: string
  status: StatusPolis
  lini: string[] | null
  versi_prodke: string[] | null
  jumlah_objek: number
  objek_diperbarui: number
  objek_tidak_ada_di_test?: string[]
  coverage_dihapus: number
  coverage_ditulis: number
  spreading_dihapus: number
  spreading_ditulis: number
  dokumen_kosong: number
  peringatan?: string[]
  galat?: string
}

/** Jawaban POST /api/konversi-coverage/jalankan. */
export type Laporan = {
  uji_coba: boolean
  mulai: string
  selesai: string
  polis: HasilPolis[]
  berhasil: number
  tidak_ditemukan: number
  gagal: number
  total_coverage: number
  total_spreading: number
}
