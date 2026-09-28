// Package archivedokumenklaimhttp adalah lapisan transport modul Archive Dokumen Klaim.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: ArchiveFile membawa KODESERVICE dan
// HITARCHIVE, jejak integrasi yang tidak ada urusannya dengan layar.
package archivedokumenklaimhttp

import (
	"time"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/archivedokumenklaim/usecase"
)

// ArchiveFileDTO adalah satu baris grid ARCHIVE FILE KLAIM.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan
// nama properti Pega: `jumlah_lembar`, bukan `aging_amount`.
type ArchiveFileDTO struct {
	ID int64 `json:"id"`

	ClaimNumber  string  `json:"nomor_klaim"`
	PolicyNumber string  `json:"nomor_polis"`
	InsuredName  string  `json:"nama_tertanggung"`
	LossDate     *string `json:"tanggal_kejadian"`
	TechnicalPIC string  `json:"pic_teknis"`

	DocumentReceivedDate *string `json:"tanggal_terima_dokumen"`
	InputDate            *string `json:"tanggal_input"`
	SheetCount           int     `json:"jumlah_lembar"`

	DocumentTypeCode string `json:"kode_tipe_dokumen"`
	DocumentTypeName string `json:"tipe_dokumen"`
	DocumentKindCode string `json:"kode_jenis_dokumen"`
	DocumentKindName string `json:"jenis_dokumen"`

	BoxName     string `json:"nama_box"`
	FillingCode string `json:"kode_filling"`
	InputUser   string `json:"user_input"`

	SentDate *string `json:"tanggal_kirim_dokumen"`

	// Keempat isian berikut dipakai bagian Kirim ke Cabang. Ia dikirim di seluruh
	// jawaban, bukan hanya di sana, supaya satu bentuk baris melayani ketiga grid —
	// tiga bentuk yang mirip adalah tiga tempat yang harus diingat berbarengan.
	GroupPanel  string `json:"group_panel"`
	Sent        bool   `json:"sudah_dikirim"`
	ServiceCode string `json:"kode_layanan"`
	ServiceNote string `json:"catatan_layanan"`
}

// ClaimCandidateDTO adalah satu baris grid Input Data Archive.
type ClaimCandidateDTO struct {
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

	// GroupPanel tidak ditampilkan di grid, tetapi IKUT dikirim balik saat menyimpan —
	// kolom GROUPPANEL itulah yang menentukan siapa melihat berkasnya pada daftar kirim
	// ke cabang.
	GroupPanel string `json:"group_panel"`
}

// OptionDTO adalah satu pilihan dropdown berkode dan berlabel.
type OptionDTO struct {
	Code  string `json:"kode"`
	Label string `json:"label"`
}

// DocumentKindDTO adalah satu pilihan Jenis Dokumen beserta tipe pemiliknya.
type DocumentKindDTO struct {
	Code     string `json:"kode"`
	Label    string `json:"label"`
	TypeCode string `json:"kode_tipe_dokumen"`
}

// FillingCodeDTO adalah satu baris pemilih "Pilih Kode".
type FillingCodeDTO struct {
	Code       string `json:"kode"`
	BoxName    string `json:"nama_box"`
	UsageCount int    `json:"jumlah_pemakaian"`
}

// BranchScopeDTO menerangkan lini bisnis yang boleh dilihat pemanggil.
//
// Ia dikirim ke layar supaya pengguna diberi tahu MENGAPA sebagian baris tidak muncul,
// alih-alih menduga daftarnya kosong karena tidak ada pekerjaan.
type BranchScopeDTO struct {
	// ExcludedGroupPanels kosong berarti seluruh lini bisnis tampak.
	ExcludedGroupPanels []string `json:"lini_disembunyikan"`
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
	ClaimSearchTypes []OptionDTO       `json:"tipe_input"`
	DocumentTypes    []OptionDTO       `json:"tipe_dokumen"`
	DocumentKinds    []DocumentKindDTO `json:"jenis_dokumen"`
	BranchScope      BranchScopeDTO    `json:"cakupan_cabang"`
	Portal           string            `json:"portal"`
}

// SearchResponse adalah jawaban grid ARCHIVE FILE KLAIM.
type SearchResponse struct {
	Files      []ArchiveFileDTO `json:"berkas"`
	Pagination PaginationDTO    `json:"halaman"`
	Portal     string           `json:"portal"`
}

// ClaimSearchResponse adalah jawaban grid Input Data Archive.
type ClaimSearchResponse struct {
	Claims []ClaimCandidateDTO `json:"klaim"`
	Portal string              `json:"portal"`
}

