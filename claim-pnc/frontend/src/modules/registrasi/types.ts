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
  /**
   * Pemanggil boleh mengerjakan tugas ini: pemiliknya, atau pemegang grup tahapnya di
   * M_LOGIN_GROUP_PNC. Dikirim hanya pada respons satu klaim.
   */
  dapat_dikerjakan?: boolean
  /** Pemanggil anggota grup Analyst (When `IsAnalisator`). Dikirim hanya pada respons satu klaim. */
  analis?: boolean
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
  /** Nama jaminan dari polis (CoverageNote). */
  nama: string
  penyebab_kerugian: string
  tsi_sen: Cents
  spreading: Spreading[]
  /** Item objek beserta estimasinya — diisi di tahap Input Estimasi. */
  item?: ObjectItem[]
  /** AdjustmentList — diisi tombol Tambah di tahap InputSurveyor. */
  adjustment?: Settlement[]
  /** Jaminan sudah ditandai Transfer ke Analyst (ISANALISTRANSFER) — hanya dikirim server. */
  sudah_transfer_analis?: boolean
  /** Isian modal "Transfer Claim ke Komite" (ClaimComitee_OC) — hanya dikirim server. */
  isian_komite?: CommitteeNote
}

/** Satu kode diagnosa (sm.m_diagnosis) — hasil Cari Kode / Desc Diagnose. */
export type DiagnosisOption = {
  kode: string
  deskripsi: string
}

export type DiagnosisResponse = {
  pilihan: DiagnosisOption[]
}

/**
 * Isian modal "Transfer Claim ke Komite" satu jaminan — kolom analisis
 * T_CLAIM_OBJECTCOVERAGE. Inisial dan tanggal komite hanya dikirim server.
 */
export interface CommitteeNote {
  kronologi_kejadian: string
  jumlah_kerugian: string
  polis_liability: string
  remarks: string
  remarks_investigasi: string
  diagnosa: string
  kode_diagnosa: string
  desc_diagnosa: string
  penerima_klaim: string
  inisial?: string
  tanggal_komite?: string
}

/** Kode Tipe Pembayaran (PAYMENTTYPE). */
export const PaymentType = {
  Final: '1',
  Interim: '2',
  Salvage: '3',
  AdjusterFee: '4',
  Adjustment: '5',
  Reject: '6',
} as const

/** Kode Tipe Resiko Sendiri (INDIVIDUAL_RISK_TYPE). */
export const RiskType = {
  OfClaim: '1',
  OfTSI: '2',
  /** Lainnya: nilai resiko sendiri diisi petugas, persennya nol. */
  Other: '3',
} as const

/** Satu baris Adjustment. Uang dalam sen; persen dikali 10.000 (100% = 1000000). */
export type Settlement = {
  tipe_pembayaran: string
  nama_tipe_pembayaran: string
  mata_uang: string
  kurs_e4: number
  nilai_propose_sen: Cents
  nilai_pengajuan_sen: Cents
  loc: PercentE4
  nilai_salvage_sen: Cents
  nilai_salvage_b_sen: Cents
  nilai_interim_sen: Cents
  nilai_estimasi_sen: Cents
  tipe_resiko: string
  persen_resiko: PercentE4
  nilai_resiko_sen: Cents
  nilai_gross_sen: Cents
  share_asm: PercentE4
  nilai_asm_sen: Cents
  nilai_akseptasi_sen: Cents
  kronologi: string
  catatan: string
  status_akseptasi: string
  nomor_akseptasi: string
  /** STATUSAKSEPTASILOD — `.AcceptationStatusLOD`: "" belum, "1" disetujui, "0" tidak. */
  status_akseptasi_lod?: string
  /** Print LOD terakhir (PRINTLOD_DATE, RFC3339) dan jenisnya (PDFTYPE) — baca saja di form akseptasi. */
  tanggal_cetak_lod?: string
  tipe_pdf_lod?: string
  nama_tipe_pdf_lod?: string
  /** Transfer Kasir: TRANSFER_CASHIER_DATE, IDCHASIER, dan penanda sudah ditransfer. */
  tanggal_transfer_kasir?: string
  case_id_kasir?: string
  sudah_transfer_kasir?: boolean
  /** Terisi begitu baris ditransfer ke komite (CASEIDKOMITE); baris itu tidak dapat ditransfer ulang. */
  komite_id?: string
  tanggal_transfer_komite?: string
  tanggal_putusan_komite?: string
}

