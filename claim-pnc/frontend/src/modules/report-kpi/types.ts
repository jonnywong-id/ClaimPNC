/**
 * Tipe kontrak API modul Report KPI PNC — butir menu `MENU_ID 84`, pengganti harness
 * `ReportKPIHarness`.
 *
 * Nama field mengikuti bentuk JSON yang dikirim server APA ADANYA, termasuk bahasanya:
 * ia kontrak, bukan nama internal (`D-80`).
 *
 * Tipe di sini DITULIS TANGAN, bukan dihasilkan dari kontrak OpenAPI. Kontrak itu belum
 * ada di repositori ini; `09-API-STRATEGY.md` §6 menuntutnya, dan ketiadaannya adalah
 * utang teknis yang berlaku untuk seluruh modul, bukan untuk modul ini saja.
 */

/** Satu kolom grid. */
export type GridColumn = {
  kunci: string
  judul: string

  /**
   * Kolom ini HANYA digambar pada satu kelompok tab KPI Admin.
   *
   * Kosong berarti berlaku di kedua kelompok. Perhatikan judul "Tgl Terima Dokumen" ada di
   * KEDUA kelompok tetapi menunjuk kolom basis data yang berbeda — karena itu penyaringnya
   * tidak boleh dilewati.
   */
  hanya_kelompok?: string

  /**
   * Kolom ini HANYA digambar pada tipe report `ALL`.
   *
   * Meniru Pega apa adanya: grid Summary di sana dilayani DUA rule, dan hanya yang
   * menggabungkan dua kelompok (`GetSummaryKPIAdjusterALL`) yang mengembalikan kolom tipe
   * — ditulis sebagai literal `'OUTSTANDING'` dan `'FINAL'`. Pada tipe tunggal kolomnya
   * memang tidak ada, dan isinya pun tidak berarti apa-apa di sana.
   */
  hanya_tipe_gabungan?: boolean
}

/** Satu komponen penilaian KPI adjuster. */
export type Component = {
  kode: string
  judul: string

  /**
   * Nama kolom basis datanya.
   *
   * Dikirim server dan DITAMPILKAN di keterangan layar: penguji gerbang 1 membandingkan
   * angka di sini dengan angka di Pega, dan yang pertama ditanyakannya selalu "ini kolom
   * yang mana".
   */
  kolom: string
}

/** Satu pilihan dropdown "Pilih Tipe Report". */
export type ReportTypeOption = {
  kode: string
  judul: string
  keterangan?: string
}

/** Satu grid pada sebuah tab. */
export type Grid = {
  kode: string
  judul: string

  /**
   * Kolom TETAP grid — belum termasuk kesembilan komponen.
   *
   * Layar merakit kolomnya dengan menyambung daftar ini dengan `komponen` pada akar
   * jawaban. Keduanya dipisah karena kesembilan komponen itu sama di kedua grid.
   */
  kolom: GridColumn[]

  /**
   * Kolom tetap SESUDAH kesembilan komponen.
   *
   * Hari ini hanya KATEGORI pada tab Adjuster dan NOTE pada tab Admin. Keduanya berada di
   * ujung kanan grid layar lama, dan menaruhnya di `kolom` akan menggambarnya di DEPAN
   * komponen — satu kolom bergeser, dan sembilan kolom angka di sebelahnya ikut terbaca
   * salah.
   */
  kolom_akhir: GridColumn[]
}

/** Satu tab layar. */
export type Tab = {
  kode: string
  judul: string
  grid: Grid[]

  /**
   * Tab yang digambar tetapi belum dapat diisi.
   *
   * Tab terhalang tetap ditampilkan, bukan disembunyikan — kemajuan migrasi terbaca
   * langsung dari layar, dan pengguna tidak melaporkan tab yang "hilang".
   */
  terhalang: boolean
  alasan_terhalang?: string
}

/** Satu pilihan dropdown "Pilih Data KPI" pada tab KPI Admin. */
export type AdminGroupOption = {
  kode: string
  judul: string
  keterangan?: string
}

/**
 * Satu baris dropdown "Periode KPI" pada blok PA tab KPI Admin.
 *
 * `kode` dan `judul` sama — `YYYYMM` apa adanya, seperti layar lama menampilkannya.
 */
export type AdminPeriodOption = {
  kode: string
  judul: string
}

