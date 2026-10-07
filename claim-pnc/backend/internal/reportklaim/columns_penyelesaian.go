package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Akseptasi & Penyelesaian.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsKasir — 9 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportTransferKeKasir-Act.xml`.
var colsKasir = parseColumns(`
No Klaim	CaseID
Nama Tertanggung	NamaSurveyor
PIC	NIK
No Akseptasi	City
Tanggal Akseptasi	CityID
Nama Penerima	Country
ASM Share	NoKTP
ASM Share Value	NPWP
Tgl Transfer keKasir	CountryID
`)

// colsPendingLOD — 11 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataPendingLOD-Act.xml`.
var colsPendingLOD = parseColumns(`
No Klaim	ClaimNo
Nopolis	PolicyNo
Nama Tertanggung	InsuredName
Status ASM	Status
PIC	UserTeknis
Tgl Kejadian	CityID
Tgl Kirim LOD	City
Kurs	Currency
Nilai Pengajuan Tertanggung	ClientID
Nilai Klaim	ClientName
Nilai Deductible	CloseClaimNote
`)

// colsAkseptasiRinci — 42 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportAkseptasiNonMBU-Act.xml`.
var colsAkseptasiRinci = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
TGL TRF KOMITE	Email
TGL KOMITE AKSEP	District
TGL AKSEPTASI	AlasanKlaim
CURRENCY	NPWP
TTLOS	ProdKe
NO AKSEPTASI	DistrictID
SHARE ASM (%)	ASMShare
ASM SHARE VALUE	KomiteStatus
GROSS VALUE	Currency
OR ASM	Location
COAS	Status
BPPDAN	LossCoverage
FACULTATIVE	LokasiSurveyor
QS RI	IsTransferPIC
PSPL	NIK
FESPL	IsAnalisTransfer
FSPL	AnaylstRemarks
PSRSPL	AnalystDoctorRemaks
PQS RI	BranchID
FACOBLIG	BranchName
FACOBSRB	BusinessID
FACOBINDT	CASEDB
ER1	CauseOfLoss
ER2	CauseOfLossID
PSS	City
PRGBI	CityID
PFRA	ClientID
XL	ClientName
Occupation	TreatyName
`)

// colsAkseptasiRingkas — 26 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportAkseptasiNonMBU-Act.xml`.
var colsAkseptasiRingkas = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
TGL TRF KOMITE	Email
TGL KOMITE AKSEP	District
TGL AKSEPTASI	AlasanKlaim
CURRENCY	NPWP
TTLOS	ProdKe
NO AKSEPTASI	DistrictID
SHARE ASM (%)	ASMShare
ASM SHARE VALUE	KomiteStatus
GROSS VALUE	Currency
OR	Location
QS	IsTransferPIC
SURPLUS	ReporterName
OTHER	RefNo
`)

// colsOSKomiteRinci — 43 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSKomiteNonMBU-Act.xml`.
var colsOSKomiteRinci = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
TGL TRF KOMITE	Email
TGL KOMITE AKSEP	District
CURRENCY	NPWP
TTLOS	ProdKe
BALANCE OS	FlagASO
SHARE ASM (%)	ASMShare
STS COAS	Status
GROSS VALUE	Currency
ASM SHARE VALUE	KomiteStatus
NILAI COAS	TKA
OR ASM	Location
BPPDAN	LossCoverage
FACULTATIVE	LokasiSurveyor
QS RI	IsTransferPIC
PSPL	NIK
FESPL	IsAnalisTransfer
FSPL	AnaylstRemarks
PSRSPL	AnalystDoctorRemaks
PQS RI	BranchID
FACOBLIG	BranchName
FACOBSRB	BusinessID
FACOBINDT	CASEDB
ER1	CauseOfLoss
ER2	CauseOfLossID
PSS	City
PRGBI	CityID
PFRA	ClientID
XL	ClientName
STATUS	TKI
DOMINAN FACTOR	DominanName
`)

// colsOSKomiteRingkas — 27 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSKomiteNonMBU-Act.xml`.
var colsOSKomiteRingkas = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
TGL TRF KOMITE	Email
TGL KOMITE AKSEP	District
CURRENCY	NPWP
TTLOS	ProdKe
BALANCE OS	FlagASO
SHARE ASM (%)	ASMShare
GROSS VALUE	Currency
ASM SHARE VALUE	KomiteStatus
NILAI COAS	TKA
OR	Location
QS	IsTransferPIC
SURPLUS	ReporterName
OTHER	RefNo
DOMINAN FACTOR	DominanName
`)

// colsOSBelumKomiteRinci — 36 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSBlmKomiteNonMBU-Act.xml`.
var colsOSBelumKomiteRinci = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
CURRENCY	NPWP
TTLOS	ProdKe
SHARE ASM (%)	ASMShare
STS COAS	Status
ASM SHARE VALUE	KomiteStatus
OR ASM	Location
BPPDAN	LossCoverage
FACULTATIVE	LokasiSurveyor
QS RI	IsTransferPIC
PSPL	NIK
FESPL	IsAnalisTransfer
FSPL	AnaylstRemarks
PSRSPL	AnalystDoctorRemaks
PQS RI	BranchID
FACOBLIG	BranchName
FACOBSRB	BusinessID
FACOBINDT	CASEDB
ER1	CauseOfLoss
ER2	CauseOfLossID
PSS	City
PRGBI	CityID
PFRA	ClientID
XL	ClientName
`)

// colsOSBelumKomiteRingkas — 21 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSBlmKomiteNonMBU-Act.xml`.
var colsOSBelumKomiteRingkas = parseColumns(`
CLAIMNO	CaseID
NOPOLIS	NoKTP
QQNAME	ClaimID
PICTEKNIK	Conveyance
BUSINESSNAME	ClaimNo
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
REGISTERDATE	Country
CURRENCY	NPWP
TTLOS	ProdKe
SHARE ASM (%)	ASMShare
ASM SHARE VALUE	KomiteStatus
OR	Location
QS	IsTransferPIC
SURPLUS	ReporterName
OTHER	RefNo
`)