/** Keputusan anggota komite (STATUSAPPROVE) dan status akseptasi hasil komite. */
export const CommitteeDecision = {
  pending: '0',
  approve: '1',
  reject: '2',
} as const

/** Satu kasus komite (baris T_CLAIM_KOMITE_LIST ber-KOMITE_ID sama). */
export type Committee = {
  id: string
  nomor_klaim: string
  status: 'berjalan' | 'disetujui' | 'ditolak'
  /** Operator jenjang yang sedang ditunggu. */
  menunggu?: string
  anggota: {
    jenjang: number
    komite: string
    keputusan: string
    catatan: string
    tanggal_putusan?: string
  }[]
}

/** Badan tombol Transfer Komite. Indeks berbasis 1. */
export type CommitteeTransferRequest = {
  tugas_id: string
  objek: number
  jaminan: number
  adjustment: number
  /** Penerima Klaim modal Transfer Claim ke Komite (Travel); kosong = TEMPRECEIVER tersimpan. */
  penerima_klaim?: string
}

export type CommitteeTransferResponse = {
  klaim: Claim
  komite: Committee
}

/** Satu putusan komite yang menunggu pengguna ini. */
export type CommitteeItem = {
  komite_id: string
  jenjang: number
  jumlah_jenjang: number
  klaim_id: string
  nomor_klaim: string
  nomor_polis: string
  nama_tertanggung: string
  nama_objek: string
  nama_coverage: string
  adjustment: number
  nama_tipe_pembayaran: string
  mata_uang: string
  nilai_asm_sen: Cents
  nilai_komite_sen: Cents
  tanggal_transfer: string
}

export type CommitteeListResponse = { komite: CommitteeItem[] }

/** Hasil hitungan baris yang belum disimpan, beserta spreading jaminannya. */
export type SettlementPreviewResponse = {
  adjustment: Settlement
  spreading: Spreading[]
}

/** Badan tombol Tambah pada grid Adjustment. */
export type SettlementRequest = {
  tugas_id: string
  objek: number
  jaminan: number
  /** Nomor baris (berbasis 1) yang diubah — hanya rute ubah. */
  adjustment?: number
  tipe_pembayaran: string
  mata_uang: string
  nilai_propose_sen: Cents
  nilai_pengajuan_sen: Cents
  loc: PercentE4
  nilai_salvage_sen: Cents
  nilai_salvage_b_sen: Cents
  tipe_resiko: string
  persen_resiko: PercentE4
  nilai_resiko_sen: Cents
  professional_fee_sen: Cents
  survey_expenses_sen: Cents
  vat: PercentE4
  tipe_vat: string
  kronologi: string
  catatan: string
}

/** Kode Tipe Estimasi (ESTIMATIONTYPE). */
export const EstimationType = {
  Claim: '1',
  Adjuster: '2',
} as const

export type Estimation = {
  tipe: string
  /** Kode mata uang POOLDATA.CURRENCY, mis. 10026. */
  mata_uang: string
  tanggal: string
  nilai_sen: Cents
  /** Dihitung server: kurs x 10.000 dan nilai rupiahnya. */
  kurs_e4: number
  nilai_idr_sen: Cents
  /** Sudah dibuatkan Claim Face Sheet — baris terkunci. Dikirim server saja. */
  sudah_cfs?: boolean
}

export type ObjectItem = {
  nama: string
  deskripsi: string
  /** Kelompok item properti polis (PROPERTYITEMGROUP), lini Fire. */
  kelompok: string
  estimasi: Estimation[]
}

export type EstimateRequest = {
  tugas_id: string
  kembali: boolean
  objek: { coverage: { item: ObjectItem[] }[] }[]
  /** Catatan ke PIC Teknis; tidak dikirim berarti catatan tersimpan dibiarkan. */
  catatan_pic_teknis?: string
}

export type CurrencyOption = { id: string; nama: string }

