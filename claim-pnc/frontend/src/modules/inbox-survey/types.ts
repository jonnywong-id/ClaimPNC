/**
 * Bentuk data layar My Work (MENU_ID 50) — antrean kerja Surveyor dan Loss Adjuster.
 *
 * Nama fieldnya mengikuti apa yang dikirim server apa adanya (`D-80`); yang berbahasa Inggris
 * di sini BUKAN kelalaian — ketiga belas kolomnya memang berjudul Inggris di layar Pega, dan
 * `D-13` menetapkan teks yang dilihat pengguna mengikuti layar lama.
 */

/** Satu baris antrean — satu janji survei yang menunggu. */
export type TugasSurvei = {
  /**
   * Kunci teknis objek survei (`T_SURVEYORLIST.CASEID`). TIDAK ditampilkan sebagai kolom —
   * isinya memuat nama kelas internal Pega — tetapi dipakai sebagai kunci baris.
   */
  survei_id: string

  /** Kunci klaim induknya, dipakai tautan baris membuka klaimnya. */
  klaim_id: string

  /**
   * Janji keberapa pada klaim yang sama.
   *
   * Tidak digambar. Satu klaim dapat punya beberapa janji survei, dan tanpa index ini kedua
   * barisnya tidak dapat dibedakan sebagai kunci React.
   */
  index_survei: string

  appointment_no: string
  reference_no: string
  claim_no: string
  policy_no: string
  insured_name: string
  cob: string
  cause_of_loss: string
  location: string
  pic_asm: string
  pic_loss_adjuster: string

  /** YYYY-MM-DD, sudah dalam WIB. Dikonversi server, bukan di peramban (`R-12`). */
  date_of_loss: string

  /**
   * Kolom "Aging" — DIBACA dari kolom `AGING`, bukan dihitung.
   *
   * `null` berarti belum dihitung, dan itu BERBEDA dari nol hari. Menggambar keduanya sama
   * akan menampilkan "0" pada baris yang sebenarnya tidak punya angka.
   */
  aging: number | null

  status_asm: string

  /** `SURVEYORTYPE_1` — `"1"` internal, `"2"` loss adjuster. Tidak digambar sebagai kolom. */
  jenis_surveyor: string

  /** `STS_SURVEY`, dibawa apa adanya dan tidak dipakai menyaring. */
  status_survei: string
}

/** Satu judul kolom, datang dari server. */
export type KolomLayar = {
  kunci: string
  judul: string
  /** Keterangan yang ditempelkan pada judul; kosong bila tidak ada. */
  keterangan?: string
}

/** Satu tab beserta judulnya. */
export type TabLayar = {
  kunci: string
  judul: string
  keterangan?: string
}

/**
 * Identitas surveyor pemanggil.
 *
 * Cakupannya ikut dikirim karena seorang leader melihat pekerjaan anggotanya. Tanpa menyebut
 * cakupan, pengguna yang melihat baris atas nama orang lain tidak punya cara menjelaskan
 * kenapa — dan yang pertama kali terpikir adalah "layarnya bocor".
 */
export type IdentitasSurveyor = {
  login: string
  nama: string
  leader: boolean
  cakupan: string[]
  jumlah_tim: number
}

/** Keterangan layar — judul kolom, judul tab, selisih terencana, dan keterbatasan. */
export type KeteranganResponse = {
  portal: string
  kolom: KolomLayar[]
  tab: TabLayar[]
  kolom_kpi: KolomLayar[]
  tab_bawaan: string
  jenis_kpi: string[]
  ukuran_halaman: number
  selisih_terencana: string[]
  keterbatasan: string[]
}

/** Satu halaman antrean. */
export type DaftarResponse = {
  portal: string
  identitas: IdentitasSurveyor
  data: TugasSurvei[]

  /** Jumlah SELURUH baris yang cocok, bukan jumlah baris di halaman ini. */
  total: number

  /**
   * Paginasi dan tab yang BENAR-BENAR dipakai server, bukan yang diminta.
   *
   * Tab yang tidak dikenal dijatuhkan ke Outstanding, dan tanpa mengembalikan yang dipakai,
   * bilah tab akan menyorot tab yang salah.
   */
  lewati: number
  batas: number
  tab: string
  cari: string
}

/** Jumlah baris satu tab. */
export type JumlahTab = {
  kunci: string
  total: number
}

/** Jumlah baris ketujuh tab, untuk bilah tab. */
export type JumlahTabResponse = {
  portal: string
  identitas: IdentitasSurveyor
  tab: JumlahTab[]
}

/** Satu baris ringkasan KPI. */
export type BarisKPI = {
  /** Nama adjuster, atau TAHUN pada ringkasan kuartal. */
  kelompok: string

  penjadwalan_survey: number
  immediate_advice: number
  preliminary_advice: number
  interim_report: number
  update_progress: number
  tanggapan_komunikasi: number
  propose_adjustment: number
  final_report: number
  nilai: number
}

/** Ringkasan KPI adjuster. */
export type KPIResponse = {
  portal: string
  identitas: IdentitasSurveyor
  jenis: string
  kategori: string
  tahun: string
  data: BarisKPI[]
}
