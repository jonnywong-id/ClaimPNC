// Package masterpasalaihttp adalah lapisan transport modul Master Pasal AI.
//
// Ia yang mengubah permintaan HTTP menjadi pemanggilan layanan, dan galat domain menjadi
// kode status. Tidak ada satu pun aturan bisnis di sini.
package masterpasalaihttp

import "claim-pnc/internal/masterpasalai"

// ClauseDTO adalah bentuk satu baris wording polis di kawat.
//
// # Kenapa nama field-nya berbahasa Indonesia, sedangkan tipe Go-nya Inggris
//
// Karena ia KONTRAK, bukan nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama — jadi ia tidak ikut beralih bahasa bersama penamaan kode.
//
// Nama field-nya mengikuti **judul kolom di layar**, bukan nama kolom basis data maupun nama
// properti klipboard Pega. Ketiganya berbeda di modul ini, dan hanya judul kolom yang
// bermakna bagi pembaca:
//
//	layar       kolom basis data   properti Pega (warisan, menyesatkan)
//	No Pasal    WP_PASAL           .City
//	Ayat        WP_AYAT            .CityID
//	Kejadian    WP_KEJADIAN        .District
type ClauseDTO struct {
	// ID adalah kolom WP_ID — kunci baris.
	//
	// TIDAK digambar sebagai kolom: layar lamanya hanya punya tiga. Ia dikirim sebagai
	// kunci baris tabel, dan kueri lamanya pun memilihnya (`WP_ID AS "CaseID"`) serta
	// mengurutkan dengannya.
	ID string `json:"id"`

	// Number adalah kolom WP_PASAL — di grid berlabel "No Pasal".
	Number string `json:"no_pasal"`

	// Paragraph adalah kolom WP_AYAT — di grid berlabel "Ayat".
	Paragraph string `json:"ayat"`

	// Event adalah kolom WP_KEJADIAN — di grid berlabel "Kejadian".
	//
	// Teks panjang: di layar lama ia satu-satunya yang digambar `pxTextArea`.
	Event string `json:"kejadian"`
}

// PageDTO adalah keterangan halaman yang menyertai setiap daftar.
//
// Ia ada karena paginasi modul ini dikerjakan di SERVER — berbeda dari seluruh layar master
// lain di aplikasi ini, yang memaginasi di peramban. Tanpa keterangan ini, layar tidak punya
// cara mengetahui ada berapa halaman.
//
// Itu bukan pilihan kami: layar lamanya pun sudah begitu. Grid Pega-nya ber-`pyPageMode =
// None` dan jendelanya dihitung activity lewat `FirstRow`/`LastRow`.
type PageDTO struct {
	// Number adalah nomor halaman yang dikembalikan, mulai dari 1.
	Number int `json:"halaman"`

	// Size adalah banyaknya baris per halaman — selalu masterpasalai.PageSize.
	//
	// Ikut dikirim, bukan ditanam di layar, supaya keduanya tidak berpisah diam-diam bila
	// angkanya kelak berubah.
	Size int `json:"ukuran_halaman"`

	// Count adalah banyaknya halaman yang tersedia, minimal 1.
	Count int `json:"jumlah_halaman"`

	// Total adalah cacah SELURUH baris yang cocok dengan kata kuncinya.
	//
	// Bukan cacah baris pada halaman ini. Layar lama memuatnya lewat kueri terpisah
	// (`CountDataPasalAI`), dan angkanya yang menggerakkan paginator.
	Total int `json:"jumlah_baris"`
}

// ListResponse adalah jawaban GET /master/pasal-ai.
type ListResponse struct {
	Clause []ClauseDTO `json:"pasal_ai"`
	Page   PageDTO     `json:"paginasi"`
	Portal string      `json:"portal"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — sehingga klien tidak menghadapi dua
// bentuk galat yang berbeda.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toDTO mengubah satu baris domain menjadi bentuk kawat.
func toDTO(one masterpasalai.Clause) ClauseDTO {
	return ClauseDTO{
		ID:        one.ID,
		Number:    one.Number,
		Paragraph: one.Paragraph,
		Event:     one.Event,
	}
}

// toListDTO mengubah satu halaman domain menjadi daftar bentuk kawat.
//
// Ia mengembalikan senarai KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan `null` — klien
// tidak perlu membedakan keduanya.
func toListDTO(list []masterpasalai.Clause) []ClauseDTO {
	result := make([]ClauseDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toDTO(one))
	}
	return result
}

// toPageDTO menyusun keterangan paginasi dari satu halaman domain.
func toPageDTO(page masterpasalai.Page) PageDTO {
	return PageDTO{
		Number: page.Number,
		Size:   masterpasalai.PageSize,
		Count:  page.PageCount(),
		Total:  page.Total,
	}
}
