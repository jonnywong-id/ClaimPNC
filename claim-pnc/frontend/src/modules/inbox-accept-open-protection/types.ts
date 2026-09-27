/**
 * Bentuk data layar Inbox Accept Open Protection.
 *
 * Nama field mengikuti kontrak JSON backend apa adanya — berbahasa Indonesia, sesuai lima
 * pengecualian `D-80`.
 */

/**
 * Antrean akseptasi.
 *
 * Layar lama memisahkannya lewat SYARAT TAMPIL grid, bukan lewat parameter: grid PREMI
 * hanya muncul bagi access group `GCNMFW:PncCollection`, grid NON PREMI bagi peran lain.
 *
 * Di sini ia pilihan tab, dan itu KELEMAHAN YANG DISADARI — sampai tabel peran
 * (`TKT-F3-004`) dapat diisi, tidak ada yang mencegah pengguna membuka antrean yang bukan
 * haknya. Begitu peran tersedia, pemilihannya pindah ke server.
 */
export type Queue = 'premi' | 'non-premi'

/** Satu baris pada antrean akseptasi. */
export type Protection = {
  nomor_proteksi: string // kolom "No Proteksi"
  nomor_polis: string // kolom "No Polis"
  nomor_klaim: string // kolom "No Klaim"

  /** Kolom "Tipe Proteksi" — KODE (`PROTECTION_TYPE_ID`). Ia yang menentukan antrean. */
  tipe_proteksi: string

  /**
   * Nama tipe dari master `POOLDATA.M_CLAIM_PROTECTION_TYPE`.
   *
   * KOSONG bila kodenya tidak terdaftar — lihat `protectionTypeLabel`.
   */
  nama_tipe_proteksi: string

  /** YYYY-MM-DD dalam WIB; backend yang mengonversinya dari UTC. */
  tanggal_proteksi: string // kolom "Tanggal Proteksi Dibuat"

  keterangan: string // kolom "Keterangan"
  user_create: string // kolom "User Create"

  /** Antrean tempat baris ini berada; diturunkan server dari tipe proteksi. */
  antrean: Queue
}

/** Satu proteksi beserta data polis dan keadaan akseptasinya. */
export type ProtectionDetail = Protection & {
  nama_tertanggung: string
  polis_mulai: string
  polis_akhir: string

  /** Kode yang sama dengan yang disimpan: `""` belum, `"1"` disetujui, `"2"` ditolak. */
  status_akseptasi: string

  tanggal_akseptasi: string
  diaksep_oleh: string

  /**
   * Diturunkan server supaya layar tidak menafsirkan kode status sendiri.
   *
   * Tombol setuju dan tolak hanya muncul ketika ia bernilai `true`.
   */
  menunggu_keputusan: boolean
}

export type ProtectionListResponse = {
  proteksi: Protection[]
  total: number
  antrean: Queue
}

/** Keputusan yang dikirim saat mengakseptasi. */
export type Decision = 'setuju' | 'tolak'

export type ProtectionFilter = {
  queue: Queue
  search?: string
  offset?: number
}

/**
 * Menampilkan nama tipe proteksi, atau kodenya apa adanya bila namanya tidak ada.
 *
 * # Kenapa daftar label di berkas ini DIHAPUS
 *
 * Sampai 2026-09-24 berkas ini memuat salinan tiga label yang artinya terbukti dari export,
 * SENGAJA diulang dari modul sebelah karena fitur tidak boleh mengimpor dari fitur lain
 * (`08-TECHNICAL-STRATEGY.md` §3 aturan 1).
 *
 * Master `M_CLAIM_PROTECTION_TYPE` kemudian diterima berisi kesembilan tipe beserta namanya,
 * dan backend mengirimkannya bersama setiap baris. Duplikasinya hilang dengan sendirinya —
 * bukan karena aturannya dilonggarkan, melainkan karena yang diduplikasi sudah tidak ada.
 *
 * # Kenapa kode, bukan tanda hubung
 *
 * Petugas yang menyetujui pembukaan proteksi berhak tahu bahwa tipe yang dihadapinya tidak
 * dikenal sistem. Tanda hubung menyembunyikan itu.
 */
export function protectionTypeLabel(code: string, name?: string): string {
  const nama = name?.trim()
  return nama ? nama : code
}
