// Package inboxclaimtreatyprophttp adalah lapisan transport modul Inbox Claim Treaty Prop.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa WorkKey dan
// Reference, dua kunci teknis Pega yang tidak pernah ditampilkan.
package inboxclaimtreatyprophttp

import (
	"claim-pnc/internal/inboxclaimtreatyprop"
	"claim-pnc/internal/inboxclaimtreatyprop/usecase"
)

// WorkItemDTO adalah satu baris pekerjaan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan nama
// properti Pega: `ceding_co`, bukan `cari8`.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
type WorkItemDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan tombol rincian klaim.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	ClaimID        string `json:"claim_id"`
	MasterID       string `json:"id_master"`
	PolicyNumber   string `json:"no_polis"`
	LossDate       string `json:"tanggal_kejadian"`
	BusinessName   string `json:"nama_bisnis"`
	BusinessSource string `json:"sumber_bisnis"`
	CedingCompany  string `json:"ceding_co"`
	InsuredName    string `json:"nama_tertanggung"`
	Subjectivity   string `json:"subjectivity"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`
}

// TabDTO adalah satu antrean kerja beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	Columns []ColumnDTO `json:"kolom"`

	// Kedua penanda berikut menentukan kontrol mana yang digambar layar. Ia dikirim
	// server, bukan ditentukan layar sendiri, supaya bentuk layar punya satu sumber
	// kebenaran — dan supaya checkbox "See All Claim" tidak digambar pada tab yang tidak
	// mengenalnya.
	ScopedToCaller bool `json:"hanya_milik_saya"`
	SupportsSeeAll bool `json:"pakai_lihat_semua"`

	// Ketiga isian berikut menyatakan tab yang digambar tetapi belum dapat diisi.
	//
	// Ia dikirim sebagai DATA, bukan ditulis tetap di layar, supaya hilang dengan
	// sendirinya begitu penghalangnya hilang — tanpa menyunting frontend.
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-claim-treaty-prop/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string `json:"selisih_terencana"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal
	// yang sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// AppliedFilterDTO adalah penyaring yang BENAR-BENAR dipakai.
//
// Ia dikirim balik karena tidak selalu sama dengan yang diminta: tab yang tidak mengenal
// "See All Claim" mengembalikan false, dan layar dapat mematikan centangnya alih-alih
// menampilkan centang yang tampak aktif padahal tidak mengubah apa pun.
type AppliedFilterDTO struct {
	SeeAll bool `json:"lihat_semua"`
}

// ListResponse adalah jawaban GET /api/inbox-claim-treaty-prop.
type ListResponse struct {
	Tab        TabDTO           `json:"tab"`
	Items      []WorkItemDTO    `json:"baris"`
	Pagination PaginationDTO    `json:"paginasi"`
	Filter     AppliedFilterDTO `json:"penyaring"`
	Portal     string           `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toWorkItemDTO mengubah satu baris.
func toWorkItemDTO(item inboxclaimtreatyprop.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:      item.Reference,
		ClaimID:        item.ClaimID,
		MasterID:       item.MasterID,
		PolicyNumber:   item.PolicyNumber,
		LossDate:       item.LossDate,
		BusinessName:   item.BusinessName,
		BusinessSource: item.BusinessSource,
		CedingCompany:  item.CedingCompany,
		InsuredName:    item.InsuredName,
		Subjectivity:   item.Subjectivity,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxclaimtreatyprop.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxclaimtreatyprop.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:           tab.Code,
		Name:           tab.Name,
		Description:    tab.Description,
		Columns:        columns,
		ScopedToCaller: tab.ScopedToCaller,
		SupportsSeeAll: tab.SupportsSeeAll,
		Blocked:        tab.Blocked,
		BlockedReason:  tab.BlockedReason,
		BlockedOwner:   tab.BlockedOwner,
	}
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	differences := make([]string, 0, len(meta.PlannedDifferences))
	differences = append(differences, meta.PlannedDifferences...)

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		PlannedDifferences: differences,
		Portal:             portalAlias,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxclaimtreatyprop.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// toListResponse merakit jawaban isi satu tab.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	return ListResponse{
		Tab:        toTabDTO(listed.Query.Tab),
		Items:      toWorkItemListDTO(listed.Page.Items),
		Pagination: toPaginationDTO(listed.Page),
		Filter:     AppliedFilterDTO{SeeAll: listed.Query.SeeAll},
		Portal:     portalAlias,
	}
}
