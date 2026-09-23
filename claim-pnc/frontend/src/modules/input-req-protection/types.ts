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
   * Kolom "Tipe Proteksi" — KODE, bukan label.
   *
   * Label untuk sebagian besar nilai tidak diketahui: daftarnya tinggal di rule Property
   * Pega yang tidak ikut diekspor (`R-16`). Yang terbukti hanya tiga — lihat
   * `PROTECTION_TYPE_LABEL`.
   */
  tipe_proteksi: string

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
   * Klaim yang benar-benar DITEMUKAN, bukan yang diketik.
   *
   * Kosongnya berarti pengguna baru mengetik nomor klaim tanpa mencarinya, dan backend
   * menolak penyimpanan dengan pesan "Silakan Tulis dan Cari Ulang No Klaim".
   */
  referensi_klaim: string

  detail_perubahan: ChangeDetail
}

export type ProtectionListResponse = {
  proteksi: Protection[]
  total: number
}

/** Isian form yang dikirim saat menyimpan. */
export type ProtectionFields = {
  nomor_polis: string
  nomor_klaim: string
  referensi_klaim: string
  tipe_proteksi: string
  keterangan: string
  detail_perubahan: ChangeDetail
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
 * Label tipe proteksi yang boleh ditampilkan.
 *
 * # Kenapa daftarnya pendek, dan kenapa itu disengaja
 *
 * Nilai `1`, `3`, `4`, `5`, dan `6` juga dipakai di sistem lama, tetapi LABELNYA TIDAK
 * DIKETAHUI — daftarnya ada di rule Property yang tidak ikut diekspor (`R-16`).
 *
 * Kode yang tidak ada di sini ditampilkan APA ADANYA. Menebak labelnya akan lebih buruk
 * daripada menampilkan kode: kode mentah di layar segera ditanyakan pengguna, label yang
 * salah diterima begitu saja.
 *
 * Begitu Work Owner menyerahkan daftarnya, isian ini pindah ke master data (`F-4`) dan
 * konstanta ini dibuang.
 */
export const PROTECTION_TYPE_LABEL: Record<string, string> = {
  [TYPE_PREMIUM]: 'Proteksi Klaim PREMI',
  [TYPE_CHANGE_LOSS_DATE]: 'Perubahan DOL',
  [TYPE_CHANGE_CAUSE_OF_LOSS]: 'Perubahan Cause Of Loss',
}

/** Menampilkan label tipe proteksi bila diketahui, atau kodenya apa adanya bila tidak. */
export function protectionTypeLabel(code: string): string {
  return PROTECTION_TYPE_LABEL[code] ?? code
}
