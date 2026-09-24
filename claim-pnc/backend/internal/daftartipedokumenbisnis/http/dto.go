// Package daftartipedokumenbisnishttp adalah lapisan transport HTTP modul Daftar Tipe
// Dokumen Bisnis.
//
// Nama paketnya berbeda dari nama foldernya supaya ia tidak menutupi `net/http` di berkas
// yang mengimpor keduanya. Pola yang sama dipakai seluruh modul lain.
package daftartipedokumenbisnishttp

import "claim-pnc/internal/daftartipedokumenbisnis"

// BusinessDTO adalah satu lini bisnis pada jawaban daftar.
//
// Nama field JSON berbahasa Indonesia (`D-80`): ia KONTRAK, bukan nama internal, dan
// mengubahnya adalah perubahan yang merusak klien — bukan penggantian nama.
type BusinessDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama_bisnis"`

	// ExcludedFromBulkSelect menandai bisnis yang dilewati tombol "Pilih semua".
	//
	// Selalu false pada grid tingkat pertama; ia hanya bermakna pada daftar pilihan.
	ExcludedFromBulkSelect bool `json:"dikecualikan_pilih_semua"`
}

// DocumentRuleDTO adalah satu aturan kelengkapan dokumen.
type DocumentRuleDTO struct {
	ID string `json:"id"`

	BusinessID   string `json:"id_bisnis"`
	BusinessName string `json:"nama_bisnis"`

	DocumentTypeID   string `json:"id_tipe_dokumen"`
	DocumentTypeName string `json:"tipe_dokumen"`

	ObjectDocID   string `json:"id_object_dokumen"`
	ObjectDocName string `json:"object_dokumen"`

	DetailTypeDocID string `json:"id_detail_dokumen"`
	DetailDocument  string `json:"detail_dokumen"`

	Mandatory   bool `json:"status_wajib"`
	MinDocument int  `json:"minimum_dokumen"`

	// Coverages hanya terisi pada pengambilan SATU baris. Pada daftar ia selalu senarai
	// kosong — bukan null — supaya klien tidak perlu membedakan "belum dimuat" dari
	// "tidak punya jaminan" dengan memeriksa null.
	Coverages []string `json:"jenis_klaim"`
}

