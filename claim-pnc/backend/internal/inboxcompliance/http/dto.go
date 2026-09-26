// Package inboxcompliancehttp adalah lapisan transport modul Inbox Compliance.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: WorkItem membawa Reference, kunci
// teknis Pega yang tidak pernah ditampilkan.
package inboxcompliancehttp

import (
	"time"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/clock"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Tak satu pun kolom di kedua grid menampilkan jam, dan
// mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang dipakai
// menampilkannya — keputusan yang seharusnya hanya ada di satu tempat (`F-5`).
const (
	dateLayout = "2006-01-02"

	// dateTimeLayout dipakai SATU kolom saja: Tanggal Kirim Audit Compliance pada tab
	// Post Audit, yang di layar Pega menampilkan jamnya. Lihat WorkItemDTO.PostAuditSent.
	dateTimeLayout = "2006-01-02 15:04"
)

// WorkItemDTO adalah satu baris pekerjaan.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar memilih kolom mana yang
// digambar dari `kolom` pada tab yang sedang terbuka — bukan dari ada-tidaknya isian,
// karena isian yang kebetulan kosong pada seluruh baris halaman ini akan membuat kolomnya
// menghilang begitu saja.
type WorkItemDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan tombol buka detail klaim.
	//
	// Ia dikirim tetapi TIDAK ditampilkan sebagai kolom.
	Reference string `json:"referensi"`

	CaseID       string `json:"nomor_case"`
	ClaimNumber  string `json:"no_klaim"`
	PolicyNumber string `json:"no_polis"`
	InsuredName  string `json:"nama_tertanggung"`
	BusinessName string `json:"nama_bisnis"`
	BranchName   string `json:"nama_cabang"`
	AdminName    string `json:"nama_admin"`

	ComplianceSent *string `json:"tanggal_kirim_compliance"`

	// PostAuditSent membawa TANGGAL DAN JAM — `2026-04-22 13:46` — berbeda dari kolom
	// tanggal lain di modul ini yang hanya membawa tanggalnya.
	//
	// Bukan ketidakseragaman yang terlewat: layar Pega menampilkan kolom ini sebagai
	// `22/04/25 13:46`, dan `D-13` menetapkan tampilan ditiru. Kolom tanggal lain memang
	// tidak menampilkan jam di sana.
	PostAuditSent *string `json:"tanggal_kirim_post_audit"`

	ComplianceRemarks string `json:"catatan_compliance"`

	// Aging adalah teks yang digambar di kolom Aging, mengikuti bentuk sistem lama —
	// "5 hours ago", "2 days 3 hours ago". Kosong bila tidak dapat dihitung.
	Aging string `json:"aging"`

	// AgingHours adalah angka mentahnya, sudah dipotong akhir pekan.
	//
	// Ia dikirim BERDAMPINGAN dengan teksnya, bukan menggantikannya, karena teks "2 days
	// 3 hours ago" tidak dapat dibandingkan sebagai angka — mengurutkannya sebagai teks
	// menaruh "10 hours ago" sebelum "2 days ago". Layar hari ini belum mengurutkan kolom
	// Aging (lihat catatan di InboxCompliancePage.tsx), dan angka ini yang membuatnya
	// dapat dilakukan kelak tanpa mengubah kontrak.
	//
	// null berarti tidak dapat dihitung — dan itu BERBEDA dari 0, yang berarti baru saja
	// masuk antrean (`P-5` butir 13).
	AgingHours *float64 `json:"aging_jam"`

	// Outstanding adalah kolom "OutStanding" pada tab Post Audit — "1 year 5 months ago".
	//
	// Ia BUKAN kolom Aging dengan nama lain: dasarnya waktu kalender apa adanya, sedangkan
	// Aging memotong akhir pekan. Lihat inboxcompliance.FormatElapsed.
	Outstanding string `json:"outstanding"`
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

	// Available menyatakan tab ini sudah dapat dilayani. Tab yang belum dapat dilayani
	// tetap dikirim — lihat inboxcompliance.Tab.Available.
	Available bool `json:"tersedia"`

	// Blocker menyebut apa yang kurang dan siapa pemiliknya. Kosong bila tersedia.
	Blocker string `json:"penghalang,omitempty"`
}

// MetadataResponse adalah jawaban GET /api/inbox-compliance/tab.
type MetadataResponse struct {
	Tabs       []TabDTO `json:"tab"`
	DefaultTab string   `json:"tab_bawaan"`

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

// ListResponse adalah jawaban GET /api/inbox-compliance.
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

// limitations adalah keterbatasan modul ini yang perlu diketahui pengguna.
//
// Keduanya bukan cacat, melainkan akibat keadaan yang sudah tercatat. Menuliskannya di layar
// membuat pengguna tidak melaporkannya berulang kali sebagai kerusakan.
var limitations = []string{
	"Kolom Aging memotong hari Sabtu dan Minggu, tetapi TIDAK memotong hari libur " +
		"nasional dan tidak mengenal jam kerja. Itu perilaku yang sama dengan sistem " +
		"lama, dan angkanya karena itu tidak dapat dibandingkan dengan angka TAT pada " +
		"laporan KPI yang memakai dasar berbeda.",

	// Ini bukan keterbatasan perkakas melainkan pertanyaan terbuka yang akibatnya
	// TERLIHAT pengguna, sehingga tempatnya di sini — bukan hanya di dokumen. Tabel
	// POOLDATA.T_CLAIM_COMPLIANCE_H tidak punya kolom status, sehingga penyaring
	// `pyStatusWork = "New"` milik Report Definition lama tidak dapat direplikasi.
	"Tab Post Audit menampilkan seluruh baris POOLDATA.T_CLAIM_COMPLIANCE_H. Sistem " +
		"lama menyaringnya ke pemeriksaan yang berstatus baru; tabel itu tidak menyimpan " +
		"status, sehingga penyaringnya tidak dapat dibawa. Bila daftar ini terasa lebih " +
		"panjang daripada di Pega, itu sebabnya.",
}

// toWorkItemDTO mengubah satu baris.
func toWorkItemDTO(item inboxcompliance.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		Reference:         item.Reference,
		CaseID:            item.CaseID,
		ClaimNumber:       item.ClaimNumber,
		PolicyNumber:      item.PolicyNumber,
		InsuredName:       item.InsuredName,
		BusinessName:      item.BusinessName,
		BranchName:        item.BranchName,
		AdminName:         item.AdminName,
		ComplianceSent:    toDateString(item.ComplianceSentDate),
		PostAuditSent:     toDateTimeString(item.PostAuditSentDate),
		ComplianceRemarks: item.ComplianceRemarks,
		Aging:             item.AgingLabel(),
		AgingHours:        item.AgingHours,
		Outstanding:       item.Outstanding,
	}
}

