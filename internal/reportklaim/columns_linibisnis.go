package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Lini Bisnis & Mitra Kerja Sama.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsKlaimHE — 27 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimHE_act-Act.xml`.
var colsKlaimHE = []Column{
	{Header: "No Klaim", Field: "ClaimNo"},
	{Header: "No Polis", Field: "PolicyNo"},
	{Header: "PIC Teknik", Field: "PICRekanan"},
	{Header: "Register Date", Field: "RegisterDate"},
	{Header: "Date Of Loss", Field: "DateOfLoss"},
	{Header: "QQ Name", Field: "UserName"},
	{Header: "Cause Of Loss", Field: "CauseOfLoss"},
	{Header: "Sumber Bisnis", Field: "BusinessName"},
	{Header: "Lokasi", Field: "Location"},
	{Header: "Sparepart", Field: "AlasanKlaim"},
	{Header: "Diskon Sparepart", Field: "AlasanTerlambat"},
	{Header: "Net Sparepart", Field: "City"},
	{Header: "Jasa", Field: "District"},
	{Header: "Diskon Jasa", Field: "BranchName"},
	{Header: "Net Jasa", Field: "ConsultantName"},
	{Header: "Bengkel", Field: "InsuredName"},
	{Header: "Lokasi Bengkel", Field: "NamaDokumen"},
	{Header: "Supplier", Field: "NamaSurveyor"},
	{Header: "Okupasi", Field: "Occupation"},
	{Header: "Model", Field: "UserTeknis"},
	{Header: "Merek", Field: "UserTeknisEmail"},
	{Header: "Tahun", Field: "TreatyName"},
	{Header: "No Mesin", Field: "TreatyYear"},
	{Header: "No Rangka", Field: "ClientName"},
	{Header: "Reserve", Field: "Remark"},
	{Header: "Total Klaim", Field: "EmailTertanggung"},
	{Header: "Total Net Biaya Klaim", Field: "Province"},
}

// colsRegistSimasOnline — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimHE_act-Act.xml`.
var colsRegistSimasOnline = []Column{
	{Header: "Tanggal Lapor", Field: "EDMDATE"},
	{Header: "NO RCV", Field: "EDMNO"},
	{Header: "NO Klaim", Field: "IDPEGA"},
	{Header: "NO Polis", Field: "NOPOLIS"},
	{Header: "Nama Tertanggung", Field: "QQNAME"},
	{Header: "Tanggal Kejadian", Field: "STARTDATE"},
	{Header: "Bisnis", Field: "SOBNAME"},
	{Header: "Close Date", Field: "ENDDATE"},
	{Header: "Close Note", Field: "BUSINESSCODE"},
	{Header: "Cabang", Field: "MARKETINGNAME"},
	{Header: "Status", Field: "CLIENTID"},
	{Header: "PIC", Field: "WARRANTYNO"},
}

// Tiga susunan kolom berdiri di sini sampai 2026-10-13 — colsKlaimAsuransiKredit,
// colsKlaimPerBisnis, dan colsKlaimTravel. Ketiganya dicabut bersama panelnya, yang
// tidak ada di Pega produksi. Lihat catatan di kepala catalogLiniBisnis.
