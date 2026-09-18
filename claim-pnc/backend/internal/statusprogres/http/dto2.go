package statusprogreshttp

import "claim-pnc/internal/statusprogres"

// StatusProgres2DTO adalah bentuk satu baris master tingkat 2 yang dikirim ke peramban.
//
// Terpisah dari statusprogres.StatusProgres2 supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type StatusProgres2DTO struct {
	// ID adalah kolom ID_MST. Diterbitkan server; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Nama adalah kolom STS_PROGRESS2.
	Nama string `json:"nama"`

	// IDInduk adalah kolom ID_PROGRESS — Status Progres 1 yang menaungi baris ini.
	IDInduk string `json:"id_induk"`

	// NamaInduk adalah kolom STS_PROGRESS1: SALINAN nama induk pada saat baris ini
	// disimpan.
	//
	// Ia dikirim apa adanya, tidak digantikan hasil join ke tabel induk. Itu keputusan
	// Work Owner 2026-09-18 — perilakunya dijalankan as-is seperti sistem lama. Nilainya
	// karena itu dapat berbeda dari nama induk yang berlaku sekarang bila induknya pernah
	// diganti nama lewat layar Master Status Progres 1.
	NamaInduk string `json:"nama_induk"`

	// Tipe adalah kolom TIPE, dibaca apa adanya dan tidak pernah ditulis.
	//
	// Artinya tidak diketahui: di seluruh export ia hanya muncul pada dua SELECT, tanpa
	// satu pun INSERT, UPDATE, maupun penyaring. Ia tetap dikirim supaya nilai yang
	// benar-benar tersimpan dapat dilihat petugas — kosong pada baris yang disisipkan
	// sistem lama maupun aplikasi ini.
	Tipe string `json:"tipe"`
}

// IndukDTO adalah satu pilihan pada dropdown "Status Progres 1".
//
// Ia memuat ID dan nama saja — persis kedua kolom yang dibaca
// `RDB List/BrowseMstProgress1-SQL.xml`. Kode posisi milik induk sengaja tidak ikut:
// layar tingkat 2 tidak memakainya, dan mengirim lebih dari yang dipakai membuat kontrak
// lebih sulit diubah kelak.
type IndukDTO struct {
	ID   string `json:"id"`
	Nama string `json:"nama"`
}

// ResponsDaftar2 adalah jawaban GET /api/master/status-progres-2.
type ResponsDaftar2 struct {
	StatusProgres2 []StatusProgres2DTO `json:"status_progres_2"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini, dengan alasan
	// yang sama seperti ResponsDaftar tingkat 1.
	Portal string `json:"portal"`
}

// ResponsSatu2 adalah jawaban penambahan.
type ResponsSatu2 struct {
	StatusProgres2 StatusProgres2DTO `json:"status_progres_2"`
	Portal         string            `json:"portal"`
}

// ResponsInduk adalah jawaban GET /api/master/status-progres-2/induk.
type ResponsInduk struct {
	Induk []IndukDTO `json:"induk"`

	// Portal ikut disebut karena daftar induk BERBEDA antarentitas — ia dibaca dari
	// tabel tingkat 1 milik portal yang bersangkutan, bukan daftar tetap milik aplikasi
	// seperti halnya posisi klaim.
	Portal string `json:"portal"`
}

// PermintaanSimpan2 adalah badan permintaan penambahan tingkat 2.
//
// Dua isian saja, persis seperti layar lama. ID diturunkan dari isi tabel; NamaInduk
// disalin server dari baris induk; Tipe tidak pernah ditulis. Menerima ketiganya dari
// peramban berarti mempercayai klien atas nilai yang sudah ada di basis data.
type PermintaanSimpan2 struct {
	Nama    string `json:"nama"`
	IDInduk string `json:"id_induk"`
}

// keDTO2 mengubah baris domain tingkat 2 menjadi bentuk yang dikirim ke peramban.
func keDTO2(sp statusprogres.StatusProgres2) StatusProgres2DTO {
	return StatusProgres2DTO{
		ID:        sp.ID,
		Nama:      sp.Nama,
		IDInduk:   sp.IDInduk,
		NamaInduk: sp.NamaInduk,
		Tipe:      sp.Tipe,
	}
}

// keDaftarDTO2 mengubah sekumpulan baris domain tingkat 2.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null`.
func keDaftarDTO2(daftar []statusprogres.StatusProgres2) []StatusProgres2DTO {
	hasil := make([]StatusProgres2DTO, 0, len(daftar))
	for _, sp := range daftar {
		hasil = append(hasil, keDTO2(sp))
	}
	return hasil
}

// keDaftarIndukDTO mengubah baris tingkat 1 menjadi pilihan dropdown.
func keDaftarIndukDTO(daftar []statusprogres.StatusProgres) []IndukDTO {
	hasil := make([]IndukDTO, 0, len(daftar))
	for _, sp := range daftar {
		hasil = append(hasil, IndukDTO{ID: sp.ID, Nama: sp.Nama})
	}
	return hasil
}
