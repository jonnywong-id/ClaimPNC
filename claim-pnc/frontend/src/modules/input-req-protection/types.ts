/**
 * Bentuk data layar Input Req Protection.
 *
 * Nama field mengikuti kontrak JSON backend apa adanya — berbahasa Indonesia, sesuai lima
 * pengecualian `D-80`. Nama tipe dan nama variabel di sekitarnya berbahasa Inggris.
 */

/** Satu baris pada daftar: sebuah permintaan proteksi yang belum diakseptasi. */
export type Protection = {
  /**
   * Kolom "No Proteksi".
   *
   * Dua bentuk hidup berdampingan: `OPC-XXX` warisan Pega dan `OPCN.YY.xxxx` terbitan
   * aplikasi ini. Layar menampilkan keduanya apa adanya.
   */
  nomor_proteksi: string

  nomor_polis: string // kolom "No Polis"

  /** Kolom "No Klaim". Kosong berarti proteksi belum ditautkan ke klaim mana pun. */
  nomor_klaim: string

  /**
   * Kolom "Tipe Proteksi" — KODE (`PROTECTION_TYPE_ID`).
   *
   * Tetap dikirim meski namanya sudah ada: kodenya yang dipakai form saat menyunting, dan
   * kodenya yang menentukan antrean akseptasi.
   */
  tipe_proteksi: string

  /**
   * Nama tipe dari master `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
   *
   * KOSONG bila kodenya tidak terdaftar di master — lihat `protectionTypeLabel`.
   */
  nama_tipe_proteksi: string

  /** YYYY-MM-DD dalam WIB; backend yang mengonversinya dari UTC. */
  tanggal_proteksi: string // kolom "Tanggal Proteksi Dibuat"

  keterangan: string // kolom "Keterangan"
  user_create: string // kolom "User Create"

  /**
   * Menggantikan `pyDisabledWhen` pada layar lama: tautan baris mati begitu nomor klaim
   * terisi.
   *
   * Dihitung di SERVER. Layar hanya mematikan tautannya — aturannya tidak diulang di sini,
   * karena aturan yang hidup di dua tempat akan berbeda cepat atau lambat.
   */
  dapat_disunting: boolean

  /** Baris ini akan masuk antrean akseptasi PREMI (`tipe_proteksi === "2"`). */
  premi: boolean
}

/** Isi panel "Detail Perubahan"; terisi hanya untuk tipe `7` dan `8`. */
export type ChangeDetail = {
  /** Keduanya dipakai tipe `7` (Perubahan DOL). Kosong berarti tidak diisi. */
  dol_sebelum: string
  dol_baru: string

  /** Keduanya dipakai tipe `8` (Perubahan Cause Of Loss). */
  penyebab_kerugian: string
  penyebab_kerugian_master: string

  nama_objek: string
  nama_cabang: string
}

/** Satu permintaan proteksi beserta isian formnya. */
export type ProtectionDetail = Protection & {
  /**
   * **ClaimID**, kolom `ID_CLAIM`. HANYA DIBACA.
   *
   * Pada baris baru nilainya SAMA dengan `nomor_klaim` — server menurunkannya, dan form
   * tidak mengirimkannya (Work Owner, 2026-09-24).
   *
   * Pada baris WARISAN ia berbeda: Pega menyimpan kunci teknisnya di sana,
   * `ASM-FW-GCNMFW-WORK PNC-xxxx`. Form menampilkannya hanya ketika berbeda.
   */
  referensi_klaim: string

  detail_perubahan: ChangeDetail
}

export type ProtectionListResponse = {
  proteksi: Protection[]
  total: number
}

/**
 * Isian form yang dikirim saat menyimpan.
 *
 * # Yang TIDAK ada di sini, dan kenapa
 *
 * `nomor_polis`, `nama_objek`, `nama_cabang`, `dol_sebelum`, dan `penyebab_kerugian` semuanya
 * DITURUNKAN dari klaim — `Activity/OpenProtection-Act.xml` mengisinya saat klaim dicari,
 * dan di sistem lama pun keenamnya tidak pernah diketik.
 *
 * Mengirimnya dari sini berarti mempercayai peramban untuk menyatakan keadaan klaim yang
 * bukan miliknya. Server menurunkannya ULANG saat menyimpan, sehingga apa pun yang dikirim
 * dari sini tidak akan terpakai.
 */
