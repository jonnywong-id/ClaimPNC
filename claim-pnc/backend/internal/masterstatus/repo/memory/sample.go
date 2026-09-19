package memory

import "claim-pnc/internal/masterstatus"

// SampleList adalah 33 status klaim nyata sesuai isi POOLDATA.V_STS_CLAIM.
//
// Nilainya disalin apa adanya dari `Database/v_sts_claim.csv` yang diekspor Work Owner,
// bukan dikarang. Ia dipakai pengujian dan pengembangan tanpa basis data.
//
// # Dua hal yang terbaca dari daftar ini dan penting untuk tidak dilupakan
//
//  1. Kodenya BERURUTAN `1134`–`1166` tanpa satu pun lompatan, karena ia dibentuk
//     `id_site || lpad(urutan, 3, '0')` — situs `1`, urutan 134 sampai 166.
//  2. LegacyCode hanya terisi pada sebelas kode pertama (`1134`–`1144` → `01`–`11`).
//     Di basis data ia tersimpan sebagai CHAR berisi padding (`"01  "`); spasinya sudah
//     dibuang di sini karena tidak pernah dimaksudkan sebagai bagian nilainya.
//
// ARTI KODE TIDAK BOLEH DISIMPULKAN DARI PEMAKAIANNYA DI RULE. Tiga arti yang sempat
// disimpulkan begitu ternyata seluruhnya salah (`R-06`): `1143` bukan status awal
// melainkan Close Claim, `1150` bukan penanda terdaftar melainkan LOD Report, `1151`
// bukan Investigator melainkan Analyst. Daftar di bawah adalah isi master yang
// sebenarnya.
//
// # Selisih dengan basis data yang berjalan — belum dijelaskan
//
// Pembacaan langsung POOLDATA.M_STS_CLAIM pada 2026-09-17 menemukan **32 baris**, bukan
// 33: kode **`1165` "Rejected Chasier"** TIDAK ADA di sana, sementara ia ada di CSV.
// Selisihnya persis satu baris; 32 kode lainnya cocok seluruhnya, termasuk labelnya.
//
// Daftar ini tetap mengikuti CSV, dan tidak "diperbaiki" menjadi 32. Alasannya: CSV
// adalah artefak yang diserahkan Work Owner sebagai isi master, dan menghapus satu baris
// darinya karena satu basis data kebetulan tidak memuatnya akan menghapus pertanyaannya
// sekaligus. Sebabnya — basis data yang berbeda, portal yang berbeda, atau baris yang
// memang terhapus — adalah pertanyaan untuk Work Owner, dan dicatat di
// docs/catatan-pengembangan.md.
//
// Uji TestSelisihDenganBasisDataProduksiTercatat mengunci selisih itu supaya ia tidak
// hilang diam-diam saat daftar ini kelak disunting.
func SampleList() []masterstatus.ClaimStatus {
	return []masterstatus.ClaimStatus{
		{Code: "1134", Label: "Abbreviated Report", LegacyCode: "01"},
		{Code: "1135", Label: "Preliminary Report", LegacyCode: "02"},
		{Code: "1136", Label: "Interim Payment Report", LegacyCode: "03"},
		{Code: "1137", Label: "Final Report", LegacyCode: "04"},
		{Code: "1138", Label: "PLA Report", LegacyCode: "05"},
		{Code: "1139", Label: "DLA Report", LegacyCode: "06"},
		{Code: "1140", Label: "Partial Accepted", LegacyCode: "07"},
		{Code: "1141", Label: "Full Accepted", LegacyCode: "08"},
		{Code: "1142", Label: "Rejected Claim", LegacyCode: "09"},
		{Code: "1143", Label: "Close Claim for this object", LegacyCode: "10"},
		{Code: "1144", Label: "Cancelled Claim", LegacyCode: "11"},
		{Code: "1145", Label: "Waiting Survey", LegacyCode: ""},
		{Code: "1146", Label: "View Polis", LegacyCode: ""},
		{Code: "1147", Label: "Register", LegacyCode: ""},
		{Code: "1148", Label: "CFS Report", LegacyCode: ""},
		{Code: "1149", Label: "Claim Committee", LegacyCode: ""},
		{Code: "1150", Label: "LOD Report", LegacyCode: ""},
		{Code: "1151", Label: "Analyst", LegacyCode: ""},
		{Code: "1152", Label: "Compliance", LegacyCode: ""},
		{Code: "1153", Label: "RCL", LegacyCode: ""},
		{Code: "1154", Label: "PUCL", LegacyCode: ""},
		{Code: "1155", Label: "RCL Dokter", LegacyCode: ""},
		{Code: "1156", Label: "Investigator", LegacyCode: ""},
		{Code: "1157", Label: "Document Waiting RCL/PUCL", LegacyCode: ""},
		{Code: "1158", Label: "Inputor", LegacyCode: ""},
		{Code: "1159", Label: "Approval Commite", LegacyCode: ""},
		{Code: "1160", Label: "Rejected Commite", LegacyCode: ""},
		{Code: "1161", Label: "Acceptance Report", LegacyCode: ""},
		{Code: "1162", Label: "Transfered to Cashier", LegacyCode: ""},
		{Code: "1163", Label: "Paid", LegacyCode: ""},
		{Code: "1164", Label: "Reopen Claim", LegacyCode: ""},
		{Code: "1165", Label: "Rejected Chasier", LegacyCode: ""},
		{Code: "1166", Label: "LOD Accepted", LegacyCode: ""},
	}
}
