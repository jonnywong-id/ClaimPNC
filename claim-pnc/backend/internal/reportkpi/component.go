package reportkpi

import "strings"

// Component adalah satu komponen penilaian KPI adjuster.
//
// Kesembilannya sama persis di grid Summary dan grid Detail — itulah sebabnya keduanya
// memakai daftar yang SATU ini, bukan dua daftar yang harus dijaga kesamaannya.
type Component struct {
	// Code adalah kunci komponen pada kontrak API dan pada peta Scores.
	//
	// Ia berbahasa Indonesia karena ia KONTRAK yang dibaca layar, sama sifatnya dengan
	// nama field JSON (`D-80`).
	Code string

	// Label adalah judul kolom yang dilihat pengguna.
	//
	// Diambil dari caption section lama APA ADANYA, termasuk huruf kapitalnya (`D-13`).
	// Seluruhnya terverifikasi ada di `Section/ReportKPI_Section-Section.xml`.
	Label string

	// Column adalah nama kolom pada POOLDATA.DETAIL_KPI_ADJUSTER.
	//
	// Ia dicantumkan di lapisan domain — bukan disembunyikan di lapisan SQL — karena
	// itulah satu-satunya tempat ketiga penamaan dapat dibandingkan sekaligus: kolom,
	// caption layar, dan alias Pega yang menyesatkan. Tidak ada satu pun kueri yang
	// dirangkai dari nilai ini; berkas .sql menyebut kolomnya sendiri.
	Column string

	// LegacyAlias adalah alias yang dipakai kueri Pega untuk kolom ini.
	//
	// Dicatat supaya penelusuran ke sistem lama tetap mungkin, dan supaya terlihat betapa
	// jauhnya alias itu dari isinya — "NameOfBank" untuk nama adjuster, "AlasanKlaim"
	// untuk nilai propose. Ini utang teknis §4.2 yang `D-19` hapus dari sistem baru.
	LegacyAlias string
}

// Kode komponen KPI adjuster.
const (
	ComponentSurvey            = "penjadwalan_survey"
	ComponentImmediateAdvice   = "immediate_advice"
	ComponentPreliminaryAdvice = "preliminary_advice"
	ComponentInterimReport     = "interim_report"
	ComponentProgress          = "update_progress"
	ComponentCommunication     = "tanggapan_komunikasi"
	ComponentPropose           = "propose_adjustment"
	ComponentFinalReport       = "final_report"
	ComponentTotal             = "nilai"
)

// components adalah kesembilan komponen, DALAM URUTAN KOLOM LAYAR LAMA.
//
// Urutannya mengikuti urutan SELECT pada `RDB List/GetSummaryKPIAdjuster-SQL.xml`, yang
// sama pula dengan urutan parameter `Database/INSERT_KPIADJUSTER.prc`. Keduanya sepakat,
// dan kesepakatan itulah yang menjadikan urutan ini fakta, bukan pilihan.
//
// `ComponentTotal` sengaja ikut di dalam daftar ini meski ia bukan komponen melainkan
// NILAI AKHIR — lihat catatan di bawah.
var components = []Component{
	{
		Code: ComponentSurvey, Label: "PENJADWALAN SURVEY",
		Column: "SURVEYLAP", LegacyAlias: "ProdKe",
	},
	{
		Code: ComponentImmediateAdvice, Label: "IMMEDIATE ADVICE",
		Column: "IMMEDIATEADVICE", LegacyAlias: "CityID",
	},
	{
		Code: ComponentPreliminaryAdvice, Label: "PRELIMINARY ADVICE",
		Column: "PRELIMINARYADVICE", LegacyAlias: "ResponseNote",
	},
	{
		Code: ComponentInterimReport, Label: "INTERIM REPORT",
		Column: "INTERIM", LegacyAlias: "ReporterName",
	},
	{
		Code: ComponentProgress, Label: "UPDATE PROGRESS",
		Column: "PROGRESS", LegacyAlias: "CountryID",
	},
	{
		Code: ComponentCommunication, Label: "TANGGAPAN KOMUNIKASI",
		Column: "KOMUNIKASI", LegacyAlias: "RWID",
	},
	{
		Code: ComponentPropose, Label: "PROPOSE ADJUSTMENT",
		Column: "PROPOSE", LegacyAlias: "AlasanKlaim",
	},
	{
		Code: ComponentFinalReport, Label: "FINAL REPORT",
		Column: "FINALREPORT", LegacyAlias: "ProvinceID",
	},
	{
		// NILAI adalah kolom TERSENDIRI di basis data, bukan jumlah kedelapan di atasnya.
		//
		// Ini mudah disalahpahami, dan salah paham itu mahal: bila layar menghitungnya
		// sendiri dari kedelapan komponen, angkanya akan BERBEDA dari Pega tanpa satu pun
		// galat. Pega merata-ratakan kolom `NILAI` seperti kolom lain
		// (`round(avg(to_number(nilai)),2)`), dan bagaimana `INSERT_KPIADJUSTER` mengisinya
		// ditentukan activity penilai yang tidak dibangun di sini.
		//
		// Karena itu ia DIBACA, tidak pernah dihitung.
		Code: ComponentTotal, Label: "NILAI",
		Column: "NILAI", LegacyAlias: "Keyword",
	},
}

