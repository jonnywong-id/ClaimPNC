package reportkpi

import "strings"

// Tab KPI PIC Teknik — penilaian kinerja PIC Teknik, empat komponen per orang.
//
// # Yang membedakannya dari kedua tab lain
//
// Tab Adjuster MEMBACA nilai yang sudah dihitung Pega. Tab Admin menghitung satu kartu skor
// dari satu kueri. Tab ini berbeda dari keduanya: nilainya **dirakit di lapisan aplikasi**
// dari empat kelompok kueri, karena begitulah sistem lama melakukannya — perhitungannya ada
// di langkah-langkah activity, bukan di dalam SQL.
//
// Akibatnya modul inilah yang memikul aturan bisnisnya, dan bukan basis data. Itu justru
// yang dikehendaki `D-02`; yang perlu dijaga adalah **kesetaraan hasilnya** (`P-5`).
//
// # Empat komponen, dan arah pitanya yang TIDAK seragam
//
// Setiap komponen menghasilkan satu persentase, lalu persentase itu dicari di tabel pita
// `POOLDATA.M_KPI_PNC` untuk memperoleh nilai 1–5. Yang mudah terlewat: **dua dari empat
// pita tersusun menurun** — makin kecil persentasenya, makin tinggi nilainya.
//
//	UPDATE PROGRESS KLAIM   0–20 → 5  …  35–100 → 1     MENURUN
//	SLA KLAIM               0–20 → 5  …  35–100 → 1     MENURUN
//	ANALISA KLAIM          80–100 → 5 …   0–65  → 1     menaik
//	AKSEPTASI KLAIM        80–100 → 5 …   0–65  → 1     menaik
//
// Pita menurun berarti tabelnya disusun untuk **persentase TERLAMBAT**. Tetapi activity lama
// mengirim persentase **TEPAT WAKTU** untuk keempatnya. Perilaku itu direplikasi apa adanya
// (`P-5`) dan dinyatakan sebagai selisih terencana — lihat `PICTeknikPlannedDifferences`.
//
// # Tidak ada satu pun JSON di tab ini
//
// Keempat kelompok kueri membaca **kolom relasional langsung** — `T_CLAIM_PNC`,
// `T_CLAIM_ADJUSTMENT`, `PEGA_DASHBOARDPNC`, `GCNM_PROGRESS_CLAIM`. Tidak ada `JSON_VALUE`
// maupun `JSON_TABLE` di jalur ini, sehingga `DB-3` tidak tertekan sama sekali di sini.

// BandTable adalah tabel tangga nilai tab KPI PIC Teknik.
//
// Ia BERBEDA dari `SourceTable` milik tab Adjuster, dan disebut terpisah karena penguji yang
// membandingkan nilai 1–5 akan mencari tabel inilah.
const BandTable = "POOLDATA.M_KPI_PNC"

// Nama JOB pada `POOLDATA.M_KPI_PNC`, dipakai apa adanya sebagai kunci pencarian pita.
//
// Nilainya ditulis PERSIS seperti di basis data, termasuk huruf besarnya: ia kunci
// pencocokan, bukan label. Salah satu huruf berarti pita tidak ditemukan dan nilainya kosong
// — tanpa galat.
const (
	JobProgress   = "UPDATE PROGRESS KLAIM"
	JobAnalysis   = "ANALISA KLAIM"
	JobAcceptance = "AKSEPTASI KLAIM"
	JobSLA        = "SLA KLAIM"
)

// Kode komponen tab KPI PIC Teknik pada kontrak API.
const (
	PICComponentProgress   = "update_progress"
	PICComponentAnalysis   = "analisa_klaim"
	PICComponentAcceptance = "akseptasi_klaim"
	PICComponentSLA        = "sla_klaim"
)

// Nilai kolom `NOTE` pada baris `SLA KLAIM`, yang memilih ambang harinya.
//
// Diturunkan dari `LEADER_MEMBER` klaim, bukan dari peran pengguna.
const (
	TeamLeader = "LEADER"
	TeamMember = "MEMBER"
)

// PICComponent adalah satu komponen penilaian PIC Teknik.
type PICComponent struct {
	// Code adalah kunci komponen pada kontrak API.
	Code string

	// Label adalah judul barisnya di layar, mengikuti teks activity lama APA ADANYA.
	Label string

	// Job adalah nilai kolom `JOB` pada `M_KPI_PNC` untuk mencari pitanya.
	Job string

	// Order adalah urutan barisnya, mengikuti penanda `TKA` pada activity lama.
	Order int

	// Weighted menyatakan komponen ini melewati pembobotan `GetBobotNilaiKPIPNC`.
	//
	// Hanya Progress yang melewatinya; ketiga lainnya memakai nilai pita apa adanya. Itu
	// bukan penyederhanaan — ketiga activity lain memang tidak memanggil rule pembobotnya.
	Weighted bool

	// Weight adalah `Param.Bobot` yang dikirim ke pembobot. Berarti hanya bila Weighted.
	Weight float64

	// DescendingBand menyatakan pita komponen ini tersusun MENURUN.
	//
	// Bukan dipakai menghitung — pencarian pita tidak peduli arahnya. Ia dipakai layar dan
	// berkas ekspor untuk menandai baris yang nilainya berlawanan arah dengan persentasenya,
	// supaya pembaca tidak menyimpulkan angkanya rusak.
	DescendingBand bool
}

