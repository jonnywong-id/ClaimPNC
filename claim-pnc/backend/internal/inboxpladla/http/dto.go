// Package inboxpladlahttp adalah lapisan transport modul Inbox PLA DLA.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8).
//
// Di modul ini pemisahan itu punya satu alasan tambahan yang khas: yang membaca jawabannya
// adalah PIHAK LUAR — mitra reasuransi. Setiap isian yang tersisip ke dalam bentuk JSON
// keluar dari dinding perusahaan, sehingga isinya ditulis satu per satu di sini alih-alih
// diserahkan begitu saja dari tipe domain.
package inboxpladlahttp

import (
	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/usecase"
)

// RowDTO adalah satu baris daftar klaim.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend (`D-80`).
// Namanya mengikuti apa yang dibaca pengguna di kolom grid, BUKAN alias Pega: sepuluh
// dari sebelas alias itu tidak menyatakan isinya.
type RowDTO struct {
	// ClaimKey adalah `T_CLAIM_PNC.CLAIMID`, beralias `"TSI"` di kueri lama.
	//
	// Ia kunci objek kerja Pega, BUKAN nilai pertanggungan. Dikirim tetapi tidak
	// digambar sebagai kolom.
	ClaimKey string `json:"kunci_klaim"`

	ClaimNo      string `json:"no_klaim"`
	PolicyNo     string `json:"no_polis"`
	Insured      string `json:"nama_tertanggung"`
	BusinessName string `json:"nama_bisnis"`

	// Tanggal dikirim sebagai teks `YYYY-MM-DD`. Layar yang memformatnya ke bentuk
	// Indonesia, dan konversi zona waktunya terjadi di sana.
	RegisterDate string `json:"tanggal_register"`
	LossDate     string `json:"tanggal_kejadian"`

	PICTeknik string `json:"pic_teknik"`

	// StatusCode adalah kode status yang DIGAMBAR — sudah termasuk penggantian menjadi
	// `1139` pada daftar DLA untuk klaim yang menunggu penutupan.
	StatusCode string `json:"kode_status"`

	// StatusLabel adalah artinya. Kosong bila kodenya tidak ada di master status.
	//
	// Layar menggambar kodenya ketika label kosong — kode yang terbaca lebih berguna
	// daripada sel yang kosong.
	StatusLabel string `json:"status"`

	AdviceNo  string `json:"no_pla"`
	CloseNote string `json:"catatan_tutup"`
}

func toRowDTO(row inboxpladla.Row) RowDTO {
	return RowDTO{
		ClaimKey:     row.ClaimKey,
		ClaimNo:      row.ClaimNo,
		PolicyNo:     row.PolicyNo,
		Insured:      row.Insured,
		BusinessName: row.BusinessName,
		RegisterDate: row.RegisterDate,
		LossDate:     row.LossDate,
		PICTeknik:    row.PICTeknik,
		StatusCode:   row.StatusCode,
		StatusLabel:  row.StatusLabel,
		AdviceNo:     row.AdviceNo,
		CloseNote:    row.CloseNote,
	}
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	Key   string `json:"kunci"`
	Title string `json:"judul"`
	Date  bool   `json:"tanggal"`
}

func toColumnDTOs(columns []inboxpladla.Column) []ColumnDTO {
	out := make([]ColumnDTO, 0, len(columns))
	for _, column := range columns {
		out = append(out, ColumnDTO{
			Key:   column.Key,
			Title: column.Title,
			Date:  column.Date,
		})
	}
	return out
}

// TabDTO adalah satu daftar beserta kolom dan keterangannya.
type TabDTO struct {
	Code        string      `json:"kode"`
	Name        string      `json:"nama"`
	Description string      `json:"keterangan"`
	Columns     []ColumnDTO `json:"kolom"`
}

func toTabDTO(tab inboxpladla.Tab) TabDTO {
	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		Columns:     toColumnDTOs(tab.Columns),
	}
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"daftar"`
	DefaultTab string   `json:"daftar_bawaan"`

	// XOLColumns adalah kolom grid "DATA PLA DLA XOL KLAIM".
	XOLColumns []ColumnDTO `json:"kolom_xol"`

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string `json:"selisih_terencana"`

	// Portal adalah alias entitas yang sedang dijawab.
	Portal string `json:"portal"`
}

func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	return MetadataResponse{
		Tabs:               tabs,
		DefaultTab:         meta.DefaultTab,
		XOLColumns:         toColumnDTOs(meta.XOLColumns),
		PlannedDifferences: meta.PlannedDifferences,
		Portal:             portalAlias,
	}
}

// PageDTO adalah keterangan paginasi.
type PageDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// ListResponse adalah isi satu daftar.
type ListResponse struct {
	Tab    TabDTO   `json:"daftar"`
	Rows   []RowDTO `json:"baris"`
	Paging PageDTO  `json:"paginasi"`

	// Search adalah kata kunci yang BENAR-BENAR dipakai.
	Search string `json:"cari"`

	Portal string `json:"portal"`
}

func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	rows := make([]RowDTO, 0, len(listed.Page.Items))
	for _, row := range listed.Page.Items {
		rows = append(rows, toRowDTO(row))
	}

	return ListResponse{
		Tab:  toTabDTO(listed.Query.Tab),
		Rows: rows,
		Paging: PageDTO{
			Page:       listed.Page.Pagination.Page,
			Size:       listed.Page.Pagination.Size,
			Total:      listed.Page.Total,
			TotalPages: listed.Page.TotalPages(),
		},
		Search: listed.Query.Search,
		Portal: portalAlias,
	}
}

// StatusCountDTO adalah satu baris tabel ringkas "Status / Jumlah".
type StatusCountDTO struct {
	Code  string `json:"kode_status"`
	Label string `json:"status"`
	Total int    `json:"jumlah"`
}

// CountsResponse adalah isi tabel ringkas.
type CountsResponse struct {
	Rows   []StatusCountDTO `json:"baris"`
	Portal string           `json:"portal"`
}

func toCountsResponse(
	counts []inboxpladla.StatusCount,
	portalAlias string,
) CountsResponse {
	rows := make([]StatusCountDTO, 0, len(counts))
	for _, count := range counts {
		rows = append(rows, StatusCountDTO{
			Code:  count.Code,
			Label: count.Label,
			Total: count.Total,
		})
	}
	return CountsResponse{Rows: rows, Portal: portalAlias}
}

// XOLRowDTO adalah satu baris grid "DATA PLA DLA XOL KLAIM".
type XOLRowDTO struct {
	Year           string `json:"tahun"`
	CauseOfLoss    string `json:"penyebab_kerugian"`
	Kind           string `json:"jenis"`
	LastInsertDate string `json:"tanggal_terakhir"`
}

// XOLResponse adalah isi grid XOL.
type XOLResponse struct {
	Rows   []XOLRowDTO `json:"baris"`
	Portal string      `json:"portal"`
}

func toXOLResponse(rows []inboxpladla.XOLRow, portalAlias string) XOLResponse {
	out := make([]XOLRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, XOLRowDTO{
			Year:           row.Year,
			CauseOfLoss:    row.CauseOfLoss,
			Kind:           row.Kind,
			LastInsertDate: row.LastInsertDate,
		})
	}
	return XOLResponse{Rows: out, Portal: portalAlias}
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}
