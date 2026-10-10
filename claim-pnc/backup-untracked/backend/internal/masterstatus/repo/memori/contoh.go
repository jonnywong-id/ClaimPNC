package memori

import "claim-pnc/internal/masterstatus"

// DaftarContoh adalah 33 status klaim nyata sesuai isi POOLDATA.V_STS_CLAIM.
//
// Nilainya disalin apa adanya dari `Database/v_sts_claim.csv` yang diekspor Work Owner,
// bukan dikarang. Ia dipakai pengujian dan pengembangan tanpa basis data.
//
// # Dua hal yang terbaca dari daftar ini dan penting untuk tidak dilupakan
//
//  1. Kodenya BERURUTAN `1134`–`1166` tanpa satu pun lompatan, karena ia dibentuk
//     `id_site || lpad(urutan, 3, '0')` — situs `1`, urutan 134 sampai 166.
//  2. KodeLama hanya terisi pada sebelas kode pertama (`1134`–`1144` → `01`–`11`).
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
func DaftarContoh() []masterstatus.StatusKlaim {
	return []masterstatus.StatusKlaim{
		{Kode: "1134", Label: "Abbreviated Report", KodeLama: "01"},
		{Kode: "1135", Label: "Preliminary Report", KodeLama: "02"},
		{Kode: "1136", Label: "Interim Payment Report", KodeLama: "03"},
		{Kode: "1137", Label: "Final Report", KodeLama: "04"},
		{Kode: "1138", Label: "PLA Report", KodeLama: "05"},
		{Kode: "1139", Label: "DLA Report", KodeLama: "06"},
		{Kode: "1140", Label: "Partial Accepted", KodeLama: "07"},
		{Kode: "1141", Label: "Full Accepted", KodeLama: "08"},
		{Kode: "1142", Label: "Rejected Claim", KodeLama: "09"},
		{Kode: "1143", Label: "Close Claim for this object", KodeLama: "10"},
		{Kode: "1144", Label: "Cancelled Claim", KodeLama: "11"},
		{Kode: "1145", Label: "Waiting Survey", KodeLama: ""},
		{Kode: "1146", Label: "View Polis", KodeLama: ""},
		{Kode: "1147", Label: "Register", KodeLama: ""},
		{Kode: "1148", Label: "CFS Report", KodeLama: ""},
		{Kode: "1149", Label: "Claim Committee", KodeLama: ""},
		{Kode: "1150", Label: "LOD Report", KodeLama: ""},
		{Kode: "1151", Label: "Analyst", KodeLama: ""},
		{Kode: "1152", Label: "Compliance", KodeLama: ""},
		{Kode: "1153", Label: "RCL", KodeLama: ""},
		{Kode: "1154", Label: "PUCL", KodeLama: ""},
		{Kode: "1155", Label: "RCL Dokter", KodeLama: ""},
		{Kode: "1156", Label: "Investigator", KodeLama: ""},
		{Kode: "1157", Label: "Document Waiting RCL/PUCL", KodeLama: ""},
		{Kode: "1158", Label: "Inputor", KodeLama: ""},
		{Kode: "1159", Label: "Approval Commite", KodeLama: ""},
		{Kode: "1160", Label: "Rejected Commite", KodeLama: ""},
		{Kode: "1161", Label: "Acceptance Report", KodeLama: ""},
		{Kode: "1162", Label: "Transfered to Cashier", KodeLama: ""},
		{Kode: "1163", Label: "Paid", KodeLama: ""},
		{Kode: "1164", Label: "Reopen Claim", KodeLama: ""},
		{Kode: "1165", Label: "Rejected Chasier", KodeLama: ""},
		{Kode: "1166", Label: "LOD Accepted", KodeLama: ""},
	}
}
