package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok PLA / DLA.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsPLA — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPLA_act-Act.xml`.
var colsPLA = parseColumns(`
No Klaim	CaseID
Nopolis	NoKTP
Nama Bisnis	ClaimID
Nama Tertanggung	ClaimNo
DOL	RefNo
Tgl Regist	Remark
PIC	RW
No PLA	Conveyance
Tanggal PLA	Country
Bulan	District
Reinsurer	CountryID
Klaim Leader/Member	Email
`)

// colsDLA — 16 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataDLA_act-Act.xml`.
var colsDLA = parseColumns(`
No Klaim	CaseID
Nopolis	NoKTP
Nama Bisnis	ClaimID
Nama Tertanggung	ClaimNo
DOL	RefNo
Tgl Regist	Remark
PIC	RW
No DLA	Conveyance
Nilai DLA	Province
No Aksep	District
Revisi DLA	NIK
Tanggal DLA	Country
Bulan	Location
Reinsurer	CountryID
Tanggal Aksep	DistrictID
Klaim Leader/Member	Email
`)

// colsPengirimanPLA — 12 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPengirimanPLA_act-Act.xml`.
var colsPengirimanPLA = parseColumns(`
No Policy	AlasanKlaim
Nama Tertanggung	Keyword
DOL	Other
Nomor Klaim	CaseID
Nomor PLA	City
Nilai PLA	Amount
PLA Reinsurer	CityID
TGL PLA	District
TGL Kirim	DistrictID
TGL Terima PLA	Country
Status Kirim	CountryID
PIC	UserTeknis
`)

// colsPengirimanDLA — 9 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataPengirimanDLA_act-Act.xml`.
var colsPengirimanDLA = parseColumns(`
Nomor Klaim	CaseID
Nomor DLA	City
Nilai Share DLA	Amount
DLA Reinsurer	CityID
TGL DLA	District
TGL Kirim	DistrictID
TGL Terima DLA	Country
Statur Kirim	CountryID
PIC	Keyword
`)
