package memory

import "claim-pnc/internal/masterpasal"

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
	return []masterpasal.Business{
		{ID: "2001", Name: "Personal Accident"},
		{ID: "2002", Name: "Travel"},
		{ID: "2003", Name: "Marine Cargo"},
		{ID: "2004", Name: "Fire / Property"},
		{ID: "2005", Name: "Aneka"},
		{ID: "2006", Name: "Bonding"},
	}
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
	return []masterpasal.Clause{
		{
			ID:            "1",
			Number:        "PSL-001",
			Text:          "Penanggung menjamin kerugian atas harta benda yang dipertanggungkan akibat kebakaran.",
			Description:   "Jaminan dasar kebakaran",
			Category:      masterpasal.CategoryPolicyCoverage,
			CategoryLabel: masterpasal.CategoryLabel(masterpasal.CategoryPolicyCoverage),
			Business: []masterpasal.Business{
				{ID: "2004", Name: "Fire / Property"},
			},
		},
		{
			ID:            "2",
			Number:        "PSL-002",
			Text:          "Tidak dijamin kerugian yang timbul akibat keausan, sifat barang sendiri, atau cacat tersembunyi.",
			Description:   "Pengecualian umum",
			Category:      masterpasal.CategoryException,
			CategoryLabel: masterpasal.CategoryLabel(masterpasal.CategoryException),
			Business: []masterpasal.Business{
				{ID: "2003", Name: "Marine Cargo"},
				{ID: "2005", Name: "Aneka"},
			},
		},
		{
			ID:            "3",
			Number:        "PSL-003",
			Text:          "Tertanggung wajib memberitahukan setiap perubahan risiko kepada Penanggung.",
			Description:   "Kewajiban pemberitahuan",
			Category:      masterpasal.CategoryNotification,
			CategoryLabel: masterpasal.CategoryLabel(masterpasal.CategoryNotification),
			Business: []masterpasal.Business{
				{ID: "2001", Name: "Personal Accident"},
				{ID: "2002", Name: "Travel"},
			},
		},
	}
}

// NewSampleRepo membentuk penyimpanan berisi contoh di atas.
func NewSampleRepo() *Repo { return NewRepo(SampleList(), SampleBusiness()) }
