// Package inboxmanageradminhttp adalah lapisan transport modul Inbox Manager Admin.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata pada dua isian: `Reference`, kunci
// teknis Pega yang tidak pernah digambar, dan `RegisteredAt`, yang di domain adalah
// `*time.Time` sedangkan kontraknya teks.
package inboxmanageradminhttp

import (
	"time"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/usecase"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Seragam dengan modul lain, dan seragamnya disengaja: layar memformat tanggal lewat satu
// fungsi bersama (`shared/components/format`), yang hanya mengenali bentuk ini.
const dateLayout = "2006-01-02"

// WorkItemDTO adalah satu baris antrean.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian, karena isian
// yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya menghilang
// begitu saja.
type WorkItemDTO struct {
	// Reference adalah kunci teknis Pega yang dibutuhkan tombol rincian.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom. Di layar lama, nilai inilah yang
	// disusun `SetAssignmentInboxReg_act` menjadi kunci assignment yang dibuka tombolnya.
	Reference string `json:"referensi"`

	CaseID         string `json:"id"`
	PolicyNumber   string `json:"no_polis"`
	InsuredName    string `json:"nama_tertanggung"`
	BusinessName   string `json:"nama_bisnis"`
	BusinessSource string `json:"nama_sumber_bisnis"`

	// RegisteredAt adalah Tanggal Pendaftaran.
	//
	// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""`
	// akan terbaca layar sebagai tanggal yang gagal diformat.
	RegisteredAt *string `json:"tanggal_pendaftaran"`

	// ClaimElapsed adalah Lama Waktu Klaim, sudah berbentuk teks siap tampil —
	// misalnya "1 year 5 months ago".
	//
	// Ia dikirim sebagai TEKS, bukan sebagai jumlah hari, karena yang ditiru adalah format
	// bawaan Pega `DateTime-Frame` yang memang menyusun kalimat. Mengirim angka lalu
	// menyusun kalimatnya di layar berarti aturan bentuknya hidup di dua tempat.
	ClaimElapsed string `json:"lama_waktu_klaim"`

	AdminName string `json:"admin_pnc"`

	// ClaimStatus TIDAK digambar sebagai kolom oleh layar — grid Pega pun tidak. Ia
	// dikirim karena berkas ekspor memuatnya, dan supaya layar dapat menampilkannya kelak
	// tanpa perubahan kontrak.
	ClaimStatus string `json:"status_klaim"`
}

// ColumnDTO adalah satu kolom grid.
type ColumnDTO struct {
	// Key menyebut isian mana pada baris yang digambar.
	Key string `json:"kunci"`

	// Title adalah judul kolom yang dibaca pengguna.
	Title string `json:"judul"`
}

// TabDTO adalah satu antrean beserta bentuk gridnya.
type TabDTO struct {
	Code        string `json:"kode"`
	Name        string `json:"nama"`
	Description string `json:"keterangan"`

	Columns []ColumnDTO `json:"kolom"`

	// OrgUnit adalah unit organisasi penugasan yang disaring tab ini.
	//
	// Ia dikirim ke layar BUKAN untuk digambar sebagai kolom, melainkan untuk dipakai
	// dalam pesan saat antreannya kosong. "Tidak ada data" tidak cukup di layar ini:
	// antrean yang kosong dapat berarti unit organisasinya memang sepi, atau kolom
	// `PXASSIGNEDORGUNIT` pada POOLDATA.T_CLAIMLIST_ADMIN tidak pernah terisi — dan yang
	// kedua adalah kekeliruan konfigurasi yang tidak menghasilkan satu pun galat.
	OrgUnit string `json:"unit_organisasi"`

	// LineBusiness adalah lini bisnis yang membuka tab ini, dibaca dari
	// `M_LOGIN_PNC.LINE_BUSINESS`.
	LineBusiness string `json:"lini_bisnis"`
}

// MetadataResponse adalah jawaban GET /api/inbox-manager-admin/tab.
type MetadataResponse struct {
	// Tabs adalah tab yang BOLEH dilihat pemanggil — bukan ketiganya.
	Tabs []TabDTO `json:"tab"`

	// DefaultTab adalah tab pertama yang boleh dilihat pemanggil. Kosong bila tidak ada
	// satu pun.
	DefaultTab string `json:"tab_bawaan"`

	// AllTabs adalah KETIGA tab tanpa memandang kewenangan, supaya layar dapat menyebutkan
	// apa saja yang ada ketika pengguna tidak berhak atas satu pun.
	AllTabs []TabDTO `json:"semua_tab"`

	// ExpectedLineBusinesses adalah lini bisnis yang membuka tab.
	ExpectedLineBusinesses []string `json:"lini_bisnis_yang_diharapkan"`

	// CallerLineBusiness adalah lini bisnis pemanggil apa adanya, dipantulkan kembali
	// supaya pengguna tahu nilai APA yang terbaca sistem saat ia tidak melihat satu tab
	// pun — dan agar "kolomnya belum diisi" dapat dibedakan dari "diisi dengan nilai yang
	// tidak dikenal".
	CallerLineBusiness string `json:"lini_bisnis_anda"`

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

// ListResponse adalah jawaban GET /api/inbox-manager-admin.
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
func toWorkItemDTO(item inboxmanageradmin.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:      item.Reference,
		CaseID:         item.CaseID,
		PolicyNumber:   item.PolicyNumber,
		InsuredName:    item.InsuredName,
		BusinessName:   item.BusinessName,
		BusinessSource: item.BusinessSource,
		RegisteredAt:   toDateString(item.RegisteredAt),
		ClaimElapsed:   item.ClaimElapsed,
		AdminName:      item.AdminName,
		ClaimStatus:    item.ClaimStatus,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxmanageradmin.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxmanageradmin.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:         tab.Code,
		Name:         tab.Name,
		Description:  tab.Description,
		Columns:      columns,
		OrgUnit:      tab.OrgUnit,
		LineBusiness: tab.LineBusiness,
	}
}

// toTabListDTO mengubah sekumpulan tab.
func toTabListDTO(tabs []inboxmanageradmin.Tab) []TabDTO {
	result := make([]TabDTO, 0, len(tabs))
	for _, tab := range tabs {
		result = append(result, toTabDTO(tab))
	}
	return result
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	lines := make([]string, 0, len(meta.ExpectedLineBusinesses))
	lines = append(lines, meta.ExpectedLineBusinesses...)


	return MetadataResponse{
		Tabs:                   toTabListDTO(meta.Tabs),
		DefaultTab:             meta.DefaultTab,
		AllTabs:                toTabListDTO(meta.AllTabs),
		ExpectedLineBusinesses: lines,
		CallerLineBusiness:     meta.CallerLineBusiness,
		Portal:                 portalAlias,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxmanageradmin.Page) PaginationDTO {
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