// colsKomiteRingkas — 11 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKomites_act-Act.xml`.
var colsKomiteRingkas = parseColumns(`
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

// colsKomite — 80 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKomites_act-Act.xml`.
var colsKomite = parseColumns(`
CLAIMNO	CaseID
Komite ID	ClaimNoExt
Nama Komite	NamaPrincipal
Status Approve	StatusApprove
KomiteAccepted	Note Komite
NOPOLIS	NoKTP
QQNAME	ClaimID
BUSINESS NAME	Resources
PICTEKNIK	ConsultantName
CAUSE OF LOSS	InsuredName
COVERAGE NAME	TelpTertanggung
COB	ClaimNo
CEDING/LEADER	OwnRisk
SUMBIS	IsPLA
NAMA MO	IsSendPremi
UW YEAR	Remark
SOURCE	UserTeknisEmail
OCCUPATION CODE	UserTeknis
OCCUPATION	UserName
START DATE	RCV_ID
END DATE	RW
DATEOFLOSS	CountryID
BULAN DOL	AreaClaimId
TAHUN DOL	ContractNo
REGISTERDATE	Country
TGL CLOSE	Conveyance
BULAN CLOSE	CopyFrom
TAHUN CLOSE	CreateFrom
TGL TRF KOMITE	Email
TGL KOMITE AKSEP	District
TGL AKSEPTASI	AlasanKlaim
EX GRATIA	AnalystRemaksInvestigator
TGL TRF PIC	EmailTertanggung
REGIST - CLOSE	ExGratiaNote
LOD - TRF KASIR	FlagASO
STS ASM	DaftarObjek
CURRENCY	NPWP
TIPE AKSEPTASI	IBNR
NO AKSEPTASI	DistrictID
TSI	IdxSurveyResults
TTLOS	ProdKe
NILAI YANG DIAJUKAN TERTANGGUNG	OldNoHp
SELISIH ESTIMASI DENGAN AKSEPTASI	IDMaster
SHARE ASM (%)	ASMShare
ASM SHARE VALUE	KomiteStatus
GROSS VALUE	Currency
OR ASM	Location
COAS	Status
BPPDAN	LossCoverage
FACULTATIVE	LokasiSurveyor
QS RI	IsTransferPIC
PSPL	NIK
FESPL	IsAnalisTransfer
FSPL	AnaylstRemarks
PSRSPL	AnalystDoctorRemaks
PQS RI	BranchID
FACOBLIG	BranchName
FACOBSRB	BusinessID
FACOBINDT	CASEDB
ER1	CauseOfLoss
ER2	CauseOfLossID
PSS	City
PRGBI	CityID
PFRA	ClientID
XL	ClientName
NO REF BROKER	NoReffBroker
STATUS BANDING	Banding
SURVEY KEPUASAN	SurveyKepuasan
EFFORT CLOSE	EffortClose
KENDALA CLOSE	KendalaClose
MANUAL/PAPERLESS	Paperless
USULAN	Usulan
ADJUSTER	FlagNOLL
ADJUSTER MARINE	FlagReject
STS PROGRESS 1	AgingAmount
STS PROGRESS 2	AlasanTerlambat
KETERANGAN	pyNote
CLOSE CLAIM NOTE	NoteKasir
Alasan Tolak 1	KodeCabang
Alasan Tolak 2	KodeKBRU
`)
