package reportklaim

// Tabel kolom berkas CSV untuk laporan kelompok Akseptasi & Penyelesaian.
//
// Judul kolom disalin APA ADANYA dari export, termasuk salah ketik dan spasi
// gandanya — lihat Column.Header pada report.go untuk alasannya.

// colsKasir — 9 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportTransferKeKasir-Act.xml`.
var colsKasir = []Column{
	{Header: "No Klaim", Field: "CaseID"},
	{Header: "Nama Tertanggung", Field: "NamaSurveyor"},
	{Header: "PIC", Field: "NIK"},
	{Header: "No Akseptasi", Field: "City"},
	{Header: "Tanggal Akseptasi", Field: "CityID"},
	{Header: "Nama Penerima", Field: "Country"},
	{Header: "ASM Share", Field: "NoKTP"},
	{Header: "ASM Share Value", Field: "NPWP"},
	{Header: "Tgl Transfer keKasir", Field: "CountryID"},
}

// colsPendingLOD — 11 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ExportDataPendingLOD-Act.xml`.
var colsPendingLOD = []Column{
	{Header: "No Klaim", Field: "ClaimNo"},
	{Header: "Nopolis", Field: "PolicyNo"},
	{Header: "Nama Tertanggung", Field: "InsuredName"},
	{Header: "Status ASM", Field: "Status"},
	{Header: "PIC", Field: "UserTeknis"},
	{Header: "Tgl Kejadian", Field: "CityID"},
	{Header: "Tgl Kirim LOD", Field: "City"},
	{Header: "Kurs", Field: "Currency"},
	{Header: "Nilai Pengajuan Tertanggung", Field: "ClientID"},
	{Header: "Nilai Klaim", Field: "ClientName"},
	{Header: "Nilai Deductible", Field: "CloseClaimNote"},
}

// colsAkseptasiRinci — 42 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportAkseptasiNonMBU-Act.xml`.
var colsAkseptasiRinci = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "TGL TRF KOMITE", Field: "Email"},
	{Header: "TGL KOMITE AKSEP", Field: "District"},
	{Header: "TGL AKSEPTASI", Field: "AlasanKlaim"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "NO AKSEPTASI", Field: "DistrictID"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "GROSS VALUE", Field: "Currency"},
	{Header: "OR ASM", Field: "Location"},
	{Header: "COAS", Field: "Status"},
	{Header: "BPPDAN", Field: "LossCoverage"},
	{Header: "FACULTATIVE", Field: "LokasiSurveyor"},
	{Header: "QS RI", Field: "IsTransferPIC"},
	{Header: "PSPL", Field: "NIK"},
	{Header: "FESPL", Field: "IsAnalisTransfer"},
	{Header: "FSPL", Field: "AnaylstRemarks"},
	{Header: "PSRSPL", Field: "AnalystDoctorRemaks"},
	{Header: "PQS RI", Field: "BranchID"},
	{Header: "FACOBLIG", Field: "BranchName"},
	{Header: "FACOBSRB", Field: "BusinessID"},
	{Header: "FACOBINDT", Field: "CASEDB"},
	{Header: "ER1", Field: "CauseOfLoss"},
	{Header: "ER2", Field: "CauseOfLossID"},
	{Header: "PSS", Field: "City"},
	{Header: "PRGBI", Field: "CityID"},
	{Header: "PFRA", Field: "ClientID"},
	{Header: "XL", Field: "ClientName"},
	{Header: "Occupation", Field: "TreatyName"},
}

// colsAkseptasiRingkas — 26 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportAkseptasiNonMBU-Act.xml`.
var colsAkseptasiRingkas = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "TGL TRF KOMITE", Field: "Email"},
	{Header: "TGL KOMITE AKSEP", Field: "District"},
	{Header: "TGL AKSEPTASI", Field: "AlasanKlaim"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "NO AKSEPTASI", Field: "DistrictID"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "GROSS VALUE", Field: "Currency"},
	{Header: "OR", Field: "Location"},
	{Header: "QS", Field: "IsTransferPIC"},
	{Header: "SURPLUS", Field: "ReporterName"},
	{Header: "OTHER", Field: "RefNo"},
}

