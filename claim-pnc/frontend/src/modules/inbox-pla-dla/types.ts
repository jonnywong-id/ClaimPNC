/**
 * Bentuk jawaban API modul Inbox PLA DLA (`MENU_ID 45`) — layar milik REASURADUR.
 *
 * Nama field mengikuti kontrak yang dikirim backend — berbahasa Indonesia, sesuai
 * pengecualian `D-80`.
 */

/** Satu kolom grid, dikirim server bersama daftarnya. */
export type Kolom = {
  kunci: string
  judul: string
  /** Kolom tanggal; layar memformatnya sebagai tanggal WIB. */
  tanggal: boolean
}

/** Satu daftar beserta kolom dan keterangannya. */
export type Daftar = {
  kode: string
  nama: string
  keterangan: string
  kolom: Kolom[]
}

/** Satu baris daftar klaim. */
export type Baris = {
  /** Kunci objek kerja Pega. Tidak digambar sebagai kolom. */
  kunci_klaim: string

  no_klaim: string
  no_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  tanggal_register: string
  tanggal_kejadian: string
  pic_teknik: string

  /**
   * Kode status yang DIGAMBAR — sudah termasuk penggantian menjadi `1139` pada daftar
   * DLA untuk klaim yang menunggu penutupan.
   */
  kode_status: string

  /** Arti kodenya. Kosong bila kodenya tidak ada di master status. */
  status: string

  no_pla: string
  catatan_tutup: string
}

/** Satu baris tabel ringkas "Status / Jumlah". */
export type RingkasStatus = {
  kode_status: string
  status: string
  jumlah: number
}

/** Satu baris grid "DATA PLA DLA XOL KLAIM". */
export type BarisXOL = {
  tahun: string
  penyebab_kerugian: string
  /** `"PLA"` atau `"DLA"`. */
  jenis: string
  tanggal_terakhir: string
}

export type Paginasi = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

export type MetadataResponse = {
  daftar: Daftar[]
  daftar_bawaan: string
  kolom_xol: Kolom[]
  selisih_terencana: string[]
  portal: string
}

export type ListResponse = {
  daftar: Daftar
  baris: Baris[]
  paginasi: Paginasi
  cari: string
  portal: string
}

export type RingkasResponse = {
  baris: RingkasStatus[]
  portal: string
}

export type XOLResponse = {
  baris: BarisXOL[]
  portal: string
}
