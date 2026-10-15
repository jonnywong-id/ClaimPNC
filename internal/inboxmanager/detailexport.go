package inboxmanager

import (
	"strings"
	"time"
)

// Export Data Detail Klaim — blok "Export Data Detail Klaim" pada tab Outstanding.
//
// Asal: tombol `pyButtonLabel Export To Excel` di `Section/PNCDashboardOS`, yang memanggil
// `Activity/ExportDataDetailKlaim`. Activity itu menjalankan `GetExportDataDetailKlaim`,
// melengkapi setiap barisnya, lalu menulis berkas lewat `pxConvertResultsToCSV`.
//
// Ia ekspor yang BERDIRI SENDIRI: tidak menyaring menurut PIC yang diklik, Reinsurer, maupun
// Kategori OS — kueri dan activity-nya tidak membaca satu pun dari ketiganya. Yang
// menyaringnya hanya lini bisnis pemanggil dan rentang tanggal "Dari"/"Sampai".

// DetailExportFilename adalah nama berkasnya, apa adanya dari `FileName "Data Detail Klaim"`
// pada pemanggilan `pxConvertResultsToCSV` (`AppendTimeStampToFileName False`).
const DetailExportFilename = "Data Detail Klaim.csv"

// DetailExportColumn adalah satu kolom berkas.
type DetailExportColumn struct {
	// Title adalah judul kolom, disalin APA ADANYA dari `CSVPropHeaders` — termasuk spasi
	// di depan lima judul (" Kanwil", " Keterangan", …). Spasi itu ada di berkas lama, dan
	// membuangnya membuat makro atau rumus yang mencari judulnya berhenti menemukannya.
	Title string

	// Property adalah nama properti Pega yang mengisi kolom itu, dari `CSVProperties`
	// dengan urutan yang sama. Ia kunci baris DetailExportRow.
	//
	// Namanya menyesatkan — "UserTeknis" berisi tanggal reject, "Password" berisi nilai
	// klaim — karena kueri Pega memaksa aliasnya cocok dengan properti klipboard yang sudah
	// ada. Nama itu tetap dipakai supaya setiap kolom dapat ditelusuri balik ke activity
	// dan kuerinya tanpa tabel pemetaan.
	Property string
}

// DetailExportColumns adalah kesembilan puluh kolom berkas, berurutan seperti di Pega.
//
// Pasangannya dibaca dari `Activity/ExportDataDetailKlaim-Act.xml` — `CSVPropHeaders` dan
// `CSVProperties` pada langkah `Call pxConvertResultsToCSV`, keduanya berisi 90 butir.
var DetailExportColumns = []DetailExportColumn{
	{"No Klaim", "CaseID"},
	{"Status Work", "pyStatusMessage"},
	{"COB", "AlasanDokterRejectRCL"},
	{"No Polis", "TKI"},
	{"Cabang Polis", "Usulan"},
	{" Kanwil", "ResponseMsg"},
	{"Client Name", "AnalystDoctorRemaks"},
	{"Nama Mo", "AnalystRemaksInvestigator"},
	{"Okupasi", "AnaylstRemarks"},
	{"No Aksep", "City"},
	{"Cause Of Loss", "CauseOfLoss"},
	{"Coverage Name", "CityID"},
	{"Sts Exgratia", "ClaimEstimate"},
	{"Date Of Loss", "StatusWork"},
	{"Month DOL", "ClaimNo"},
	{"Year DOL", "CloseClaimNote"},
	{"Prepare Date", "UserTeknisEmail"},
	{"Thn Regis", "CommentKomiteClosecase"},
	{"THN POLIS", "RefNo"},
	{"Tgl Aksep", "DollarCurrencyVal"},
	{"Month Aksep", "CompliancePosAuditByr"},
	{"Year Aksep", "ComplianceRemark"},
	{"Tgl Reject", "UserTeknis"},
	{"Month Reject", "Conveyance"},
	{"Thn Reject", "Country"},
	{"Begin Date", "UserTeknisGroup"},
	{"End Date", "RemarkRecommendation"},
	{"TAT", "UserName"},
	{"TSI", "ProvinceID"},
	{"Total Aksep", "CountryID"},
	{"Below 100jt", "DokumenLengkap"},
	{"Total OS", "Currency"},
	{"Reject Klaim", "Province"},
	{"LEADER/MEMBER/FACIN", "DaftarObjek"},
	{"Risk Loc", "Location"},
	{"Risk Loc Klaim", "pyID"},
	{"PIC Klaim", "DistrictID"},
	{"Own Retension", "ExGratiaNote"},
	{"Coins", "FlagReject"},
	{"PSRSPL", "InsuredRelationship"},
	{"QS_RI", "PolicyObjectLocationRWNote"},
	{"ER1", "IsBackCFS"},
	{"ER2", "IsCFS_PNC"},
	{"SURPLUS1", "isComplianceTransfer"},
	{"SURPLUS2", "IsTransferAnalisator"},
	{"PSRQS_RI", "IsTransferPIC"},
	{"PSRQS_OR", "InsuredRelationshipOthers"},
	{"ORS", "KomiteStatus"},
	{"Facultative", "District"},
	{"Facobl", "LokasiSurveyor"},
	{"BPPDAN", "LossCoverage"},
	{"XL", "NamaDokumen"},
	{"PSS", "NamaSurveyor"},
	{"PRGBI", "NIK"},
	{"FESPL", "NoKTP"},
	{"PFRA", "ProdKe"},
	{"Nilai Klaim yang Diajukan Tertanggung", "Password"},
	{"Adjustment Klaim (Self Adjustment)", "RCV_ID"},
	{"Adjustment Klaim (Adjuster)", "ReportAddress"},
	{"Selisih Estimasi Klaim dengan Adjustment", "StatusEkonomis"},
	{"Selisih Estimasi Klaim dengan Adjuster", "Remark"},
	{"Adjuster Fee", "ReporterName"},
	{"Nilai Klaim yang Diajukan Tertanggung OR", "ReportDescription"},
	{"Adjustment Klaim (Self Adjustment) OR", "StatusClaim"},
	{"Adjustment Klaim (Adjuster) OR", "TelpTertanggung"},
	{"Selisih Estimasi Klaim dengan Adjustment OR", "UserAdmin"},
	{"Selisih Estimasi Klaim dengan Adjuster OR", "OldAddress"},
	{"Nilai Salvage", "RWID"},
	{"Status Progress 1", "AgingAmount"},
	{"Tipe Dokumen", "ReporterTelp"},
	{"Status Progress 2", "AlasanTerlambat"},
	{"Tgl Terakhir Update Progress", "ReceiverClaim"},
	{"Jumlah FU Terlambat", "ResponseNote"},
	{"Jumlah Tidak FU Terlambat", "StatusKomunikasi"},
	{" Keterangan", "pyNote"},
	{" Sumber Bisnis", "UserBusinessPA"},
	{"Ceding/Leader", "OwnRisk"},
	{"Surveyor", "SuspiciousComment"},
	{"Loss Adjuster", "StatusAnalystRemarks"},
	{" Next Follow Up", "StatusReceiver"},
	{" Nama Objek", "TypeOfAddress"},
	{" Alasan Tolak Klaim", "TypeOldNoHp"},
	{"SubjectEmail", "SubjectEmail"},
	{"Report Name", "Resources"},
	{"No Surat", "viewnotelp"},
	{"Tgl Kirim", "view_ktp"},
	{"Tgl Jatuh Tempo", "view_email"},
	{"No Polis Leader", "PolicyLeaderNo"},
	{"Pinwil", "HideKTP"},
	{"Kanwil", "HidePelapor"},
}

