/**
 * Bentuk jawaban API modul Inbox PLA, DLA, Pre DLA (`MENU_ID 44`).
 *
 * Nama field mengikuti kontrak yang dikirim backend — berbahasa Indonesia, sesuai
 * pengecualian `D-80` untuk nama field JSON. Nama tipe dan propertinya di TypeScript
 * tetap berbahasa Inggris hanya bila ia bukan bagian kontrak; di sini seluruhnya bagian
 * kontrak, sehingga seluruhnya mengikuti nama di kawat.
 */

/** Satu kolom grid, dikirim server bersama daftarnya. */
export type Kolom = {
  kunci: string
  judul: string
  /** Kolom tanggal; layar memformatnya sebagai tanggal WIB. */
  tanggal: boolean
}

/** Satu daftar beserta kolom dan penyaringnya. */
export type Daftar = {
  kode: string
  nama: string
  keterangan: string
  kolom: Kolom[]

  /** Kolom grid rincian. Kosong bila daftar ini tidak punya. */
  kolom_rincian: Kolom[]

  /**
   * Menyatakan daftar ini menggambar grid rincian.
   *
   * Dikirim TERPISAH dari panjang `kolom_rincian` supaya layar tidak menyembunyikan
   * gridnya hanya karena kolomnya kebetulan belum terisi.
   */
  punya_rincian: boolean

  /** Kolom panel "Print Pre DLA". Kosong bila daftar ini tidak punya panelnya. */
  kolom_cetak: Kolom[]

  /**
   * Menyatakan setiap BARIS daftar ini punya tombol "Print Pre DLA".
   *
   * Tab Pre DLA saja. Ia dikirim server, bukan disimpulkan layar dari kode daftarnya:
   * inventaris tombol adalah hasil pembacaan export, sama halnya dengan daftar kolom.
   */
  punya_cetak: boolean

  /**
   * Judul tombol pada kolom aksi tiap baris.
   *
   * "Rincian" pada PLA dan DLA, "Print Pre DLA" pada Pre DLA — karena yang dibuka pun
   * berbeda: dua yang pertama membuka grid rincian di bawah antrean, yang ketiga membuka
   * panel tersendiri.
   *
   * OPSIONAL, dan itu bukan kelonggaran kontrak melainkan pengakuan atas kenyataan:
   * peladen yang berjalan bisa lebih tua daripada berkas layar, dan medan ini baru ada
   * sejak panel "Print Pre DLA" dibangun. Lihat `judulTombolBaris`.
   */
  label_aksi_baris?: string

  label_pencarian: string

  /**
   * Judul rentang tanggal yang disaring daftar ini.
   *
   * BERBEDA per daftar: tanggal PLA, tanggal DLA, atau tanggal Pre-DLA. Di Pega kotaknya
   * hanya berjudul "Dari"/"Sampai", dan pengguna harus menebak tanggal apa yang sedang ia
   * batasi.
   */
  label_tanggal: string
}

/** Satu baris antrean. */
export type Baris = {
  /**
   * Kunci objek kerja Pega, berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`.
   *
   * Tidak digambar sebagai kolom; tombol "Rincian" yang memakainya. Ia memuat SPASI,
   * sehingga wajib dikodekan saat masuk ke alamat.
   */
  kunci_klaim: string

  no_klaim: string
  no_polis: string
  nama_tertanggung: string
  tanggal_register: string
  tanggal_kejadian: string
  pic_teknik: string

  /**
   * Tanggal dokumen terbaru yang belum terkirim.
   *
   * KOSONG pada klaim yang seluruh dokumennya ber-`ISKIRIM = '0'` — kejanggalan Pega yang
   * sengaja dibawa. Sel kosong di kolom ini bukan kerusakan.
   */
  tanggal_advice: string
}

/** Satu baris grid rincian. */
export type Dokumen = {
  no_advice: string
  reasuradur: string
  tipe: string
  /** Hanya terisi pada PLA. */
  revisi: string
  tanggal_dokumen: string
  /** `ISKIRIM` apa adanya: kosong, `"0"`, atau `"1"`. */
  terkirim: string
  tanggal_kirim: string
  tanggal_terima: string
  catatan: string
  email: string
  /** Hanya terisi pada DLA. */
  no_akseptasi: string

  /**
   * Menyatakan baris ini digambar bertombol **"Send"**.
   *
   * Dihitung PELADEN, dan itu bukan kerapian: syaratnya berbeda antara kedua tab —
   * `.MARKETING != '1'` pada PLA, `.MARKETING == ''` pada DLA — dan keduanya hasil
   * pembacaan section Pega. Layar yang menyimpulkannya sendiri dari `terkirim` akan
   * menyamakan keduanya tanpa satu pun galat.
   *
   * OPSIONAL, dan alasannya sama dengan `label_aksi_baris`: peladen yang berjalan bisa
   * lebih tua daripada berkas layar. Tanpa cadangan, SELURUH tombol Send lenyap pada
   * peladen lama — kegagalan yang tidak menghasilkan galat apa pun, hanya tombol yang
   * tidak ada. Lihat `bolehKirim`.
   */
  dapat_dikirim?: boolean
}

/** Satu baris panel "Print Pre DLA". */
export type DokumenPreDLA = {
  no_advice: string
  reasuradur: string
  tipe: string
  /** Tanggal KIRIM, bukan tanggal Pre-DLA. Aliasnya di Pega (`TglDLA`) menyesatkan. */
  tanggal_kirim: string
  /** `NVL(ISKIRIM, '0')`: selalu `"0"` atau `"1"`, tidak pernah kosong. */
  terkirim: string
  /**
   * Kunci berkas lampirannya.
   *
   * Tidak digambar sebagai kolom. Tombol unduh di dalam panel Pega memakainya, dan
   * tombol itu belum dibangun di sini.
   */
  kunci_lampiran: string
}

export type Paginasi = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Penyaring yang BENAR-BENAR dipakai server. */
export type Penyaring = {
  cari: string
  dari: string
  sampai: string
}

export type MetadataResponse = {
  daftar: Daftar[]
  daftar_bawaan: string
  portal: string
}

export type ListResponse = {
  daftar: Daftar
  baris: Baris[]
  paginasi: Paginasi
  penyaring: Penyaring
  portal: string
}

export type DokumenResponse = {
  daftar: Daftar
  kunci_klaim: string
  baris: Dokumen[]
  portal: string
}

export type CetakResponse = {
  daftar: Daftar
  kunci_klaim: string
  baris: DokumenPreDLA[]
  portal: string
}

/** Isian panel pencarian sebagaimana dipegang layar. */
export type FormPencarian = {
  cari: string
  dari: string
  sampai: string
}