// picComponents adalah keempat komponen, dalam urutan `TKA` activity lama.
//
// Label "max terlambat 25%" diambil dari `GetReportKPI_Progress`, yang menulis baris PIC.
// `PNCReportKPI_act` menulis "max terlambat 20%" untuk baris Leader — **kedua teks itu
// berselisih di sistem lama**, dan keduanya dibawa apa adanya. Lihat
// `PICTeknikPlannedDifferences`.
var picComponents = []PICComponent{
	{
		Code:           PICComponentProgress,
		Label:          "Update Status Progress (max terlambat 25%)",
		Job:            JobProgress,
		Order:          1,
		Weighted:       true,
		Weight:         15,
		DescendingBand: true,
	},
	{
		Code:  PICComponentAnalysis,
		Label: "Analisa Klaim (max 10 hari)",
		Job:   JobAnalysis,
		Order: 2,
	},
	{
		Code:  PICComponentAcceptance,
		Label: "Akseptasi Klaim ( 1 hari )",
		Job:   JobAcceptance,
		Order: 3,
	},
	{
		Code:           PICComponentSLA,
		Label:          "SLA Klaim",
		Job:            JobSLA,
		Order:          4,
		DescendingBand: true,
	},
}

// PICComponents mengembalikan keempat komponen dalam urutan barisnya.
func PICComponents() []PICComponent {
	out := make([]PICComponent, len(picComponents))
	copy(out, picComponents)
	return out
}

// FindPICComponent mencari satu komponen menurut kodenya.
func FindPICComponent(code string) (PICComponent, bool) {
	for _, component := range picComponents {
		if component.Code == code {
			return component, true
		}
	}
	return PICComponent{}, false
}

// BusinessLine adalah lini bisnis yang dipilih pengguna pada tab KPI PIC Teknik.
//
// Nilainya dikirim apa adanya ke kolom `TYPE_BUSINESS` pada `MST_USER_TEKNIK`, dan
// menentukan penyaring Group Panel yang dipakai keempat kueri.
type BusinessLine string

// Keempat lini, persis nilai yang di-set `PNCReportKPI_act` ke `TempLaporan.UserTeknis`.
const (
	LineNonMBU  BusinessLine = "NONMBU"
	LinePA      BusinessLine = "PA"
	LineTravel  BusinessLine = "TRAVEL"
	LineBonding BusinessLine = "BONDING"
)

// BusinessLineOption adalah satu pilihan dropdown lini bisnis.
type BusinessLineOption struct {
	Code  BusinessLine
	Title string

	// Note menjelaskan penyaringnya dengan kata-kata, bukan dengan potongan SQL.
	Note string
}

// businessLines adalah keempat pilihan beserta penyaring Group Panel-nya.
//
// Penyaringnya BUKAN daftar Group Panel yang rapi: Bonding dipisahkan lewat `GROUPBISNISID`,
// dan keempat kode itu justru DIKELUARKAN dari Non-MBU. Artinya Non-MBU dan Bonding saling
// meniadakan pada kolom yang berbeda — bukan dua nilai pada kolom yang sama.
var businessLines = []BusinessLineOption{
	{
		Code:  LineNonMBU,
		Title: "NON MBU",
		Note:  "Group Panel 003, 004, dan 006 — tanpa klaim Bonding.",
	},
	{Code: LinePA, Title: "PA", Note: "Group Panel 002."},
	{Code: LineTravel, Title: "TRAVEL", Note: "Group Panel 005."},
	{
		Code:  LineBonding,
		Title: "BONDING",
		Note:  "Dipilih lewat kode kelompok bisnis, bukan lewat Group Panel.",
	},
}

// BusinessLines mengembalikan keempat pilihan lini bisnis.
func BusinessLines() []BusinessLineOption {
	out := make([]BusinessLineOption, len(businessLines))
	copy(out, businessLines)
	return out
}

// FindBusinessLine mencari satu lini menurut kodenya, tanpa memedulikan huruf besar-kecil.
func FindBusinessLine(code string) (BusinessLineOption, bool) {
	wanted := strings.ToUpper(strings.TrimSpace(code))
	for _, line := range businessLines {
		if string(line.Code) == wanted {
			return line, true
		}
	}
	return BusinessLineOption{}, false
}

// slaExcludedPICs adalah petugas yang DIKECUALIKAN dari penilaian SLA Klaim.
//
// Disaring `@contains(.UserTeknis, …)` di `GetReportKPI_SLA` — pencocokan SEBAGIAN nama,
// bukan kesamaan penuh, dan itu direplikasi apa adanya.
//
// Ini hardcode `D-15` yang paling terang di modul ini: siapa yang dinilai ditentukan dengan
// menyebut nama orang di dalam kode. Keduanya tetap dinilai pada ketiga komponen lain.
var slaExcludedPICs = []string{
	"BAMBANGSETIADJIGUNAWAN",
	"DANIELLISWANDI",
}

