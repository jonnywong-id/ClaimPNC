package memory

import (
	_ "embed"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleList adalah baris awal untuk menjalankan aplikasi tanpa basis data.
//
// # Isinya karangan, dan itu dinyatakan terang-terangan
//
// Berbeda dari Master Status Klaim yang 33 barisnya adalah isi master sungguhan, isi
// MST_USER_TEKNIK **tidak ada di dalam export** — tidak ada CSV, tidak ada dump. Yang
// terbaca dari export hanyalah bentuk tabelnya.
//
// Karena itu baris di bawah adalah contoh yang dikarang: nama, surel, dan ID operatornya
// tidak merujuk pegawai mana pun. Mengarang data yang BERPURA-PURA nyata justru yang
// dilarang; yang dilakukan di sini sebaliknya — contoh yang jelas-jelas contoh, memakai
// domain `example.invalid` yang memang dicadangkan supaya tidak mungkin tertukar dengan
// alamat sungguhan.
//
// # Kenapa susunannya seperti ini
//
// Keempat baris dipilih supaya setiap jalur layar dapat dicoba tanpa basis data:
//
//	PICTEKNIK01  atasan, kuota besar          — muncul di daftar
//	PICTEKNIK02  bawahan, beban di bawah kuota — muncul di daftar
//	PICTEKNIK03  beban SAMA DENGAN kuota       — menguji tampilan "kuota penuh"
//	PICTEKNIK04  NONAKTIF                      — TIDAK muncul di daftar, tetapi dapat
//	                                             dibuka lewat ID dan diaktifkan kembali
//
// Baris keempat itulah yang membuat keputusan "daftar hanya menampilkan yang aktif" dapat
// diuji sungguhan, bukan hanya dipercaya.
func SampleList() []masterpicteknik.Technician {
	return sampledata.Must[[]masterpicteknik.Technician](sampleJSON, "SampleList")
}
