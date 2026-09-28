/**
 * Bentuk data modul Monitoring SLINK OJK (`MENU_ID 78`).
 *
 * Nama field mengikuti kontrak JSON backend — berbahasa Indonesia, dan itu termasuk
 * pengecualian `D-80`: ia kontrak yang dibaca kedua sisi, bukan nama internal.
 */

/** Kode segmen SLIK OJK. */
export type KodeSegmen = 'D01' | 'F06'

/**
 * Satu kolom grid, datang dari katalog backend.
 *
 * # Kenapa kolomnya DIKIRIM, bukan ditulis di sini
 *
 * Segmen F06 punya 38 kolom dan segmen D01 punya 20. Menuliskannya di TypeScript berarti
 * 58 definisi yang harus berubah bersamaan dengan katalog Go setiap kali satu kolom
 * bergeser — dan yang tertinggal tidak menghasilkan galat apa pun, hanya kolom laporan
 * regulator yang diam-diam salah.
 */
export type Kolom = {
  kunci: string
  judul: string

  /**
   * Kolomnya punya sumber data.
   *
   * `false` berarti kolomnya ADA di layar Pega tetapi TIDAK punya sumber di export —
   * 30 dari 38 kolom segmen F06 begitu keadaannya. Layar menandainya, supaya sel kosong
   * tidak salah dibaca sebagai "datanya memang kosong".
   */
  tersedia: boolean

  /** Nilainya angka; sel diratakan ke kanan. */
  angka: boolean

  /** Nama properti Pega yang dulu mengisinya; dipakai sebagai keterangan kolom. */
  properti_lama?: string
}

/** Satu segmen beserta katalog kolomnya. */
export type Segmen = {
  kode: KodeSegmen
  label: string
  jumlah_kolom: number
  jumlah_kolom_tersedia: number
  kolom: Kolom[]
}

/** Satu pilihan dropdown "Business Name". */
export type PilihanBisnis = {
  nilai: string
  label: string

  /**
   * Keterangan arti pilihannya.
   *
   * Terisi pada "SURETY BOND", yang MENIADAKAN Asuransi Kredit alih-alih memilih Surety
   * Bond. Tanpa keterangan, pengguna membaca hasilnya sebagai daftar Surety Bond — dan
   * salah membaca laporan regulator.
   */
  keterangan?: string
}

/** Keterangan layar: kedua segmen beserta katalognya. */
export type Keterangan = {
  segmen: Segmen[]
  business_name: PilihanBisnis[]

  /** Nama field kunci baris di dalam setiap baris. */
  kunci_baris: string
}

/**
 * Satu baris grid.
 *
 * Peta `kunci kolom -> teks`. Nilainya SUDAH berupa teks siap tampil: pemformatan tanggal
 * dan angka dilakukan backend, sehingga layar dan berkas CSV tidak akan pernah
 * menampilkan bentuk berbeda untuk nilai yang sama.
 */
export type Baris = Record<string, string>

/** Keterangan halaman. */
export type MetaHalaman = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban tombol "Cari Data". */
export type HasilPencarian = {
  segmen: Segmen
  baris: Baris[]
  meta: MetaHalaman
}

/**
 * Isian penyaring layar.
 *
 * # Kedua nama tanggal SENGAJA mempertahankan alias Pega
 *
 * Keputusan Work Owner 2026-09-26. Namanya MENYESATKAN dan itu disadari:
 *
 *   date_of_loss             -> isian berlabel "Dari"   (batas bawah tanggal registrasi)
 *   date_of_request_document -> isian berlabel "Sampai" (batas atas tanggal registrasi)
 *
 * Tidak satu pun berhubungan dengan Tanggal Kejadian maupun Tanggal Terima Dokumen.
 * Namanya dipertahankan supaya penelusuran ke `GetTempDataD01` tetap langsung; artinya
 * dijelaskan di setiap tempat ia muncul.
 */
export type Penyaring = {
  segmen: KodeSegmen
  business_name: string

  /** Isian "Tipe Generate" — segmen D01 saja. Lihat catatan pada TipeGenerate. */
  tipe_generate: string

  /** Isian "Dari" — batas bawah tanggal registrasi, berbentuk `YYYY-MM-DD`. */
  date_of_loss: string

  /** Isian "Sampai" — batas atas tanggal registrasi, berbentuk `YYYY-MM-DD`. */
  date_of_request_document: string
}

/** Penyaring kosong — keadaan awal layar. */
export const penyaringKosong: Penyaring = {
  segmen: 'D01',
  business_name: '',
  tipe_generate: '',
  date_of_loss: '',
  date_of_request_document: '',
}