// FillingCodeResponse adalah jawaban pemilih "Pilih Kode".
type FillingCodeResponse struct {
	Codes []FillingCodeDTO `json:"kode"`

	// Reconstructed selalu true, dan ia dikirim dengan sengaja: layar menampilkan
	// keterangan bahwa daftar ini disusun dari kode yang sudah pernah dipakai, bukan
	// dari master — karena masternya tidak ada di export (`R-16`). Tanpa keterangan itu,
	// pengguna akan menduga kode yang belum pernah dipakai memang tidak boleh dipakai.
	Reconstructed bool `json:"dari_pemakaian"`

	Portal string `json:"portal"`
}

// PendingResponse adalah jawaban daftar kirim ke cabang.
type PendingResponse struct {
	Files      []ArchiveFileDTO `json:"berkas"`
	Pagination PaginationDTO    `json:"halaman"`
	Scope      BranchScopeDTO   `json:"cakupan_cabang"`
	Portal     string           `json:"portal"`
}

// SaveRequest adalah badan permintaan penyimpanan berkas arsip.
//
// Nol pada `id` berarti baris baru. Penanda "insert"/"update" terpisah milik sistem lama
// tidak dibawa — lihat usecase.Service.Save.
type SaveRequest struct {
	ID int64 `json:"id"`

	ClaimNumber  string  `json:"nomor_klaim"`
	PolicyNumber string  `json:"nomor_polis"`
	InsuredName  string  `json:"nama_tertanggung"`
	LossDate     *string `json:"tanggal_kejadian"`
	TechnicalPIC string  `json:"pic_teknis"`
	GroupPanel   string  `json:"group_panel"`

	DocumentReceivedDate *string `json:"tanggal_terima_dokumen"`
	SheetCount           int     `json:"jumlah_lembar"`
	DocumentTypeCode     string  `json:"kode_tipe_dokumen"`
	DocumentKindCode     string  `json:"kode_jenis_dokumen"`
	BoxName              string  `json:"nama_box"`
	FillingCode          string  `json:"kode_filling"`
}

// SaveResponse adalah jawaban penyimpanan berkas arsip.
//
// Keempat isian terakhir melaporkan PENGIRIMAN yang menyertai penyimpanan. Menyimpan
// berkas langsung mengirimkannya ke layanan Arsip — perilaku sistem lama yang
// direplikasi (`SaveAttachArchiveToDatabase` langkah 5) — dan hasilnya perlu terbaca
// pengguna, bukan hanya tercatat di log.
//
// Pengiriman yang gagal TIDAK menggagalkan penyimpanan, sehingga jawaban ini tetap 201
// atau 200 dengan `terkirim` bernilai false dan `galat_kirim` terisi.
type SaveResponse struct {
	ID      int64  `json:"id"`
	Created bool   `json:"baru"`
	Message string `json:"pesan"`

	Sent        bool   `json:"terkirim"`
	ServiceCode string `json:"kode_layanan,omitempty"`
	ServiceNote string `json:"catatan_layanan,omitempty"`
	SendError   string `json:"galat_kirim,omitempty"`

	Portal string `json:"portal"`
}

