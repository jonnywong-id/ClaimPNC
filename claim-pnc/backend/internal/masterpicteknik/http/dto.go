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
	OperatorID string `json:"id_operator"`

	// Nama berasal dari direktori operator, bukan dari isian. Ia dikirim supaya layar
	// dapat menampilkannya, tetapi tidak pernah diterima kembali.
	Name string `json:"nama"`

	Email      string `json:"email"`
	BusinessLine string `json:"lini_bisnis"`
	Group       string `json:"grup"`
	Supervisor     string `json:"atasan"`
	Quota      int    `json:"kuota"`

	// KuotaLuar adalah COUNTER_QUOTA2 — beban kerja petugas yang sama di sistem lain.
	// Namanya di sistem lama, alias "OLD_OPERATOR_ID", menyesatkan: isinya angka.
	ExternalQuota int `json:"kuota_luar"`

	// GrupPanel hanya dibaca: procedure penulis lama tidak pernah menulis kolom ini.
	GrupPanel string `json:"grup_panel"`

	Active bool `json:"aktif"`
}

// ResponsDaftar adalah jawaban GET /api/master/pic-teknik.
type ListResponse struct {
	PICTeknik []PICTeknikDTO `json:"pic_teknik"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai —
	// sama dengan modul master lain, supaya layar master berperilaku seragam.
	Total int `json:"total"`
}

// ResponsSatu adalah jawaban untuk satu petugas: ambil, tambah, dan ubah.
type SingleResponse struct {
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
type SaveRequest struct {
	OperatorID string `json:"id_operator"`
	Email      string `json:"email"`
	BusinessLine string `json:"lini_bisnis"`
	Group       string `json:"grup"`
	Supervisor     string `json:"atasan"`
	Quota      int    `json:"kuota"`
	ExternalQuota  int    `json:"kuota_luar"`
	Active      bool   `json:"aktif"`
}

// PelanggaranDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form.
type ViolationDTO struct {
	Field string `json:"field"`
	Message string `json:"pesan"`
}

// ResponsGalat adalah bentuk galat modul ini.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuknya sama persis dengan masterstatushttp.ResponsGalat. Menyatukan keduanya
// menjadi satu tipe bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat
// seluruh aplikasi, dan tiket itu masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code  string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
