package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleSite adalah kode situs contoh.
//
// Bentuknya meniru `POOLDATA.M_SITE_DATABASE.ID` — kode pendek yang mendahului nomor urut
// pada setiap ID. Nilainya sengaja BUKAN "1" atau "01" supaya salah baca antara kode situs
// dan nomor urut langsung terlihat pada kunci yang diterbitkan.
const SampleSite = "SP"

// SampleSequence adalah nomor urut terakhir yang sudah dipakai contoh di bawah.
//
// Penambahan berikutnya karena itu menerbitkan SP0000000004 — melanjutkan deret, bukan
// menimpa baris yang sudah ada.
const SampleSequence int64 = 3

// sampleTime adalah stempel TGL_UPDATE_HARGA pada baris contoh.
//
// Tetap, bukan time.Now(): daftar contoh yang waktunya bergerak membuat cuplikan layar
// berubah setiap kali aplikasi dijalankan, dan uji yang membandingkannya menjadi rapuh.
func sampleTime(day int) *time.Time {
	at := time.Date(2026, time.September, day, 3, 15, 0, 0, time.UTC)
	return &at
}

// SampleCategories adalah daftar kategori contoh.
//
// Ia meniru `POOLDATA.GCNM_M_SPAREPART_CATEGORY` yang SUDAH DISETUJUI — hanya baris
// ber-APPROVAL '1' yang sampai ke layar; lihat catatan pada mastersparepart/lookup.go.
// Kategori yang masih menunggu karena itu tidak diwakili di sini sama sekali.
func SampleCategories() []mastersparepart.Category {
	return sampledata.Must[[]mastersparepart.Category](sampleJSON, "SampleCategories")
}

// SampleTypes adalah daftar tipe contoh beserta kategori induknya.
//
// Sengaja tidak merata: ENGINE punya dua tipe, HYDRAULIC dua, UNDERCARRIAGE satu. Distribusi
// yang tidak merata membuat penyaringan Tipe menurut Kategori benar-benar terlihat bekerja
// saat layar dicoba — daftar yang setiap kategorinya berisi jumlah yang sama tidak
// membuktikan apa pun.
func SampleTypes() []mastersparepart.PartType {
	return sampledata.Must[[]mastersparepart.PartType](sampleJSON, "SampleTypes")
}

// SampleList adalah daftar sparepart contoh untuk pengembangan tanpa Oracle.
//
// Ketiganya sengaja berbeda keadaan, supaya seluruh alur layar dapat dicoba:
//
//	SP0000000001  disetujui, SELURUH kolom terisi
//	SP0000000002  menunggu persetujuan, kolom pilihan KOSONG
//	SP0000000003  ditolak, dan TGL_UPDATE_HARGA-nya nil
//
// Yang kedua penting: kelima kolom yang daftar pilihannya tidak ada di export boleh kosong,
// dan layar harus tetap benar menggambarnya. Yang ketiga menutup keadaan `PriceUpdatedAt`
// nil, yang paling mudah terlupa diuji karena baris pertama selalu mengisinya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Kelima kolom penanda sengaja dibiarkan kosong; lihat catatan fungsi ini.
// PriceUpdatedAt sengaja nil: harganya nol, dan baris ini mewakili keadaan yang
// belum pernah distempel sama sekali.
func SampleList() []mastersparepart.Sparepart {
	return sampledata.Must[[]mastersparepart.Sparepart](sampleJSON, "SampleList")
}