/** Pilihan Objek item estimasi — item properti polis Fire. Lini lain: daftar kosong. */
/** id: ObjectItemID pilihan (manfaat plan Travel); kosong untuk lini lain. */
export type ItemOption = { id: string; nama: string; kelompok: string; tsi_sen: Cents }
/** bawaan: nama item untuk item baru ("Others"/"OTHERS"); kosong untuk Fire. */
export type ItemOptionsResponse = { pilihan: ItemOption[]; bawaan: string }
/**
 * Pilihan dropdown "Tambah coverage" — coverage polis milik satu objek, dibaca dari
 * POOLDATA.T_COVERAGELIST_CARGO/ANEKA/FIRE/PERSON sesuai lini bisnis polis.
 */
export type CoverageOptionsResponse = { pilihan: Coverage[] }
export type CurrenciesResponse = { pilihan: CurrencyOption[] }

export type InsuredItem = {
  id: string
  nama: string
  lokasi: string
  coverage: Coverage[]
  /** Pekerjaan dan tanggal lahir (YYYY-MM-DD) peserta PA — T_PERSONLIST polis, baca saja. */
  pekerjaan?: string
  tanggal_lahir?: string
  /** KTP/Paspor dan Status peserta Travel — T_PERSONLIST polis, baca saja. */
  ktp_paspor?: string
  status_peserta?: string
  /** Model, Merk, Nama Tipe, Nomor Chasis objek HE — T_ANEKALIST polis, baca saja. */
  model?: string
  merk?: string
  nama_tipe?: string
  nomor_chasis?: string
}

/** Satu penerima klaim. */
export type Receiver = {
  id: string
  nama: string
  alamat: string
  nama_bank: string
  nomor_rekening: string
}

/** Satu rekening Master Rekening (POOLDATA.LST_ACCOUNT) — isian No Rekening InputReceiver. */
export type BankAccount = {
  nomor_rekening: string
  nama: string
  nama_bank: string
  nama_cabang_bank: string
  alamat: string
  id_bank: string
  email: string
  telepon: string
  /** Kosong bila master belum mengisinya. */
  tanggal_approve_kasir: string
  tanggal_approve_komite: string
}

/** Badan tombol Simpan InputReceiver. `id` kosong berarti penerima baru (tombol Tambah). */
export type ReceiverRequest = {
  tugas_id: string
  id: string
  nomor_rekening: string
  email: string
  telepon: string
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
  /** Quotation.BusinessCode — When IsAneka (10140). */
  kode_bisnis?: string
  mulai_pertanggungan: string
  akhir_pertanggungan: string
  mata_uang: string
  nama_tertanggung: string
  deklarasi: boolean
  penjamin_kredit: boolean
  /** TYPEOFCOINS: 0 tanpa koasuransi, 1 member, 2 leader, F fac in. */
  jenis_koasuransi?: string
  peran_koasuransi?: string
  /** Quotation.SobName dan Quotation.BusinessName — DATA TERTANGGUNG KLAIM. */
  nama_sumbis?: string
  nama_bisnis?: string
  kode_cabang?: string
}

/** Satu baris grid Telephone dan Email (CIFData ASMTelfax). */
export type InsuredPhone = {
  jenis: string
  nama_jenis: string
  kode: string
  nomor: string
  ekstensi: string
}

/** Satu alamat tertanggung dari CIF polis (CIFData.AddressList). */
export type InsuredAddress = {
  jenis: string
  nama_jenis: string
  alamat: string
  kota: string
  nama_kota: string
  kecamatan: string
  nama_kecamatan: string
  kelurahan: string
  nama_kelurahan: string
  kode_pos: string
  telepon: InsuredPhone[]
}

/** GET /api/registrasi/klaim/{id}/tertanggung. */
export type InsuredResponse = { no_ktp: string; alamat: InsuredAddress[] }

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

/**
 * Isian awal form AcceptationLOD (AcceptationLOD_PreAct): Tipe Akseptasi beserta pilihannya,
 * Nama Komite Akseptasi, Nilai LOD (Non-MBU saja), dan peringatan yang menolak akseptasi.
 */
