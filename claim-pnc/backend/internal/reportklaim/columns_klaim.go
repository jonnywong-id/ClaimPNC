package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Klaim.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsTATPersonal — 52 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCTATReport1_Act-Act.xml`.
var colsTATPersonal = parseColumns(`
No Klaim	CaseID
RCV	RefNo
No polis	ClaimNo
Nama tertanggung	LossCoverage
Sumber bisnis	DollarCurrencyVal
Coverage	UserTeknisGroup
Status Klaim	AnalystDoctorRemaks
Posisi	TelpTertanggung
No Akseptasi	IsAnalisTransfer
Jumlah yang diakseptasi	FlagReject
Tgl receive dokumen	CommentKomiteClosecase
tgl masuk analis	KirimAnalystDate
tgl analis-invest	KirimInvestDate
Tgl registrasi	City
Tgl tf invest - analyst	IdxSurveyResults
Tgl tf analyst - PUCL/RCL	ClaimEstimate
Tgl tf RCL/PUCL-Analyst	AlasanTerlambat
RCL/PUCL/Notifikasi	ExGratia
Tgl tf analyst - compliance	Province
tgl tf complicance-analyst	IsCFS_PNC
Tgl bayar CPL	CompliancePosAuditByr
Tgl postAudit CPL	IsBackCFS
Tgl bayar CPL postAuditIs	IsTransferPIC
TransferPIC	Location
Tgl tf analyst - komite	AnaylstRemarks
invest 1/0	IsInvest
compliance 1/0	isCompliance
Tgl komite akseptasi	StatusWork
Tgl krm propose adjustment	CountryID
Tgl akseptasi	ComplianceRemark
Tgl Bayar	UserBusinessPA
Tipe Pembayaran	ReporterName
Status bayar	DistrictID
Tgl tutup klaim	ExGratiaNote
Alasan keterlambatan	Remark
Inputor	pyLabel
Tgl Cetak LOD	ResponseNote
Tgl Trf Investigator	DateKomite
Tgl Terima LOD	ResponseNote
Tgl Transfer Kasir	pyEmailAddress
Report Date	CASEDB
Start Date Polis	ClaimNoSRB
End Date Polis	NoPla
Date Of Accident	DaftarObjek
Remark	Status
Tgl Analis To Inputor	HideKTP
Risk Location	RiskLocation
DOB	DOB
Usia pada saat klaim	USIA
Diagnosa	Diagnose
Nature Of Loss	NatureOfLoss
Perhitungan jarak antara terbit polis - tanggal kejadian	PolicyRange
`)

// colsTATNonMBU — 39 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCTATReport1_Act-Act.xml`.
var colsTATNonMBU = parseColumns(`
No Klaim	CaseID
No Polis	ClaimNo
Nama tertanggung	LossCoverage
Nama bisnis	AlasanDokterRejectRCL
Sumber bisnis	DollarCurrencyVal
Klaim Leader/Member	NamaDokterRCL
Nomor Akseptasi	IsAnalisTransfer
Tahun Of Loss	CauseOfLoss
Tgl kejadian	DaftarObjek
Tgl terima dokumen	CommentKomiteClosecase
Tgl Registrasi	City
Tgl selesai registrasi	Location
Tgl tf PLA	InsuredRelationshipOthers
Lama proses regis-tf ke teknik	CloseClaimNote
Tgl permintaan kelengkapan dokumen	isComplianceTransfer
Tgl Survey	UserAdmin
Tgl aksep komite	AnaylstRemarks
Tgl Selesai Cek Komite	LokasiSurveyor
Tgl krm propose adjustment	CountryID
Tgl Terima LOD	ResponseNote
Tgl Akseptasi	ComplianceRemark
Tgl akseptasi manual	IsTransferAnalisator
Tgl DLA	RCV_ID
Tgl bayar	UserBusinessPA
Status Bayar	DistrictID
Lama regis - tanggal trf kasir	CityID
Lama aksep - trf kasir	OccupationCode
Lama akseptasi - tanggal bayar	Conveyance
Status klaim	AnalystDoctorRemaks
PIC Teknik	NoKTP
Tgl tutup kaim	ExGratiaNote
ctt tutup kaim	NIK
Alasan keterlambatan	Remark
Tgl Trf PIC	NewNoKTP
PIC Admin	pyLabel
Tgl Dok Lengkap	pyGroup
Tgl Trf Komite	pyCountryName
Tgl Trf Kasir	NewEmail
Report Date	CASEDB
`)

