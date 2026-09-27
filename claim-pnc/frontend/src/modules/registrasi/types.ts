// Bentuk data modul Registrasi Klaim.
//
// Tipe di sini adalah cerminan DTO Go di internal/registrasi/http. Bila salah satu
// berubah, yang lain wajib ikut berubah — pemeriksaan tipe TypeScript adalah jaring
// pengaman antara layar dan API.
//
// Ia berada di dalam modul, bukan di `api/tipe.ts`, karena hanya modul ini yang
// memakainya. Aturan susunan frontend menaikkan sesuatu ke `api/` hanya ketika ia
// benar-benar dipakai bersama.

import type { PercentE4, Cents } from '@/components/format'

/** Dua model penugasan yang ADR-0019 tetapkan dipertahankan. */
export const QueueKind = {
  /** Tugas milik satu orang tertentu. */
  worklist: 'WORKLIST',
  /** Antrean bersama, diambil siapa pun yang berwenang. */
  workbasket: 'WORKBASKET',
} as const

export type QueueKind = (typeof QueueKind)[keyof typeof QueueKind]

export type Task = {
  id: string
  klaim_id: string
  nomor_klaim: string
  tahap: string
  nama_tahap: string
  antrean: string
  workbasket: string
  pemilik: string
  /** Tugas masih menunggu seseorang mengambilnya. Kewenangannya tetap diperiksa server. */
  dapat_diambil: boolean
  tindakan_keluar: string
  dibuat_pada: string
}

export type Spreading = {
  jenis_treaty: string
  nama: string
  /** Persentase dikali 10.000; 100% dikirim sebagai 1000000. */
  share: PercentE4
  dihapus: boolean
  objek_fac_offer: string
}

export type Coverage = {
  id: string
  penyebab_kerugian: string
  tsi_sen: Cents
  spreading: Spreading[]
}

export type InsuredItem = {
  id: string
  nama: string
  lokasi: string
  coverage: Coverage[]
}

export type Reporter = {
  nama: string
  telepon: string
  email: string
  alamat: string
  hubungan: number
  hubungan_lainnya: string
}

export type Policy = {
  nomor: string
  lini: string
  nama_lini: string
  jenis_bisnis: string
  mulai_pertanggungan: string
  akhir_pertanggungan: string
  mata_uang: string
  nama_tertanggung: string
  deklarasi: boolean
  penjamin_kredit: boolean
}

/**
 * Wilayah kejadian — bagian bawah layar Input Register Pega
 * (Section/ViewInputRegisterDetail-Section.xml). Setiap tingkat membawa kode dan nama:
 * kode menyaring tingkat di bawahnya, nama yang ditampilkan.
 */
export type Area = {
  negara: string
  negara_id: string
  provinsi: string
  provinsi_id: string
  kota: string
  kota_id: string
  kabupaten: string
  kabupaten_id: string
  kelurahan: string
  kelurahan_id: string
  kode_pos: string
}

/** Tingkat daftar pilihan wilayah, sesuai rute /api/registrasi/wilayah/{tingkat}. */
export const AreaLevel = {
  Country: 'negara',
  Province: 'provinsi',
  City: 'kota',
  District: 'kabupaten',
  Village: 'kelurahan',
} as const
export type AreaLevel = (typeof AreaLevel)[keyof typeof AreaLevel]

export type AreaOption = {
  id: string
  nama: string
  kode_pos?: string
}

export type AreaOptionsResponse = {
  pilihan: AreaOption[]
}

/** Negara yang membuka isian Kota sampai Kode Pos (kondisi Country = 'INDONESIA'). */
export const COUNTRY_INDONESIA = 'INDONESIA'

/** Prinsip Mengenal Nasabah — ClaimData.CustomerPrinciple. NORMAL adalah bawaan. */
export const CustomerPrinciple = {
  Normal: '1',
  Suspicious: '2',
} as const

export type Claim = {
  id: string
  nomor: string
  portal: string
  polis: Policy
  tanggal_kejadian: string
  tanggal_lapor: string
  tanggal_terima_dokumen: string
  lokasi: string
  kronologi: string
  pelapor: Reporter
  wilayah: Area
  prinsip_mengenal_nasabah: string
  komentar_suspicious: string
  nilai_estimasi_sen: Cents
  mata_uang: string
  nomor_slik: string
  ex_gratia: boolean
  user_teknis: string
  rcv_id: string
  objek: InsuredItem[]
  status_pucl: number
  transfer_compliance: boolean

  /**
   * Empat konsep status yang berbeda (ADR-0018). Namanya sengaja dibuat tidak dapat
   * tertukar; di sistem lama `StatusClaim` dan `ClaimStatus` berbeda hanya pada urutan
   * kata, dan kekeliruan membacanya sulit terdeteksi.
   */
  status_proses: string
  status_klaim: string
  flag_klaim: string
  status_posisi_progres: string

  tahap_kini: string
}

export type Stage = {
  id: string
  nama: string
  antrean: string
  workbasket: string
  router: string
  tindakan_keluar: string
  pega_id: string
}

export type FlowResponse = {
  nama: string
  mulai: string
  tahap: Stage[]
}

export type InboxResponse = {
  tugas: Task[]
}

export type ClaimResponse = {
  klaim: Claim
  tugas: Task | null
  jalur: string[] | null
  jejak_keputusan: string[] | null
  large_loss: boolean
}

/** Satu aturan validasi yang dilanggar, beserta kolom yang harus diperbaiki. */
export type Violation = {
  kode: string
  field: string
  pesan: string
}

/**
 * Kode galat modul registrasi.
 *
 * Layar membedakan jenis galat lewat kode ini, TIDAK PERNAH dengan mencocokkan teks
 * pesan — teks bisa berubah kapan saja tanpa mengubah artinya.
 */
export const RegistrationErrorCode = {
  validationFailed: 'validasi_gagal',
  claimNotFound: 'klaim_tidak_ditemukan',
  taskNotFound: 'tugas_tidak_ditemukan',
  taskAlreadyClaimed: 'tugas_sudah_diambil',
  notTaskOwner: 'bukan_pemilik_tugas',
  taskAlreadyDone: 'tugas_sudah_selesai',
  stageMismatch: 'tahap_tidak_bersesuai',
  invalidAction: 'tindakan_tidak_sah',
  exchangeRateNotFound: 'kurs_tidak_ditemukan',
} as const

export type RegistrationErrorCode =
  (typeof RegistrationErrorCode)[keyof typeof RegistrationErrorCode]

/** Badan POST /api/registrasi/register. */
export type RegisterRequest = {
  tugas_id: string
  tanggal_kejadian: string
  tanggal_lapor: string
  tanggal_terima_dokumen: string
  lokasi: string
  kronologi: string
  pelapor: Reporter
  wilayah: Area
  prinsip_mengenal_nasabah: string
  komentar_suspicious: string
  nilai_estimasi_sen: Cents
  mata_uang: string
  nomor_slik: string
  ex_gratia: boolean
  user_teknis: string
  rcv_id: string
  objek: InsuredItem[]
  status_pucl: number
  transfer_compliance: boolean
  kembali: boolean
}
