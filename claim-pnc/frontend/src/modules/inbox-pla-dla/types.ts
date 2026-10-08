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

/**
 * Bentuk sebuah tampilan.
 *
 * Layar Pega punya TUJUH tampilan yang dikendalikan satu nilai (`TempView.CityID`), dan
 * tidak semuanya menggambar hal yang sama: enam menggambar daftar klaim, yang ketujuh
 * menggambar ringkasan XOL dengan kolom dan sumber yang berbeda.
 *
 * Penandanya datang dari SERVER. Layar tidak menyimpulkannya sendiri dari kode tab —
 * kesimpulan yang disalin ke layar akan tertinggal saat tampilannya bertambah.
 */
export type JenisTampilan = 'daftar-klaim' | 'xol'

/**
 * Tabel asal sebuah daftar klaim.
 *
 * Dipakai menjelaskan daftar yang KOSONG: "belum ada pemberitahuan" dan "belum ada
 * komunikasi" adalah dua sebab berbeda. Ia datang dari server, bukan disimpulkan dari
 * awalan kode tab — kesimpulan itu pecah diam-diam saat kodenya berubah.
 *
 * Kosong pada tampilan XOL.
 */
export type SumberDaftar = 'pemberitahuan' | 'komunikasi' | ''

/** Satu tampilan beserta kolom dan keterangannya. */
export type Daftar = {
  kode: string
  nama: string
  keterangan: string
  jenis: JenisTampilan
  sumber: SumberDaftar
  kolom: Kolom[]

  /** Barisnya punya tombol "Detail Claim". */
  punya_rincian: boolean
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

/* ==========================================================================
 * LAYAR RINCIAN — tombol "Detail Claim"
 * ========================================================================== */

/** Keterangan klaim di kepala layar rincian. */
export type KlaimRincian = {
  kunci_klaim: string
  no_klaim: string
  no_polis: string
  nama_tertanggung: string
  nama_bisnis: string
  tanggal_register: string
  tanggal_kejadian: string
  pic_teknik: string
  kode_status: string
  status: string
}

/** Satu baris grid PLA atau DLA pada layar rincian. */
export type BarisPemberitahuan = {
  /** `"PLA"` atau `"DLA"` — dipakai mengambil dokumennya. */
  jenis: string

  nomor: string
  tipe: string

  /**
   * Nilai pemberitahuan, dikirim sebagai TEKS.
   *
   * Ia tidak dihitung di layar — hanya digambar — dan mengubahnya menjadi `number` akan
   * memperkenalkan pembulatan pada nilai uang yang `D-51` larang justru untuk keadaan
   * seperti ini.
   */
  nilai: string

  no_akseptasi: string
  tanggal_dokumen: string
  tanggal_kirim: string
}

/** Satu baris grid dokumen. */
export type BarisDokumen = {
  id: string
  jenis_dokumen: string
  kategori_dokumen: string
  nama: string
}

/** Satu baris grid riwayat komunikasi. */
export type BarisKomunikasi = {
  id: string
  tanggal: string
  pengirim: string
  pesan: string
  balasan: string
  nama_pembalas: string
  tanggal_balasan: string
  sudah_dijawab: boolean

  /**
   * Pemanggil boleh membalas percakapan ini.
   *
   * Dihitung PELADEN. Layar tidak menyimpulkannya sendiri dari `sudah_dijawab`: keduanya
   * DAPAT berbeda pada data lama, dan kesimpulan layar akan menggambar tombol balas pada
   * percakapan yang permintaannya justru akan ditolak.
   */
  boleh_dibalas: boolean
}

export type DetailResponse = {
  klaim: KlaimRincian
  pla: BarisPemberitahuan[]
  dla: BarisPemberitahuan[]
  komunikasi: BarisKomunikasi[]

  kolom_pla: Kolom[]
  kolom_dla: Kolom[]
  kolom_dokumen: Kolom[]
  kolom_komunikasi: Kolom[]

  portal: string
}

export type DokumenResponse = {
  baris: BarisDokumen[]
  kolom: Kolom[]
  portal: string
}

export type BalasResponse = {
  pesan: string
  portal: string
}
