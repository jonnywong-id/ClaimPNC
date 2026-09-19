// Bentuk data yang dipertukarkan dengan backend.
//
// Tipe di sini adalah cerminan dto Go di internal/*/http. Bila salah satu berubah, yang
// lain wajib ikut berubah — pemeriksaan tipe TypeScript adalah jaring pengaman antara
// layar dan API.

/** Dua populasi pengguna, diverifikasi lewat dua jalur berbeda. */
export const JenisPengguna = {
  /** Diverifikasi ke API HCC/HCQ. Identitasnya NIK. */
  karyawan: 'KARYAWAN',
  /** Broker dan surveyor independen, diverifikasi ke POOLDATA.M_LOGIN_PNC. */
  nonKaryawan: 'NON_KARYAWAN',
} as const

export type JenisPengguna = (typeof JenisPengguna)[keyof typeof JenisPengguna]

export type Pengguna = {
  /** NIK untuk karyawan, LOGIN_ID untuk non-karyawan. */
  identitas: string
  nama: string
  jenis: string
  login: string
  /** Dapat kosong: POOLDATA.M_LOGIN_PNC tidak memuat surel. */
  email: string
  /** Dapat kosong: hanya diisi HCC/HCQ. */
  perusahaan: string
}

export type ResponsMasuk = {
  token: string
  tipe_token: string
  berlaku_sampai: string
  pengguna: Pengguna
}

export type ResponsSaya = {
  pengguna: Pengguna
  berlaku_sampai: string
}

export type ResponsPerpanjang = {
  berlaku_sampai: string
}

/** Satu entitas yang dilayani aplikasi ini (ADR-0030). */
export type Portal = {
  id: string
  nama: string
  alias: string
  /**
   * Koneksi basis data portal ini sudah dapat dipakai.
   *
   * Portal yang belum siap tetap ditampilkan — ditandai, bukan disembunyikan — supaya
   * pengguna melihat seluruh entitas yang direncanakan.
   */
  siap: boolean
}

export type ResponsDaftarPortal = {
  portal: Portal[]
  /** Alias portal yang basis datanya melayani sesi dan lookup pra-login. */
  utama: string
}

/**
 * Satu baris Master Status Klaim.
 *
 * Status Klaim adalah salah satu dari empat konsep status yang `D-18` tetapkan memang
 * berbeda. Ia menjawab "klaim ini berada di keadaan bisnis apa" — Register, Claim
 * Committee, Paid, dan seterusnya. Domainnya 33 kode `1134`–`1166`.
 */
export type StatusKlaim = {
  /** Dibuat sistem saat status ditambahkan; tidak pernah berubah sesudahnya. */
  kode: string
  /** Teks yang dibaca pengguna. Di layar Pega ia berlabel "Status". */
  label: string
  /**
   * Penomoran lama `01`–`11` yang melekat pada sebelas kode pertama (`1134`–`1144`).
   * Kosong untuk 22 kode sisanya, dan tidak pernah diisi untuk status baru.
   */
  kode_lama: string
}

export type ResponsDaftarStatusKlaim = {
  status_klaim: StatusKlaim[]
  /** Datang dari server, bukan dihitung dari panjang senarai. */
  total: number
}

export type ResponsStatusKlaim = {
  status_klaim: StatusKlaim
}

/**
 * Posisi sebuah laporan klaim dalam perjalanannya menjadi klaim.
 *
 * Tahap DIHITUNG server dari isi laporan, tidak disimpan sebagai kolom — sama seperti
 * sistem lama menurunkannya dari kombinasi `PNCCASEID` dan `STATUSLOCK`. Layar TIDAK
 * menghitungnya sendiri: aturan yang hidup di dua tempat akan berselisih pada perubahan
 * berikutnya.
 */
export const ReportStage = {
  notTransferred: 'BELUM_TRANSFER',
  notRegistered: 'BELUM_REGISTRASI',
  registered: 'SUDAH_REGISTRASI',
  accepted: 'SUDAH_AKSEPTASI',
  rejected: 'DITOLAK',
} as const

export type ReportStage = (typeof ReportStage)[keyof typeof ReportStage]

/**
 * Satu laporan klaim — laporan kerugian yang masuk sebelum klaim diregistrasi.
 *
 * Menggantikan case `ASM-FW-GCNMFW-Work-ReceiveDocument`, yang di menu portal Pega
 * berjudul **"Inbox Laporan Klaim"**.
 */
export type ClaimReport = {
  nomor: string

  nama_pelapor: string
  email_pengirim: string
  telepon_pengirim: string
  nama_kurir: string
  subjek_email: string

  /** Polis dan tertanggung SEBAGAIMANA DISEBUT PELAPOR — bukan snapshot polis. */
  nomor_polis: string
  nama_tertanggung: string
  email_tertanggung: string
  kode_bisnis: string
  group_panel: string
  nomor_referensi: string

  /** Tanggal kalender, `YYYY-MM-DD`. Kosong berarti belum diisi. */
  tanggal_kejadian: string
  lokasi_kejadian: string
  kronologi: string
  rincian_kerusakan: string
  sim_pengendara: string
  /** Teks desimal, bukan angka — presisi penuh tanpa pembulatan floating point. */
  nilai_estimasi: string
  tipe_klaim: string

  jumlah_dokumen: number
  tanggal_terima_dokumen: string

  nomor_klaim: string
  ditransfer: boolean
  /** Waktu peristiwa, RFC 3339 UTC. Kosong berarti belum terjadi. */
  tanggal_transfer: string
  tanggal_registrasi: string
  alasan_belum_transfer: string
  catatan_belum_registrasi: string

  /** Dihitung server dari isi laporan. Layar tidak menghitungnya sendiri. */
  tahap: string
  tahap_label: string

  /** Dihitung server dari aturan yang sama yang ditegakkan usecase. */
  dapat_ditransfer: boolean
  dapat_diubah: boolean

  kode_cabang: string
  diinput_oleh: string
  diinput_pada: string
  diubah_pada: string
}

