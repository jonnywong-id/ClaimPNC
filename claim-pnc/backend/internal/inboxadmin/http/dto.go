// Package inboxadminhttp adalah lapisan transport modul Inbox Admin.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, kunci
// teknis Pega yang tidak pernah ditampilkan.
package inboxadminhttp

import (
	"time"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/usecase"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Tak satu pun kolom di kedelapan grid menampilkan jam, dan
// mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang dipakai
// menampilkannya — keputusan yang seharusnya hanya ada di satu tempat (`F-5`).
const dateLayout = "2006-01-02"

// WorkItemDTO adalah satu baris pekerjaan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan nama
// properti Pega: `sumber_bisnis`, bukan `rcvid`.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang
// digambar dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian,
// karena isian yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya
// menghilang begitu saja.
type WorkItemDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan tombol "Lihat Detail Klaim".
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID         string  `json:"case_id"`
	PolicyNumber   string  `json:"no_polis"`
	InsuredName    string  `json:"nama_tertanggung"`
	BusinessName   string  `json:"nama_bisnis"`
	BusinessSource string  `json:"sumber_bisnis"`
	BranchName     string  `json:"nama_cabang"`
	ClaimBranch    string  `json:"cabang_klaim"`
	Creator        string  `json:"pembuat"`
	LossDate       *string `json:"tanggal_kejadian"`
	ReportDate     *string `json:"tanggal_lapor"`
	InputDate      *string `json:"tanggal_input"`
	Note           string  `json:"catatan"`
	ClaimPosition  string  `json:"posisi_klaim"`
	ClaimStatus    string  `json:"status_klaim"`
	LODStatus      string  `json:"status_lod"`

	RequestDate  *string `json:"tanggal_request"`
	PolicyBranch string  `json:"cabang_polis"`
	SurveyBranch string  `json:"cabang_survei"`
	TechnicalPIC string  `json:"pic_klaim"`
	Surveyor     string  `json:"surveyor"`
	SurveyNumber string  `json:"no_survei"`

	InboxDate       *string `json:"tanggal_masuk_inbox"`
	AnalystNote     string  `json:"deskripsi_analis"`
	RCLPUCLStatus   string  `json:"status_rcl_pucl"`
	LetterPrintDate *string `json:"tanggal_cetak_surat"`
	ClaimAge        string  `json:"lama_klaim"`
	ExpiryStatus    string  `json:"status_kadaluarsa"`

	// Keempat angka Aging dihitung server, bukan layar, supaya ia memakai satu jam yang
	// sama untuk seluruh baris dan dapat diuji secara deterministik.
	//
	// Satuannya HARI KALENDER, bukan hari kerja — lihat inboxadmin.WorkItem.LODAgingDays.
	ReportAgingDays  *int `json:"aging_lapor"`
	TotalAgingDays   *int `json:"aging_total"`
	LODAgingDays     *int `json:"aging_lod"`
	RequestAgingDays *int `json:"aging_request"`
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

	// Ketiga penanda berikut menentukan kontrol mana yang digambar layar. Ia dikirim
	// server, bukan ditentukan layar sendiri, supaya bentuk layar punya satu sumber
	// kebenaran — dan supaya kotak cari tidak digambar pada tab yang tidak menyaringnya.
	ScopedToCaller         bool `json:"hanya_milik_saya"`
	SupportsSearch         bool `json:"pakai_pencarian"`
	SupportsBusinessFilter bool `json:"pakai_lini_bisnis"`
}

// BusinessLineDTO adalah satu pilihan pada dropdown lini bisnis.
type BusinessLineDTO struct {
	Code  string `json:"kode"`
	Label string `json:"label"`
}

// DisabledTabDTO adalah satu tab sistem lama yang sengaja tidak dibangun.
//
// Ia dikirim ke layar, bukan disembunyikan: pengguna yang mencari tab "Not Answered"
// memperoleh jawaban alih-alih menduga modulnya belum selesai.
type DisabledTabDTO struct {
	Code   string `json:"kode"`
	Name   string `json:"nama"`
	Reason string `json:"alasan"`
}

