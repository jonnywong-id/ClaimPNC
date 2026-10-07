package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleTasks mengembalikan daftar contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh isinya KARANGAN, dan itu wajib
//
// Tidak ada satu pun nomor polis, nama tertanggung, nama peserta, maupun nomor klaim nyata
// di sini. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan itu
// berlaku pada data contoh persis seperti pada dokumen.
//
// **Bentuknya** ditiru dari data nyata yang diperiksa pada 2026-09-24 — nomor kasus
// `PNC-xxxx` empat digit, nomor polis 14 digit, dan tanggal registrasi yang berjarak
// tahunan dari hari ini — supaya layar diuji menghadapi rupa yang benar-benar akan
// ditemuinya. **Isinya** tidak.
//
// # Yang sengaja diwakili
//
// Keenam baris di bawah dipilih supaya setiap keadaan yang dapat dihadapi layar muncul
// sekurang-kurangnya sekali:
//
//	baris 1   lengkap, menunggu paling lama            -> tampil paling atas
//	baris 2   lengkap, menunggu beberapa bulan
//	baris 3   TANPA Nama Peserta                       -> polisnya tidak ditemukan
//	baris 4   TANPA Date Of Loss                       -> kolom tanggal yang dapat kosong
//	baris 5   TANPA tanggal registrasi                 -> jatuh ke AKHIR daftar
//	baris 6   ClaimKey KOSONG                          -> klaim yatim; Submit DITOLAK
//
// Dua baris terakhir yang paling perlu ada.
//
// Baris 5 membuktikan baris bertanggal kosong jatuh di akhir, sejajar dengan NULL pada
// `ORDER BY ... ASC` — dan bahwa kolom Aging-nya ditulis sebagai tanda hubung, bukan
// "0 hari" yang akan terbaca seolah klaimnya baru masuk hari ini.
//
// Baris 6 membuktikan jalur yang paling mudah terlewat: gabungan ke `T_CLAIM_PNC` sengaja
// LEFT, sehingga pekerjaan yang klaimnya tidak ditemukan TETAP TAMPIL — persis seperti di
// Pega — tetapi Submit atasnya ditolak dengan ErrClaimMissing.
//
// # Urutan penulisannya sengaja TIDAK terurut
//
// Baris-barisnya ditulis acak supaya pengurutan benar-benar diuji. Bila daftar contoh sudah
// terurut sejak awal, fungsi pengurutan yang rusak pun akan tampak benar.
//
// # Seluruhnya SUDAH merupakan klaim TKA yang belum selesai
//
// Tidak ada baris yang tanggal kelengkapan dokumennya terisi dan tidak ada yang berstatus
// selesai, karena kedua penyaring itu hidup di dalam kueri SQL — bukan di dalam Filter.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Polisnya tidak ditemukan di tabel polis, sehingga nama pesertanya kosong.
// Barisnya tetap dapat dikerjakan — yang hilang hanya satu sel.
// Date Of Loss kosong.
// Tanggal registrasi kosong — kolom sumbernya VARCHAR2, dan bentuk yang tidak
// dikenali diurai menjadi nil alih-alih tanggal karangan.
// Klaim YATIM: pekerjaannya ada di tabel kerja Pega, klaimnya tidak ada di
// tabel bisnis. Ia TAMPIL, dan Submit atasnya ditolak.
func SampleTasks() []inboxreceivetka.Task {
	return sampledata.Must[[]inboxreceivetka.Task](sampleJSON, "SampleTasks")
}

// at mengurai waktu contoh, dan panik bila penulisannya salah.
//
// Panik di sini aman: nilainya konstanta di dalam berkas ini, bukan masukan pengguna,
// sehingga kesalahan penulisannya adalah cacat yang harus terlihat pada uji pertama.
func at(value string) *time.Time {
	moment, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic("inboxreceivetka/memory: waktu contoh salah tulis: " + value)
	}
	return &moment
}
