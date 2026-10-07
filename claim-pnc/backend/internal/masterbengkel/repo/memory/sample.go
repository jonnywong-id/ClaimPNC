package memory

import (
	_ "embed"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// Isi contoh untuk pengembangan tanpa Oracle.
//
// SELURUHNYA KARANGAN. Tidak ada satu pun nama bengkel, alamat, nomor rekening, NPWP,
// maupun alamat surel yang disalin dari data nyata — `D-69` melarang data nasabah dan
// alamat surel ditulis di berkas yang di-commit, dan bengkel adalah pihak ketiga yang
// datanya diperlakukan sama.
//
// Yang ditiru dari data nyata hanyalah BENTUKNYA: panjang kode, bentuk ID_BENGKEL, dan
// sebaran status — supaya layar yang dicoba saat pengembangan berperilaku seperti layar
// yang dipakai di produksi.

// SampleSite adalah kode situs contoh.
//
// Di produksi ia dibaca dari `POOLDATA.M_SITE_DATABASE` dan panjangnya tidak diketahui
// (R-08). Dua digit dipilih karena ID_BENGKEL yang dihasilkan menjadi dua belas karakter
// — cukup panjang untuk memperlihatkan bahwa kuncinya bukan angka berurut biasa.
const SampleSite = "01"

// SampleSequence adalah nomor urut terakhir yang dianggap sudah dipakai.
//
// Bukan nol, supaya ID yang diterbitkan saat pengembangan tidak tampak seperti nomor
// urut pertama dan tidak bertabrakan dengan baris contoh di bawah.
const SampleSequence int64 = 3

// SampleDocumentYear adalah dua digit tahun pada DATAID contoh.
//
// Dipatok, bukan diambil dari jam berjalan — lihat catatan pada field documentYear.
const SampleDocumentYear = "26"

// SampleList adalah bengkel contoh, satu per status persetujuan.
//
// Ketiga tab layar karena itu terisi tanpa perlu menambah apa pun lebih dulu — termasuk
// tab Reject, yang paling mudah terlupa diuji.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Non-rekanan, dan karena itu TANPA login aplikasi. Ia ada supaya jalur yang
// paling mudah salah dapat dicoba tanpa basis data: bengkel non-rekanan tidak
// wajib punya login, dan dua bengkel tanpa login tidak boleh saling menolak.
func SampleList() []masterbengkel.Workshop {
	return sampledata.Must[[]masterbengkel.Workshop](sampleJSON, "SampleList")
}

// SampleBranches adalah cabang contoh untuk dropdown Cabang.
func SampleBranches() []masterbengkel.Branch {
	return sampledata.Must[[]masterbengkel.Branch](sampleJSON, "SampleBranches")
}

// SampleCities adalah kota contoh untuk lookup Kota.
//
// Kodenya memakai bentuk kode wilayah empat digit, sama seperti yang tersimpan di kolom
// CITY_ID — supaya panjang isian yang terlihat saat pengembangan tidak menyesatkan.
func SampleCities() []masterbengkel.City {
	return sampledata.Must[[]masterbengkel.City](sampleJSON, "SampleCities")
}

// SampleBanks adalah bank contoh untuk dropdown Bank.
func SampleBanks() []masterbengkel.Bank {
	return sampledata.Must[[]masterbengkel.Bank](sampleJSON, "SampleBanks")
}
