package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok PLA / DLA.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsPLA — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPLA_act-Act.xml`.
var colsPLA = []Column{
	{Header: "No Klaim", Field: "CaseID"},
	{Header: "Nopolis", Field: "NoKTP"},
	{Header: "Nama Bisnis", Field: "ClaimID"},
	{Header: "Nama Tertanggung", Field: "ClaimNo"},
	{Header: "DOL", Field: "RefNo"},
	{Header: "Tgl Regist", Field: "Remark"},
	{Header: "PIC", Field: "RW"},
	{Header: "No PLA", Field: "Conveyance"},
	{Header: "Tanggal PLA", Field: "Country"},
	{Header: "Bulan", Field: "District"},
	{Header: "Reinsurer", Field: "CountryID"},
	{Header: "Klaim Leader/Member", Field: "Email"},
}

// colsDLA — 16 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataDLA_act-Act.xml`.
var colsDLA = []Column{
	{Header: "No Klaim", Field: "CaseID"},
	{Header: "Nopolis", Field: "NoKTP"},
	{Header: "Nama Bisnis", Field: "ClaimID"},
	{Header: "Nama Tertanggung", Field: "ClaimNo"},
	{Header: "DOL", Field: "RefNo"},
	{Header: "Tgl Regist", Field: "Remark"},
	{Header: "PIC", Field: "RW"},
	{Header: "No DLA", Field: "Conveyance"},
	{Header: "Nilai DLA", Field: "Province"},
	{Header: "No Aksep", Field: "District"},
	{Header: "Revisi DLA", Field: "NIK"},
	{Header: "Tanggal DLA", Field: "Country"},
	{Header: "Bulan", Field: "Location"},
	{Header: "Reinsurer", Field: "CountryID"},
	{Header: "Tanggal Aksep", Field: "DistrictID"},
	{Header: "Klaim Leader/Member", Field: "Email"},
}

// colsPengirimanPLA — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPengirimanPLA_act-Act.xml`.
var colsPengirimanPLA = []Column{
	{Header: "No Policy", Field: "AlasanKlaim"},
	{Header: "Nama Tertanggung", Field: "Keyword"},
	{Header: "DOL", Field: "Other"},
	{Header: "Nomor Klaim", Field: "CaseID"},
	{Header: "Nomor PLA", Field: "City"},
	{Header: "Nilai PLA", Field: "Amount"},
	{Header: "PLA Reinsurer", Field: "CityID"},
	{Header: "TGL PLA", Field: "District"},
	{Header: "TGL Kirim", Field: "DistrictID"},
	{Header: "TGL Terima PLA", Field: "Country"},
	{Header: "Status Kirim", Field: "CountryID"},
	{Header: "PIC", Field: "UserTeknis"},
}

// colsPengirimanDLA — 9 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPengirimanDLA_act-Act.xml`.
var colsPengirimanDLA = []Column{
	{Header: "Nomor Klaim", Field: "CaseID"},
	{Header: "Nomor DLA", Field: "City"},
	{Header: "Nilai Share DLA", Field: "Amount"},
	{Header: "DLA Reinsurer", Field: "CityID"},
	{Header: "TGL DLA", Field: "District"},
	{Header: "TGL Kirim", Field: "DistrictID"},
	{Header: "TGL Terima DLA", Field: "Country"},
	{Header: "Statur Kirim", Field: "CountryID"},
	{Header: "PIC", Field: "Keyword"},
}