// MetadataResponse adalah jawaban GET /api/inbox-admin/tab.
type MetadataResponse struct {
	Tabs          []TabDTO          `json:"tab"`
	DefaultTab    string            `json:"tab_bawaan"`
	BusinessLines []BusinessLineDTO `json:"lini_bisnis"`
	DisabledTabs  []DisabledTabDTO  `json:"tab_dinonaktifkan"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal
	// yang sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`

	// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta
	// alasannya, dalam kalimat yang dapat langsung ditampilkan ke pengguna.
	//
	// Ia dikirim sebagai data, bukan ditulis tetap di layar, supaya ia hilang dengan
	// sendirinya begitu penghalangnya hilang — tanpa menyunting frontend.
	Limitations []string `json:"keterbatasan"`
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
// Ia dikirim balik karena tidak selalu sama dengan yang diminta: tab yang tidak mendukung
// pencarian mengembalikan kata kunci kosong, dan layar dapat membersihkan kotaknya alih-alih
// menampilkan kata kunci yang tampak aktif padahal tidak menyaring apa pun.
type AppliedFilterDTO struct {
	Business string `json:"bisnis"`
	Keyword  string `json:"cari"`
}

// ListResponse adalah jawaban GET /api/inbox-admin.
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
func toWorkItemDTO(item inboxadmin.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:      item.Reference,
		CaseID:         item.CaseID,
		PolicyNumber:   item.PolicyNumber,
		InsuredName:    item.InsuredName,
		BusinessName:   item.BusinessName,
		BusinessSource: item.BusinessSource,
		BranchName:     item.BranchName,
		ClaimBranch:    item.ClaimBranch,
		Creator:        item.Creator,
		LossDate:       toDateString(item.LossDate),
		ReportDate:     toDateString(item.ReportDate),
		InputDate:      toDateString(item.InputDate),
		Note:           item.Note,
		ClaimPosition:  item.ClaimPosition,
		ClaimStatus:    item.ClaimStatus,
		LODStatus:      item.LODStatus,

		RequestDate:  toDateString(item.RequestDate),
		PolicyBranch: item.PolicyBranch,
		SurveyBranch: item.SurveyBranch,
		TechnicalPIC: item.TechnicalPIC,
		Surveyor:     item.Surveyor,
		SurveyNumber: item.SurveyNumber,

		InboxDate:       toDateString(item.InboxDate),
		AnalystNote:     item.AnalystNote,
		RCLPUCLStatus:   item.RCLPUCLStatus,
		LetterPrintDate: toDateString(item.LetterPrintDate),
		ClaimAge:        item.ClaimAge,
		ExpiryStatus:    item.ExpiryStatus,

		ReportAgingDays:  item.ReportAgingDays,
		TotalAgingDays:   item.TotalAgingDays,
		LODAgingDays:     item.LODAgingDays,
		RequestAgingDays: item.RequestAgingDays,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxadmin.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxadmin.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:                   tab.Code,
		Name:                   tab.Name,
		Description:            tab.Description,
		Columns:                columns,
		ScopedToCaller:         tab.ScopedToCaller,
		SupportsSearch:         tab.SupportsSearch,
		SupportsBusinessFilter: tab.SupportsBusinessFilter,
	}
}

// limitations adalah keterbatasan modul ini yang perlu diketahui pengguna.
//
// Ketiganya bukan cacat, melainkan akibat keputusan yang sudah diambil. Menuliskannya di
// layar membuat pengguna tidak melaporkannya berulang kali sebagai kerusakan.
var limitations = []string{
	"Kolom Aging dihitung dalam hari kalender, bukan hari kerja. Kalender libur dibaca " +
		"sistem lama lewat sambungan ke basis data HRD yang belum punya API pengganti.",
	"Penyaring Cabang dan Korwil belum aktif, sehingga daftar ini belum dibatasi cabang " +
		"Anda. Sumbernya sama dengan kalender libur di atas.",
	"Kotak cari hanya menelusuri Case ID dan No Polis — sama seperti di sistem lama.",
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	lines := make([]BusinessLineDTO, 0, len(meta.BusinessLines))
	for _, line := range meta.BusinessLines {
		lines = append(lines, BusinessLineDTO{Code: string(line), Label: line.Label()})
	}

	disabled := make([]DisabledTabDTO, 0, len(meta.DisabledTabs))
	for _, tab := range meta.DisabledTabs {
		disabled = append(disabled, DisabledTabDTO{
			Code: tab.Code, Name: tab.Name, Reason: tab.Reason,
		})
	}

	return MetadataResponse{
		Tabs:          tabs,
		DefaultTab:    meta.DefaultTab,
		BusinessLines: lines,
		DisabledTabs:  disabled,
		Portal:        portalAlias,
		Limitations:   limitations,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxadmin.Page) PaginationDTO {
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
		Filter: AppliedFilterDTO{
			Business: string(listed.Query.Business),
			Keyword:  listed.Query.Keyword,
		},
		Portal: portalAlias,
	}
}

// toDateString memformat tanggal, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}