/** Jumlah laporan pada satu tahap, untuk lencana di atas tabnya. */
export type StageSummary = {
  tahap: string
  label: string
  jumlah: number
}

export type ClaimReportListResponse = {
  laporan: ClaimReport[]
  /** Banyaknya baris yang cocok SEBELUM dipotong paginasi. */
  jumlah: number
  batas: number
  lewati: number
  /** Selalu memuat kelima tahap, termasuk yang jumlahnya nol. */
  ringkasan: StageSummary[]
}

export type ClaimReportResponse = {
  laporan: ClaimReport
}

/**
 * Kode galat modul Pelaporan Klaim.
 *
 * Terpisah dari KodeGalat karena ia milik satu modul, sementara KodeGalat mengikat seluruh
 * aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const ClaimReportErrorCode = {
  notFound: 'laporan_klaim_tidak_ditemukan',
  alreadyTransferred: 'laporan_sudah_ditransfer',
  alreadyRegistered: 'laporan_sudah_diregistrasi',
  numberTaken: 'nomor_laporan_sudah_dipakai',
} as const

export type ClaimReportErrorCode =
  (typeof ClaimReportErrorCode)[keyof typeof ClaimReportErrorCode]

/** Satu aturan yang dilanggar beserta kolom yang melanggarnya. */
export type PelanggaranField = {
  field: string
  pesan: string
}

/**
 * Kode galat yang dikirim backend.
 *
 * Layar membedakan jenis galat lewat kode ini, TIDAK PERNAH dengan mencocokkan teks
 * pesan — teks bisa berubah kapan saja tanpa mengubah artinya.
 */
export const KodeGalat = {
  kredensialSalah: 'kredensial_salah',
  penggunaTidakAktif: 'pengguna_tidak_aktif',
  identitasPutus: 'sistem_identitas_tidak_terhubung',
  sesiTidakSah: 'sesi_tidak_sah',
  sesiKedaluwarsa: 'sesi_kedaluwarsa',
  permintaanCacat: 'permintaan_cacat',
  galatInternal: 'galat_internal',

  // Milik modul master data.
  statusKlaimTidakDitemukan: 'status_klaim_tidak_ditemukan',
  validasiGagal: 'validasi_gagal',
  labelStatusSudahDipakai: 'label_status_sudah_dipakai',
  kodeStatusSudahDipakai: 'kode_status_sudah_dipakai',
} as const

export type KodeGalat = (typeof KodeGalat)[keyof typeof KodeGalat]

/**
 * Posisi sebuah rekening dalam alur persetujuan komite.
 *
 * Sandinya "0"/"1"/"2" mengikuti kolom APPROVAL pada POOLDATA.LST_ACCOUNT. Ia tidak
 * dapat dipilih bebas selama tabel yang sama masih dibaca sistem lama.
 */
export const StatusRekening = {
  menunggu: '0',
  disetujui: '1',
  ditolak: '2',
} as const

export type StatusRekening = (typeof StatusRekening)[keyof typeof StatusRekening]

/** Satu baris master rekening tujuan pembayaran klaim. */
export type Rekening = {
  nomor_rekening: string
  nama_pemilik: string
  nama_bank: string
  cabang_bank: string
  alamat_bank: string
  kode_bank: string
  tipe_rekening: string
  aktif: boolean

  email: string
  telepon: string
  nik: string
  catatan: string
  id_dokumen: string
  diinput_oleh: string

  status: string
  /** Sebutan status dalam bahasa yang dibaca pengguna, dihitung server. */
  status_label: string
  komite_approval: string

  diinput_pada: string
  diputuskan_pada?: string

  status_layanan: string
  id_rekening_kasir: string
  respons_kasir: string

  /**
   * Boleh tidak rekening ini menerima pembayaran klaim.
   *
   * Dihitung server dari dua syarat — disetujui komite DAN masih aktif — supaya
   * keduanya tidak perlu diulang di setiap layar.
   */
  dapat_dipakai: boolean
}

export type Bank = {
  kode: string
  nama: string
}

export type ResponsDaftarRekening = {
  rekening: Rekening[]
  /** Banyaknya baris yang cocok SEBELUM dipotong paginasi. */
  jumlah: number
  batas: number
  lewati: number
}

export type ResponsDaftarBank = {
  bank: Bank[]
}

/**
 * Kode galat modul Master Rekening.
 *
 * Terpisah dari KodeGalat karena ia milik satu modul, sementara KodeGalat mengikat
 * seluruh aplikasi. Keduanya dibaca dari field `kode` yang sama.
 */
export const KodeGalatRekening = {
  tidakDitemukan: 'rekening_tidak_ditemukan',
  sudahAda: 'nomor_rekening_sudah_ada',
  sudahDiputuskan: 'keputusan_sudah_diambil',
  isianTidakSah: 'isian_tidak_sah',
} as const

export type KodeGalatRekening =
  (typeof KodeGalatRekening)[keyof typeof KodeGalatRekening]