// colsOSKomiteRinci — 43 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSKomiteNonMBU-Act.xml`.
var colsOSKomiteRinci = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "TGL TRF KOMITE", Field: "Email"},
	{Header: "TGL KOMITE AKSEP", Field: "District"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "BALANCE OS", Field: "FlagASO"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "STS COAS", Field: "Status"},
	{Header: "GROSS VALUE", Field: "Currency"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "NILAI COAS", Field: "TKA"},
	{Header: "OR ASM", Field: "Location"},
	{Header: "BPPDAN", Field: "LossCoverage"},
	{Header: "FACULTATIVE", Field: "LokasiSurveyor"},
	{Header: "QS RI", Field: "IsTransferPIC"},
	{Header: "PSPL", Field: "NIK"},
	{Header: "FESPL", Field: "IsAnalisTransfer"},
	{Header: "FSPL", Field: "AnaylstRemarks"},
	{Header: "PSRSPL", Field: "AnalystDoctorRemaks"},
	{Header: "PQS RI", Field: "BranchID"},
	{Header: "FACOBLIG", Field: "BranchName"},
	{Header: "FACOBSRB", Field: "BusinessID"},
	{Header: "FACOBINDT", Field: "CASEDB"},
	{Header: "ER1", Field: "CauseOfLoss"},
	{Header: "ER2", Field: "CauseOfLossID"},
	{Header: "PSS", Field: "City"},
	{Header: "PRGBI", Field: "CityID"},
	{Header: "PFRA", Field: "ClientID"},
	{Header: "XL", Field: "ClientName"},
	{Header: "STATUS", Field: "TKI"},
	{Header: "DOMINAN FACTOR", Field: "DominanName"},
}

// colsOSKomiteRingkas — 27 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSKomiteNonMBU-Act.xml`.
var colsOSKomiteRingkas = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "TGL TRF KOMITE", Field: "Email"},
	{Header: "TGL KOMITE AKSEP", Field: "District"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "BALANCE OS", Field: "FlagASO"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "GROSS VALUE", Field: "Currency"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "NILAI COAS", Field: "TKA"},
	{Header: "OR", Field: "Location"},
	{Header: "QS", Field: "IsTransferPIC"},
	{Header: "SURPLUS", Field: "ReporterName"},
	{Header: "OTHER", Field: "RefNo"},
	{Header: "DOMINAN FACTOR", Field: "DominanName"},
}

// colsOSBelumKomiteRinci — 36 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSBlmKomiteNonMBU-Act.xml`.
var colsOSBelumKomiteRinci = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "STS COAS", Field: "Status"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "OR ASM", Field: "Location"},
	{Header: "BPPDAN", Field: "LossCoverage"},
	{Header: "FACULTATIVE", Field: "LokasiSurveyor"},
	{Header: "QS RI", Field: "IsTransferPIC"},
	{Header: "PSPL", Field: "NIK"},
	{Header: "FESPL", Field: "IsAnalisTransfer"},
	{Header: "FSPL", Field: "AnaylstRemarks"},
	{Header: "PSRSPL", Field: "AnalystDoctorRemaks"},
	{Header: "PQS RI", Field: "BranchID"},
	{Header: "FACOBLIG", Field: "BranchName"},
	{Header: "FACOBSRB", Field: "BusinessID"},
	{Header: "FACOBINDT", Field: "CASEDB"},
	{Header: "ER1", Field: "CauseOfLoss"},
	{Header: "ER2", Field: "CauseOfLossID"},
	{Header: "PSS", Field: "City"},
	{Header: "PRGBI", Field: "CityID"},
	{Header: "PFRA", Field: "ClientID"},
	{Header: "XL", Field: "ClientName"},
}

// colsOSBelumKomiteRingkas — 21 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/ReportOSBlmKomiteNonMBU-Act.xml`.
var colsOSBelumKomiteRingkas = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "PICTEKNIK", Field: "Conveyance"},
	{Header: "BUSINESSNAME", Field: "ClaimNo"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "OR", Field: "Location"},
	{Header: "QS", Field: "IsTransferPIC"},
	{Header: "SURPLUS", Field: "ReporterName"},
	{Header: "OTHER", Field: "RefNo"},
}

// colsKomiteRingkas — 11 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKomites_act-Act.xml`.
var colsKomiteRingkas = []Column{
	{Header: "No Klaim", Field: "CaseID"},
	{Header: "Nopolis", Field: "NoKTP"},
	{Header: "QQName", Field: "ClaimID"},
	{Header: "Nama Bisnis", Field: "ClaimNo"},
	{Header: "PIC Teknik", Field: "Conveyance"},
	{Header: "Tgl Registrasi", Field: "Country"},
	{Header: "Tgl Kejadian", Field: "CountryID"},
	{Header: "Tgl Akseptasi", Field: "District"},
	{Header: "No Akseptasi", Field: "DistrictID"},
	{Header: "Tgl Close", Field: "Email"},
	{Header: "Bulan Close", Field: "FlagASO"},
}

