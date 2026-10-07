package memory

import (
	_ "embed"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// Isi contoh untuk pengembangan tanpa Oracle.
//
// SELURUHNYA KARANGAN. Tidak ada satu pun nama supplier, alamat, NPWP, nomor rekening,
// maupun alamat surel yang disalin dari data nyata — `D-69` melarang data nasabah dan
// alamat surel ditulis di berkas yang di-commit, dan supplier adalah pihak ketiga yang
// datanya diperlakukan sama.
//
// Yang ditiru dari data nyata hanyalah BENTUKNYA: panjang kode, bentuk ID, dan sebaran
// status — supaya layar yang dicoba saat pengembangan berperilaku seperti layar yang
// dipakai di produksi.

// SampleSite adalah kode situs contoh.
//
// Di produksi ia dibaca dari `POOLDATA.M_SITE_DATABASE` dan panjangnya tidak diketahui
// (R-08). Dua digit dipilih supaya ID yang dihasilkan menjadi tiga belas karakter —
// sebelas digit nomor urut ditambah dua digit situs, persis bentuk
// `PEGA_M_SUPPLIER.prc:21`.
const SampleSite = "01"

// SampleSequence adalah nomor urut terakhir yang dianggap sudah dipakai.
//
// Bukan nol, supaya ID yang diterbitkan saat pengembangan tidak tampak seperti nomor urut
// pertama dan tidak bertabrakan dengan baris contoh di bawah.
const SampleSequence int64 = 3

// SampleList adalah supplier contoh.
//
// Ketiganya sengaja berbeda pada hal yang paling menentukan perilaku layar:
//
//	baris 1  aktif, rekanan, supplier Heavy Equipment
//	baris 2  aktif, bukan HE — memperlihatkan pasangan JENIS_STATUS/SUPPLIER_HE yang "0"
//	baris 3  TIDAK aktif — satu-satunya jalur yang menyimpan tanpa meminta persetujuan
//
// Baris ketiga yang paling mudah terlupa diuji, dan justru ia yang mengubah keadaan tanpa
// melewati antrean siapa pun.
func SampleList() []mastersupplier.Supplier {
	return sampledata.Must[[]mastersupplier.Supplier](sampleJSON, "SampleList")
}

// SampleBranches adalah cabang contoh untuk dropdown Cabang.
func SampleBranches() []mastersupplier.Branch {
	return sampledata.Must[[]mastersupplier.Branch](sampleJSON, "SampleBranches")
}

// SampleCities adalah kota contoh untuk lookup Kota.
//
// Kodenya meniru bentuk kode wilayah empat digit, sama seperti yang dipakai contoh Master
// Bengkel — supaya keduanya terlihat berasal dari tabel CITY yang sama.
func SampleCities() []mastersupplier.City {
	return sampledata.Must[[]mastersupplier.City](sampleJSON, "SampleCities")
}

// SampleCountries adalah negara contoh untuk isian Negara.
func SampleCountries() []mastersupplier.Country {
	return sampledata.Must[[]mastersupplier.Country](sampleJSON, "SampleCountries")
}

// SampleBanks adalah bank contoh untuk dropdown Bank.
func SampleBanks() []mastersupplier.Bank {
	return sampledata.Must[[]mastersupplier.Bank](sampleJSON, "SampleBanks")
}
