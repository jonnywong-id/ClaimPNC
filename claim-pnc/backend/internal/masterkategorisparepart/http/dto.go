// Package masterkategorispareparthttp — lihat catatan nama di bawah — adalah lapisan
// transport modul Master Kategori Sparepart.
//
// Namanya panjang, dan itu konsekuensi `D-81`: folder modul memakai nama modul bisnis apa
// adanya ("Master Kategori Sparepart"), dan paket transportnya menambahkan akhiran `http`
// tanpa tanda hubung karena Go tidak mengizinkannya. Pola yang sama dipakai
// `masterspareparthttp`, `masterpanelhttp`, dan `masterbengkelhttp`.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package masterkategorispareparthttp

import "claim-pnc/internal/masterkategorisparepart"

// PartCategoryDTO adalah satu baris kategori sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode —
// dan nama field JSON termasuk di dalamnya, karena mengubahnya adalah perubahan yang
// merusak klien, bukan penggantian nama.
//
// # Kenapa hanya empat field
//
// Karena tabelnya hanya punya tiga kolom, dan yang keempat — `status_label` — diturunkan
// dari yang ketiga supaya layar tidak perlu menyimpan petanya sendiri. Tidak ada
// `user_update` maupun `tanggal_update`: `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya
// kolomnya. Mengirim keduanya sebagai string kosong akan membuat layar menggambar kolom
// yang selamanya kosong.
type PartCategoryDTO struct {
	// ID adalah kolom PART_CATEGORY_ID. Teks, meski isinya angka — lihat catatan pada
	// masterkategorisparepart.PartCategory.
	ID string `json:"id_kategori_sparepart"`

	// Name adalah kolom PART_CATEGORY_NAME.
	Name string `json:"nama_kategori_sparepart"`

	// Status adalah kolom APPROVAL, dikirim apa adanya sebagai "0", "1", atau "2".
	//
	// Sandinya yang dikirim, bukan labelnya, supaya klien membandingkan nilai dan bukan
	// teks. Label ikut dikirim terpisah.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status yang dibaca pengguna.
	StatusLabel string `json:"status_label"`
}

// ListResponse adalah jawaban GET /master/kategori-sparepart.
type ListResponse struct {
	// Category selalu berupa array, tidak pernah null — layar tidak perlu menjaga dua
	// bentuk kosong yang berbeda. Lihat toListDTO.
	Category []PartCategoryDTO `json:"kategori_sparepart"`

	// Status adalah penyaring yang benar-benar dipakai, dikembalikan supaya layar dapat
	// memastikan jawaban yang tiba memang milik tab yang sedang dibuka.
	Status string `json:"status"`

	// Portal adalah entitas yang menjawab. Ia dikirim pada SETIAP jawaban, dan itu bukan
	// hiasan: satu aplikasi melayani empat badan hukum dengan basis data terpisah, dan
	// "data siapa ini" tidak boleh hanya diandaikan (ADR-0030, R-20).
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban satu baris — dipakai Get, Create, dan Save.
type SingleResponse struct {
	Category PartCategoryDTO `json:"kategori_sparepart"`
	Portal   string          `json:"portal"`
}

// DecisionResponse adalah jawaban keputusan borongan.
type DecisionResponse struct {
	// Changed adalah jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dipilih.
	// Keduanya dapat berbeda bila daftar di layar sudah basi, dan pengguna berhak tahu.
	Changed int `json:"jumlah_berubah"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Portal      string `json:"portal"`
}

// SaveRequest adalah badan permintaan tambah dan simpan.
//
// SATU isian saja. ID tidak ada di sini: pada penambahan ia diterbitkan server, dan pada
// penyimpanan ia diambil dari jalur URL. Status juga tidak ada — menyimpan selalu
// mengembalikan baris ke antrean persetujuan, dan menerimanya dari klien akan membuka
// jalan memutuskan persetujuan lewat jalur simpan.
type SaveRequest struct {
	Name string `json:"nama_kategori_sparepart"`
}

// toInput mengubah badan permintaan menjadi nilai domain.
func (r SaveRequest) toInput() masterkategorisparepart.Input {
	return masterkategorisparepart.Input{Name: r.Name}
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// TANPA catatan: tabelnya tidak punya kolom penampungnya. Lihat
// usecase.Service.Decide.
type DecisionRequest struct {
	ID     []string `json:"id_kategori_sparepart"`
	Status string   `json:"status"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — ditambah `detail` untuk pelanggaran
// per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan mencocokkan
// teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterbengkel, masterpanel, mastersparepart,
// masterstatusprogres, dan masterautoclaim. Penyeragamannya dengan masterstatus — yang
// memakai `field` — adalah TKT-F1-004 yang masih terhalang. Frontend sudah menampung
// keduanya lewat `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah satu baris domain menjadi bentuk yang dikirim ke klien.
func toDTO(c masterkategorisparepart.PartCategory) PartCategoryDTO {
	return PartCategoryDTO{
		ID:          c.ID,
		Name:        c.Name,
		Status:      string(c.Status),
		StatusLabel: c.Status.Label(),
	}
}

// toListDTO mengubah daftar domain menjadi daftar DTO.
//
// Selalu mengembalikan slice yang TIDAK nil, sehingga JSON-nya `[]` dan bukan `null`.
// Tanpa ini, layar harus menjaga dua bentuk kosong yang berbeda — dan satu layar yang lupa
// akan gagal menggambar tabel kosong.
func toListDTO(list []masterkategorisparepart.PartCategory) []PartCategoryDTO {
	result := make([]PartCategoryDTO, 0, len(list))
	for _, c := range list {
		result = append(result, toDTO(c))
	}
	return result
}
