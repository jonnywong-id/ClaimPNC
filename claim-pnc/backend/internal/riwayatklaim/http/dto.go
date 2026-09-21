// Package riwayatklaimhttp adalah lapisan transport modul View History Claim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8 dan aturan 8 di
// README). Memakai tipe domain langsung sebagai bentuk JSON membuat perubahan internal
// bocor ke klien dan sebaliknya — dan di modul ini bocornya akan nyata: ClaimHistory
// membawa Reference, kunci teknis Pega yang justru tidak boleh ditampilkan.
package riwayatklaimhttp

import (
	"time"

	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/usecase"
)

// ClaimDTO adalah satu baris hasil pencarian.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan
// nama properti Pega: `nomor_klaim`, bukan `edmno`.
type ClaimDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan tombol "Lihat Detail Klaim".
	//
	// Ia dikirim tetapi TIDAK ditampilkan. Layar rincian klaim adalah `MENU_ID 75`
	// "View Claim" yang belum dibangun; begitu ia ada, inilah yang dipakai membukanya.
	Reference string `json:"referensi"`

	Number        string  `json:"nomor_klaim"`
	PolicyNumber  string  `json:"nomor_polis"`
	InsuredName   string  `json:"nama_tertanggung"`
	LossDate      *string `json:"tanggal_kejadian"`
	BusinessName  string  `json:"bisnis"`
	BranchName    string  `json:"cabang"`
	WorkStatus    string  `json:"status"`
	ClaimPosition string  `json:"posisi_klaim"`
	CloseDate     *string `json:"tanggal_close"`
	CloseNote     string  `json:"catatan_close"`
	TechnicalPIC  string  `json:"pic_teknis"`

	// Keempat isian berikut hanya terisi pada sebagian tipe pencarian. Layar
	// menyembunyikan kolomnya mengikuti `kolom_tambahan` pada tipe yang dipilih.
	AcceptanceNumber string  `json:"nomor_akseptasi"`
	AuctionHouseID   string  `json:"nomor_balai_lelang"`
	InsuredItemName  string  `json:"nama_objek"`
	BirthDate        *string `json:"tanggal_lahir"`
}

// SearchTypeDTO adalah satu pilihan pada dropdown "Tipe Pencarian".
type SearchTypeDTO struct {
	Code  string `json:"kode"`
	Label string `json:"label"`

	// Ketiga penanda berikut menentukan isian mana yang digambar layar. Ia dikirim
	// server, bukan ditentukan layar sendiri, supaya bentuk formulir punya satu sumber
	// kebenaran — lihat riwayatklaim.SearchType.
	ShowsText       bool `json:"pakai_teks"`
	ShowsSearchDate bool `json:"pakai_tanggal_pencarian"`
	ShowsBirthDate  bool `json:"pakai_tanggal_lahir"`

	// ExtraColumns adalah kolom tambahan yang dibawa tipe ini.
	ExtraColumns []string `json:"kolom_tambahan"`

	Available   bool   `json:"tersedia"`
	Unavailable string `json:"alasan_belum_tersedia,omitempty"`
}

// AccessDTO adalah keadaan gerbang proteksi data.
//
// Ia dikirim ke layar supaya sisa jatah terbaca SEBELUM habis, bukan hanya diketahui
// lewat penolakan. Pengguna yang tahu jatahnya tinggal dua akan bersikap berbeda dari
// pengguna yang tiba-tiba ditolak.
type AccessDTO struct {
	QuotaTotal     int `json:"jatah_total"`
	QuotaUsed      int `json:"jatah_terpakai"`
	QuotaRemaining int `json:"jatah_sisa"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// OpenResponse adalah jawaban saat layar dibuka.
type OpenResponse struct {
	SearchTypes []SearchTypeDTO `json:"tipe_pencarian"`
	Access      AccessDTO       `json:"proteksi"`
	Portal      string          `json:"portal"`
}

// SearchResponse adalah jawaban satu pencarian.
type SearchResponse struct {
	Claims     []ClaimDTO    `json:"klaim"`
	Pagination PaginationDTO `json:"halaman"`
	Access     AccessDTO     `json:"proteksi"`
	Portal     string        `json:"portal"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai isian yang salah, bukan sekadar menampilkan
// satu pesan di atas formulir.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Code dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Code — bukan dengan mencocokkan teks Message.
//
// Bentuknya sama persis dengan respons galat modul lain. Menyatukannya menjadi satu tipe
// bersama adalah lingkup `TKT-F1-004`, kontrak galat yang mengikat seluruh aplikasi, dan
// tiket itu masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Details hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Details []ViolationDTO `json:"detail,omitempty"`
}

