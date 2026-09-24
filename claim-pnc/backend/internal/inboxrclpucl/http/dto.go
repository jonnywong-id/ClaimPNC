// Package inboxrclpuclhttp adalah lapisan transport modul Inbox RCL/PUCL.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, isian yang
// tidak pernah digambar sebagai kolom.
package inboxrclpuclhttp

import (
	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// WorkItemDTO adalah satu baris pada grid.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
//
// Di layar ini sifat itu bukan kehalusan: `tanggal_cetak_surat` SELALU kosong pada tab
// "Cetak Surat" — penyaringnya `IS NULL` — dan kolomnya tetap harus digambar karena layar
// lama menggambarnya.
type WorkItemDTO struct {
	// Reference adalah kunci teknis Pega yang dibutuhkan tombol rincian.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID       string `json:"no_case"`
	PolicyNumber string `json:"no_polis"`
	InsuredName  string `json:"nama_tertanggung"`

	// InboxEntryAt adalah "Tanggal Masuk Inbox" — tanggal klaimnya DIKIRIM ke jalur
	// RCL/PUCL, bukan tanggal objek kerjanya dibuat. Keduanya kolom yang berbeda, dan yang
	// dipakai mengurutkan justru yang TIDAK dikirim ke sini.
	InboxEntryAt string `json:"tanggal_masuk_inbox"`

	AnalystNote string `json:"deskripsi_analyst"`

	// Track adalah jalur penanganan — "RCL" atau "PUCL".
	//
	// Kosong bila kode jalurnya tidak dikenali, meniru `CASE` tanpa `ELSE` di sistem lama.
	// Judul kolomnya berbeda antartab: "Status RCL/PUCL" pada dua tab pertama, "Status"
	// pada tab Klaim MSIG — dan perbedaan itu datang dari `kolom`, bukan dari sini.
	Track string `json:"status_rcl_pucl"`

	// LetterPrintedAt SELALU kosong pada tab "Cetak Surat", dan itu bukan data hilang —
	// justru itulah arti tab tersebut.
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

// TabDTO adalah satu tab beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	Columns []ColumnDTO `json:"kolom"`

	// HasDateRangeReport menyatakan tombol ekspor tab ini menghasilkan LAPORAN HARIAN
	// berbasis rentang tanggal, bukan salinan tabel.
	//
	// Layar memakainya untuk menampilkan kedua isian tanggal — dan untuk menjelaskan bahwa
	// isian itu TIDAK menyaring tabel di bawahnya.
	HasDateRangeReport bool `json:"punya_laporan_rentang_tanggal"`

	// Notice adalah keterangan yang berlaku pada tab ini saja, digambar di atas grid.
	// Kosong berarti tidak ada.
	Notice string `json:"catatan,omitempty"`

	// Ketiga isian berikut menyatakan tab yang digambar tetapi belum dapat diisi.
	//
	// Tidak ada tab terhalang di layar ini. Isian tetap dikirim sebagai DATA supaya
	// penambahan tab terhalang kelak tidak menuntut perubahan kontrak API maupun suntingan
	// frontend.
	Blocked       bool   `json:"terhalang"`
	BlockedReason string `json:"alasan_terhalang,omitempty"`
	BlockedOwner  string `json:"pemilik_penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-rcl-pucl/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// ReportColumns adalah kolom berkas laporan harian.
	//
	// Dikirim supaya layar dapat menyebutkan isi berkasnya SEBELUM diunduh — isinya
	// berbeda dari tabel yang sedang dilihat, dan perbedaan itu tidak boleh baru ketahuan
	// setelah berkasnya dibuka.
	ReportColumns []ColumnDTO `json:"kolom_laporan"`

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

// ListResponse adalah jawaban GET /api/inbox-rcl-pucl.
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
func toWorkItemDTO(item inboxrclpucl.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:       item.Reference,
		CaseID:          item.CaseID,
		PolicyNumber:    item.PolicyNumber,
		InsuredName:     item.InsuredName,
		InboxEntryAt:    item.InboxEntryAt,
		AnalystNote:     item.AnalystNote,
		Track:           item.Track,
		LetterPrintedAt: item.LetterPrintedAt,
		ClaimAge:        item.ClaimAge,
		ExpiryStatus:    item.ExpiryStatus,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxrclpucl.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toColumnListDTO mengubah senarai kolom.
func toColumnListDTO(columns []inboxrclpucl.Column) []ColumnDTO {
	result := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		result = append(result, ColumnDTO{Key: column.Key, Title: column.Title})
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxrclpucl.Tab) TabDTO {
	return TabDTO{
		Code:               tab.Code,
		Name:               tab.Name,
		Description:        tab.Description,
		Columns:            toColumnListDTO(tab.Columns),
		HasDateRangeReport: tab.HasDateRangeReport,
		Notice:             tab.Notice,
		Blocked:            tab.Blocked,
		BlockedReason:      tab.BlockedReason,
		BlockedOwner:       tab.BlockedOwner,
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
		ReportColumns:      toColumnListDTO(meta.ReportColumns),
		PlannedDifferences: differences,
		Portal:             portalAlias,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxrclpucl.Page) PaginationDTO {
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
