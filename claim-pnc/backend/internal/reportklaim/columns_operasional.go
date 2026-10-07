package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Operasional.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsMitra — 13 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCMitraReport_Act-Act.xml`.
var colsMitra = parseColumns(`
Login Aplikasi	QQNAME
Posisi	NOPOLIS
Atasan	THEINSURED
No Klaim	IDPEGA
Jenis Klaim	BUSINESSCODE
Tgl Awal	SOBLEADER0
Tgl Akhir	SOBLEADER1
AGING	SOBLEADER2
Sesuai SLA	AUTOCANCELPRINTSTATUS
Jml Sesuai SLA	BRANCHNAME
Jml Tdk Sesuai SLA	ACCUMCODE
Persentase Sesuai SLA	BUSINESSNAME
Total Produktivitas	BRANCHCODE
`)

// colsProduksiPA — 15 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportProduksiPA_act-Act.xml`.
var colsProduksiPA = parseColumns(`
Tahun	CityID
Resiko A	AlasanDokterRejectRCL
Jumlah Klaim	CompliancePosAuditByr
Tahun	CaseID
Resiko B	Country
Jumlah Klaim	ComplianceRemark
Tahun	City
Resiko D	AnalystDoctorRemaks
Jumlah Klaim	Conveyance
Tahun	AnaylstRemarks
Resiko MC	AnalystRemaksInvestigator
Jumlah Klaim	CustomerPrinciple
Tahun	ReporterName
Resiko LAINNYA	CloseClaimNote
Jumlah Klaim	District
`)

// colsCompliance — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCComplianceReport_Act-Act.xml`.
var colsCompliance = parseColumns(`
No Klaim	CaseID
No Polis	ClaimNo
Tanggal Mulai Polis	AnaylstRemarks
Tanggal Berakhir Polis	CountryID
Tanggal Kerugian	AnalystDoctorRemaks
Nama Tertanggung	CityID
Lokasi Kerugian	Location
Nama Claimant	ReporterName
Nilai KlaimPropose	ClaimEstimate
Nilai Klaim Paid	Country
Komentar Compliance	Remark
Tanggal Compliance	NoteKasir
`)

// colsAdjuster — 10 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCAdjusterReport_Act-Act.xml`.
var colsAdjuster = parseColumns(`
No Klaim	IDSurvey
No Polis	IDObject
Nama Tertanggung	InsuredPIC
Nama Bisnis	CouseOfLos
Nilai Reserve Klaim ASM	Salvage
Tgl Pengajuan Survey	BodyLetterOP
Nama	SurveyorName
Catatan	KeteranganLain
PIC Teknis	AdjusterPIC
Status	AdjusterStatus
`)

// colsKomunikasiKlaim — 7 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKominukasiAdjusterKlaim-Act.xml`.
var colsKomunikasiKlaim = parseColumns(`
Nomor Klaim	pzInsKey
Tanggal kirim	CloseClaimDate
Message	Email
Pengirim	UserName
Message Balasan	CloseClaimNote
Tanggal Balas	AnalystTransferDate
Nama_Replay	UserAdmin
`)