// DetailExportEmptyColumns adalah properti yang SELALU kosong di berkas, beserta sebabnya.
//
// Ia data, bukan komentar, supaya uji dapat memastikan tidak ada kolom LAIN yang kosong tanpa
// sebab yang tercatat.
var DetailExportEmptyColumns = map[string]string{
	"IsCFS_PNC": "Kosong juga di Pega: kuerinya memberi alias `IsCFS`, berkasnya meminta " +
		"`IsCFS_PNC`. Dibiarkan seperti Pega (ketetapan Work Owner 2026-09-25).",
	"SubjectEmail": "Pega membacanya dari objek kerja (`ClaimData.SubjectEmail`). " +
		"`T_CLAIM_PNC.SUBJECTEMAIL` ada, tetapi kosong pada seluruh klaim terbuka (2026-10-14).",
	"PolicyLeaderNo": "Pega membacanya dari objek kerja (`ClaimData.PolicyLeaderNo`). " +
		"`T_CLAIM_PNC.POLISLEADER` ada, tetapi kosong pada seluruh klaim terbuka (2026-10-14).",
}

// DetailExportRow adalah satu baris berkas, dikunci DetailExportColumn.Property.
type DetailExportRow map[string]string

// PortalSimasnet adalah portal yang di Pega bernama aplikasi `SIMASNET`.
//
// `ExportDataDetailKlaim` langkah "Jika SIMASNET, remove filter bisnis" mengosongkan saringan
// lini bisnis bila `TempGetApp.LSC_ID=="SIMASNET"`. Padanan portalnya ASI — sama dengan yang
// dipakai modul Registrasi (`registrasi.CashierCompany`).
const PortalSimasnet = "ASI"

// DetailExportInput adalah isian mentah blok ekspor.
type DetailExportInput struct {
	// From dan Until adalah isian "Dari" dan "Sampai", berbentuk "YYYY-MM-DD".
	From  string
	Until string
}

// DetailExportQuery adalah permintaan ekspor yang sudah tervalidasi.
type DetailExportQuery struct {
	// From adalah batas bawah tanggal klaim dibuat, INKLUSIF. Nol berarti tanpa saringan
	// tanggal.
	From time.Time

	// Until adalah batas atas, EKSKLUSIF — sehari sesudah isian "Sampai". Lihat Period.
	Until time.Time

	// LineBusiness adalah lini bisnis yang menyaring baris — padanan `OperatorID.pyPosition`.
	LineBusiness string

	// AllLines melepas saringan lini bisnis — pemanggil di portal SIMASNET.
	AllLines bool

	// Caller adalah identitas pemanggil.
	Caller Caller
}

