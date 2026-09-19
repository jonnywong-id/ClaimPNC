package masterstatusprogreshttp

import "claim-pnc/internal/masterstatusprogres"

// ProgressStatus2DTO adalah bentuk satu baris master tingkat 2 yang dikirim ke peramban.
//
// Terpisah dari masterstatusprogres.ProgressStatus2 supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
type ProgressStatus2DTO struct {
	// ID adalah kolom ID_MST. Diterbitkan server; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Nama adalah kolom STS_PROGRESS2.
	Name string `json:"nama"`

	// ParentID adalah kolom ID_PROGRESS — Status Progres 1 yang menaungi baris ini.
	ParentID string `json:"id_induk"`

	// ParentName adalah kolom STS_PROGRESS1: SALINAN nama induk pada saat baris ini
	// disimpan.
	//
	// Ia dikirim apa adanya, tidak digantikan hasil join ke tabel induk. Itu keputusan
	// Work Owner 2026-09-18 — perilakunya dijalankan as-is seperti sistem lama. Nilainya
	// karena itu dapat berbeda dari nama induk yang berlaku sekarang bila induknya pernah
	// diganti nama lewat layar Master Status Progres 1.
	ParentName string `json:"nama_induk"`

	// Tipe adalah kolom TIPE, dibaca apa adanya dan tidak pernah ditulis.
	//
	// Artinya tidak diketahui: di seluruh export ia hanya muncul pada dua SELECT, tanpa
	// satu pun INSERT, UPDATE, maupun penyaring. Ia tetap dikirim supaya nilai yang
	// benar-benar tersimpan dapat dilihat petugas — kosong pada baris yang disisipkan
	// sistem lama maupun aplikasi ini.
	Kind string `json:"tipe"`
}

// ParentDTO adalah satu pilihan pada dropdown "Status Progres 1".
//
// Ia memuat ID dan nama saja — persis kedua kolom yang dibaca
// `RDB List/BrowseMstProgress1-SQL.xml`. Kode posisi milik induk sengaja tidak ikut:
// layar tingkat 2 tidak memakainya, dan mengirim lebih dari yang dipakai membuat kontrak
// lebih sulit diubah kelak.
type ParentDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ListResponse2 adalah jawaban GET /api/master/status-progres-2.
type ListResponse2 struct {
	ProgressStatus2 []ProgressStatus2DTO `json:"status_progres_2"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini, dengan alasan
	// yang sama seperti ResponsDaftar tingkat 1.
	Portal string `json:"portal"`
}

// SingleResponse2 adalah jawaban penambahan.
type SingleResponse2 struct {
	ProgressStatus2 ProgressStatus2DTO `json:"status_progres_2"`
	Portal          string             `json:"portal"`
}

// ParentListResponse adalah jawaban GET /api/master/status-progres-2/induk.
type ParentListResponse struct {
	Parent []ParentDTO `json:"induk"`

	// Portal ikut disebut karena daftar induk BERBEDA antarentitas — ia dibaca dari
	// tabel tingkat 1 milik portal yang bersangkutan, bukan daftar tetap milik aplikasi
	// seperti halnya posisi klaim.
	Portal string `json:"portal"`
}

// SaveRequest2 adalah badan permintaan penambahan tingkat 2.
//
// Dua isian saja, persis seperti layar lama. ID diturunkan dari isi tabel; ParentName
// disalin server dari baris induk; Tipe tidak pernah ditulis. Menerima ketiganya dari
// peramban berarti mempercayai klien atas nilai yang sudah ada di basis data.
type SaveRequest2 struct {
	Name     string `json:"nama"`
	ParentID string `json:"id_induk"`
}

// toDTO2 mengubah baris domain tingkat 2 menjadi bentuk yang dikirim ke peramban.
func toDTO2(sp masterstatusprogres.ProgressStatus2) ProgressStatus2DTO {
	return ProgressStatus2DTO{
		ID:         sp.ID,
		Name:       sp.Name,
		ParentID:   sp.ParentID,
		ParentName: sp.ParentName,
		Kind:       sp.Kind,
	}
}

// toListDTO2 mengubah sekumpulan baris domain tingkat 2.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim
// sebagai `[]` dan bukan `null`.
func toListDTO2(list []masterstatusprogres.ProgressStatus2) []ProgressStatus2DTO {
	result := make([]ProgressStatus2DTO, 0, len(list))
	for _, sp := range list {
		result = append(result, toDTO2(sp))
	}
	return result
}

// toParentListDTO mengubah baris tingkat 1 menjadi pilihan dropdown.
func toParentListDTO(list []masterstatusprogres.ProgressStatus) []ParentDTO {
	result := make([]ParentDTO, 0, len(list))
	for _, sp := range list {
		result = append(result, ParentDTO{ID: sp.ID, Name: sp.Name})
	}
	return result
}