// toWorkItemListDTO mengubah satu halaman baris.
//
// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien, dan yang kedua
// memaksa setiap layar memeriksanya lebih dulu.
func toWorkItemListDTO(items []inboxcompliance.WorkItem) []WorkItemDTO {
	result := make([]WorkItemDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toWorkItemDTO(item))
	}
	return result
}

// toTabDTO mengubah satu tab.
func toTabDTO(tab inboxcompliance.Tab) TabDTO {
	columns := make([]ColumnDTO, 0, len(tab.Columns))
	for _, column := range tab.Columns {
		columns = append(columns, ColumnDTO{Key: column.Key, Title: column.Title})
	}

	return TabDTO{
		Code:        tab.Code,
		Name:        tab.Name,
		Description: tab.Description,
		Columns:     columns,
		Available:   tab.Available,
		Blocker:     tab.Blocker,
	}
}

// toMetadataResponse merakit jawaban keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	tabs := make([]TabDTO, 0, len(meta.Tabs))
	for _, tab := range meta.Tabs {
		tabs = append(tabs, toTabDTO(tab))
	}

	return MetadataResponse{
		Tabs:        tabs,
		DefaultTab:  meta.DefaultTab,
		Portal:      portalAlias,
		Limitations: limitations,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page inboxcompliance.Page) PaginationDTO {
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

// toDateString memformat tanggal menurut TANGGAL WIB, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
//
// # Kenapa WIB, bukan UTC
//
// Karena yang dikirim adalah TANGGAL tanpa jam, dan tanggal hanya punya arti setelah zona
// waktunya ditetapkan. Klaim yang masuk pukul 06.00 WIB tanggal 2 adalah pukul 23.00 UTC
// tanggal 1 — memformatnya sebagai UTC akan menampilkan tanggal kemarin kepada petugas yang
// baru saja memasukkannya pagi itu.
//
// Pergeseran zonanya tetap terjadi di satu tempat saja, yakni clock.ZoneWIB (`F-5`).
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.In(clock.ZoneWIB).Format(dateLayout)
	return &formatted
}

// toDateTimeString memformat waktu berikut jamnya menurut WIB, atau nil bila kosong.
//
// Zonanya WIB dengan alasan yang sama seperti toDateString, dan di sini akibatnya lebih
// besar: yang bergeser bukan hanya tanggalnya melainkan jam yang benar-benar terbaca di
// kolom. Pukul 13.46 WIB adalah pukul 06.46 UTC, dan petugas yang mengirimnya siang hari
// akan melihat jam pagi.
func toDateTimeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.In(clock.ZoneWIB).Format(dateTimeLayout)
	return &formatted
}

// SendPostAuditRequest adalah badan permintaan POST /api/inbox-compliance/post-audit.
type SendPostAuditRequest struct {
	// Reference adalah kunci klaim yang dikirim, yakni isian `referensi` pada baris tab
	// Compliance. Layar mengirimkannya kembali apa adanya; ia tidak pernah diketik.
	Reference string `json:"referensi"`

	// Remarks adalah Catatan. Boleh kosong — kolomnya nullable dan tidak ada bukti di
	// export bahwa ia wajib.
	Remarks string `json:"catatan"`
}

// SendPostAuditResponse adalah jawaban pengiriman yang berhasil.
type SendPostAuditResponse struct {
	// CaseID adalah nomor yang terbit, misalnya `CPL-100001`.
	CaseID string `json:"nomor_case"`

	ClaimNumber  string `json:"no_klaim"`
	InsuredName  string `json:"nama_tertanggung"`
	PolicyNumber string `json:"no_polis"`
	Remarks      string `json:"catatan"`

	// SentAt membawa tanggal berikut jam, sama seperti kolomnya di tab Post Audit.
	SentAt *string `json:"tanggal_kirim_post_audit"`

	Portal string `json:"portal"`
}

// toSendPostAuditResponse merakit jawaban pengiriman.
func toSendPostAuditResponse(sent usecase.Sent, portalAlias string) SendPostAuditResponse {
	at := sent.Entry.SentAt

	return SendPostAuditResponse{
		CaseID:       sent.Entry.CaseID,
		ClaimNumber:  sent.Entry.ClaimNumber,
		InsuredName:  sent.Entry.InsuredName,
		PolicyNumber: sent.Entry.PolicyNumber,
		Remarks:      sent.Entry.Remarks,
		SentAt:       toDateTimeString(&at),
		Portal:       portalAlias,
	}
}
