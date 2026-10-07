package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
	"claim-pnc/internal/riwayatklaim"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SELURUH ISI BERKAS INI KARANGAN.
//
// Tidak ada satu pun nomor polis, nama tertanggung, nomor klaim, atau tanggal lahir yang
// berasal dari data nyata. `D-69` melarang data nasabah masuk ke berkas yang di-commit,
// dan larangan itu tidak mengenal pengecualian "hanya untuk contoh".
//
// Nomor klaimnya sengaja memuat KEDUA format yang hidup berdampingan selama masa paralel:
// `PNC-xxxx` terbitan Pega dan `PNCN.YY.xxxx` terbitan sistem baru (`D-71`). Dengan begitu
// pencarian No Klaim dapat dicoba terhadap keduanya — dan perubahan pada kueri tipe 7
// (lihat riwayatklaim.sql) terbukti bekerja, bukan hanya diyakini.

// SampleClaims mengembalikan riwayat klaim contoh.
//
// Isinya dipilih supaya KESEBELAS tipe pencarian yang tersedia dapat dicoba tanpa Oracle:
// ada baris ber-nomor PLA, ber-nomor DLA, ber-ID balai lelang, ber-nomor survei, dan satu
// baris Personal Accident yang membawa peserta beserta tanggal lahirnya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Satu-satunya baris yang membawa peserta. Ia ada supaya pencarian Tanggal
// Lahir PUNYA sasaran yang seharusnya ditemukan — dan uji dapat membuktikan
// bahwa yang membuatnya tidak ditemukan adalah cacat yang direplikasi, bukan
// data contoh yang kebetulan kosong.
// Nomor terbitan SISTEM BARU — tanpa awalan Pega pada Reference-nya,
// persis yang ditetapkan `D-22` dan `D-71`.
func SampleClaims() []Claim { return sampledata.Must[[]Claim](sampleJSON, "SampleClaims") }

// SampleProtections mengembalikan baris proteksi data contoh.
//
// # Kenapa ini ada, dan kenapa ia penting
//
// Tanpa baris proteksi, gerbang menolak SETIAP pengguna dan layar tidak dapat dibuka
// sama sekali — itulah perilaku sistem lama, dan Work Owner memutuskan 2026-09-20 ia
// dibangun penuh. Di Oracle, barisnya didaftarkan lewat Master Proteksi Data milik sistem
// lama. Di memori, tidak ada yang mendaftarkannya, sehingga contohnya disediakan di sini.
//
// Ketiga baris di bawah sengaja berbeda keadaan supaya ketiga jalur gerbang dapat dicoba:
// lolos, hampir habis, dan sudah habis. Pengguna tiruan yang TIDAK disebut di sini —
// `profilbolong` — mewakili jalur keempat: belum terdaftar sama sekali.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Jatahnya tinggal satu: membuka layar sekali lagi menghabiskannya, dan
// pembukaan berikutnya ditolak. Jalur "jatah habis" karena itu dapat dicoba
// tanpa menunggu lima puluh kali percobaan.
func SampleProtections() []riwayatklaim.Protection {
	return sampledata.Must[[]riwayatklaim.Protection](sampleJSON, "SampleProtections")
}
