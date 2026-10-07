package memory

import (
	_ "embed"

	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleBusiness mengembalikan master lini bisnis contoh — padanan POOLDATA.BUSINESS.
//
// # Yang NYATA dan yang DIKARANG di sini, dinyatakan terang-terangan
//
// Nama lini bisnisnya nyata: keenamnya dibaca dari `CONTEXT.md` bagian Lini Bisnis &
// Segmentasi, yang diturunkan dari Group Panel di sistem lama.
//
// Kodenya DIKARANG. Isi POOLDATA.BUSINESS belum pernah diterima — ia tabel milik ruleset
// GISFW, dan DDL maupun isinya tidak ada di export (`R-08`). Kode di bawah karena itu
// hanya boleh dipakai untuk mencoba layar, TIDAK PERNAH sebagai rujukan.
//
// Tidak ada satu pun data nasabah di sini, dan memang tidak boleh ada (`D-69`).
func SampleBusiness() []masterpasal.Business {
	return sampledata.Must[[]masterpasal.Business](sampleJSON, "SampleBusiness")
}

// SampleList mengembalikan pasal kerugian contoh.
//
// Ketiganya mencerminkan ketiga kategori yang ada — Jaminan Polis, Pengecualian, dan
// Notifikasi — supaya kolom Kategori dapat dilihat bekerja tanpa harus menambah baris
// lebih dulu.
//
// Baris ketiga sengaja BERKATEGORI KOSONG. Itulah bentuk "Notifikasi" yang sebenarnya:
// cabang `else` pada ekspresi Pega, bukan sebuah kode tersendiri — lihat
// masterpasal.CategoryNotification. Dengan begitu layar yang dicoba saat pengembangan
// menampakkan bentuk data yang benar-benar akan ditemui, bukan bentuk yang dirapikan.
//
// Isi pasalnya dikarang dan sengaja dibuat pendek. Ia hanya perlu cukup untuk melihat
// kolom "Isi Pasal" terisi.
func SampleList() []masterpasal.Clause {
	return sampledata.Must[[]masterpasal.Clause](sampleJSON, "SampleList")
}

// NewSampleRepo membentuk penyimpanan berisi contoh di atas.
func NewSampleRepo() *Repo { return NewRepo(SampleList(), SampleBusiness()) }
