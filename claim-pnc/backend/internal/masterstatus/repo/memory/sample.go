package memory

import (
	_ "embed"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

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
	return sampledata.Must[[]masterstatus.ClaimStatus](sampleJSON, "SampleList")
}
