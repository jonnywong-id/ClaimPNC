package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Lini Bisnis & Mitra Kerja Sama.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsKlaimHE — 27 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimHE_act-Act.xml`.
var colsKlaimHE = parseColumns(`
No Klaim	ClaimNo
No Polis	PolicyNo
PIC Teknik	PICRekanan
Register Date	RegisterDate
Date Of Loss	DateOfLoss
QQ Name	UserName
Cause Of Loss	CauseOfLoss
Sumber Bisnis	BusinessName
Lokasi	Location
Sparepart	AlasanKlaim
Diskon Sparepart	AlasanTerlambat
Net Sparepart	City
Jasa	District
Diskon Jasa	BranchName
Net Jasa	ConsultantName
Bengkel	InsuredName
Lokasi Bengkel	NamaDokumen
Supplier	NamaSurveyor
Okupasi	Occupation
Model	UserTeknis
Merek	UserTeknisEmail
Tahun	TreatyName
No Mesin	TreatyYear
No Rangka	ClientName
Reserve	Remark
Total Klaim	EmailTertanggung
Total Net Biaya Klaim	Province
`)

// colsRegistSimasOnline — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimHE_act-Act.xml`.
var colsRegistSimasOnline = parseColumns(`
Tanggal Lapor	EDMDATE
NO RCV	EDMNO
NO Klaim	IDPEGA
NO Polis	NOPOLIS
Nama Tertanggung	QQNAME
Tanggal Kejadian	STARTDATE
Bisnis	SOBNAME
Close Date	ENDDATE
Close Note	BUSINESSCODE
Cabang	MARKETINGNAME
Status	CLIENTID
PIC	WARRANTYNO
`)

// colsKlaimAsuransiKredit — 3 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimKredit_act-Act.xml`.
var colsKlaimAsuransiKredit = parseColumns(`
User Input	EDMNO
Sumber Bisnis	PRODKE
Total Klaim	CLIENTID
`)

// colsKlaimPerBisnis — 14 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataClaimBusiness-Act.xml`.
var colsKlaimPerBisnis = parseColumns(`
NO KLAIM	ClaimNo
NO POLIS	PolicyNo
SUMBIS	SIM
PERIODE AWAL POLIS	City
PERIODE AKHIR POLIS	CityID
TANGGAL REGIST	Country
DOL	CountryID
PIC	UserTeknis
TSI	TKI
NILAI ESTIMASI	CaseID
NILAI AKSEPTASI	RefNo
STATUS KLAIM	StatusClaim
EMAIL	NewEmail
No. TELP	NewTelpTertanggung
`)

// colsKlaimTravel — 6 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataClaimTravel-Act.xml`.
var colsKlaimTravel = parseColumns(`
NO POLIS	PolicyNo
NO KLAIM	ClaimNo
TGL REGIST	BranchID
NAMA PESERTA	BranchName
NO AKSEPTASI	CaseID
NILAI AKSEP	CauseOfLoss
`)