// HasPeriod menyatakan apakah ekspor ini disaring tanggal.
func (q DetailExportQuery) HasPeriod() bool {
	return !q.From.IsZero()
}

// NewDetailExportQuery membentuk permintaan ekspor yang sah, atau menyatakan apa yang salah.
//
// # Kedua tanggal kosong SAH, satu saja TIDAK
//
// Activity-nya melewati saringan tanggal hanya bila KEDUANYA kosong
// (`TempBisnis.EDMDATE=="" && TempBisnis.ENDDATE==""`). Bila satu saja terisi, ia merangkai
// `to_date` atas teks kosong untuk yang lain — NULL — sehingga kuerinya mengembalikan NOL baris tanpa
// satu pun pesan. Berkas kosong itu tidak dibawa: permintaannya ditolak dengan alasan yang
// terbaca.
//
// # Kewenangan
//
// Ekspor ini milik tab Outstanding, sehingga hanya pemanggil yang boleh melihat tab itu yang
// boleh menjalankannya.
func NewDetailExportQuery(
	input DetailExportInput,
	caller Caller,
	portalAlias string,
) (DetailExportQuery, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return DetailExportQuery{}, ErrCallerUnknown
	}

	tab, _ := FindTab(TabOutstanding)
	if !CanSee(clean, tab) {
		return DetailExportQuery{}, ErrTabNotAllowed
	}

	query := DetailExportQuery{
		LineBusiness: clean.LineBusiness,
		AllLines:     strings.EqualFold(strings.TrimSpace(portalAlias), PortalSimasnet),
		Caller:       clean,
	}

	fromText, untilText := strings.TrimSpace(input.From), strings.TrimSpace(input.Until)
	if fromText == "" && untilText == "" {
		return query, nil
	}

	violations := []Violation{}
	if fromText == "" || untilText == "" {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Isi tanggal \"Dari\" dan \"Sampai\" keduanya, atau kosongkan keduanya untuk seluruh periode.",
		})
		return DetailExportQuery{}, NewValidationError(violations)
	}

	from, okFrom := parseDate(fromText)
	if !okFrom {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Dari\" tidak dikenali. Contoh yang benar: 2026-09-01.",
		})
	}
	until, okUntil := parseDate(untilText)
	if !okUntil {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Sampai\" tidak dikenali. Contoh yang benar: 2026-09-30.",
		})
	}
	if okFrom && okUntil && until.Before(from) {
		violations = append(violations, Violation{
			Field:   FieldPeriod,
			Message: "Tanggal \"Sampai\" tidak boleh mendahului tanggal \"Dari\".",
		})
	}
	if len(violations) > 0 {
		return DetailExportQuery{}, NewValidationError(violations)
	}

	// Batas atas digeser satu hari supaya EKSKLUSIF sementara tanggal "Sampai" tetap ikut —
	// padanan `trunc(c.pxcreatedatetime) <= to_date(sampai)` tanpa membungkus kolomnya.
	query.From = from
	query.Until = until.AddDate(0, 0, 1)
	return query, nil
}

// FeeBand adalah satu pita tangga fee adjuster (`POOLDATA.GCNM_FEE_SCALE`).
type FeeBand struct {
	Index      int
	LossAmount float64
	Fee        float64
}

// FeeScale menghitung fee adjuster dari sebuah nilai kerugian — pengganti
// `POOLDATA.Get_InterpolasiPNC` (`D-02`). Jawaban `false` berarti tidak dapat dihitung.
type FeeScale interface {
	Fee(lossValue float64) (float64, bool)
}

// ProgressNames menyusun nama tahapan progres dari JSON `JSONSTATUS_PROGRESS2` — pengganti
// `POOLDATA.GET_POSISI_PROGRESS2` (`D-02`).
type ProgressNames interface {
	Positions(rawJSON string) string
}

// DetailRules adalah aturan bisnis yang dipakai ekspor tetapi DIMILIKI modul lain.
//
// # Kenapa ia seam, bukan impor
//
// Kedua fungsi basis data itu sudah ditulis ulang di Go oleh modul Report Klaim
// (`reportklaim.FeeScale`, `reportklaim.ProgressNames`), lengkap dengan ujinya. Menulisnya
// sekali lagi di sini berarti dua salinan aturan uang yang dapat menyimpang tanpa ketahuan;
// mengimpornya langsung melanggar batas modul. Yang dipilih adalah cara yang ditetapkan:
// modul ini menyatakan apa yang dibutuhkannya, dan `cmd/claimpnc` merangkainya.
//
// Fungsi yang nil membuat kolom yang bergantung padanya KOSONG — bukan nol — dan itu yang
// terjadi pada penyimpanan memori dan uji.
type DetailRules struct {
	NewFeeScale      func(bands []FeeBand) FeeScale
	NewProgressNames func(names map[string]string) ProgressNames
}