// colsTATBonding — 34 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCTATReport1_Act-Act.xml`.
var colsTATBonding = parseColumns(`
No Klaim	CaseID
No Polis	ClaimNo
Nama tertanggung	LossCoverage
Nama bisnis	AlasanDokterRejectRCL
Sumber bisnis	DollarCurrencyVal
Klaim Leader/Member	NamaDokterRCL
Nomor Akseptasi	IsAnalisTransfer
Tahun Of Loss	CauseOfLoss
Tgl kejadian	DaftarObjek
Tgl terima dokumen	CommentKomiteClosecase
Tgl Registrasi	City
Tgl selesai registrasi	AnalystRemaksInvestigator
Tgl tf ke PIC Teknik	Location
Tgl tf PLA	InsuredRelationshipOthers
Lama proses regis-tf ke teknik	CloseClaimNote
Tgl permintaan kelengkapan dokumen	isComplianceTransfer
Tgl Survey	UserAdmin
Tgl aksep komite	AnaylstRemarks
Tgl Selesai Cek Komite	LokasiSurveyor
Tgl krm propose adjustment	CountryID
Tgl Terima LOD	ResponseNote
Tgl Akseptasi	ComplianceRemark
Tgl akseptasi manual	IsTransferAnalisator
Tgl DLA	RCV_ID
Tgl bayar	UserBusinessPA
Status Bayar	DistrictID
Lama regis - request bayar	CityID
Lama akseptasi - request bayar	Conveyance
Status klaim	AnalystDoctorRemaks
PIC Teknik	NoKTP
Tgl tutup kaim	ExGratiaNote
ctt tutup kaim	NIK
Alasan keterlambatan	Remark
Report Date	CASEDB
`)

// colsKlaimHarianRingkas — 10 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportHarian_act1-Act.xml`.
var colsKlaimHarianRingkas = parseColumns(`
Tanggal regist	City
No Polis	AlasanDokterRejectRCL
Atas Nama tertanggung	AlasanTerlambat
No Klaim	AnalystDoctorRemaks
DOL	AnaylstRemarks
Tanggal Transfer PIC	District
Tanggal Terima Dokumen	DistrictID
Nilai Klaim(100%)	CloseClaimNote
Nilai Klaim(Share ASM)	ComplianceRemark
PIC Teknik	ExGratiaNote
`)

// colsKlaimHarian — 30 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportHarian_act1-Act.xml`.
var colsKlaimHarian = parseColumns(`
No Polis	NoKTP
PNC CaseID	CaseID
Nama Tertanggung	NamaSurveyor
COB	City
PIC Teknik	UserAdmin
Email	AnalystTransferDate
Tgl Regis	Email
Bulan Klaim	DateOfLoss
Date of Loss	CityID
SOB	AgingAmount
Leader/ Member/Fac In	Country
Nature of Loss	Country
Cause of Loss	AlasanTerlambat
ASM Share	CloseClaimNote
Deductible	CommentKomiteClosecase
Nilai Share ASM	ComplianceRemark
Nilai Klaim 100%	Conveyance
Adjuster fee 100 % Share	UserName
Nilai Klaim Net 100 %	UserTeknis
Nilai Klaim Net ASM Share	CompliancePosAuditByr
Lack of Doc/Salvage/Recovery/Subrogaration	AnalystRemaksInvestigator
Cabang	ClaimNo
Status	LokasiSurveyor
Kronologis	NIK
Tgl terima dokumen	NPWP
Tgl tf ke PIC Teknik	TransferPICDate
Remark	AnaylstRemarks
PIC Admin Regist	NewEmail
Location	Location
No Reff Broker	NoReffBroker
`)

// colsRejectKlaim — 33 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataReject_act-Act.xml`.
var colsRejectKlaim = parseColumns(`
No Klaim	CaseID
Nopolis	NoKTP
QQName	ClaimID
Nama Bisnis	ClaimNo
PIC Teknik	Conveyance
Tgl Registrasi	Country
Tgl Kejadian	CountryID
Tgl Reject	Email
Bulan Reject	FlagASO
Alasan Reject	CityID
Estimasi Value	Resources
Share ASM	ClaimAmount
Persen OR	OwnRisk
Reject Note	CloseClaimNote
Coins	FlagReject
PSRSPL	InsuredRelationship
QS_RI	PolicyObjectLocationRWNote
ER1	IsBackCFS
ER2	IsCFS
SURPLUS1	isComplianceTransfer
SURPLUS2	IsTransferAnalisator
PSRQS_RI	IsTransferPIC
PSRQS_OR	InsuredRelationshipOthers
ORS	KomiteStatus
Facultative	District
Facobl	LokasiSurveyor
BPPDAN	LossCoverage
XL	NamaDokumen
PSS	NamaSurveyor
PRGBI	NIK
FESPL	AnalystRemaksInvestigator
PFRA	ProdKe
Fac_Out	AlasanDokterRejectRCL
`)