// colsKomite — 80 kolom.
//
// Disalin dari `CSVProperties` dan `CSVPropHeaders` milik langkah
// `pxConvertResultsToCSV` pada `Activity/PNCReportDataKomites_act-Act.xml`.
var colsKomite = []Column{
	{Header: "CLAIMNO", Field: "CaseID"},
	{Header: "Komite ID", Field: "ClaimNoExt"},
	{Header: "Nama Komite", Field: "NamaPrincipal"},
	{Header: "Status Approve", Field: "StatusApprove"},
	{Header: "KomiteAccepted", Field: "Note Komite"},
	{Header: "NOPOLIS", Field: "NoKTP"},
	{Header: "QQNAME", Field: "ClaimID"},
	{Header: "BUSINESS NAME", Field: "Resources"},
	{Header: "PICTEKNIK", Field: "ConsultantName"},
	{Header: "CAUSE OF LOSS", Field: "InsuredName"},
	{Header: "COVERAGE NAME", Field: "TelpTertanggung"},
	{Header: "COB", Field: "ClaimNo"},
	{Header: "CEDING/LEADER", Field: "OwnRisk"},
	{Header: "SUMBIS", Field: "IsPLA"},
	{Header: "NAMA MO", Field: "IsSendPremi"},
	{Header: "UW YEAR", Field: "Remark"},
	{Header: "SOURCE", Field: "UserTeknisEmail"},
	{Header: "OCCUPATION CODE", Field: "UserTeknis"},
	{Header: "OCCUPATION", Field: "UserName"},
	{Header: "START DATE", Field: "RCV_ID"},
	{Header: "END DATE", Field: "RW"},
	{Header: "DATEOFLOSS", Field: "CountryID"},
	{Header: "BULAN DOL", Field: "AreaClaimId"},
	{Header: "TAHUN DOL", Field: "ContractNo"},
	{Header: "REGISTERDATE", Field: "Country"},
	{Header: "TGL CLOSE", Field: "Conveyance"},
	{Header: "BULAN CLOSE", Field: "CopyFrom"},
	{Header: "TAHUN CLOSE", Field: "CreateFrom"},
	{Header: "TGL TRF KOMITE", Field: "Email"},
	{Header: "TGL KOMITE AKSEP", Field: "District"},
	{Header: "TGL AKSEPTASI", Field: "AlasanKlaim"},
	{Header: "EX GRATIA", Field: "AnalystRemaksInvestigator"},
	{Header: "TGL TRF PIC", Field: "EmailTertanggung"},
	{Header: "REGIST - CLOSE", Field: "ExGratiaNote"},
	{Header: "LOD - TRF KASIR", Field: "FlagASO"},
	{Header: "STS ASM", Field: "DaftarObjek"},
	{Header: "CURRENCY", Field: "NPWP"},
	{Header: "TIPE AKSEPTASI", Field: "IBNR"},
	{Header: "NO AKSEPTASI", Field: "DistrictID"},
	{Header: "TSI", Field: "IdxSurveyResults"},
	{Header: "TTLOS", Field: "ProdKe"},
	{Header: "NILAI YANG DIAJUKAN TERTANGGUNG", Field: "OldNoHp"},
	{Header: "SELISIH ESTIMASI DENGAN AKSEPTASI", Field: "IDMaster"},
	{Header: "SHARE ASM (%)", Field: "ASMShare"},
	{Header: "ASM SHARE VALUE", Field: "KomiteStatus"},
	{Header: "GROSS VALUE", Field: "Currency"},
	{Header: "OR ASM", Field: "Location"},
	{Header: "COAS", Field: "Status"},
	{Header: "BPPDAN", Field: "LossCoverage"},
	{Header: "FACULTATIVE", Field: "LokasiSurveyor"},
	{Header: "QS RI", Field: "IsTransferPIC"},
	{Header: "PSPL", Field: "NIK"},
	{Header: "FESPL", Field: "IsAnalisTransfer"},
	{Header: "FSPL", Field: "AnaylstRemarks"},
	{Header: "PSRSPL", Field: "AnalystDoctorRemaks"},
	{Header: "PQS RI", Field: "BranchID"},
	{Header: "FACOBLIG", Field: "BranchName"},
	{Header: "FACOBSRB", Field: "BusinessID"},
	{Header: "FACOBINDT", Field: "CASEDB"},
	{Header: "ER1", Field: "CauseOfLoss"},
	{Header: "ER2", Field: "CauseOfLossID"},
	{Header: "PSS", Field: "City"},
	{Header: "PRGBI", Field: "CityID"},
	{Header: "PFRA", Field: "ClientID"},
	{Header: "XL", Field: "ClientName"},
	{Header: "NO REF BROKER", Field: "NoReffBroker"},
	{Header: "STATUS BANDING", Field: "Banding"},
	{Header: "SURVEY KEPUASAN", Field: "SurveyKepuasan"},
	{Header: "EFFORT CLOSE", Field: "EffortClose"},
	{Header: "KENDALA CLOSE", Field: "KendalaClose"},
	{Header: "MANUAL/PAPERLESS", Field: "Paperless"},
	{Header: "USULAN", Field: "Usulan"},
	{Header: "ADJUSTER", Field: "FlagNOLL"},
	{Header: "ADJUSTER MARINE", Field: "FlagReject"},
	{Header: "STS PROGRESS 1", Field: "AgingAmount"},
	{Header: "STS PROGRESS 2", Field: "AlasanTerlambat"},
	{Header: "KETERANGAN", Field: "pyNote"},
	{Header: "CLOSE CLAIM NOTE", Field: "NoteKasir"},
	{Header: "Alasan Tolak 1", Field: "KodeCabang"},
	{Header: "Alasan Tolak 2", Field: "KodeKBRU"},
}
