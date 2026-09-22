// Package masterspareparthttp adalah lapisan transport modul Master Sparepart.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// masterpanel/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterspareparthttp` supaya tidak menutupi `net/http`.
package masterspareparthttp

import (
	"time"

	"claim-pnc/internal/mastersparepart"
)

// SparepartDTO adalah bentuk satu baris master sparepart yang dikirim ke peramban.
//
// Terpisah dari mastersparepart.Sparepart supaya perubahan internal tidak bocor ke klien
// dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK, bukan nama internal (`D-80`). Yang
// dipakai adalah nama yang MENCERMINKAN ISI dan mengikuti caption layar lama — `NO_SPART`
// menjadi `nomor_sparepart` karena captionnya memang "Nomor Sparepart".
type SparepartDTO struct {
	// ID adalah kolom ID — kunci baris ini, diterbitkan server.
	ID string `json:"id_sparepart"`

	Name   string `json:"nama_sparepart"`
	Number string `json:"nomor_sparepart"`
	Code   string `json:"kode_sparepart"`

	// SellingPrice adalah kolom HARGA_JUAL, dikirim sebagai TEKS apa adanya.
	//
	// Bukan angka JSON, dan itu disengaja: JSON number diurai peramban sebagai IEEE-754
	// double, sehingga nilai uang dengan banyak digit dapat berubah hanya karena melewati
	// jaringan. Teks melewatinya tanpa disentuh (`I-12`, `D-51`).
	SellingPrice string `json:"harga_jual"`

	// CategoryID adalah kolom KATEGORI_SPART, berisi PART_CATEGORY_ID.
	CategoryID string `json:"kategori_sparepart"`

	// CategoryName adalah nama kategorinya, DIHITUNG server dari daftar acuan.
	//
	// Ia tidak ada di tabel — kolomnya hanya menyimpan ID. Dikirim supaya grid dan form
	// dapat menampilkan nama tanpa memuat ulang daftar acuan untuk setiap baris.
	//
	// Kosong berarti kategorinya tidak ada di daftar acuan yang disetujui: baris lama dapat
	// menunjuk kategori yang kemudian ditolak, dan itu keadaan yang harus terlihat alih-alih
	// disamarkan menjadi nama kosong yang tampak wajar.
	CategoryName string `json:"nama_kategori_sparepart,omitempty"`

	// TypeID adalah kolom TIPE_SPART, berisi PART_SECTION_ID.
	TypeID string `json:"tipe_sparepart"`

	// TypeName adalah nama tipenya, dihitung server dengan alasan yang sama.
	TypeName string `json:"nama_tipe_sparepart,omitempty"`

	Weight        string `json:"berat"`
	Length        string `json:"panjang"`
	Width         string `json:"lebar"`
	Height        string `json:"tinggi"`
	MinStock      string `json:"stock_minimal"`
	MaxStock      string `json:"stock_maximal"`
	OrderQuantity string `json:"kuantitas_pesanan"`

	// ProductionDate adalah kolom PROD_DATE — TEKS, bukan tanggal; lihat catatan pada
	// mastersparepart.Sparepart.
	ProductionDate string `json:"tanggal_produksi"`

	Substitute string `json:"part_substitusi"`

	Kind         string `json:"jenis_sparepart"`
	Unit         string `json:"satuan"`
	ActiveStatus string `json:"status_aktif"`
	PartStatus   string `json:"status_sparepart"`

	// UpdatedBy adalah kolom USER_UPDATE — login petugas yang terakhir menyimpannya.
	//
	// Baca-saja di layar: ia diisi penyimpanan, tidak pernah dari isian.
	UpdatedBy string `json:"user_update"`

	// PriceUpdatedAt adalah kolom TGL_UPDATE_HARGA dalam RFC 3339 UTC.
	//
	// Kosong berarti belum pernah terisi. Konversi ke WIB dilakukan di layar, di satu
	// tempat — tidak ada satu pun penambahan 7 jam di sini (`F-5`).
	PriceUpdatedAt string `json:"tanggal_update_harga,omitempty"`

	// DocumentID adalah kolom DOKUMENID.
	//
	// Ia dikirim meski layar ini tidak menyediakan unggahan lampiran: baris lama dapat
	// memilikinya, dan menyembunyikannya berarti petugas tidak punya cara mengetahui bahwa
	// lampirannya masih ada.
	DocumentID string `json:"id_dokumen"`

	// Status adalah kolom APPROVAL: "0" menunggu, "1" disetujui, "2" ditolak.
	Status string `json:"status"`

	// StatusLabel adalah sebutan status dalam bahasa yang dibaca pengguna, dihitung server
	// supaya layar tidak menyimpan salinan ketiga sandinya.
	StatusLabel string `json:"status_label"`
}

