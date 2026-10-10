// Package inboxosclaimpercabanghttp adalah lapisan transport modul Inbox OS Claim per Cabang.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe domain
// langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya — dan
// di modul ini bocornya akan nyata: ExportRow membawa 24 pembagian treaty beserta kunci
// internal klaim, dan tak satu pun dari semuanya pernah digambar di layar.
package inboxosclaimpercabanghttp

import (
	"time"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/platform/clock"
)

// WorkItemDTO adalah satu baris grid.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan alias
// Pega: `tanggal_update_progres`, bukan `tanggalterlambat`.
//
// SELURUH isian selalu dikirim, termasuk yang kosong. Layar tidak boleh menyimpulkan kolom
// mana yang ada dari isian yang kebetulan terisi pada halaman ini.
type WorkItemDTO struct {
	BranchName     string `json:"cabang"`
	BusinessSource string `json:"sumbis"`
	BusinessName   string `json:"cob"`
	PolicyNumber   string `json:"no_polis"`

	// InsuredName KOSONG di layar Pega — kolomnya digambar, tetapi tidak satu pun rule
	// mengisinya. Di sini diisi dari sumber yang sudah dipakai berkas ekspor layar yang sama.
	// Lihat inboxosclaimpercabang.WorkItem.InsuredName.
	InsuredName string `json:"nama_insured"`
	ClaimNumber string `json:"no_klaim"`

	// Kedua tanggal dikirim sebagai teks ISO, atau kosong bila memang tidak ada.
	//
	// Pemformatan ke `dd/MM/yyyy` dikerjakan layar, bukan di sini — dan bukan pula di SQL.
	// `09-DATABASE-STRATEGY.md` §3.2 menghapus 411 pemakaian `TO_CHAR` karena tanggal yang
	// dikirim sebagai teks berformat membuat pengurutan menjadi pengurutan TEKS.
	RegisterDate string `json:"tanggal_registrasi"`
	LossDate     string `json:"tanggal_kejadian"`

	// EstimationValue adalah kolom berjudul "Reserve Claim ASM Share" di layar.
	//
	// Nilainya BUKAN porsi ASM — lihat inboxosclaimpercabang.WorkItem.EstimationValue.
	// Selisih itu dinyatakan kepada pengguna lewat `selisih_terencana`.
	//
	// Bentuknya TEKS desimal kanonik (`"250000.00"`), bukan angka JSON. Itu kontrak yang
	// sama dengan modul Ambang Komite dan Inbox Komite, dan layar membacanya dengan
	// `formatRupiah` dari `@/lib/money`. Angka JSON adalah IEEE 754 ganda, dan nilai uang
	// yang melewatinya kehilangan ketepatan tanpa satu pun tanda.
	EstimationValue string `json:"nilai_estimasi"`

	LastProgressAt  string `json:"tanggal_update_progres"`
	ProgressStatus1 string `json:"status_progres_1"`
	ProgressStatus2 string `json:"status_progres_2"`
	TechnicalPIC    string `json:"pic"`
	AdjusterName    string `json:"adjuster"`
	CauseOfLoss     string `json:"col"`

	// AgingDays ANGKA, bukan teks: layar mengurutkannya dan menandai yang melewati ambang,
	// dan keduanya menuntut bilangan.
	AgingDays int `json:"aging_hari"`

	// NeedsAttention menyatakan baris ini digambar merah.
	//
	// Ia dikirim SERVER, bukan dihitung ulang layar. Aturannya —
	// `aging > 180 || progres mandek` — menyangkut peringatan operasional, dan aturan yang
	// hidup di dua tempat akan berbeda antara grid dan berkas ekspor tanpa ada yang
	// menyadarinya.
	NeedsAttention bool `json:"perlu_perhatian"`

	// ProgressStalled dikirim terpisah supaya layar dapat menjelaskan MENGAPA sebuah baris
	// merah. Merah tanpa sebab yang dapat dibaca hanya memindahkan pertanyaannya.
	ProgressStalled bool `json:"progres_mandek"`

	// ProgressNote tidak digambar sebagai kolom, tetapi dibutuhkan panel rincian —
	// `OutstandingKlaimperCabang_Harness` mengirimnya ke layar Detail sebagai `notepic`.
	ProgressNote string `json:"catatan_progres"`
}