/** Jawaban GET /api/report-kpi/tab. */
export type MetadataResponse = {
  tab: Tab[]
  tab_bawaan: string
  komponen: Component[]
  tipe_report: ReportTypeOption[]
  kelompok_admin: AdminGroupOption[]
  lini_bisnis: BusinessLineOption[]
  komponen_pic: PICComponent[]

  /** Isi dropdown "Pilih Data KPI" — dikirim peladen, bukan ditulis di layar. */
  data_kpi: PICExportKind[]

  /**
   * Petugas yang DIKECUALIKAN dari penilaian SLA di sistem lama.
   *
   * Ditampilkan di layar. Pengecualian yang tidak terlihat adalah pengecualian yang tidak
   * dapat dipertanyakan — dan ini pengecualian berbasis nama orang di dalam kode.
   */
  pic_dikecualikan_sla: string[]

  /**
   * Isi dropdown "Periode KPI" — penyaring blok PA pada tab KPI Admin.
   *
   * Dikirim peladen karena isinya diturunkan dari TAHUN BERJALAN, dan tahun yang berlaku
   * harus tahun peladen. Menyusunnya di browser berarti jam klien yang salah setel
   * diam-diam mengubah bulan yang dapat dipilih.
   */
  periode_kpi: AdminPeriodOption[]

  tabel_sumber: string

  /** Tabel tangga nilai tab KPI PIC Teknik — berbeda dari `tabel_sumber`. */
  tabel_tangga_nilai: string
}

/** Satu pilihan dropdown lini bisnis pada tab KPI PIC Teknik. */
export type BusinessLineOption = {
  kode: string
  judul: string
  keterangan?: string
}

/** Satu komponen penilaian PIC Teknik. */
export type PICComponent = {
  kode: string
  judul: string

  /**
   * Tangga nilainya MENURUN — makin kecil persentasenya, makin tinggi nilainya.
   *
   * Dipakai layar untuk menandai baris itu. Tanpa penanda, pembaca yang melihat
   * "95% → nilai 1" akan menyimpulkan angkanya rusak — padahal itulah yang sedang
   * berjalan di Pega hari ini.
   */
  tangga_menurun?: boolean

  /** Komponen ini punya nilai berbobot terpisah, berskala 0–15. */
  berbobot?: boolean
}

/** Satu baris penilaian pada kartu skor PIC. */
export type PICRow = {
  komponen: string
  judul: string
  total: number
  tercapai: number

  /** `null` berarti tidak dapat dihitung, BUKAN nol. */
  persentase: number | null
  nilai: number | null
}

/** Kartu skor satu PIC. */
export type PICScorecard = {
  pic: string
  leader: boolean
  baris: PICRow[]

  /** Nilai berbobot komponen Progress, berskala 0–15. */
  nilai_berbobot: number | null
}

/** Penyaring tab KPI PIC Teknik yang benar-benar dipakai server. */
export type PICFilter = {
  lini_bisnis: string
  dari: string
  sampai: string
}

/** Jawaban GET /api/report-kpi/pic-teknik. */
export type PICTeknikResponse = {
  kartu_skor: PICScorecard[]
  rekapitulasi: PICScorecard
  penyaring: PICFilter
  portal: string
}

/** Isian penyaring tab KPI PIC Teknik sebagaimana dipegang layar. */
/** Satu pilihan dropdown "Pilih Data KPI". */
export type PICExportKind = {
  kode: string
  judul: string
}

export type PICFilterInput = {
  lini: string
  dari: string
  sampai: string

  /**
   * dataKPI memilih BERKAS mana yang diunduh — bukan apa yang tampil di layar.
   *
   * Di Pega ia isian tersendiri di sebelah tombol Export, dan nilainya "1"…"4".
   * Kosong berarti bawaan ("1", Export KPI Progress), persis seperti
   * @if(TempLaporan.City==0,"1",TempLaporan.City) pada activity-nya.
   */
  dataKPI: string
}

/** Kepala kartu skor — siapa yang dinilai. */
export type AdminIdentity = {
  kategori: string
  nama_koordinator: string
  nik: string
  unit_kerja: string
}

/**
 * Satu baris terukur pada kartu skor.
 *
 * `nilai` `null` berarti metriknya TIDAK dapat dihitung — misalnya karena tidak ada satu
 * pun klaim pada periode yang dipilih, sehingga pembaginya nol. Bukan berarti nol.
 */
export type Metric = {
  kode: string
  judul: string
  nilai: number | null

  /** `cacah` · `persen` · `nilai` · `desimal` — menentukan cara angkanya digambar. */
  bentuk: string
}

/** Penyaring tab KPI Admin yang benar-benar dipakai server. */
export type AdminFilter = {
  kelompok: string
  dari: string
  sampai: string
}

/** Jawaban GET /api/report-kpi/admin/kartu-skor. */
export type ScorecardResponse = {
  identitas: AdminIdentity
  tanggal_efektif: string
  metrik: Metric[]

  /** Kosong pada kelompok PA — kartu skornya memang tidak punya baris kesimpulan. */
  achievement?: string

  penyaring: AdminFilter
  portal: string
}