// ListResponse adalah jawaban GET /api/master/sparepart.
type ListResponse struct {
	Sparepart []SparepartDTO `json:"sparepart"`

	// Status menyebut penyaring yang benar-benar dipakai, bukan yang diminta. Keduanya sama
	// pada jalur normal; menyebutkannya membuat layar dapat memastikan tab yang ditampilkan
	// memang tab yang dijawab.
	Status string `json:"status"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang milik
	// entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan hukum, "data
	// siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// SingleResponse adalah jawaban penambahan, pengambilan, dan penyimpanan.
type SingleResponse struct {
	Sparepart SparepartDTO `json:"sparepart"`
	Portal    string       `json:"portal"`
}

// DecisionResponse adalah jawaban keputusan borongan.
type DecisionResponse struct {
	// Changed adalah jumlah baris yang BENAR-BENAR berubah, bukan jumlah yang dikirim.
	//
	// Keduanya dapat berbeda: baris yang sudah berstatus itu, atau yang sudah tidak ada,
	// tidak ikut terhitung. Menyebutkannya membuat layar dapat mengatakan "3 dari 5"
	// alih-alih melaporkan keberhasilan atas baris yang tidak tersentuh.
	Changed int `json:"jumlah_berubah"`

	Status      string `json:"status"`
	StatusLabel string `json:"status_label"`
	Portal      string `json:"portal"`
}

// CategoryDTO adalah satu pilihan Kategori Sparepart.
type CategoryDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`
}

// TypeDTO adalah satu pilihan Tipe Sparepart.
type TypeDTO struct {
	ID   string `json:"kode"`
	Name string `json:"nama"`

	// CategoryID adalah kategori induk tipe ini.
	//
	// Layar memakainya untuk mempersempit daftar Tipe begitu Kategori dipilih. Tanpa itu,
	// petugas memilih dari seluruh tipe yang ada dan dapat menautkan tipe milik kategori
	// lain — keadaan yang tidak dapat dilihat dari layar setelah tersimpan.
	CategoryID string `json:"kode_kategori"`
}