// dateLayout adalah bentuk tanggal pada kontrak API.
//
// ISO 8601 tanggal saja, bukan `dd/mm/yyyy` seperti sistem lama. Pemformatan untuk layar
// dilakukan di frontend (`09-DATABASE-STRATEGY.md` §3.2 memindahkan pemformatan keluar
// dari lapisan data), dan tanggal berformat ISO dapat diurutkan sebagai teks — yang
// berformat `dd/mm/yyyy` tidak.
const dateLayout = "2006-01-02"

// toClaimDTO mengubah satu baris domain menjadi bentuk yang dikirim ke peramban.
func toClaimDTO(claim riwayatklaim.ClaimHistory) ClaimDTO {
	return ClaimDTO{
		Reference:        claim.Reference,
		Number:           claim.Number,
		PolicyNumber:     claim.PolicyNumber,
		InsuredName:      claim.InsuredName,
		LossDate:         toDateString(claim.LossDate),
		BusinessName:     claim.BusinessName,
		BranchName:       claim.BranchName,
		WorkStatus:       claim.WorkStatus,
		ClaimPosition:    claim.ClaimPosition,
		CloseDate:        toDateString(claim.CloseDate),
		CloseNote:        claim.CloseNote,
		TechnicalPIC:     claim.TechnicalPIC,
		AcceptanceNumber: claim.AcceptanceNumber,
		AuctionHouseID:   claim.AuctionHouseID,
		InsuredItemName:  claim.InsuredItemName,
		BirthDate:        toDateString(claim.BirthDate),
	}
}

// toClaimListDTO selalu mengembalikan potongan yang tidak nil.
//
// Potongan nil terserialisasi menjadi `null`, dan layar yang melakukan `.map` atasnya akan
// gagal — bukan menampilkan daftar kosong. Daftar kosong adalah keadaan yang sah dan
// sering di layar pencarian.
func toClaimListDTO(list []riwayatklaim.ClaimHistory) []ClaimDTO {
	content := make([]ClaimDTO, 0, len(list))
	for _, claim := range list {
		content = append(content, toClaimDTO(claim))
	}
	return content
}

// toSearchTypeListDTO mengubah daftar tipe pencarian.
func toSearchTypeListDTO(list []riwayatklaim.SearchType) []SearchTypeDTO {
	content := make([]SearchTypeDTO, 0, len(list))
	for _, searchType := range list {
		columns := make([]string, 0, len(searchType.ExtraColumns))
		for _, column := range searchType.ExtraColumns {
			columns = append(columns, string(column))
		}
		content = append(content, SearchTypeDTO{
			Code:            searchType.Code,
			Label:           searchType.Label,
			ShowsText:       searchType.ShowsText,
			ShowsSearchDate: searchType.ShowsSearchDate,
			ShowsBirthDate:  searchType.ShowsBirthDate,
			ExtraColumns:    columns,
			Available:       searchType.Available,
			Unavailable:     searchType.Unavailable,
		})
	}
	return content
}

// toAccessDTO mengubah keadaan gerbang.
func toAccessDTO(access riwayatklaim.Access) AccessDTO {
	return AccessDTO{
		QuotaTotal:     access.QuotaTotal,
		QuotaUsed:      access.QuotaUsed,
		QuotaRemaining: access.QuotaRemaining,
	}
}

// toPaginationDTO mengubah keterangan halaman.
func toPaginationDTO(page riwayatklaim.Page) PaginationDTO {
	return PaginationDTO{
		Page:       page.Pagination.Page,
		Size:       page.Pagination.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// toSearchResponse merakit jawaban pencarian.
func toSearchResponse(found usecase.Found, portalAlias string) SearchResponse {
	return SearchResponse{
		Claims:     toClaimListDTO(found.Page.Claims),
		Pagination: toPaginationDTO(found.Page),
		Access:     toAccessDTO(found.Access),
		Portal:     portalAlias,
	}
}

// toDateString memformat tanggal, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""`
// akan terbaca layar sebagai tanggal yang gagal diformat.
func toDateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateLayout)
	return &formatted
}

// parseDate membaca tanggal dari parameter query.
//
// Nilai kedua false bila teksnya ada tetapi tidak dapat dibaca — dibedakan dari teks
// kosong, yang berarti isiannya memang tidak dikirim.
func parseDate(raw string) (*time.Time, bool) {
	if raw == "" {
		return nil, true
	}
	parsed, err := time.Parse(dateLayout, raw)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}
