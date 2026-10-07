package memory

import (
	_ "embed"

	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleList adalah daftar contoh untuk pengembangan tanpa basis data.
//
// ============================================================================
// ISI DI BAWAH INI **BUKAN** ISI MASTER YANG SEBENARNYA.
// ============================================================================
//
// Ini berbeda dari SampleList milik modul Master Status Klaim, yang nilainya disalin apa
// adanya dari `Database/v_sts_claim.csv` yang diserahkan Work Owner. Untuk
// `M_DOMINAN_FACTOR` **tidak ada artefak setara**: export tidak memuat isinya, dan tidak
// ada satu pun CSV master untuk tabel ini.
//
// Daftar di bawah karena itu DIKARANG sebagai bahan mencoba layar — bukan sebagai
// pernyataan tentang faktor dominan apa yang benar-benar dipakai Asuransi Sinar Mas.
// Jangan mengutipnya ke dokumen mana pun, dan jangan memakainya sebagai dasar uji
// kesetaraan.
//
// Isi yang sebenarnya perlu diminta ke DBA dengan satu kueri:
//
//	SELECT ID, NAME FROM POOLDATA.M_DOMINAN_FACTOR ORDER BY TO_NUMBER(ID);
//
// Permintaan itu dicatat sebagai hal terbuka di `claim-pnc/docs/catatan-pengembangan.md`.
// Begitu hasilnya diterima, berkas ini diganti isinya dan komentar ini dicabut.
//
// # Kenapa daftarnya tetap ada, dan tidak dibiarkan kosong
//
// Daftar kosong membuat tiga hal tidak dapat dicoba sama sekali tanpa Oracle: pengurutan
// numerik `9` sebelum `10`, tombol Ubah, dan keadaan tabel yang berisi. Yang dikarang di
// sini hanyalah TEKS-nya; perilaku yang diujinya nyata.
//
// Sepuluh baris dipilih dengan sengaja supaya ID mencapai dua digit — itulah yang
// membuktikan pengurutan numerik benar-benar bekerja, dan bukan kebetulan karena semua
// ID masih satu digit.
func SampleList() []masterdominanfactor.DominantFactor {
	return sampledata.Must[[]masterdominanfactor.DominantFactor](sampleJSON, "SampleList")
}