// ExcludedFromSLA menyatakan seorang PIC tidak dinilai pada komponen SLA Klaim.
func ExcludedFromSLA(pic string) bool {
	name := strings.ToUpper(strings.TrimSpace(pic))
	for _, excluded := range slaExcludedPICs {
		if strings.Contains(name, excluded) {
			return true
		}
	}
	return false
}

// SLAExcludedPICs mengembalikan daftar yang dikecualikan, untuk ditampilkan di keterangan.
func SLAExcludedPICs() []string {
	out := make([]string, len(slaExcludedPICs))
	copy(out, slaExcludedPICs)
	return out
}

// PICTeknikTab mengembalikan tab KPI PIC Teknik.
func PICTeknikTab() Tab {
	tab, _ := FindTab(TabPICTeknik)
	return tab
}

// PICTeknikGrid mengembalikan grid tab KPI PIC Teknik.
func PICTeknikGrid() Grid {
	for _, grid := range PICTeknikTab().Grids {
		if grid.Code == GridPICScorecard {
			return grid
		}
	}
	return Grid{}
}

// Band adalah satu pita nilai dari `POOLDATA.M_KPI_PNC`.
type Band struct {
	// Job adalah nama komponennya pada tabel pita.
	Job string

	// Value adalah nilai 1–5 yang diberikan bila persentasenya masuk pita ini.
	Value float64

	// Bottom dan Top adalah batas pitanya, KEDUANYA INKLUSIF — mengikuti
	// `BOTTOM <= x AND TOP >= x` pada kueri lama.
	//
	// Akibat langsungnya: pita bertetangga BERTINDIH di titik batasnya. Nilai tepat 20
	// memenuhi pita 0–20 dan pita 20–25 sekaligus. Lihat BandFor.
	Bottom float64
	Top    float64

	// Note adalah kolom `NOTE`; pada `SLA KLAIM` ia berisi LEADER atau MEMBER.
	Note string
}

// Matches menyatakan sebuah persentase masuk pita ini.
func (b Band) Matches(percent float64) bool {
	return b.Bottom <= percent && b.Top >= percent
}

// BandFor mencari nilai untuk sebuah persentase, meniru `GetNilaiKPIPIC`.
//
// # Kenapa ia mengambil yang PERTAMA cocok, bukan yang paling tepat
//
// Kueri lama tidak punya `ORDER BY` dan activity-nya memakai `pxResults(1)` — baris pertama
// yang dikembalikan basis data. Karena pita bertetangga bertindih di titik batas, hasilnya
// pada nilai seperti 20 atau 80 **tidak dapat ditentukan** di sistem lama: ia bergantung pada
// urutan baris yang kebetulan dikembalikan Oracle.
//
// Di sini urutannya dibuat tetap — pita dibaca terurut menurut `ID`, dan yang pertama cocok
// yang dipakai. Itu **mempersempit** perilaku lama, bukan mengubahnya: salah satu dari dua
// jawaban yang sama-sama mungkin di Pega menjadi satu-satunya jawaban di sini. Dinyatakan
// sebagai selisih terencana, karena pada titik batas hasilnya dapat berbeda dari Pega.
func BandFor(bands []Band, percent float64) (Band, bool) {
	for _, band := range bands {
		if band.Matches(percent) {
			return band, true
		}
	}
	return Band{}, false
}

// PICRow adalah satu baris penilaian — satu komponen untuk satu PIC.
type PICRow struct {
	// PIC adalah `OPERATOR_ID` petugasnya, atau "Leader" pada baris rekapitulasi.
	PIC string

	// Component adalah kode komponennya.
	Component string

	// Label adalah judul barisnya.
	Label string

	// Total adalah pembagi — berapa banyak yang dinilai.
	Total float64

	// Achieved adalah pembilang — berapa banyak yang tepat waktu.
	Achieved float64

	// Percent adalah persentase yang dipakai mencari pita.
	Percent Score

	// Value adalah nilai 1–5 hasil pencarian pita, atau kosong bila pitanya tidak ditemukan.
	Value Score
}

// PICScorecard adalah kartu skor satu PIC: keempat komponennya beserta rekapitulasinya.
type PICScorecard struct {
	PIC    string
	Leader bool
	Rows   []PICRow

	// Weighted adalah nilai berbobot komponen Progress, berskala 0–15.
	//
	// Ia BUKAN pengganti nilai pita pada baris Progress — keduanya disimpan di kolom yang
	// berbeda di sistem lama dan keduanya ditampilkan. Kosong bila tidak dapat dihitung.
	Weighted Score
}

// PICTeknikResult adalah keluaran lengkap tab: kartu skor tiap PIC, lalu baris Leader.
type PICTeknikResult struct {
	Line BusinessLine

	// Scorecards adalah kartu skor per PIC, terurut menurut `OPERATOR_ID`.
	Scorecards []PICScorecard

	// Leader adalah rekapitulasi seluruh PIC, meniru `TempKPILeader` activity lama.
	Leader PICScorecard
}