// ReferenceDTO adalah satu pilihan pada isian yang merujuk master lain.
type ReferenceDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`

	// ParentID hanya terisi pada daftar rincian dokumen, dan di sana ia tahap dokumen
	// pemiliknya. Layar memakainya untuk menyempitkan pilihan Detail Dokumen mengikuti
	// Tipe Dokumen yang sudah dipilih pada baris yang sama — lihat Reference.ParentID di
	// lapisan domain.
	//
	// Kosong pada kedua daftar lain, dan itu bukan kekurangan: keduanya memang tidak
	// bergantung pada pilihan lain.
	ParentID string `json:"id_induk"`
}

// BusinessListResponse adalah jawaban GET /api/master/tipe-dokumen-bisnis.
type BusinessListResponse struct {
	Business []BusinessDTO `json:"bisnis"`
	Total    int           `json:"total"`
	Portal   string        `json:"portal"`

	// MayBulkSelect menyatakan apakah tombol "Pilih semua" ditampilkan.
	//
	// Ia penyembunyian TAMPILAN, bukan kewenangan — di Pega pun `pyVisible`, bukan
	// privilege. Bisnis yang sama tetap dapat dipilih satu per satu oleh siapa pun yang
	// membuka layarnya, sehingga tidak ada kemampuan yang dijaga penanda ini.
	MayBulkSelect bool `json:"boleh_pilih_semua"`
}

// RuleListResponse adalah jawaban GET /api/master/tipe-dokumen-bisnis/bisnis/{id}.
type RuleListResponse struct {
	Rules  []DocumentRuleDTO `json:"tipe_dokumen_bisnis"`
	Total  int               `json:"total"`
	Portal string            `json:"portal"`
}

// SingleResponse adalah jawaban satu baris aturan.
type SingleResponse struct {
	Rule   DocumentRuleDTO `json:"tipe_dokumen_bisnis"`
	Portal string          `json:"portal"`
}

// CreateResponse adalah jawaban penyimpanan yang menghasilkan BANYAK baris.
//
// Berbeda bentuk dari SingleResponse karena penambahan di layar ini memang menghasilkan
// perkalian bisnis kali baris dokumen — lihat BatchInput di lapisan domain.
type CreateResponse struct {
	Rules  []DocumentRuleDTO `json:"tipe_dokumen_bisnis"`
	Total  int               `json:"total"`
	Portal string            `json:"portal"`
}

// ReferenceListResponse adalah jawaban keempat rute daftar pilihan.
type ReferenceListResponse struct {
	Items  []ReferenceDTO `json:"pilihan"`
	Total  int            `json:"total"`
	Portal string         `json:"portal"`
}

// SaveRequest adalah badan permintaan penambahan.
//
// # Yang TIDAK dapat dikirim klien, dan kenapa
//
// Tidak ada `id`, `user_edit`, maupun `tgl_edit`. Ketiganya diisi server: ID dari urutan
// basis data, dan kedua jejak simpan dari sesi beserta seam Clock. Decoder menolak field
// tak dikenal, sehingga badan permintaan yang menyertakannya dijawab 400 — tanpa itu
// USER_EDIT, kolom yang justru dipakai menelusuri siapa mengubah apa, dapat diaku-aku.
type SaveRequest struct {
	// BusinessIDs adalah lini bisnis yang dituju, satu atau lebih.
	BusinessIDs []string `json:"bisnis"`

	// Rules adalah baris aturan yang dibuat pada SETIAP bisnis di atas.
	Rules []RuleRequest `json:"dokumen"`
}

// UpdateRequest adalah badan permintaan penyuntingan satu baris.
//
// Tanpa `bisnis`: lini bisnis sebuah aturan tidak dapat diubah — procedure lama pun tidak
// mengubahnya (`PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41`). Menerimanya lalu
// mengabaikannya akan membuat klien mengira perpindahan berhasil.
type UpdateRequest struct {
	RuleRequest
}

// RuleRequest adalah satu baris aturan yang dikirim layar.
type RuleRequest struct {
	DocumentTypeID  string `json:"id_tipe_dokumen"`
	ObjectDocID     string `json:"id_object_dokumen"`
	DetailTypeDocID string `json:"id_detail_dokumen"`
	DetailDocument  string `json:"detail_dokumen"`
	Mandatory       bool   `json:"status_wajib"`
	MinDocument     int    `json:"minimum_dokumen"`
}

// CoverageRequest adalah badan permintaan penambahan jaminan.
type CoverageRequest struct {
	CoverageID string `json:"id_jenis_klaim"`
}

// ErrorResponse adalah bentuk jawaban galat modul ini.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toInput mengubah badan permintaan menjadi nilai domain.
func toInput(request RuleRequest) daftartipedokumenbisnis.Input {
	return daftartipedokumenbisnis.Input{
		DocumentTypeID:  request.DocumentTypeID,
		ObjectDocID:     request.ObjectDocID,
		DetailTypeDocID: request.DetailTypeDocID,
		DetailDocument:  request.DetailDocument,
		Mandatory:       request.Mandatory,
		MinDocument:     request.MinDocument,
	}
}

// toBatchInput mengubah badan permintaan penambahan menjadi nilai domain.
func toBatchInput(request SaveRequest) daftartipedokumenbisnis.BatchInput {
	batch := daftartipedokumenbisnis.BatchInput{
		BusinessIDs: request.BusinessIDs,
		Rules:       make([]daftartipedokumenbisnis.Input, 0, len(request.Rules)),
	}
	for _, rule := range request.Rules {
		batch.Rules = append(batch.Rules, toInput(rule))
	}
	return batch
}

// toDTO mengubah satu aturan menjadi bentuk jawaban.
func toDTO(rule daftartipedokumenbisnis.DocumentRule) DocumentRuleDTO {
	// Senarai jaminan SELALU dibentuk, meski kosong, supaya ia terkirim sebagai [] dan
	// bukan null.
	coverages := make([]string, 0, len(rule.Coverages))
	for _, coverage := range rule.Coverages {
		coverages = append(coverages, coverage.ID)
	}

	return DocumentRuleDTO{
		ID:               rule.ID,
		BusinessID:       rule.BusinessID,
		BusinessName:     rule.BusinessName,
		DocumentTypeID:   rule.DocumentTypeID,
		DocumentTypeName: rule.DocumentTypeName,
		ObjectDocID:      rule.ObjectDocID,
		ObjectDocName:    rule.ObjectDocName,
		DetailTypeDocID:  rule.DetailTypeDocID,
		DetailDocument:   rule.DetailDocument,
		Mandatory:        rule.Mandatory,
		MinDocument:      rule.MinDocument,
		Coverages:        coverages,
	}
}

// toRuleListDTO mengubah daftar aturan menjadi bentuk jawaban.
func toRuleListDTO(list []daftartipedokumenbisnis.DocumentRule) []DocumentRuleDTO {
	result := make([]DocumentRuleDTO, 0, len(list))
	for _, rule := range list {
		result = append(result, toDTO(rule))
	}
	return result
}

// toBusinessListDTO mengubah daftar bisnis menjadi bentuk jawaban.
func toBusinessListDTO(list []daftartipedokumenbisnis.Business) []BusinessDTO {
	result := make([]BusinessDTO, 0, len(list))
	for _, business := range list {
		result = append(result, BusinessDTO{
			ID:                     business.ID,
			Name:                   business.Name,
			ExcludedFromBulkSelect: business.ExcludedFromBulkSelect,
		})
	}
	return result
}

// toReferenceListDTO mengubah daftar pilihan menjadi bentuk jawaban.
func toReferenceListDTO(list []daftartipedokumenbisnis.Reference) []ReferenceDTO {
	result := make([]ReferenceDTO, 0, len(list))
	for _, item := range list {
		result = append(result, ReferenceDTO{ID: item.ID, Name: item.Name, ParentID: item.ParentID})
	}
	return result
}
