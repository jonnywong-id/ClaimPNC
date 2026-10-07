package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/inboxinvestigator"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleTasks mengembalikan antrean contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh isinya KARANGAN, dan itu wajib
//
// Tidak ada satu pun nomor polis, nama tertanggung, nama peserta, maupun nomor klaim nyata
// di sini. `D-69` melarang data nasabah ditulis ke berkas yang di-commit, dan larangan itu
// berlaku pada data contoh persis seperti pada dokumen.
//
// Nomor klaimnya memakai kedua format yang hidup berdampingan selama masa paralel —
// `PNC-xxxx` dari Pega dan `PNCN.YY.xxxx` dari sistem baru (`D-71`) — supaya layar terbukti
// menampilkan keduanya tanpa membedakan.
//
// # Yang sengaja diwakili
//
// Kedelapan baris di bawah dipilih supaya setiap keadaan yang dapat dihadapi layar muncul
// sekurang-kurangnya sekali:
//
//	baris 1   lengkap, baru masuk beberapa jam
//	baris 2   lengkap, tanggal survei beberapa hari lalu
//	baris 3   TANPA tanggal survei                    -> kolom kesembilan KOSONG
//	baris 4   TANPA Nama Peserta                      -> klaim yang objeknya belum terisi
//	baris 5   nomor klaim format baru PNCN.YY.xxxx
//	baris 6   Nama Admin kosong                       -> kasus yang dibuat proses, bukan orang
//	baris 7   tanggal survei jauh di belakang
//	baris 8   Tanggal Pendaftaran kosong              -> kolom tanggal yang dapat NULL
//
// Baris 3 yang paling perlu ada: ia satu-satunya yang membuktikan klaim tanpa baris survei
// TETAP tampil di antrean — hanya kolom kesembilannya yang kosong. Tanpa baris itu, jalur
// tersebut tidak akan pernah tercoba.
//
// # Seluruhnya SUDAH merupakan antrean investigator yang belum selesai
//
// Tidak ada baris yang berstatus selesai dan tidak ada yang berada di workbasket lain,
// karena kedua penyaring itu hidup di dalam kueri SQL — bukan di dalam Filter. Repo memori
// karena itu menganggap isinya sudah tersaring; lihat catatan pada Repo.List.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Nama Peserta ada, tetapi surveinya belum dijadwalkan sama sekali.
// TANPA tanggal survei. Kolom kesembilan wajib tampil KOSONG, barisnya TETAP ada.
// Klaim yang objek pertanggungannya belum terisi — T_CLAIM_OBJECTLIST kosong,
// sehingga subquery mengembalikan NULL.
// Nomor klaim format BARU (`D-71`), hidup berdampingan dengan format Pega.
// Nama Admin kosong: kasus yang dibuat proses terjadwal, bukan orang. Job
// harian memang membuat kasus tanpa pengguna sama sekali (`D-57`).
// Tanggal Pendaftaran kosong. Tidak semestinya terjadi pada baris yang sah,
// tetapi tidak ada DDL yang membuktikan kolomnya NOT NULL (`R-08`).
func SampleTasks() []inboxinvestigator.Task {
	return sampledata.Must[[]inboxinvestigator.Task](sampleJSON, "SampleTasks")
}

// at membaca waktu contoh, dan panik bila teksnya salah.
//
// Panik di sini aman: teksnya konstanta di dalam berkas ini, bukan masukan pengguna,
// sehingga kesalahannya adalah cacat pemrograman yang harus terlihat saat pertama
// dijalankan.
//
// Seluruhnya UTC, sama seperti yang disimpan basis data (`DB-8`). Pengubahan ke WIB terjadi
// di layar, bukan di sini — dan tidak pernah dengan menambahkan tujuh jam secara manual
// (`F-5`).
func at(text string) *time.Time {
	moment, err := time.Parse(time.RFC3339, text)
	if err != nil {
		panic("inboxinvestigator/memory: waktu contoh tidak dapat dibaca: " + text)
	}
	return &moment
}
