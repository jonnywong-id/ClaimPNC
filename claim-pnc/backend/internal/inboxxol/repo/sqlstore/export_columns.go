package sqlstore

// Daftar kolom berkas "Export to Excel", disalin dari parameter `pxConvertResultsToCSV`
// pada `Activity/GenerateDetailClaimBusinessXOL-Act.xml`.
//
// BERKAS INI DIBANGKITKAN dari activity tersebut, bukan diketik ulang. Empat puluh dua
// kolom yang disalin tangan adalah empat puluh dua peluang salah ketik yang tidak akan
// ketahuan sampai ada yang membuka berkasnya di Excel.
//
// # properties dan headers berpasangan menurut URUTAN
//
// `properties[i]` adalah nama kolom yang dikembalikan kueri (alias Pega), `headers[i]`
// adalah judul yang dibaca pengguna. Panjang keduanya WAJIB sama — diperiksa
// TestKolomExportBerpasangan.
//
// # Satu properti boleh muncul beberapa kali
//
// Pada varian per-business, `Currency` muncul EMPAT kali: Curr RESERVE, Curr ACCEPTED,
// Curr SALVAGE, dan Cur ADJUSTER FEE semuanya membaca properti yang sama. Itu disengaja
// di sistem lama, dan pembacaan berbasis nama membuatnya bekerja tanpa perlakuan khusus.

var exportMBUColumns = exportColumns{
	properties: []string{"ClaimNo", "PolicyNo", "BranchID", "AlasanKlaim", "CoverageNote", "Message", "StatusClaim", "Currency", "ClaimAmount", "ClaimEstimate", "ASMShare", "BusinessID", "District", "DistrictID"},
	headers:    []string{"No Klaim", "No Polis", "Date of loss", "QQ Name", "Coverage", "Merk/Tipe", "Status Klaim", "Currency", "Reserve Amount", "Reserve Share ASM", "Aksep Amount", "Aksep Share ASM", "OS Salvage", "Aksep Salvage"},
}

var exportPerBusinessColumns = exportColumns{
	properties: []string{"AlasanKlaim", "Location", "PolicyNo", "ClaimNo", "AnalystDoctorRemaks", "AllBusinessFlag", "Currency", "Keyword", "ASMShare", "ClaimAmount", "BranchID", "BranchName", "BusinessID", "BusinessName", "CASEDB", "CaseID", "Currency", "CauseOfLossID", "City", "CityID", "ClaimEstimate", "ClaimID", "ClaimNoSRB", "ClientID", "ClientName", "CloseClaimNote", "ConsultantID", "ConsultantName", "Conveyance", "ContractNo", "Country", "Currency", "District", "Currency", "Email", "UserTeknis", "EmailTertanggung", "FlagASO", "IDMaster", "IsPLA", "Remark", "ReporterName"},
	headers:    []string{"INSURED", "RISK LOCATION", "ASM POLICY", "CLAIM NO", "DOL", "LEADER", "Curr RESERVE", "RESERVE AMOUNT", "ASM SHARE (%)", "ASM RESERVE VALUE", "OR OS", "BPPDAN OS", "Fac-Out OS", "FSPL OS", "PSPL OS", "QS (R/I) OS", "Curr ACCEPTED", "ACCEPTED", "OR ACCEPT", "BPPDAN ACCEPT", "Fac-Out ACCEPT", "FSPL ACCEPT", "PSPL ACCEPT", "QS (R/I) ACCEPT", "BALANCE OS", "BALANCE OR", "BALANCE BPPDAN", "BALANCE Fac-Out", "BALANCE FSPL", "BALANCE PSPL", "BALANCE QS (R/I)", "Curr SALVAGE", "AMOUNT SALVAGE", "Cur ADJUSTER FEE", "AMOUNT ADJUSTERFEE", "PIC", "LEADER/MEMBER", "SUMBIS", "LOSS ADJUSTER", "OKUPASI", "REMARKS", "LAST STATUS"},
}

var exportTreatyInwardColumns = exportColumns{
	properties: []string{"pyCompany", "Currency", "IsReservedClaim", "OwnRiskValue", "QsTreaty", "DeductibleValue", "ClaimAmount", "ASMShare", "ClaimAmountShareASM", "PaidClaimAmountShare", "BalanceClaimASMShare", "DateOfLoss", "CauseOfLoss"},
	headers:    []string{"COMPANY", "CURRENCY", "ISRESERVEDCLAIM", "OWNRISKVALUE", "QSTREATY", "DEDUCTIBLEVALUE", "CLAIMAMOUNT", "ASMSHARE", "CLAIMAMOUNTSHAREASM", "PAIDCLAIMAMOUNTSHARE", "BALANCECLAIMASMSHARE", "DATEOFLOSS", "CAUSEOFLOSS"},
}

// exportColumns memasangkan nama kolom kueri dengan judul yang dibaca pengguna.
type exportColumns struct {
	properties []string
	headers    []string
}
