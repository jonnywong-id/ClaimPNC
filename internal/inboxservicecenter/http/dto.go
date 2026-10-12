// Package inboxservicecenterhttp adalah lapisan transport modul Inbox Service Center.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe domain
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya — dan
// di modul ini bocornya akan nyata: ServiceClaim membawa isian yang tidak pernah digambar
// sebagai kolom.
package inboxservicecenterhttp

import (
	"time"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/usecase"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Kolom "Tanggal Input" pada grid Pega tidak menampilkan jam, dan
// mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang dipakai
// menampilkannya — keputusan yang seharusnya hanya ada di satu tempat (`F-5`).
const dateLayout = "2006-01-02"

// ServiceClaimDTO adalah satu baris klaim portal rekanan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan nama
// properti Pega: `nasabah`, bukan `username`.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang digambar
// dari `kolom` pada tab yang sedang terbuka.
type ServiceClaimDTO struct {
	ID           string  `json:"id"`
	InputDate    *string `json:"tanggal_input"`
	PolicyNumber string  `json:"no_polis"`
	CustomerName string  `json:"nasabah"`
	Type         string  `json:"tipe"`
	TechnicalPIC string  `json:"pic"`

	// Keempat isian berikut TIDAK digambar sebagai kolom — grid Pega hanya punya enam.
	//
	// Ia tetap dikirim karena dibutuhkan hal lain: `repair_id` untuk membuka rincian kelak,
	// `no_klaim` dan `imei` karena keduanya ikut dicari kotak "Cari", sehingga pengguna dapat
	// melihat mengapa sebuah baris cocok.
	RepairID    string `json:"repair_id"`
	ClaimNumber string `json:"no_klaim"`
	IMEI        string `json:"imei"`

	// Kedua status dikirim sebagai KODE dan LABEL sekaligus.
	//
	// Kode supaya layar dapat membandingkannya tanpa mencocokkan teks yang dapat berubah;
	// label supaya layar tidak perlu memuat tabel terjemahannya sendiri — tabel itu dibaca
	// dari rule Pega dan tempatnya di backend (`inboxservicecenter/tab.go`).
	RepairStatus        string `json:"status_perbaikan"`
	RepairStatusLabel   string `json:"status_perbaikan_label"`
	ApprovalStatus      string `json:"status_persetujuan"`
	ApprovalStatusLabel string `json:"status_persetujuan_label"`
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

	// PegaParam adalah nilai `stsapprove` sistem lama untuk tab ini.
	//
	// Ia ikut dikirim demi ketelusuran, bukan untuk digambar: saat uji kesetaraan gerbang 1
	// menemukan selisih, inilah yang menghubungkan tab di layar dengan langkah activity di
	// export. Teks kosong pada tab pertama memang begitu adanya.
	PegaParam string `json:"parameter_pega"`

	Columns []ColumnDTO `json:"kolom"`
}

// MetadataResponse adalah jawaban GET /api/inbox-service-center/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal yang
	// sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`

	// Limitations menyatakan hal yang BELUM berjalan penuh di modul ini beserta alasannya,
	// dalam kalimat yang dapat langsung ditampilkan ke pengguna.
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

	// Active menyatakan hasilnya memang dipotong per halaman.
	//
	// Ia bernilai false saat pengguna sedang mencari — sistem lama mematikan paginasinya
	// sendiri begitu kotak cari terisi. Layar memakainya untuk menyembunyikan tombol halaman
	// alih-alih menggambar tombol yang tidak melakukan apa pun.
	Active bool `json:"aktif"`
}

// AppliedFilterDTO adalah penyaring yang BENAR-BENAR dipakai.
//
// Ia dikirim balik supaya kotak cari di layar selalu memperlihatkan kata kunci yang benar-
// benar menyaring — bukan yang sempat diketik lalu tidak jadi terkirim.
type AppliedFilterDTO struct {
	Keyword string `json:"cari"`
}

// ListResponse adalah jawaban GET /api/inbox-service-center.
type ListResponse struct {
	Tab        TabDTO            `json:"tab"`
	Items      []ServiceClaimDTO `json:"baris"`
	Pagination PaginationDTO     `json:"paginasi"`
	Filter     AppliedFilterDTO  `json:"penyaring"`
	Portal     string            `json:"portal"`
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

// toServiceClaimDTO mengubah satu baris.
func toServiceClaimDTO(claim inboxservicecenter.ServiceClaim) ServiceClaimDTO {
	return ServiceClaimDTO{
		ID:           claim.ID,
		InputDate:    toDateString(claim.InputDate),
		PolicyNumber: claim.PolicyNumber,
		CustomerName: claim.CustomerName,
		Type:         claim.Type,
		TechnicalPIC: claim.TechnicalPIC,

		RepairID:    claim.RepairID,
		ClaimNumber: claim.ClaimNumber,
		IMEI:        claim.IMEI,

		RepairStatus:        claim.RepairStatus,
		RepairStatusLabel:   inboxservicecenter.RepairStatusLabel(claim.RepairStatus),
		ApprovalStatus:      claim.ApprovalStatus,
		ApprovalStatusLabel: inboxservicecenter.ApprovalStatusLabel(claim.ApprovalStatus),
	}
}

// toServiceClaimListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toServiceClaimListDTO(claims []inboxservicecenter.ServiceClaim) []ServiceClaimDTO {
	result := make([]ServiceClaimDTO, 0, len(claims))
	for _, claim := range claims {
		result = append(result, toServiceClaimDTO(claim))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxservicecenter.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		PegaParam:   tab.PegaParam,
		Columns:     columns,
	}
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	limitations := make([]string, 0, len(meta.Limitations))
	limitations = append(limitations, meta.Limitations...)

	return MetadataResponse{
		Tabs:        tabs,
		DefaultTab:  meta.DefaultTab,
		Portal:      portalAlias,
		Limitations: limitations,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxservicecenter.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
		Active:     page.Paginated,
	}
}

// toListResponse merakit jawaban isi satu tab.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	return ListResponse{
		Tab:        toTabDTO(listed.Query.Tab),
		Items:      toServiceClaimListDTO(listed.Page.Items),
		Pagination: toPaginationDTO(listed.Page),
		Filter:     AppliedFilterDTO{Keyword: listed.Query.Keyword},
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
