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

// TabDTO adalah satu tampilan beserta kolom dan keterangannya.
type TabDTO struct {
	Code        string      `json:"kode"`
	Name        string      `json:"nama"`
	Description string      `json:"keterangan"`
	Columns     []ColumnDTO `json:"kolom"`

	// Kind menyatakan BENTUK tampilannya — daftar klaim atau ringkasan XOL.
	//
	// Ia dikirim SERVER, bukan disimpulkan layar dari kodenya, dengan alasan yang sama
	// seperti senarai kolom: inventaris tampilan adalah hasil pembacaan export, dan
	// menyalinnya ke layar berarti keputusan yang sama hidup di dua tempat — dengan yang
	// di layar tertinggal saat tampilannya bertambah.
	Kind string `json:"jenis"`

	// HasDetailAction menyatakan barisnya punya tombol "Detail Claim".
	HasDetailAction bool `json:"punya_rincian"`
}

func toTabDTO(tab inboxpladla.Tab) TabDTO {
	return TabDTO{
		Code:            tab.Code,
		Name:            tab.Name,
		Description:     tab.Description,
		Columns:         toColumnDTOs(tab.Columns),
		Kind:            string(tab.Kind),
		HasDetailAction: tab.HasDetailAction,
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

// ============================================================================
// LAYAR RINCIAN — tombol "Detail Claim"
// ============================================================================

// ClaimHeaderDTO adalah keterangan klaim di kepala layar rincian.
type ClaimHeaderDTO struct {
	ClaimKey     string `json:"kunci_klaim"`
	ClaimNo      string `json:"no_klaim"`
	PolicyNo     string `json:"no_polis"`
	Insured      string `json:"nama_tertanggung"`
	BusinessName string `json:"nama_bisnis"`
	RegisterDate string `json:"tanggal_register"`
	LossDate     string `json:"tanggal_kejadian"`
	PICTeknik    string `json:"pic_teknik"`
	StatusCode   string `json:"kode_status"`
	StatusLabel  string `json:"status"`
}

// AdviceRowDTO adalah satu baris grid PLA atau DLA.
type AdviceRowDTO struct {
	// Kind bernilai `"PLA"` atau `"DLA"`.
	Kind string `json:"jenis"`

	No   string `json:"nomor"`
	Type string `json:"tipe"`

	// Amount dikirim sebagai TEKS. Ia hanya digambar — tidak satu pun dihitung di layar —
	// dan mengirimnya sebagai angka pecahan memperkenalkan pembulatan pada nilai uang
	// yang `D-51` larang justru untuk keadaan seperti ini.
	Amount string `json:"nilai"`

	AcceptanceNo string `json:"no_akseptasi"`
	AdviceDate   string `json:"tanggal_dokumen"`
	SentDate     string `json:"tanggal_kirim"`
}

func toAdviceRowDTOs(rows []inboxpladla.AdviceRow) []AdviceRowDTO {
	out := make([]AdviceRowDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdviceRowDTO{
			Kind:         string(row.Kind),
			No:           row.No,
			Type:         row.Type,
			Amount:       row.Amount,
			AcceptanceNo: row.AcceptanceNo,
			AdviceDate:   row.AdviceDate,
			SentDate:     row.SentDate,
		})
	}
	return out
}

// DocumentRowDTO adalah satu baris grid dokumen.
type DocumentRowDTO struct {
	ID          string `json:"id"`
	Category    string `json:"jenis_dokumen"`
	SubCategory string `json:"kategori_dokumen"`
	Name        string `json:"nama"`
}

func toDocumentDTOs(rows []inboxpladla.DocumentRow) []DocumentRowDTO {
	out := make([]DocumentRowDTO, 0, len(rows))
	for _, row := range rows {
		// MimeType TIDAK ikut dikirim. Ia menentukan bagaimana berkasnya diserahkan saat
		// diunduh, dan itu keputusan peladen — bukan keterangan yang berguna di layar.
		out = append(out, DocumentRowDTO{
			ID:          row.ID,
			Category:    row.Category,
			SubCategory: row.SubCategory,
			Name:        row.Name,
		})
	}
	return out
}

// ConversationDTO adalah satu baris grid riwayat komunikasi.
type ConversationDTO struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"tanggal"`
	SenderName  string `json:"pengirim"`
	Message     string `json:"pesan"`
	Reply       string `json:"balasan"`
	ReplierName string `json:"nama_pembalas"`
	RepliedAt   string `json:"tanggal_balasan"`
	Answered    bool   `json:"sudah_dijawab"`

	// CanReply dihitung PELADEN. Layar menggambar tombol balas menurut nilai ini, bukan
	// menurut kesimpulannya sendiri — kesimpulan layar akan menggambar tombol pada
	// percakapan yang permintaannya akan ditolak.
	CanReply bool `json:"boleh_dibalas"`
}

func toConversationDTOs(rows []inboxpladla.Conversation) []ConversationDTO {
	out := make([]ConversationDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, ConversationDTO{
			ID:          row.ID,
			CreatedAt:   row.CreatedAt,
			SenderName:  row.SenderName,
			Message:     row.Message,
			Reply:       row.Reply,
			ReplierName: row.ReplierName,
			RepliedAt:   row.RepliedAt,
			Answered:    row.Answered,
			CanReply:    row.CanReply,
		})
	}
	return out
}

// DetailResponse adalah seluruh isi layar rincian satu klaim.
type DetailResponse struct {
	Claim ClaimHeaderDTO `json:"klaim"`

	PLA []AdviceRowDTO `json:"pla"`
	DLA []AdviceRowDTO `json:"dla"`

	Conversations []ConversationDTO `json:"komunikasi"`

	// Kolom keempat grid dikirim bersama isinya, sama seperti pada layar induk.
	ColumnsPLA          []ColumnDTO `json:"kolom_pla"`
	ColumnsDLA          []ColumnDTO `json:"kolom_dla"`
	ColumnsDocument     []ColumnDTO `json:"kolom_dokumen"`
	ColumnsConversation []ColumnDTO `json:"kolom_komunikasi"`

	Portal string `json:"portal"`
}

func toDetailResponse(detail usecase.Detail, portalAlias string) DetailResponse {
	return DetailResponse{
		Claim: ClaimHeaderDTO{
			ClaimKey:     detail.Claim.Header.ClaimKey,
			ClaimNo:      detail.Claim.Header.ClaimNo,
			PolicyNo:     detail.Claim.Header.PolicyNo,
			Insured:      detail.Claim.Header.Insured,
			BusinessName: detail.Claim.Header.BusinessName,
			RegisterDate: detail.Claim.Header.RegisterDate,
			LossDate:     detail.Claim.Header.LossDate,
			PICTeknik:    detail.Claim.Header.PICTeknik,
			StatusCode:   detail.Claim.Header.StatusCode,
			StatusLabel:  detail.Claim.Header.StatusLabel,
		},
		PLA:                 toAdviceRowDTOs(detail.Claim.PLA),
		DLA:                 toAdviceRowDTOs(detail.Claim.DLA),
		Conversations:       toConversationDTOs(detail.Claim.Conversations),
		ColumnsPLA:          toColumnDTOs(detail.AdviceColumnsPLA),
		ColumnsDLA:          toColumnDTOs(detail.AdviceColumnsDLA),
		ColumnsDocument:     toColumnDTOs(detail.DocumentColumns),
		ColumnsConversation: toColumnDTOs(detail.ConversationColumns),
		Portal:              portalAlias,
	}
}

// DocumentsResponse adalah isi grid dokumen satu nomor pemberitahuan.
type DocumentsResponse struct {
	Rows    []DocumentRowDTO `json:"baris"`
	Columns []ColumnDTO      `json:"kolom"`
	Portal  string           `json:"portal"`
}

// ReplyRequest adalah badan permintaan balasan komunikasi.
type ReplyRequest struct {
	ConversationID string `json:"percakapan"`
	Message        string `json:"balasan"`
}

// ReplyResponse adalah jawaban balasan yang berhasil tersimpan.
//
// Ia membawa KALIMAT, bukan sekadar status kosong: yang membalas adalah pihak luar, dan
// ia perlu tahu balasannya benar-benar sampai — bukan sekadar bahwa permintaannya
// diterima.
type ReplyResponse struct {
	Message string `json:"pesan"`
	Portal  string `json:"portal"`
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