// Components mengembalikan salinan daftar komponen, dalam urutan kolom layar.
//
// Salinan, bukan slice aslinya: pemanggil di lapisan transport menyusunnya menjadi
// jawaban JSON, dan slice yang dibagikan dapat diubah tanpa sengaja oleh salah satunya.
func Components() []Component {
	result := make([]Component, len(components))
	copy(result, components)
	return result
}

// ComponentCodes mengembalikan kode kesembilan komponen dalam urutan kolom layar.
func ComponentCodes() []string {
	codes := make([]string, 0, len(components))
	for _, c := range components {
		codes = append(codes, c.Code)
	}
	return codes
}

// FindComponent mencari komponen menurut kodenya.
func FindComponent(code string) (Component, bool) {
	wanted := strings.TrimSpace(code)
	for _, c := range components {
		if c.Code == wanted {
			return c, true
		}
	}
	return Component{}, false
}

// ReportType adalah tipe laporan yang dipilih pengguna pada dropdown "Pilih Tipe Report".
//
// Nilainya SAMA PERSIS dengan isi kolom `TIPE` di basis data, dan itu disengaja: ia
// dipakai sebagai pembanding kueri apa adanya. Menerjemahkannya menjadi kode lain hanya
// menambah satu tabel pemetaan yang dapat salah tanpa ketahuan.
type ReportType string

// Ketiga tipe laporan yang dikenal.
//
// `TypeOutstanding` dan `TypeFinal` benar-benar ada sebagai nilai kolom `TIPE`;
// `TypeAll` TIDAK — ia bukan nilai kolom melainkan permintaan MENGGABUNGKAN keduanya,
// dan di Pega dilayani kueri tersendiri yang ber-`UNION ALL`
// (`RDB List/GetSummaryKPIAdjusterALL-SQL.xml`).
//
// Perbedaan itu penting: pada `TypeAll` satu adjuster muncul DUA baris, dan baris keduanya
// bukan duplikat.
const (
	TypeOutstanding ReportType = "OUTSTANDING"
	TypeFinal       ReportType = "FINAL"
	TypeAll         ReportType = "ALL"
)

// ReportTypeOption adalah satu pilihan pada dropdown "Pilih Tipe Report".
type ReportTypeOption struct {
	Code  ReportType
	Label string

	// Note menjelaskan apa yang dilihat pengguna bila memilihnya.
	//
	// Ia ADA karena satu pilihan berperilaku berbeda dari dua lainnya, dan perbedaan itu
	// tidak terbaca dari namanya sendiri.
	Note string
}

// reportTypes adalah ketiga pilihan, dalam urutan yang sama dengan layar lama.
var reportTypes = []ReportTypeOption{
	{
		Code: TypeOutstanding, Label: "OUTSTANDING",
		Note: "Kasus survei yang masih berjalan.",
	},
	{
		Code: TypeFinal, Label: "FINAL",
		Note: "Kasus survei yang laporan akhirnya sudah masuk.",
	},
	{
		Code: TypeAll, Label: "ALL",
		Note: "Keduanya sekaligus. Satu adjuster tampil dua baris — satu per tipe — " +
			"dan itu bukan baris ganda.",
	},
}

// ReportTypes mengembalikan salinan ketiga pilihan tipe laporan.
func ReportTypes() []ReportTypeOption {
	result := make([]ReportTypeOption, len(reportTypes))
	copy(result, reportTypes)
	return result
}

// FindReportType mencari tipe laporan menurut kodenya, mengabaikan besar-kecil huruf.
//
// Besar-kecil huruf diabaikan pada PEMBACAAN saja; yang dikembalikan selalu bentuk baku
// huruf besar, karena itulah yang dibandingkan dengan isi kolom `TIPE`.
func FindReportType(code string) (ReportType, bool) {
	wanted := strings.ToUpper(strings.TrimSpace(code))
	for _, t := range reportTypes {
		if string(t.Code) == wanted {
			return t.Code, true
		}
	}
	return "", false
}

// SplitsIntoTypes menyebut tipe mana saja yang benar-benar dibaca dari kolom `TIPE`.
//
// `TypeAll` menghasilkan dua; dua lainnya menghasilkan dirinya sendiri. Ia dipakai
// pengisi seam memori supaya perilakunya sama dengan `UNION ALL` di SQL tanpa
// menduplikasi aturannya.
func (t ReportType) SplitsIntoTypes() []ReportType {
	if t == TypeAll {
		return []ReportType{TypeOutstanding, TypeFinal}
	}
	return []ReportType{t}
}