// SendResponse adalah jawaban pengiriman ke layanan Arsip.
type SendResponse struct {
	ID          int64  `json:"id"`
	ServiceCode string `json:"kode_layanan"`
	ServiceNote string `json:"catatan_layanan"`
	Message     string `json:"pesan"`
	Portal      string `json:"portal"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Code dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Code — bukan dengan mencocokkan teks Message.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// dateLayout adalah bentuk tanggal pada kontrak API modul ini.
//
// Hanya tanggal, tanpa jam dan tanpa zona: keenam isian tanggal di layar ini semuanya
// tanggal kalender, dan mengirimkan jam akan membuat klien di zona waktu berbeda
// menggeser tanggalnya sendiri (`F-5`).
const dateLayout = "2006-01-02"

// formatDate menuliskan tanggal, atau nil bila tidak ada.
func formatDate(value *time.Time) *string {
	if value == nil {
		return nil
	}
	text := value.UTC().Format(dateLayout)
	return &text
}

// toArchiveFileDTO memetakan satu baris arsip.
func toArchiveFileDTO(file archivedokumenklaim.ArchiveFile) ArchiveFileDTO {
	return ArchiveFileDTO{
		ID:                   file.ID,
		ClaimNumber:          file.ClaimNumber,
		PolicyNumber:         file.PolicyNumber,
		InsuredName:          file.InsuredName,
		LossDate:             formatDate(file.LossDate),
		TechnicalPIC:         file.TechnicalPIC,
		DocumentReceivedDate: formatDate(file.DocumentReceivedDate),
		InputDate:            formatDate(file.InputDate),
		SheetCount:           file.SheetCount,
		DocumentTypeCode:     file.DocumentTypeCode,
		DocumentTypeName:     file.DocumentTypeName,
		DocumentKindCode:     file.DocumentKindCode,
		DocumentKindName:     file.DocumentKindName,
		BoxName:              file.BoxName,
		FillingCode:          file.FillingCode,
		InputUser:            file.InputUser,
		SentDate:             formatDate(file.SentDate),
		GroupPanel:           file.GroupPanel,
		Sent:                 file.BranchStatus == archivedokumenklaim.BranchStatusSent,
		ServiceCode:          file.ServiceCode,
		ServiceNote:          file.ServiceNote,
	}
}

// toArchiveFileListDTO memetakan satu halaman baris arsip.
//
// Ia mengembalikan slice kosong, bukan nil, supaya JSON-nya `[]` dan bukan `null` — klien
// yang memetakan `null` akan gagal, dan gagalnya terjadi tepat saat hasil pencarian
// kosong.
func toArchiveFileListDTO(files []archivedokumenklaim.ArchiveFile) []ArchiveFileDTO {
	list := make([]ArchiveFileDTO, 0, len(files))
	for _, file := range files {
		list = append(list, toArchiveFileDTO(file))
	}
	return list
}

// toClaimListDTO memetakan hasil pencarian klaim.
func toClaimListDTO(claims []archivedokumenklaim.ClaimCandidate) []ClaimCandidateDTO {
	list := make([]ClaimCandidateDTO, 0, len(claims))
	for _, claim := range claims {
		list = append(list, ClaimCandidateDTO{
			Number:        claim.Number,
			PolicyNumber:  claim.PolicyNumber,
			InsuredName:   claim.InsuredName,
			LossDate:      formatDate(claim.LossDate),
			BusinessName:  claim.BusinessName,
			BranchName:    claim.BranchName,
			WorkStatus:    claim.WorkStatus,
			ClaimPosition: claim.ClaimPosition,
			CloseDate:     formatDate(claim.CloseDate),
			CloseNote:     claim.CloseNote,
			TechnicalPIC:  claim.TechnicalPIC,
			GroupPanel:    claim.GroupPanel,
		})
	}
	return list
}

// toPaginationDTO memetakan keterangan halaman.
func toPaginationDTO(page archivedokumenklaim.ArchivePage) PaginationDTO {
	clean := page.Pagination.Normalize()
	return PaginationDTO{
		Page:       clean.Page,
		Size:       clean.Size,
		Total:      page.Total,
		TotalPages: page.TotalPages(),
	}
}

// toBranchScopeDTO memetakan cakupan lini bisnis.
func toBranchScopeDTO(scope archivedokumenklaim.BranchScope) BranchScopeDTO {
	list := make([]string, 0, len(scope.ExcludedGroupPanels))
	list = append(list, scope.ExcludedGroupPanels...)
	return BranchScopeDTO{ExcludedGroupPanels: list}
}

// toOpenResponse memetakan keadaan layar saat dibuka.
func toOpenResponse(opened usecase.Opened, portalAlias string) OpenResponse {
	searchTypes := make([]OptionDTO, 0, len(opened.ClaimSearchTypes))
	for _, option := range opened.ClaimSearchTypes {
		searchTypes = append(searchTypes, OptionDTO{
			Code:  string(option.Code),
			Label: option.Label,
		})
	}

	documentTypes := make([]OptionDTO, 0, len(opened.DocumentTypes))
	for _, option := range opened.DocumentTypes {
		documentTypes = append(documentTypes, OptionDTO{Code: option.Code, Label: option.Name})
	}

	documentKinds := make([]DocumentKindDTO, 0, len(opened.DocumentKinds))
	for _, option := range opened.DocumentKinds {
		documentKinds = append(documentKinds, DocumentKindDTO{
			Code:     option.Code,
			Label:    option.Name,
			TypeCode: option.TypeCode,
		})
	}

	return OpenResponse{
		ClaimSearchTypes: searchTypes,
		DocumentTypes:    documentTypes,
		DocumentKinds:    documentKinds,
		BranchScope:      toBranchScopeDTO(opened.BranchScope),
		Portal:           portalAlias,
	}
}

// toFillingCodeListDTO memetakan isi pemilih "Pilih Kode".
func toFillingCodeListDTO(
	codes []archivedokumenklaim.FillingCodeOption,
) []FillingCodeDTO {
	list := make([]FillingCodeDTO, 0, len(codes))
	for _, code := range codes {
		list = append(list, FillingCodeDTO{
			Code:       code.Code,
			BoxName:    code.BoxName,
			UsageCount: code.UsageCount,
		})
	}
	return list
}
