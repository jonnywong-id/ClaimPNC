package memory

import (
	_ "embed"

	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// NewSampleRepo membentuk penyimpanan berisi contoh yang mencakup SELURUH jalur layar.
//
// # Seluruh isinya karangan
//
// Tidak satu pun nomor, nama, atau nilai di berkas ini berasal dari data produksi
// (`D-69`: data nasabah tidak pernah ditulis ke berkas yang di-commit). Yang ditiru
// adalah BENTUK dan KEADAANNYA, bukan isinya.
//
// # Enam keadaan yang sengaja dibuat berbeda
//
// Supaya setiap cabang layar dapat dicoba tanpa Oracle — termasuk empat yang paling mudah
// terlewat kalau contohnya "semua normal":
//
//  1. Perjanjian dengan dua group business dan klaim di dua tanggal → jalur utama.
//  2. Perjanjian yang salah satu group business-nya TIDAK punya nama di master → menguji
//     penggantian menjadi "TREATY INWARD" (`GET_GROUPBUSINESS_XOL`).
//  3. Perjanjian TANPA group business sama sekali → menguji jalur daftar kosong yang
//     BUKAN galat, yaitu master yang baru dibuat dan belum diisi.
//  4. Baris treaty inward yang kursnya TIDAK ditemukan → menguji RateMissing, pengganti
//     `RETURN 1` yang `D-49` butir 5 perbaiki.
//  5. Pemberitahuan yang sudah direvisi → menguji perakitan "nomor / revisi".
//  6. Pemberitahuan yang belum disetujui di kedua tipe → menguji antrean tab Komite.
func NewSampleRepo() *Repo {
	return NewRepo(
		WithMasters(SampleMasters()...),
		WithCauseOfLoss(SampleCauseOfLoss()...),
		WithSummaries("2024", SampleSummaries2024()...),
		WithSummaries("2023", SampleSummaries2023()...),
		WithBreakdown("12/03/2024", "BANJIR", SampleBreakdownBanjir()...),
		WithTreatyInward("12/03/2024", "BANJIR", SampleTreatyBanjir()...),
		WithBreakdown("28/07/2024", "KEBAKARAN", SampleBreakdownKebakaran()...),
		WithTreatyInward("28/07/2024", "KEBAKARAN", SampleTreatyKebakaran()...),
		WithAdvices(SampleAdvices()...),
	)
}

// SampleMasters adalah tiga perjanjian XOL contoh.
//
// Nilai kursnya dibuat berbeda supaya pembagian ke mata uang perjanjian benar-benar
// terlihat: kalau kursnya sama, kekeliruan membagi dua kali tidak akan terbaca.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Sudah disetujui komite — tidak muncul di antrean tab Komite.
// Nama kosong: di sistem lama GET_GROUPBUSINESS_XOL menggantinya dengan
// "TREATY INWARD". Di sini penggantian itu terjadi di Go.
// Kurs sudah diisi, group business belum. Perjanjian seperti ini NYATA —
// ia baru dibuat — dan layarnya harus menampilkan daftar kosong, bukan galat.
func SampleMasters() []inboxxol.MasterXOL {
	return sampledata.Must[[]inboxxol.MasterXOL](sampleJSON, "SampleMasters")
}

// SampleCauseOfLoss adalah isi dropdown Penyebab Kerugian.
//
// Yang tersimpan di kolom CAUSEOFLOSS pada tabel klaim adalah DESKRIPSI-nya, bukan
// kodenya — karena itu deskripsi di sini sama persis dengan yang dipakai kunci contoh
// rincian di bawah.
func SampleCauseOfLoss() []inboxxol.CauseOfLoss {
	return sampledata.Must[[]inboxxol.CauseOfLoss](sampleJSON, "SampleCauseOfLoss")
}

// SampleSummaries2024 adalah akumulasi klaim perjanjian 2024, MASIH DALAM RUPIAH.
//
// Nilainya sengaja dibiarkan rupiah persis seperti yang dikembalikan basis data:
// pembagian dengan kurs adalah tugas usecase, dan mengisi contoh ini dengan nilai yang
// sudah terbagi akan menyembunyikan kekeliruan bila pembagiannya kelak terlewat.
func SampleSummaries2024() []inboxxol.ClaimSummary {
	return sampledata.Must[[]inboxxol.ClaimSummary](sampleJSON, "SampleSummaries2024")
}

// SampleSummaries2023 adalah akumulasi klaim perjanjian 2023.
func SampleSummaries2023() []inboxxol.ClaimSummary {
	return sampledata.Must[[]inboxxol.ClaimSummary](sampleJSON, "SampleSummaries2023")
}

// SampleBreakdownBanjir adalah rincian klaim milik sendiri, masih dalam rupiah.
func SampleBreakdownBanjir() []inboxxol.BusinessBreakdown {
	return sampledata.Must[[]inboxxol.BusinessBreakdown](sampleJSON, "SampleBreakdownBanjir")
}

// SampleTreatyBanjir adalah baris treaty inward yang kursnya LENGKAP.
//
// Nilainya SUDAH dalam mata uang perjanjian — baris treaty inward dikonversi di kuerinya
// sendiri dan tidak ikut dibagi kurs di usecase.
func SampleTreatyBanjir() []inboxxol.BusinessBreakdown {
	return sampledata.Must[[]inboxxol.BusinessBreakdown](sampleJSON, "SampleTreatyBanjir")
}

// SampleBreakdownKebakaran adalah rincian klaim milik sendiri untuk tanggal kedua.
func SampleBreakdownKebakaran() []inboxxol.BusinessBreakdown {
	return sampledata.Must[[]inboxxol.BusinessBreakdown](sampleJSON, "SampleBreakdownKebakaran")
}

// SampleTreatyKebakaran adalah baris treaty inward yang KURSNYA TIDAK DITEMUKAN.
//
// Inilah keadaan yang di sistem lama menghasilkan angka salah tanpa satu pun tanda:
// `GETCURRENCYSTANDARD` mengembalikan `1`, sehingga nilai valuta asing diperlakukan satu
// banding satu terhadap rupiah. Di sini nilainya nol dan barisnya ditandai, sehingga
// layar menyatakan kursnya tidak tersedia alih-alih menampilkan angka yang salah.
func SampleTreatyKebakaran() []inboxxol.BusinessBreakdown {
	return sampledata.Must[[]inboxxol.BusinessBreakdown](sampleJSON, "SampleTreatyKebakaran")
}

// SampleAdvices adalah pemberitahuan PLA dan DLA contoh.
//
// Dua di antaranya belum disetujui (`ApprovalStatus` `"0"`) pada tipe yang berbeda,
// sehingga antrean tab Komite berisi dua baris — satu PLA, satu DLA.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Sudah direvisi sekali — nomor yang dibaca pengguna menjadi
// "PLA/XOL/2024/0002 / 1". Perakitannya terjadi di Go, bukan di SQL.
// Surel kosong pada baris pemberitahuan; di SQL ia jatuh ke
// T_REINSURER.EMAIL lewat COALESCE. Di contoh ini hasil jatuhnya sudah
// terisi, karena penyimpanan memori tidak punya tabel reasuradur.
func SampleAdvices() []inboxxol.Advice {
	return sampledata.Must[[]inboxxol.Advice](sampleJSON, "SampleAdvices")
}
