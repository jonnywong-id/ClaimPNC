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

  // Milik modul Komite.
  liniTidakDikenal: 'lini_tidak_dikenal',
  nilaiKlaimCacat: 'nilai_klaim_cacat',
} as const

export type KodeGalat = (typeof KodeGalat)[keyof typeof KodeGalat]

/**
 * Satu baris master ambang komite — satu jenjang persetujuan untuk satu lini bisnis.
 *
 * Nilai uang datang sebagai **teks desimal kanonik** (`"50000001.00"`), bukan angka JSON.
 * Angka JSON adalah floating point ganda di peramban, dan `I-12` menetapkan nilai uang
 * disimpan dengan presisi penuh. Pemformatannya lewat `@/lib/money`.
 */
export type KomiteThreshold = {
  id: string
  nama: string
  operator_id: string
  lini: string
  /**
   * Kolom `TYPE_KOMITE` apa adanya, dan kolom itu memikul **dua arti**: pita nilai di
   * Non-MBU, varian jalur (PA reguler versus PA TKI) di PA. Jangan menafsirkannya di
   * layar — pakai `berpita_nilai` pada respons penjenjangan.
   */
  jenis_komite: string
  batas_bawah: string
  /**
   * `LIMIT_TOP`. **Tidak pernah** dipakai memilih baris — memakainya untuk menyaring
   * akan mengembalikan tepat satu baris dan menghapus penjenjangan seluruhnya (`D-47`).
   * Ia hanya bahan pemeriksaan integritas master.
   */
  batas_atas: string
  /** `DEGREE` — penentu urutan, bukan jumlah jenjang. Boleh berulang dan boleh melompat. */
  jenjang: number
  aktif: boolean
  untuk_adjustment: boolean
  untuk_registrasi: boolean
  untuk_penolakan: boolean
  sedang_absen: boolean
  /** Kesimpulan server: baris ini benar-benar ikut menyetujui nilai klaim. */
  jenjang_persetujuan: boolean
}

/** Aturan pita nilai yang berlaku untuk sebuah lini. Hanya Non-MBU yang memakainya. */
export type KomiteBandPolicy = {
  lini: string
  batas: string
  pita_bawah: string
  pita_atas: string
}

export type KomiteThresholdListResponse = {
  ambang: KomiteThreshold[]
  total: number
  /** Hanya baris yang benar-benar ikut menyetujui. Selalu lebih kecil dari `total`. */
  total_jenjang: number
  /** Datang dari data, bukan dari daftar tetap di dalam kode (`D-15`). */
  lini: string[]
  kebijakan_pita: KomiteBandPolicy[]
  /**
   * Mode penjenjangan portal ini — `"kumulatif"` atau `"satu-penyetuju"`.
   *
   * Ditentukan per **portal**, bukan per lini. Layar memakainya untuk menentukan apakah
   * isian Operator ID pengaju perlu ditampilkan sama sekali.
   */
  mode: string
}

/** Satu hal yang ditemukan pada master ambang. */
export type KomiteFinding = {
  /** `"cacat"` atau `"peringatan"`. Dibedakan lewat nilai ini, bukan teks pesannya. */
  tingkat: string
  jenis: string
  lini: string
  pita?: string
  pesan: string
  id_ambang: string[]
}

export type KomiteIntegrityResponse = {
  temuan: KomiteFinding[]
  jumlah_cacat: number
  jumlah_peringatan: number
}

/** Satu orang yang harus menyetujui sebuah nilai klaim. */
export type KomiteApprover = {
  /** Selalu berurutan tanpa lompatan, 1 sampai jumlah penyetuju. */
  urutan: number
  /** `DEGREE` dari master. Boleh berulang — lihat `urutan_tidak_pasti`. */
  jenjang: number
  nama: string
  operator_id: string
  /** Ambang yang membuat orang ini ikut — alasannya, bukan sekadar hasilnya. */
  batas_bawah: string
  sedang_absen: boolean
  id_ambang: string
}

/** Mode penjenjangan, ditentukan per portal. */
export const KomiteMode = {
  /** Non-MBU, PA, Travel, Bonding: setiap jenjang yang terlampaui ikut menyetujui. */
  cumulative: 'kumulatif',
  /** Simasnet: dipilih tepat satu penyetuju, diacak, penginput dikecualikan. */
  singleApprover: 'satu-penyetuju',
} as const

export type KomiteMode = (typeof KomiteMode)[keyof typeof KomiteMode]

export type KomiteTieringResponse = {
  nilai: string
  lini: string
  mode: string
  /** Membedakan "lini ini tidak memakai pita" dari "pitanya gagal dihitung". */
  berpita_nilai: boolean
  pita?: string
  penyetuju: KomiteApprover[]
  /**
   * Hanya terisi pada mode satu-penyetuju: seluruh orang yang **layak** dipilih pada
   * jenjang terendah, sebelum satu di antaranya diacak.
   *
   * Yang dapat diperiksa pada mode itu bukan siapa yang terpilih — itu acak — melainkan
   * apakah kumpulan yang layak sudah benar.
   */
  kandidat?: KomiteApprover[]
  /** Operator yang **diminta** dikecualikan karena dialah yang mengajukan. */
  dikecualikan_penginput?: string
  /**
   * Orang yang **benar-benar** keluar karena pengecualian itu.
   *
   * Dibedakan dari `dikecualikan_penginput` dengan sengaja: penginput yang bukan anggota
   * komite tidak mengubah apa pun. Isinya paling penting justru saat `penyetuju` kosong —
   * ia satu-satunya keterangan yang menjelaskan kenapa.
   */
  tersingkir?: KomiteApprover[]
  jumlah_jenjang: number
  /** Keadaan, bukan galat: temuan tentang isi master yang harus dilihat Work Owner. */
  tanpa_penyetuju: boolean
  /**
   * Dua penyetuju ber-`DEGREE` sama. Kueri sistem lama mengurutkan dengan
   * `ORDER BY DEGREE` saja, sehingga saat seri urutannya ditentukan basis data.
   */
  urutan_tidak_pasti: boolean
}
