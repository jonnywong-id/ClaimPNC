// Package masterpasalhttp adalah lapisan transport modul Master Pasal Kerugian.
//
// Ia yang mengubah permintaan HTTP menjadi pemanggilan layanan, dan galat domain menjadi
// kode status. Tidak ada satu pun aturan bisnis di sini.
package masterpasalhttp

import "claim-pnc/internal/masterpasal"

// ClauseDTO adalah bentuk satu pasal kerugian di kawat.
//
// # Kenapa nama field-nya berbahasa Indonesia, sedangkan tipe Go-nya Inggris
//
// Karena ia KONTRAK, bukan nama internal (`D-80`). Mengubahnya adalah perubahan yang
// merusak klien, bukan penggantian nama — jadi ia tidak ikut beralih bahasa bersama
// penamaan kode.
//
// # Kenapa DTO terpisah dari tipe domain
//
// Memakai tipe domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke
// klien dan sebaliknya (§2 aturan 4 `08-TECHNICAL-STRATEGY.md`). Di modul ini bedanya
// terlihat langsung: domain menyimpan daftar lini bisnis sebagai nil pada hasil daftar,
// dan DTO menjadikannya array kosong supaya klien tidak perlu membedakan `null` dari `[]`.
type ClauseDTO struct {
	// ID adalah kolom IDDATA. Diterbitkan server; tidak pernah diisi pengguna.
	ID string `json:"id"`

	// Number adalah kolom IDPASAL — di layar berlabel "No Pasal".
	//
	// Ia TIDAK dijamin unik; kuncinya adalah `id`.
	Number string `json:"no_pasal"`

	// Text adalah `$.DESCRIPTION` — di grid berlabel "ISI PASAL".
	Text string `json:"isi_pasal"`

	// Description adalah `$.OLD_D_COL_ID` — di grid berlabel "Deskripsi".
	//
	// Perhatikan pasangannya dengan `isi_pasal` di atas memang terbalik dari dugaan yang
	// wajar; itu bentuk aslinya, bukan salah petakan.
	Description string `json:"deskripsi"`

	// Category adalah `$.pyCountry` — kode kategori apa adanya.
	//
	// `"1"` Jaminan Polis · `"2"` Pengecualian · selain itu Notifikasi. Kode lain yang
	// ditemui pada baris lama dikirim apa adanya, bukan dinormalkan, supaya layar dapat
	// mengembalikannya utuh saat menyimpan.
	Category string `json:"kategori"`

	// CategoryLabel adalah sebutan kategori yang dibaca pengguna.
	//
	// Ia DITURUNKAN server dari `kategori`, dan dikirim supaya layar tidak perlu memuat
	// aturan penurunannya sendiri — dua tempat yang menurunkan hal yang sama akan
	// berbeda begitu salah satunya disunting.
	CategoryLabel string `json:"kategori_label"`

	// Business adalah lini bisnis tempat pasal ini berlaku.
	//
	// KOSONG pada hasil daftar, dan itu bukan kelalaian: grid layar lama pun tidak
	// menampilkannya, dan memuatnya untuk setiap baris berarti satu pembacaan master
	// lini bisnis per pasal demi kolom yang tidak ada. Ia terisi pada `GET` satu baris.
	Business []BusinessDTO `json:"bisnis"`
}

// BusinessDTO adalah satu lini bisnis — POOLDATA.BUSINESS.
type BusinessDTO struct {
	// ID adalah kolom ID. Ia BOLEH kosong pada butir yang diketik bebas di layar lama.
	ID string `json:"id"`

	// Name adalah kolom NOTE.
	Name string `json:"nama"`
}

// CategoryDTO adalah satu pilihan pada dropdown Kategori.
type CategoryDTO struct {
	// Code mengisi `$.pyCountry`. Ia KOSONG untuk "Notifikasi" — lihat
	// masterpasal.CategoryNotification.
	Code string `json:"kode"`

	Label string `json:"nama"`
}

