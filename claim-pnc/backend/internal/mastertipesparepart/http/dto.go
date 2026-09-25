// Package mastertipespareparthttp — lihat catatan nama di bawah — adalah lapisan transport
// modul Master Tipe Sparepart.
//
// Namanya panjang, dan itu konsekuensi `D-81`: folder modul memakai nama modul bisnis apa
// adanya ("Master Tipe Sparepart"), dan paket transportnya menambahkan akhiran `http`
// tanpa tanda hubung karena Go tidak mengizinkannya. Pola yang sama dipakai
// `masterkategorispareparthttp`, `masterspareparthttp`, dan `masterpanelhttp`.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package mastertipespareparthttp

import "claim-pnc/internal/mastertipesparepart"

// PartTypeDTO adalah satu baris tipe sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode — dan
// nama field JSON termasuk di dalamnya, karena mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama.
//
// # Kenapa enam field untuk tabel berkolom empat
//
// Dua di antaranya bukan kolom tabel ini. `nama_kategori_sparepart` milik tabel kategori
// dan ikut dibaca lewat JOIN — grid layar lama memang menampilkan keduanya berdampingan.
// `status_label` diturunkan dari `status` supaya layar tidak perlu menyimpan petanya
// sendiri.
//
// Tidak ada `user_update` maupun `tanggal_update`: `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak
// punya kolomnya. Mengirim keduanya sebagai string kosong akan membuat layar menggambar
// kolom yang selamanya kosong.
type PartTypeDTO struct {
	// ID adalah kolom PART_SECTION_ID. Teks, meski isinya angka — lihat catatan pada
	// mastertipesparepart.PartType.
	ID string `json:"id_tipe_sparepart"`

	// Name adalah kolom PART_SECTION_NAME.
	Name string `json:"nama_tipe_sparepart"`

	// CategoryID adalah kolom PART_CATEGORY_ID — induk tipe ini.
	CategoryID string `json:"id_kategori_sparepart"`

	// CategoryName adalah PART_CATEGORY_NAME milik tabel kategori, dibaca lewat JOIN.
	//
	// Ia dapat KOSONG, dan itu bukan galat: kuerinya memakai LEFT JOIN, sehingga tipe yang
	// menunjuk kategori yang tidak ada tetap terkirim dengan nama kategori kosong. Di sistem
	// lama baris seperti itu justru HILANG dari daftar. Layar menampilkannya sebagai tanda
	// "—" beserta keterangan, bukan sebagai sel kosong yang tidak dapat dijelaskan.
	CategoryName string `json:"nama_kategori_sparepart"`

	// Status adalah kolom APPROVAL, dikirim apa adanya sebagai "0", "1", atau "2".
	//
	// Sandinya yang dikirim, bukan labelnya, supaya klien membandingkan nilai dan bukan
	// teks. Label ikut dikirim terpisah.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status yang dibaca pengguna.
	StatusLabel string `json:"status_label"`
}

// ListResponse adalah jawaban GET /master/tipe-sparepart.
type ListResponse struct {
	// Type selalu berupa array, tidak pernah null — layar tidak perlu menjaga dua bentuk
	// kosong yang berbeda. Lihat toListDTO.
	Type []PartTypeDTO `json:"tipe_sparepart"`

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
	Type   PartTypeDTO `json:"tipe_sparepart"`
	Portal string      `json:"portal"`
}