// OptionsResponse adalah jawaban GET /api/master/sparepart/pilihan.
//
// Berbeda dari Master Panel yang daftar pilihannya konstanta di dalam kode, isi di sini
// DIBACA DARI BASIS DATA entitas yang sedang dibuka — kategori dan tipe suku cadang berbeda
// antarentitas. Karena itu rutenya ikut dipasangi pemeriksaan portal; lihat Mount.
type OptionsResponse struct {
	Category []CategoryDTO `json:"kategori"`
	Type     []TypeDTO     `json:"tipe"`
	Portal   string        `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan DAN penyimpanan.
//
// # Kenapa satu bentuk untuk dua jalur
//
// Karena isiannya memang sama: `Section/BrowseMasterSparepartHEApproval-Section.xml`
// memakai form yang sama untuk menambah dan mengubah, dan
// `Activity/UpdateSparepartHE_act` melayani keduanya — yang membedakannya hanya
// `@IF(TempSparepart.ID!="", TempSparepart.ID, "UnknownID")`.
//
// # Yang TIDAK ada di sini, dan kenapa
//
//	id_sparepart          kunci baris. Pada penambahan ia diterbitkan server dari kode
//	                      situs dan sequence; pada penyimpanan ia diambil dari jalur URL.
//	status                bukan isian melainkan akibat. Penambahan dan penyimpanan SELALU
//	                      menghasilkan status menunggu; keputusan menempuh endpoint sendiri.
//	user_update           diambil dari identitas pemanggil, bukan dari badan permintaan —
//	                      kalau tidak, klien dapat mengaku sebagai orang lain.
//	tanggal_update_harga  distempel server; lihat usecase.Service.
//	id_dokumen            hasil unggah lampiran, jalur yang tidak dibawa modul ini.
type SaveRequest struct {
	Name   string `json:"nama_sparepart"`
	Number string `json:"nomor_sparepart"`
	Code   string `json:"kode_sparepart"`

	SellingPrice string `json:"harga_jual"`

	CategoryID string `json:"kategori_sparepart"`
	TypeID     string `json:"tipe_sparepart"`

	Weight        string `json:"berat"`
	Length        string `json:"panjang"`
	Width         string `json:"lebar"`
	Height        string `json:"tinggi"`
	MinStock      string `json:"stock_minimal"`
	MaxStock      string `json:"stock_maximal"`
	OrderQuantity string `json:"kuantitas_pesanan"`

	ProductionDate string `json:"tanggal_produksi"`
	Substitute     string `json:"part_substitusi"`

	Kind         string `json:"jenis_sparepart"`
	Unit         string `json:"satuan"`
	ActiveStatus string `json:"status_aktif"`
	PartStatus   string `json:"status_sparepart"`
}

// DecisionRequest adalah badan permintaan keputusan borongan.
//
// Bentuknya — DAFTAR kunci ditambah satu status — mengikuti sistem lama:
// `Activity/SetApprovalAllMaster` menelusuri baris yang dicentang lalu menetapkan status
// yang sama pada seluruhnya. Satu permintaan per baris akan mengubah operasi yang di Pega
// berupa satu tindakan menjadi sederet tindakan yang dapat gagal separuh jalan.
//
// TANPA catatan, berbeda dari Master Panel: `POOLDATA.SPAREPART_HE` tidak punya kolom
// penampungnya. Lihat usecase.Service.Decide.
type DecisionRequest struct {
	// ID adalah kunci baris yang dicentang pengguna.
	ID []string `json:"id_sparepart"`

	// Status adalah keputusan yang dikehendaki: "1" approve, "2" reject.
	//
	// "0" juga diterima — ia yang mengembalikan baris ke antrean, dan
	// `SetApprovalAllMaster` pun menerima nilai apa pun lewat `Param.approval`.
	Status string `json:"status"`
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
// Nama kuncinya `kolom`, mengikuti masterbengkel, masterpanel, masterstatusprogres, dan
// masterautoclaim. Ketiga modul master belum sepakat menamainya — masterstatus memakai
// `field` — dan penyeragamannya adalah TKT-F1-004 yang masih terhalang. Frontend sudah
// menampung keduanya lewat `APIError.violations()`.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// nameIndex adalah pemetaan kode acuan ke namanya.
//
// Dipakai toListDTO dan toDTO untuk mengisi CategoryName dan TypeName. Ia dibangun SEKALI
// per permintaan, bukan sekali per baris — pada daftar berisi ratusan sparepart, membangunnya
// ulang setiap baris berarti menelusuri daftar acuan ratusan kali untuk hasil yang sama.
type nameIndex struct {
	category map[string]string
	partType map[string]string
}

// newNameIndex menyusun pemetaan dari kedua daftar acuan.
//
// Kunci yang sudah ada TIDAK ditimpa: bila daftar acuan memuat dua baris berkode sama —
// keadaan yang mungkin karena tabelnya tidak punya constraint unik yang diketahui (`R-08`)
// — yang pertama yang menang, dan itu berperilaku sama setiap kali dipanggil.
func newNameIndex(category []mastersparepart.Category, partType []mastersparepart.PartType) nameIndex {
	index := nameIndex{
		category: make(map[string]string, len(category)),
		partType: make(map[string]string, len(partType)),
	}
	for _, one := range category {
		if _, exists := index.category[one.ID]; !exists {
			index.category[one.ID] = one.Name
		}
	}
	for _, one := range partType {
		if _, exists := index.partType[one.ID]; !exists {
			index.partType[one.ID] = one.Name
		}
	}
	return index
}

// toDTO mengubah baris domain menjadi bentuk yang dikirim ke peramban.
func toDTO(s mastersparepart.Sparepart, index nameIndex) SparepartDTO {
	dto := SparepartDTO{
		ID:             s.ID,
		Name:           s.Name,
		Number:         s.Number,
		Code:           s.Code,
		SellingPrice:   s.SellingPrice,
		CategoryID:     s.CategoryID,
		CategoryName:   index.category[s.CategoryID],
		TypeID:         s.TypeID,
		TypeName:       index.partType[s.TypeID],
		Weight:         s.Weight,
		Length:         s.Length,
		Width:          s.Width,
		Height:         s.Height,
		MinStock:       s.MinStock,
		MaxStock:       s.MaxStock,
		OrderQuantity:  s.OrderQuantity,
		ProductionDate: s.ProductionDate,
		Substitute:     s.Substitute,
		Kind:           s.Kind,
		Unit:           s.Unit,
		ActiveStatus:   s.ActiveStatus,
		PartStatus:     s.PartStatus,
		UpdatedBy:      s.UpdatedBy,
		DocumentID:     s.DocumentID,
		Status:         string(s.Status),
		StatusLabel:    s.Status.Label(),
	}

	if s.PriceUpdatedAt != nil {
		dto.PriceUpdatedAt = s.PriceUpdatedAt.UTC().Format(time.RFC3339)
	}

	return dto
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya tabel kosong terkirim sebagai
// `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri, dan satu
// layar yang lupa akan gagal saat tabelnya masih kosong.
func toListDTO(list []mastersparepart.Sparepart, index nameIndex) []SparepartDTO {
	result := make([]SparepartDTO, 0, len(list))
	for _, s := range list {
		result = append(result, toDTO(s, index))
	}
	return result
}

// toInput mengubah badan permintaan menjadi isian domain.
//
// Ia dipakai jalur tambah DAN jalur simpan, supaya keduanya tidak pernah berbeda soal isian
// mana yang diterima.
func (r SaveRequest) toInput() mastersparepart.Input {
	return mastersparepart.Input{
		Name:           r.Name,
		Number:         r.Number,
		Code:           r.Code,
		SellingPrice:   r.SellingPrice,
		CategoryID:     r.CategoryID,
		TypeID:         r.TypeID,
		Weight:         r.Weight,
		Length:         r.Length,
		Width:          r.Width,
		Height:         r.Height,
		MinStock:       r.MinStock,
		MaxStock:       r.MaxStock,
		OrderQuantity:  r.OrderQuantity,
		ProductionDate: r.ProductionDate,
		Substitute:     r.Substitute,
		Kind:           r.Kind,
		Unit:           r.Unit,
		ActiveStatus:   r.ActiveStatus,
		PartStatus:     r.PartStatus,
	}
}

// toOptionsDTO menyusun kedua daftar acuan.
//
// Kedua slice selalu dibuat, tidak pernah dibiarkan nil, dengan alasan yang sama seperti
// toListDTO.
func toOptionsDTO(
	source []mastersparepart.Category,
	kind []mastersparepart.PartType,
	portalAlias string,
) OptionsResponse {
	category := make([]CategoryDTO, 0, len(source))
	for _, one := range source {
		category = append(category, CategoryDTO{ID: one.ID, Name: one.Name})
	}

	partType := make([]TypeDTO, 0, len(kind))
	for _, one := range kind {
		partType = append(partType, TypeDTO{
			ID:         one.ID,
			Name:       one.Name,
			CategoryID: one.CategoryID,
		})
	}

	return OptionsResponse{Category: category, Type: partType, Portal: portalAlias}
}
