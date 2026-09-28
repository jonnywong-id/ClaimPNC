package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Operasional.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsMitra — 13 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCMitraReport_Act-Act.xml`.
var colsMitra = []Column{
	{Header: "Login Aplikasi", Field: "QQNAME"},
	{Header: "Posisi", Field: "NOPOLIS"},
	{Header: "Atasan", Field: "THEINSURED"},
	{Header: "No Klaim", Field: "IDPEGA"},
	{Header: "Jenis Klaim", Field: "BUSINESSCODE"},
	{Header: "Tgl Awal", Field: "SOBLEADER0"},
	{Header: "Tgl Akhir", Field: "SOBLEADER1"},
	{Header: "AGING", Field: "SOBLEADER2"},
	{Header: "Sesuai SLA", Field: "AUTOCANCELPRINTSTATUS"},
	{Header: "Jml Sesuai SLA", Field: "BRANCHNAME"},
	{Header: "Jml Tdk Sesuai SLA", Field: "ACCUMCODE"},
	{Header: "Persentase Sesuai SLA", Field: "BUSINESSNAME"},
	{Header: "Total Produktivitas", Field: "BRANCHCODE"},
}

// colsProduksiPA — 15 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportProduksiPA_act-Act.xml`.
var colsProduksiPA = []Column{
	{Header: "Tahun", Field: "CityID"},
	{Header: "Resiko A", Field: "AlasanDokterRejectRCL"},
	{Header: "Jumlah Klaim", Field: "CompliancePosAuditByr"},
	{Header: "Tahun", Field: "CaseID"},
	{Header: "Resiko B", Field: "Country"},
	{Header: "Jumlah Klaim", Field: "ComplianceRemark"},
	{Header: "Tahun", Field: "City"},
	{Header: "Resiko D", Field: "AnalystDoctorRemaks"},
	{Header: "Jumlah Klaim", Field: "Conveyance"},
	{Header: "Tahun", Field: "AnaylstRemarks"},
	{Header: "Resiko MC", Field: "AnalystRemaksInvestigator"},
	{Header: "Jumlah Klaim", Field: "CustomerPrinciple"},
	{Header: "Tahun", Field: "ReporterName"},
	{Header: "Resiko LAINNYA", Field: "CloseClaimNote"},
	{Header: "Jumlah Klaim", Field: "District"},
}

// colsCompliance — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCComplianceReport_Act-Act.xml`.
var colsCompliance = []Column{
	{Header: "No Klaim", Field: "CaseID"},
	{Header: "No Polis", Field: "ClaimNo"},
	{Header: "Tanggal Mulai Polis", Field: "AnaylstRemarks"},
	{Header: "Tanggal Berakhir Polis", Field: "CountryID"},
	{Header: "Tanggal Kerugian", Field: "AnalystDoctorRemaks"},
	{Header: "Nama Tertanggung", Field: "CityID"},
	{Header: "Lokasi Kerugian", Field: "Location"},
	{Header: "Nama Claimant", Field: "ReporterName"},
	{Header: "Nilai KlaimPropose", Field: "ClaimEstimate"},
	{Header: "Nilai Klaim Paid", Field: "Country"},
	{Header: "Komentar Compliance", Field: "Remark"},
	{Header: "Tanggal Compliance", Field: "NoteKasir"},
}

// colsAdjuster — 10 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCAdjusterReport_Act-Act.xml`.
var colsAdjuster = []Column{
	{Header: "No Klaim", Field: "IDSurvey"},
	{Header: "No Polis", Field: "IDObject"},
	{Header: "Nama Tertanggung", Field: "InsuredPIC"},
	{Header: "Nama Bisnis", Field: "CouseOfLos"},
	{Header: "Nilai Reserve Klaim ASM", Field: "Salvage"},
	{Header: "Tgl Pengajuan Survey", Field: "BodyLetterOP"},
	{Header: "Nama", Field: "SurveyorName"},
	{Header: "Catatan", Field: "KeteranganLain"},
	{Header: "PIC Teknis", Field: "AdjusterPIC"},
	{Header: "Status", Field: "AdjusterStatus"},
}

// colsKomunikasiKlaim — 7 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKominukasiAdjusterKlaim-Act.xml`.
var colsKomunikasiKlaim = []Column{
	{Header: "Nomor Klaim", Field: "pzInsKey"},
	{Header: "Tanggal kirim", Field: "CloseClaimDate"},
	{Header: "Message", Field: "Email"},
	{Header: "Pengirim", Field: "UserName"},
	{Header: "Message Balasan", Field: "CloseClaimNote"},
	{Header: "Tanggal Balas", Field: "AnalystTransferDate"},
	{Header: "Nama_Replay", Field: "UserAdmin"},
}