// CategoryDTO adalah satu pilihan pada dropdown Kategori.
//
// Nama kuncinya `kode` dan `nama`, mengikuti `mastersparepart.CategoryDTO` yang lebih dulu
// menamai hal yang sama. Dua layar yang menawarkan daftar kategori karena itu memakai
// bentuk yang identik, dan komponen dropdown-nya tidak perlu tahu dari endpoint mana
// daftarnya datang.
type CategoryDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`
}

// OptionsResponse adalah jawaban GET /master/tipe-sparepart/pilihan.
//
// Berbeda dari Master Panel yang daftar pilihannya konstanta di dalam kode, isi di sini
// DIBACA DARI BASIS DATA entitas yang sedang dibuka — kategori suku cadang berbeda
// antarentitas. Karena itu rutenya ikut dipasangi pemeriksaan portal; lihat Mount.
type OptionsResponse struct {
	Category []CategoryDTO `json:"kategori"`

	// Truncated menyatakan daftarnya terpotong pada batas lookup.
	//
	// Dikirim supaya layar dapat mengatakannya kepada pengguna. Tanpa itu, pemotongan
	// terjadi diam-diam dan terbaca sebagai "kategorinya belum dibuat".
	Truncated bool `json:"terpotong"`

	Portal string `json:"portal"`
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
// DUA isian. ID tidak ada di sini: pada penambahan ia diterbitkan server, dan pada
// penyimpanan ia diambil dari jalur URL. Status juga tidak ada — menyimpan selalu
// mengembalikan baris ke antrean persetujuan, dan menerimanya dari klien akan membuka jalan
// memutuskan persetujuan lewat jalur simpan.
//
// `nama_kategori_sparepart` juga tidak ada, dan itu disengaja: ia milik tabel kategori dan
// tidak pernah ditulis modul ini. Menerimanya dari klien akan mengundang layar mengirim
// nama yang sudah basi, dan membuat pembaca menyangka namanya ikut tersimpan.
type SaveRequest struct {
	Name       string `json:"nama_tipe_sparepart"`
	CategoryID string `json:"id_kategori_sparepart"`
}

// toInput mengubah badan permintaan menjadi nilai domain.
func (r SaveRequest) toInput() mastertipesparepart.Input {
	return mastertipesparepart.Input{
		Name:       r.Name,
		CategoryID: r.CategoryID,
	}
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// TANPA catatan: tabelnya tidak punya kolom penampungnya. Lihat usecase.Service.Decide.
type DecisionRequest struct {
	ID     []string `json:"id_tipe_sparepart"`
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
// Nama kuncinya `kolom`, mengikuti masterkategorisparepart, masterbengkel, masterpanel,
// mastersparepart, masterstatusprogres, dan masterautoclaim. Penyeragamannya dengan
// masterstatus — yang memakai `field` — adalah TKT-F1-004 yang masih terhalang. Frontend
// sudah menampung keduanya lewat `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// toDTO mengubah satu baris domain menjadi bentuk yang dikirim ke klien.
func toDTO(t mastertipesparepart.PartType) PartTypeDTO {
	return PartTypeDTO{
		ID:           t.ID,
		Name:         t.Name,
		CategoryID:   t.CategoryID,
		CategoryName: t.CategoryName,
		Status:       string(t.Status),
		StatusLabel:  t.Status.Label(),
	}
}

// toListDTO mengubah daftar domain menjadi daftar DTO.
//
// Selalu mengembalikan slice yang TIDAK nil, sehingga JSON-nya `[]` dan bukan `null`. Tanpa
// ini, layar harus menjaga dua bentuk kosong yang berbeda — dan satu layar yang lupa akan
// gagal menggambar tabel kosong.
func toListDTO(list []mastertipesparepart.PartType) []PartTypeDTO {
	result := make([]PartTypeDTO, 0, len(list))
	for _, t := range list {
		result = append(result, toDTO(t))
	}
	return result
}

// toOptionsDTO menyusun daftar acuan kategori.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, dengan alasan yang sama seperti
// toListDTO.
func toOptionsDTO(
	source []mastertipesparepart.Category,
	truncated bool,
	portalAlias string,
) OptionsResponse {
	category := make([]CategoryDTO, 0, len(source))
	for _, one := range source {
		category = append(category, CategoryDTO{ID: one.ID, Name: one.Name})
	}
	return OptionsResponse{
		Category:  category,
		Truncated: truncated,
		Portal:    portalAlias,
	}
}