// colsCloseKlaimRingkas — 11 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataClose_act-Act.xml`.
var colsCloseKlaimRingkas = parseColumns(`
No Klaim	CaseID
Nopolis	NoKTP
QQName	ClaimID
Nama Bisnis	ClaimNo
PIC Teknik	Conveyance
Tgl Registrasi	Country
Tgl Kejadian	CountryID
Tgl Akseptasi	District
No Akseptasi	DistrictID
Tgl Close	Email
Bulan Close	FlagASO
`)

// colsCloseKlaim — 81 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataClose_act-Act.xml`.
var colsCloseKlaim = parseColumns(`
No Klaim	CaseID
Bisnis	KodeCabang
Remark PIC	AlasanDokterRejectRCL
No Polis	TKI
Client Name	AnalystDoctorRemaks
Nama Mo	AnalystRemaksInvestigator
Okupasi	AnaylstRemarks
No Aksep	City
Cause Of Loss	CauseOfLoss
Coverage Name	CityID
Sts Exgratia	ClaimEstimate
Date Of Loss	StatusWork
Month DOL	ClaimNo
Year DOL	CloseClaimNote
Prepare Date	UserTeknisEmail
Thn Regis	CommentKomiteClosecase
THN POLIS	RefNo
Tgl Aksep	DollarCurrencyVal
Month Aksep	CompliancePosAuditByr
Year Aksep	ComplianceRemark
Tgl Reject	UserTeknis
Month Reject	Conveyance
Thn Reject	Country
Begin Date	UserTeknisGroup
End Date	RemarkRecommendation
TAT	UserName
TSI	ProvinceID
Total Aksep	CountryID
Below 100jt	DokumenLengkap
Total OS	Currency
Reject Klaim	Province
LEADER/MEMBER/FACIN	DaftarObjek
Risk Loc	Location
Risk Loc Klaim	pyID
PIC Klaim	DistrictID
Own Retension	ExGratiaNote
Coins	FlagReject
PSRSPL	InsuredRelationship
QS_RI	Investigasi
ER1	IsBackCFS
ER2	IsCFS_PNC
SURPLUS1	isComplianceTransfer
SURPLUS2	IsTransferAnalisator
PSRQS_RI	IsTransferPIC
PSRQS_OR	InsuredRelationshipOthers
ORS	KomiteStatus
Facultative	District
Facobl	LokasiSurveyor
BPPDAN	LossCoverage
XL	NamaDokumen
PSS	NamaSurveyor
PRGBI	NIK
FESPL	NoKTP
PFRA	ProdKe
Nilai Klaim yang Diajukan Tertanggung	Password
Adjustment Klaim (Self Adjustment)	RCV_ID
Adjustment Klaim (Adjuster)	ReportAddress
Selisih Estimasi Klaim dengan Adjustment	Remark
Adjuster Fee	ReporterName
Nilai Klaim yang Diajukan Tertanggung OR	ReportDescription
Adjustment Klaim (Self Adjustment) OR	StatusClaim
Adjustment Klaim (Adjuster) OR	TelpTertanggung
Selisih Estimasi Klaim dengan Adjustment OR	UserAdmin
Nilai Salvage	RWID
Dokumen Penalti	ReporterTelp
Status Progress 1	AgingAmount
Status Progress 2	AlasanTerlambat
Tgl Terakhir Update Progress	ReceiverClaim
Jumlah FU Terlambat	ResponseNote
Jumlah Tidak FU Terlambat	StatusKomunikasi
Keterangan	pyNote
Sumber Bisnis	UserBusinessPA
Ceding/Leader	OwnRisk
Surveyor	SuspiciousComment
Loss Adjuster	StatusAnalystRemarks
Next Follow Up	StatusReceiver
Tgl Regist	NewNoKTP
Tgl Close Claim	NewEmail
Tgl Tf PIC	NewTelpTertanggung
Alasan Reject	AlasanDokterRejectRCL
Dominan Factor	DominanName
`)

// colsAIKlaim — 14 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportHasilDataAIKlaim-Act.xml`.
var colsAIKlaim = parseColumns(`
No Klaim	CaseID
No Polis	PolicyNo
Tertangung	City
Bisnis Name	CityID
Tgl Kejadian	NoKTP
Start Polis	CityID
End Polis	Country
ResultsAi	CountryID
Coverage AI Final	AlasanKlaim
Cause Of Los (AI)	NoteKasir
Coverage Kronologi (AI)	FlagASO
Tipe Note AI	NamaSurveyor
Note AI	DistrictID
	District
`)