// BranchDTO adalah cabang yang barisnya sedang ditampilkan.
//
// Layar menyusun judulnya dari sini — "Inbox Outstanding Claim per Cabang ( CABANG <nama> )" —
// bukan dari profil sesi. Nama pada judul dan kode pada penyaring harus berasal dari satu
// jawaban yang sama.
type BranchDTO struct {
	Code string `json:"kode"`
	Name string `json:"nama"`
}

// PaginationDTO adalah keterangan halaman.
type PaginationDTO struct {
	Page       int `json:"halaman"`
	Size       int `json:"ukuran"`
	Total      int `json:"total"`
	TotalPages int `json:"total_halaman"`
}

// ListResponse adalah jawaban GET /api/inbox-os-claim-per-cabang.
type ListResponse struct {
	Items      []WorkItemDTO `json:"data"`
	Branch     BranchDTO     `json:"cabang"`
	Pagination PaginationDTO `json:"paginasi"`

	// AgingThreshold dikirim supaya layar dapat MENJELASKAN pewarnaannya — "lebih dari 180
	// hari" — tanpa menuliskan angkanya sendiri di dua tempat.
	AgingThreshold int `json:"ambang_aging"`

	// Portal ikut dikirim supaya layar dapat memastikan jawabannya memang milik portal yang
	// sedang dipilih — bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// ErrorResponse adalah bentuk galat yang dibaca frontend.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toListResponse menyusun jawaban daftar.
func toListResponse(listed usecase.Listed, portalAlias string) ListResponse {
	items := make([]WorkItemDTO, 0, len(listed.Page.Items))
	for _, item := range listed.Page.Items {
		items = append(items, toWorkItemDTO(item))
	}

	return ListResponse{
		Items: items,
		Branch: BranchDTO{
			Code: listed.Query.Branch.Code,
			Name: listed.Query.Branch.Name,
		},
		Pagination: PaginationDTO{
			Page:       listed.Page.Pagination.Page,
			Size:       listed.Page.Pagination.Size,
			Total:      listed.Page.Total,
			TotalPages: listed.Page.TotalPages(),
		},
		AgingThreshold:     inboxosclaimpercabang.AgingThreshold,
		Portal:             portalAlias,
	}
}

// toWorkItemDTO menyalin satu baris ke bentuk yang dibaca layar.
func toWorkItemDTO(item inboxosclaimpercabang.WorkItem) WorkItemDTO {
	return WorkItemDTO{
		BranchName:      item.BranchName,
		BusinessSource:  item.BusinessSource,
		BusinessName:    item.BusinessName,
		PolicyNumber:    item.PolicyNumber,
		InsuredName:     item.InsuredName,
		ClaimNumber:     item.ClaimNumber,
		RegisterDate:    isoDate(item.RegisterDate),
		LossDate:        isoDate(item.LossDate),
		EstimationValue: item.EstimationValue.String(),
		LastProgressAt:  isoDate(item.LastProgressAt),
		ProgressStatus1: item.ProgressStatus1,
		ProgressStatus2: item.ProgressStatus2,
		TechnicalPIC:    item.TechnicalPIC,
		AdjusterName:    item.AdjusterName,
		CauseOfLoss:     item.CauseOfLoss,
		AgingDays:       item.AgingDays,
		NeedsAttention:  item.NeedsAttention(),
		ProgressStalled: item.ProgressStalled,
		ProgressNote:    item.ProgressNote,
	}
}

// isoDate menuliskan sebuah tanggal sebagai teks, atau kosong bila tidak ada.
//
// Yang dikirim adalah TANGGAL WIB-nya, bukan cap waktu lengkap: ketiga kolom tanggal di layar
// ini digambar tanpa jam, dan mengirim cap waktu UTC membuat tanggal yang tersimpan menjelang
// tengah malam WIB tampil mundur satu hari di peramban.
func isoDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.In(clock.ZoneWIB).Format("2006-01-02")
}
