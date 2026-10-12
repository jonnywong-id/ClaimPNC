/** Keadaan satu koneksi — tanpa kata sandi. */
export type ConnectionInfo = {
  alamat: string
  lengkap: boolean
  kurang: string[]
}

/** Jawaban GET /api/konversi-object-item-fire/keterangan. */
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
  jumlah_objek: number
  objek_diperbarui: number
  objek_tidak_ada_di_test?: string[]
  item_dihapus: number
  item_ditulis: number
  dokumen_kosong: number
  peringatan?: string[]
  galat?: string
}

/** Pembatas objek yang dikonversi. Isian kosong berarti tanpa batas. */
export type ObjectFilter = {
  prodke: string
  indexobject: string
}

/** Jawaban POST /api/konversi-object-item-fire/jalankan. */
export type Report = {
  uji_coba: boolean
  /** Filter yang dipakai; tidak ada berarti seluruh PRODKE / seluruh INDEXOBJECT. */
  filter_prodke?: string | undefined
  filter_indexobject?: string | undefined
  mulai: string
  selesai: string
  polis: PolicyResult[]
  berhasil: number
  tidak_ditemukan: number
  gagal: number
  total_item: number
}