/** Satu baris grid rincian tab KPI Admin. */
export type AdminDetailRow = {
  no_klaim: string
  no_polis: string
  business: string
  tgl_regist_klaim: string
  tgl_terima_dokumen: string
  flag: string
  aging_regist_klaim: number | null
  tgl_terima_dokumen_pa: string
  tgl_terima_lod: string
  tgl_pembayaran: string
  aging_pembayaran_klaim: number | null
  status_sla_regist: string
  status_sla_pembayaran: string
}

/** Jawaban GET /api/report-kpi/admin. */
export type AdminDetailResponse = {
  baris: AdminDetailRow[]
  paginasi: Pagination
  penyaring: AdminFilter
  portal: string
}

/** Isian penyaring tab KPI Admin sebagaimana dipegang layar. */
export type AdminFilterInput = {
  kelompok: string
  dari: string
  sampai: string
}

/** Penyaring yang BENAR-BENAR dipakai server menjawab permintaan. */
export type Filter = {
  tipe_report: string
  adjuster: string
  dari: string
  sampai: string
}

/**
 * Peta nilai komponen.
 *
 * `null` berarti komponen itu tidak punya satu pun nilai yang terbaca — BUKAN bernilai
 * nol. Perbedaan itu nyata pada laporan penilaian kinerja, dan layar menggambarnya
 * sebagai tanda hubung.
 */
export type Scores = Record<string, number | null>

/** Satu baris grid Summary — satu adjuster. */
export type SummaryRow = {
  adjuster: string
  tipe: string
  nilai: Scores

  /** Kolom KATEGORI. Kosong berarti nilainya tidak masuk satu pita pun. */
  kategori: string
}

/** Satu baris grid Detail — satu kasus survei. */
export type DetailRow = {
  adjuster: string
  no_case: string
  tipe: string

  /** Berbentuk `YYYY-MM-DD`. Kosong berarti kolomnya kosong di basis data. */
  tanggal: string
  nilai: Scores

  /** Kolom KATEGORI, dicari dengan cara yang sama seperti pada Summary. */
  kategori: string
}

/** Keterangan halaman. */
export type Pagination = {
  halaman: number
  ukuran: number
  total: number
  total_halaman: number
}

/** Jawaban GET /api/report-kpi/adjuster/ringkasan. */
export type SummaryResponse = {
  baris: SummaryRow[]
  penyaring: Filter
  portal: string
}

/** Jawaban GET /api/report-kpi/adjuster. */
export type DetailResponse = {
  baris: DetailRow[]
  paginasi: Pagination
  penyaring: Filter
  portal: string
}

/** Jawaban GET /api/report-kpi/adjuster/pilihan. */
export type AdjusterListResponse = {
  adjuster: string[]
  portal: string
}

/**
 * Isian penyaring sebagaimana dipegang layar.
 *
 * Ia TERPISAH dari `Filter` di atas, dan itu disengaja: yang ini isian yang sedang
 * diketik pengguna — boleh belum lengkap — sedangkan `Filter` adalah penyaring yang sudah
 * diterima server. Menyatukan keduanya membuat layar tidak dapat membedakan "yang sedang
 * saya ketik" dari "yang sedang ditampilkan".
 */
export type FilterInput = {
  /**
   * tipeReport memuat pilihan **"Pilih Status Survey"** di layar — ALL, FINAL, atau
   * OUTSTANDING.
   *
   * # Kenapa namanya tidak cocok dengan labelnya
   *
   * Ia dikirim ke peladen sebagai parameter `tipe_report`, dan nama itu **kontrak**
   * (`D-80`) — menggantinya memutuskan klien yang sudah ada. Yang dikoreksi 2026-10-08
   * adalah LABELNYA: layar Pega menyebut isian ini "Pilih Status Survey"
   * (`TempAdjComp.ASMFull`), sementara "Pilih Tipe Report" di Pega adalah isian LAIN —
   * lihat `variant` di bawah.
   *
   * Jadi ketidakcocokan ini disengaja dan terbatas pada nama internal; yang dilihat
   * pengguna sudah sama dengan Pega (`D-13`).
   */
  tipeReport: string

  /**
   * variant memuat pilihan **"Pilih Tipe Report"** milik Pega: DATA SUMMARY atau
   * DATA DETAIL (`TempAdjComp.AcceptedNo`).
   *
   * Ia menentukan grid mana yang digambar DAN berkas mana yang diunduh — di Pega satu
   * tombol "Export Data" melayani keduanya, karena pilihannya sudah dinyatakan di sini.
   */
  variant: string

  adjuster: string
  dari: string
  sampai: string
}
