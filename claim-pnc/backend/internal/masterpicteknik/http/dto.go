// Package masterpicteknikhttp adalah lapisan transport modul Master PIC Teknik: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp,
// portalhttp, dan masterstatushttp: foldernya `http` supaya letaknya seragam antarmodul,
// nama paketnya `masterpicteknikhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterpicteknikhttp

// PICTeknikDTO adalah bentuk satu petugas teknik yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari masterpicteknik.PICTeknik. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4).
type PICTeknikDTO struct {
	IDOperator string `json:"id_operator"`

	// Nama berasal dari direktori operator, bukan dari isian. Ia dikirim supaya layar
	// dapat menampilkannya, tetapi tidak pernah diterima kembali.
	Nama string `json:"nama"`

	Email      string `json:"email"`
	LiniBisnis string `json:"lini_bisnis"`
	Grup       string `json:"grup"`
	Atasan     string `json:"atasan"`
	Kuota      int    `json:"kuota"`

	// KuotaLuar adalah COUNTER_QUOTA2 — beban kerja petugas yang sama di sistem lain.
	// Namanya di sistem lama, alias "OLD_OPERATOR_ID", menyesatkan: isinya angka.
	KuotaLuar int `json:"kuota_luar"`

	// GrupPanel hanya dibaca: procedure penulis lama tidak pernah menulis kolom ini.
	GrupPanel string `json:"grup_panel"`

	Aktif bool `json:"aktif"`
}

// ResponsDaftar adalah jawaban GET /api/master/pic-teknik.
type ResponsDaftar struct {
	PICTeknik []PICTeknikDTO `json:"pic_teknik"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai —
	// sama dengan modul master lain, supaya layar master berperilaku seragam.
	Total int `json:"total"`
}

// ResponsSatu adalah jawaban untuk satu petugas: ambil, tambah, dan ubah.
type ResponsSatu struct {
	PICTeknik PICTeknikDTO `json:"pic_teknik"`
}

// PermintaanSimpan adalah isian form tambah dan ubah.
//
// Dua hal yang TIDAK diterima, dan itu disengaja:
//
//   - Nama. Ia dimiliki direktori operator dan dicari server setiap kali disimpan.
//     Menerimanya berarti master ini dapat memuat nama yang tidak cocok dengan
//     direktori, dan setiap layar penugasan akan menyebut orang yang berbeda dari yang
//     sesungguhnya bertugas.
//   - GrupPanel. Procedure lama tidak pernah menulisnya; menerimanya di sini berarti
//     menambah perilaku yang tidak pernah ada.
//
// IDOperator hanya dipakai pada penambahan. Pada perubahan ia diambil dari jalur URL —
// dua sumber untuk satu nilai berarti keduanya dapat berbeda.
type PermintaanSimpan struct {
	IDOperator string `json:"id_operator"`
	Email      string `json:"email"`
	LiniBisnis string `json:"lini_bisnis"`
	Grup       string `json:"grup"`
	Atasan     string `json:"atasan"`
	Kuota      int    `json:"kuota"`
	KuotaLuar  int    `json:"kuota_luar"`
	Aktif      bool   `json:"aktif"`
}

// PelanggaranDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form.
type PelanggaranDTO struct {
	Field string `json:"field"`
	Pesan string `json:"pesan"`
}

// ResponsGalat adalah bentuk galat modul ini.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuknya sama persis dengan masterstatushttp.ResponsGalat. Menyatukan keduanya
// menjadi satu tipe bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat
// seluruh aplikasi, dan tiket itu masih terhalang keputusan Work Owner.
type ResponsGalat struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []PelanggaranDTO `json:"detail,omitempty"`
}