export type AcceptanceDefaults = {
  tipe_akseptasi: string
  pilihan_tipe_akseptasi: { id: string; nama: string }[]
  nama_komite_akseptasi: string
  nilai_lod_sen?: number
  peringatan: string[]
}

/** Satu pilihan Penyebab Kerugian: id D_COL_ID, nama DESCRIPTION. */
export type CauseOfLossOption = {
  id: string
  nama: string
}

export type CauseOfLossOptionsResponse = {
  pilihan: CauseOfLossOption[]
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
  /** Isian InputRegisterDetail2_sect: EMAIL_LOD, REMARKRECOMENDATION, SUBJECTEMAIL, STSSALVAGE. */
  email_lod?: string
  rekomendasi?: string
  subjek_email?: string
  status_salvage?: string
  nilai_estimasi_sen: Cents
  mata_uang: string
  nomor_slik: string
  ex_gratia: boolean
  user_teknis: string
  /** Catatan ke PIC Teknis layar Input Estimasi (T_CLAIM_PNC.REMARK). */
  catatan_pic_teknis?: string
  rcv_id: string
  objek: InsuredItem[]
  /** ClaimData.ReceiverClaim — penerima klaim (T_CLAIM_RECEIVER). */
  penerima_klaim?: Receiver[]
  status_pucl: number
  transfer_compliance: boolean

  /**
   * Empat konsep status yang berbeda (ADR-0018). Namanya sengaja dibuat tidak dapat
   * tertukar; di sistem lama `StatusClaim` dan `ClaimStatus` berbeda hanya pada urutan
   * kata, dan kekeliruan membacanya sulit terdeteksi.
   */
  status_proses: string
  status_klaim: string
  /** Nama Status Klaim dari master V_STS_CLAIM, mis. "Register". */
  status_klaim_nama?: string
  /** Klaim sudah pernah ditransfer ke Analyst (ANALYST_TRANSFERDATE terisi). */
  sudah_transfer_analis?: boolean
  /** Klaim ditutup sementara (ISPENDINGCLOSE) — tombol Tutup Klaim tidak tampil. */
  tutup_sementara?: boolean
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
  email_lod: string
  rekomendasi: string
  subjek_email: string
  status_salvage: string
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

// ── Tab pendamping tahap Input Estimasi — hanya membaca tabel warisan ────────────

export type Survey = {
  kasus_id: string
  /** 1 = survey internal, 2 = loss adjuster. */
  tipe: string
  nama_surveyor: string
  tanggal_survey: string
  lokasi_survey: string
  nama_objek: string
  lokasi_objek: string
  urutan: string
  status: string
  keterangan: string
  tanggal_input: string
}
export type SurveysResponse = { survey: Survey[] }

export type DocumentRow = {
  id: string
  jenis_id: string
  nama: string
  wajib: boolean
  minimal: string
  terunggah: number
}
export type DocumentCategory = { kode: string; nama: string; dokumen: DocumentRow[] }
export type Attachment = {
  id: string
  nama: string
  jenis_berkas: string
  catatan: string
  kategori: string
  sub_kategori: string
  tersimpan: boolean
  diunggah_oleh: string
  /** RFC 3339, WIB. */
  diunggah_pada: string
  /** Tombol Delete tampil: berkas diunggah pemanggil sendiri dan tersimpan di penyimpanan (dinilai server). */
  bisa_dihapus?: boolean
}
export type DocumentsResponse = { kategori: DocumentCategory[]; berkas: Attachment[] }
/** Alamat baca satu lampiran — tombol Lihat dokumen. berlaku_sampai RFC 3339 WIB, boleh kosong. */
export type DocumentLink = { url: string; berlaku_sampai: string }

export type ProgressEntry = {
  urutan: number
  tanggal_input: string
  status_1: string
  status_1_nama: string
  status_2: string
  status_2_nama: string
  keterangan: string
  tindak_lanjut: string
  diinput_oleh: string
}
export type Communication = {
  kasus_id: string
  id: string
  tanggal: string
  pengirim: string
  pesan: string
  balasan: string
  penjawab: string
  tanggal_balasan: string
  /** COMMUNICATE_FROM; "SENDTOINPUTOR" untuk catatan tombol Kirim ke Inputor. */
  kanal?: string
}
export type ProgressResponse = { progres: ProgressEntry[]; komunikasi: Communication[] }

/** Badan POST /api/registrasi/klaim/{id}/cfs. Objek dan jaminan berbasis 1. */
export type FaceSheetRequest = {
  tugas_id: string
  objek: number
  jaminan: number
}

/**
 * Badan rute PLA (daftar, catatan, cetak). Objek dan jaminan berbasis 1. nomor memilih satu PLA
 * untuk dicetak (kosong = seluruhnya); catatan berisi REMARKS per nomor PLA.
 */
export type PLARequest = FaceSheetRequest & {
  nomor?: string
  catatan?: Record<string, string>
  /** Isian Email per nomor PLA (`.pyEmailAddress`); hanya nomor yang dikirim yang diubah. */
  email?: Record<string, string>
}

/** Satu baris grid layar PrintPLA_dtl. */
export type PLARow = {
  nomor: string
  penerima: string
  tipe: string
  catatan: string
  email: string
  tanggal: string
}

export type PLAListResponse = {
  revisi_cfs: number
  baru_terbit: number
  pla: PLARow[]
}

/**
 * Alamat satu baris adjustment untuk dialog Print DLA. Indeks berbasis 1. Nomor memilih satu
 * DLA untuk dicetak (kosong = seluruhnya); catatan berisi REMARKS per nomor yang belum dicetak.
 */
export type DLARequest = {
  tugas_id: string
  objek: number
  jaminan: number
  adjustment: number
  nomor?: string
  catatan?: Record<string, string>
  sesuai_polis?: boolean
}

/** Satu baris grid layar PrintDLA. */
export type DLARow = {
  nomor: string
  penerima: string
  tipe: string
  catatan: string
  email: string
  tanggal: string
  nilai: string
  mata_uang: string
  sudah_cetak: boolean
  sudah_kirim: boolean
}

/** Alamat satu baris adjustment untuk Transfer Kasir. Indeks berbasis 1. */
export type CashierRequest = {
  tugas_id: string
  objek: number
  jaminan: number
  adjustment: number
  /** "Tipe Transfer Kasir": 1 Pembayaran Biasa, 2 Join Placement, 3 Fronting. */
  tipe_transfer?: string
  /** No DLA FAC OUT yang dicentang "Pilih Fac-out Tidak Dibayar". */
  fac_out_tidak_dibayar?: string[]
}

/** Isi dialog konfirmasi Transfer Kasir; masalah berisi galat validasi pertama. */
export type CashierPreview = {
  nomor_akseptasi: string
  penerima: string
  nomor_rekening: string
  nama_bank: string
  email: string
  nilai_nett_sen: number
  mata_uang: string
  masalah: string
  /** Kalimat konfirmasi Pega (Pre_AlertTransferkasir). */
  konfirmasi: string
  /** Pilihan "Tipe Transfer Kasir" (property JoinPlacement). */
  tipe_transfer: { id: string; nama: string }[]
  /** DLA FAC OUT adjustment ini (GetdataFacoutJoinPlacement). */
  fac_out: { nomor_dla: string; nama_facout: string; nilai_bayar: string; mata_uang: string }[]
}

export type DLAListResponse = {
  baru_terbit: number
  ex_gratia: boolean
  peringatan: string[]
  dla: DLARow[]
}

/** Satu baris grid Status Penerimaan Komite (InputAdjustment_sect, .ComiteeClaim). */
export type CommitteeStatusRow = {
  jenjang: number
  nama_komite: string
  /** 0 menunggu, 1 setuju, 2 tolak. */
  status: string
  tanggal: string
  komentar: string
}

/** Satu baris grid Histori Transfer Kasir (InputAdjustment_sect, TempDataLogKasir). */
export type CashierHistoryRow = {
  pic_teknik: string
  tanggal: string
  status_kasir: string
  komentar: string
}

export type SettlementHistoryResponse = { komite: CommitteeStatusRow[]; kasir: CashierHistoryRow[] }

/** Jawaban GET /klaim/{id}/aging — `.PaymentData.AgingAmount` dari layanan premi. */
export type PremiumAgingResponse = { aging_amount: string | null; tersedia: boolean }
