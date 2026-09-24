// Package inboxmanagerreceivepuclhttp adalah lapisan transport modul Inbox Manager
// Receive / PUCL.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, isian yang
// tidak pernah digambar sebagai kolom.
package inboxmanagerreceivepuclhttp

import (
	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxmanagerreceivepucl/usecase"
)

// WorkItemDTO adalah satu baris pekerjaan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: sembilan dari enam belas isian memang hanya
// berlaku pada salah satu tab, dan satu — `jumlah_lembar_dokumen` — SELALU kosong karena
// sumbernya tidak ada di basis data mana pun.
type WorkItemDTO struct {
	// Reference adalah kunci teknis Pega yang dibutuhkan tombol rincian.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID       string `json:"no_case"`
	PolicyNumber string `json:"no_polis"`
	ClaimNumber  string `json:"no_klaim_pnc"`
	InsuredName  string `json:"nama_tertanggung"`
	LossDate     string `json:"tanggal_kejadian"`

	// ClaimType diturunkan dari Group Panel, bukan dibaca dari isian aslinya — lihat
	// inboxmanagerreceivepucl.ClaimTypeOf. Ia kosong pada tab RCL/PUCL.
	ClaimType string `json:"jenis_klaim"`

	SenderName           string `json:"nama_pengirim"`
	DocumentReceivedDate string `json:"tanggal_terima_dokumen"`

	// DocumentSheetCount SELALU kosong: tidak ada kolom basis data untuknya di seluruh
	// export. Ia tetap dikirim supaya kolomnya dapat digambar dan ketiadaannya terlihat.
	DocumentSheetCount string `json:"jumlah_lembar_dokumen"`

	InboxEntryAt    string `json:"tanggal_masuk_inbox"`
	AnalystNote     string `json:"deskripsi_analyst"`
	Track           string `json:"rcl_pucl"`
	TrackStatus     string `json:"status_rcl_pucl"`
	LetterPrintedAt string `json:"tanggal_cetak_surat"`

	// ClaimAge dikirim sebagai TEKS, bukan angka. Satuannya tidak diketahui: tidak satu pun
	// kueri di export menghitungnya, dan tidak ada DDL yang menyatakan tipenya (`R-08`).
	ClaimAge string `json:"lama_klaim"`

	ExpiryStatus string `json:"status_kadaluarsa"`
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

	// FromWorkbasket menyatakan barisnya diambil dari antrean BERSAMA, bukan dari penugasan
	// per orang. Layar memakainya untuk menjelaskan antrean yang kosong menurut sebabnya.
	FromWorkbasket bool `json:"antrean_bersama"`

	// Ketiga isian berikut menyatakan tab yang digambar tetapi belum dapat diisi.
	//
	// Tidak ada tab terhalang di layar ini hari ini. Isian tetap dikirim sebagai DATA
	// supaya penambahan tab terhalang kelak tidak menuntut perubahan kontrak API maupun
	// suntingan frontend.
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-manager-receive-pucl/tab.
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

// ListResponse adalah jawaban GET /api/inbox-manager-receive-pucl.
type ListResponse struct {
	Tab        TabDTO        `json:"tab"`
	Items      []WorkItemDTO `json:"baris"`
	Pagination PaginationDTO `json:"paginasi"`
	Portal     string        `json:"portal"`
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
func toWorkItemDTO(item inboxmanagerreceivepucl.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:            item.Reference,
		CaseID:               item.CaseID,
		PolicyNumber:         item.PolicyNumber,
		ClaimNumber:          item.ClaimNumber,
		InsuredName:          item.InsuredName,
		LossDate:             item.LossDate,
		ClaimType:            item.ClaimType,
		SenderName:           item.SenderName,
		DocumentReceivedDate: item.DocumentReceivedDate,
		DocumentSheetCount:   item.DocumentSheetCount,
		InboxEntryAt:         item.InboxEntryAt,
		AnalystNote:          item.AnalystNote,
		Track:                item.Track,
		TrackStatus:          item.TrackStatus,
		LetterPrintedAt:      item.LetterPrintedAt,
		ClaimAge:             item.ClaimAge,
		ExpiryStatus:         item.ExpiryStatus,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxmanagerreceivepucl.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxmanagerreceivepucl.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:           tab.Code,
		Name:           tab.Name,
		Description:    tab.Description,
		Columns:        columns,
		FromWorkbasket: tab.FromWorkbasket,
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
func toPaginationDTO(page inboxmanagerreceivepucl.Page) PaginationDTO {
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
		Portal:     portalAlias,
	}
}