export type ProtectionFields = {
  nomor_klaim: string
  tipe_proteksi: string
  keterangan: string
  detail_perubahan: ChangeRequestFields
}

/** Bagian panel Detail Perubahan yang benar-benar DIPILIH pengguna. */
export type ChangeRequestFields = {
  /** "Next Date Of Loss" — wajib untuk tipe `7`. */
  dol_baru: string

  /** "Next Cause Of Loss" — wajib untuk tipe `8`. */
  penyebab_kerugian_baru: string
}

/** Hasil pencarian klaim; seluruhnya HANYA DIBACA di layar. */
export type ClaimLookup = {
  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string

  /** "Current Date Of Loss". Kosong bila klaimnya tidak punya DOL tercatat. */
  dol: string

  /** "Cause Of Loss Dipilih". */
  penyebab_kerugian: string

  nama_objek: string
  nama_cabang: string
}

/** Penyaring daftar. Seluruhnya opsional; kosong berarti tidak menyaring. */
export type ProtectionFilter = {
  search?: string
  offset?: number
}

/**
 * Kode tipe proteksi yang ARTINYA terbukti dari export Pega.
 *
 * Ketiganya diturunkan dari perilaku yang dapat dilihat, bukan dari daftar nilai:
 *
 *   `2` dipisahkan menjadi antrean tersendiri milik peran penagihan premi
 *       (`InboxOpenProtection2_RD_collection` menyaring `= "2"`)
 *   `7` memunculkan panel "Detail Perubahan DOL"
 *       (`Section/InputProtectionSection-Section.xml:2728`)
 *   `8` memunculkan panel "Detail Perubahan Cause Of Loss" (`:3638`)
 */
export const TYPE_PREMIUM = '2'
export const TYPE_CHANGE_LOSS_DATE = '7'
export const TYPE_CHANGE_CAUSE_OF_LOSS = '8'

/**
 * Satu pilihan tipe proteksi, dari master `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
 *
 * Fieldnya `kode` dan `nama`, bukan `id` dan `label`: keduanya menyebut apa yang dilihat
 * pengguna, dan `kode` itulah yang tampil di layar ketika namanya tidak ada.
 */
export type ProtectionType = {
  kode: string
  nama: string
}

/** Tanggapan endpoint master tipe. */
export type ProtectionTypeListResponse = {
  tipe: ProtectionType[]
}

/**
 * Menampilkan nama tipe proteksi, atau kodenya apa adanya bila namanya tidak ada.
 *
 * # Kenapa daftarnya TIDAK lagi ada di berkas ini
 *
 * Sampai 2026-09-24 berkas ini memuat `PROTECTION_TYPE_LABEL` berisi tiga label yang
 * artinya terbukti dari export. Lima nilai lain dipakai tanpa label yang diketahui
 * (`R-16`), dan kodenya ditampilkan apa adanya.
 *
 * Master `M_CLAIM_PROTECTION_TYPE` kemudian diterima berisi KESEMBILAN tipe beserta
 * namanya — termasuk `9` "Nama Rekening Tidak Sesuai", yang dipakai 25 baris produksi dan
 * NOL kemunculan di export. Daftar tebakan mana pun akan melewatkannya.
 *
 * Karena itu nama datang dari server, dan berkas ini tidak lagi memetakan kode ke nama.
 * Yang tersisa hanyalah aturan tampilannya: nama bila ada, kode bila tidak.
 *
 * # Kenapa kode, bukan tanda hubung
 *
 * Kode tipe yang tidak terdaftar adalah data yang perlu ditanyakan manusia. Menggantinya
 * dengan tanda hubung membuatnya tidak pernah ditanyakan.
 */
export function protectionTypeLabel(code: string, name?: string): string {
  const nama = name?.trim()
  return nama ? nama : code
}