// ListResponse adalah jawaban daftar pasal kerugian.
type ListResponse struct {
	Clause []ClauseDTO `json:"pasal_kerugian"`

	// Portal menyebutkan entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia bukan gema dari yang dikirim klien: satu aplikasi melayani empat badan hukum
	// dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya diandaikan
	// (ADR-0030, R-20).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban satu baris.
type SingleResponse struct {
	Clause ClauseDTO `json:"pasal_kerugian"`
	Portal string    `json:"portal"`
}

// BusinessListResponse adalah jawaban pencarian lini bisnis.
type BusinessListResponse struct {
	Business []BusinessDTO `json:"bisnis"`
	Portal   string        `json:"portal"`
}

// CategoryListResponse adalah jawaban daftar pilihan Kategori.
//
// Ia TIDAK menyebutkan portal, dan itu disengaja: isinya milik aplikasi, bukan dibaca
// dari basis data entitas mana pun. Menyebutkan portal di sini akan membuat pembaca
// menduga daftarnya berbeda antarentitas.
type CategoryListResponse struct {
	Category []CategoryDTO `json:"kategori"`
}

// DeleteResponse adalah jawaban penghapusan.
//
// Ia memuat ID yang benar-benar terhapus dan portalnya, bukan badan kosong: penghapusan
// di modul ini PERMANEN dan tidak meninggalkan jejak apa pun di basis data (lihat
// masterpasal.Repo), sehingga jawaban yang menyebutkan apa yang hilang adalah satu-satunya
// catatan yang sampai ke pemanggil.
type DeleteResponse struct {
	ID     string `json:"id"`
	Portal string `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan dan penyuntingan.
//
// ID tidak ada di sini: ia diterbitkan server saat menambah, dan datang dari jalur URL
// saat mengubah. Menerimanya dari badan permintaan berarti membuka kemungkinan badan dan
// jalur menyebut baris yang berbeda.
//
// `kategori_label` juga tidak ada: ia turunan, dan menerimanya dari layar berarti sebutan
// yang tersimpan dapat berbeda dari kodenya.
type SaveRequest struct {
	Number      string                `json:"no_pasal"`
	Text        string                `json:"isi_pasal"`
	Description string                `json:"deskripsi"`
	Category    string                `json:"kategori"`
	Business    []BusinessSaveRequest `json:"bisnis"`
}

// BusinessSaveRequest adalah satu butir lini bisnis yang dikirim layar.
//
// Keduanya dikirim — kode DAN nama — karena keduanya memang tersimpan di dalam dokumen.
// Nama TIDAK diturunkan ulang dari kode di server, supaya butir yang diketik bebas (yang
// kodenya kosong) tetap dapat tersimpan, persis seperti layar lama yang
// ber-`pyAllowFreeFormInput=true`.
type BusinessSaveRequest struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama persis dengan modul lain — `{kode, pesan, detail}` — supaya klien tidak
// menghadapi dua bentuk galat yang berbeda.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// toDTO mengubah satu baris domain menjadi bentuk kawat.
func toDTO(clause masterpasal.Clause) ClauseDTO {
	return ClauseDTO{
		ID:            clause.ID,
		Number:        clause.Number,
		Text:          clause.Text,
		Description:   clause.Description,
		Category:      clause.Category,
		CategoryLabel: clause.CategoryLabel,
		Business:      toBusinessListDTO(clause.Business),
	}
}

// toListDTO mengubah daftar baris menjadi bentuk kawat.
//
// Ia selalu mengembalikan array, tidak pernah nil, supaya klien menerima `[]` alih-alih
// `null` pada daftar kosong — dua bentuk yang menuntut dua penanganan berbeda di sisi
// klien untuk arti yang sama.
func toListDTO(list []masterpasal.Clause) []ClauseDTO {
	result := make([]ClauseDTO, 0, len(list))
	for _, clause := range list {
		result = append(result, toDTO(clause))
	}
	return result
}

// toBusinessListDTO mengubah daftar lini bisnis menjadi bentuk kawat.
func toBusinessListDTO(list []masterpasal.Business) []BusinessDTO {
	result := make([]BusinessDTO, 0, len(list))
	for _, business := range list {
		result = append(result, BusinessDTO{ID: business.ID, Name: business.Name})
	}
	return result
}

// toCategoryListDTO mengubah daftar pilihan Kategori menjadi bentuk kawat.
func toCategoryListDTO(list []masterpasal.Category) []CategoryDTO {
	result := make([]CategoryDTO, 0, len(list))
	for _, category := range list {
		result = append(result, CategoryDTO{Code: category.Code, Label: category.Label})
	}
	return result
}

// toInput mengubah badan permintaan menjadi isian domain.
//
// Ia TIDAK membersihkan dan TIDAK memeriksa apa pun — keduanya milik domain, dan
// mengerjakannya di sini berarti dua tempat yang memutuskan hal yang sama.
func toInput(request SaveRequest) masterpasal.Input {
	input := masterpasal.Input{
		Number:      request.Number,
		Text:        request.Text,
		Description: request.Description,
		Category:    request.Category,
	}
	for _, business := range request.Business {
		input.Business = append(input.Business, masterpasal.Business{
			ID:   business.ID,
			Name: business.Name,
		})
	}
	return input
}
