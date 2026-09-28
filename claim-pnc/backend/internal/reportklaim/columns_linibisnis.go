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

// colsKlaimAsuransiKredit — 3 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportKlaimKredit_act-Act.xml`.
var colsKlaimAsuransiKredit = []Column{
	{Header: "User Input", Field: "EDMNO"},
	{Header: "Sumber Bisnis", Field: "PRODKE"},
	{Header: "Total Klaim", Field: "CLIENTID"},
}

// colsKlaimPerBisnis — 14 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataClaimBusiness-Act.xml`.
var colsKlaimPerBisnis = []Column{
	{Header: "NO KLAIM", Field: "ClaimNo"},
	{Header: "NO POLIS", Field: "PolicyNo"},
	{Header: "SUMBIS", Field: "SIM"},
	{Header: "PERIODE AWAL POLIS", Field: "City"},
	{Header: "PERIODE AKHIR POLIS", Field: "CityID"},
	{Header: "TANGGAL REGIST", Field: "Country"},
	{Header: "DOL", Field: "CountryID"},
	{Header: "PIC", Field: "UserTeknis"},
	{Header: "TSI", Field: "TKI"},
	{Header: "NILAI ESTIMASI", Field: "CaseID"},
	{Header: "NILAI AKSEPTASI", Field: "RefNo"},
	{Header: "STATUS KLAIM", Field: "StatusClaim"},
	{Header: "EMAIL", Field: "NewEmail"},
	{Header: "No. TELP", Field: "NewTelpTertanggung"},
}

// colsKlaimTravel — 6 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataClaimTravel-Act.xml`.
var colsKlaimTravel = []Column{
	{Header: "NO POLIS", Field: "PolicyNo"},
	{Header: "NO KLAIM", Field: "ClaimNo"},
	{Header: "TGL REGIST", Field: "BranchID"},
	{Header: "NAMA PESERTA", Field: "BranchName"},
	{Header: "NO AKSEPTASI", Field: "CaseID"},
	{Header: "NILAI AKSEP", Field: "CauseOfLoss"},
}
