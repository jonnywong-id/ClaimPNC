// Package inboxxolhttp adalah lapisan transport modul Inbox XOL.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Advice membawa alamat surel PIC dan
// reasuradur, yang tidak setiap grid membutuhkannya.
package inboxxolhttp

import (
	"claim-pnc/internal/inboxxol"
	"claim-pnc/internal/inboxxol/usecase"
)

// MasterDTO adalah satu perjanjian XOL.
//
// Nama field JSON berbahasa Indonesia — ia KONTRAK yang dibaca frontend, dan termasuk
// pengecualian `D-80`. Namanya mengikuti apa yang dibaca pengguna di kolom grid, bukan
// nama properti Pega: `tahun`, bukan `currencyName`.
type MasterDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
	Year string `json:"tahun"`

	// ExchangeRate adalah kurs perjanjian. Ia dikirim supaya layar dapat menyatakan
	// dasar konversi yang dipakai — nilai "USD" yang tidak dapat ditelusuri kursnya
	// tidak dapat diperiksa siapa pun.
	ExchangeRate float64 `json:"kurs"`

	// MinLimit dan MinLimitIDR mengisi kolom "Min Limit" dan "Min Limit IDR" pada grid
	// layar rincian. Yang kedua adalah yang pertama dikali kurs — perkalian yang di
	// sistem lama dirangkai di dalam SQL (`min(limit)*A.KURSVALUE`).
	//
	// Keduanya dikirim, bukan hanya hasil kalinya: kolomnya memang dua, dan layar tidak
	// boleh membagi kembali untuk memperoleh yang pertama — pembagian dengan kurs nol
	// menghasilkan angka tak hingga yang tampil sebagai nilai biasa.
	MinLimit    float64 `json:"min_limit"`
	MinLimitIDR float64 `json:"min_limit_idr"`

	Type string `json:"tipe"`

	// BusinessGroups adalah nama group business yang sudah dirangkai, persis seperti
	// kolom "Group Business" di grid lama.
	BusinessGroups string `json:"group_business"`

	// BusinessGroupCount memberi layar cara membedakan "belum diisi" dari "ada tetapi
	// namanya kosong" tanpa mengirim seluruh senarainya.
	BusinessGroupCount int `json:"jumlah_group_business"`

	CommitteeStatus   string `json:"status_komite"`
	AwaitingCommittee bool   `json:"menunggu_komite"`
	CommitteeNote     string `json:"catatan_komite"`
	PIC               string `json:"pic"`
	PICNote           string `json:"catatan_pic"`
}

// toMasterDTO menyalin satu perjanjian ke bentuk wire.
//
// Alamat surel PIC TIDAK ikut dikirim. Tidak satu pun grid menampilkannya, dan `D-69`
// menetapkan alamat surel tidak ditulis ke tempat yang tidak membutuhkannya.
func toMasterDTO(master inboxxol.MasterXOL) MasterDTO {
	return MasterDTO{
		ID:                 master.ID,
		Name:               master.Name,
		Year:               master.Year,
		ExchangeRate:       master.ExchangeRate,
		MinLimit:           master.MinLimit,
		MinLimitIDR:        master.MinLimit * master.ExchangeRate,
		Type:               master.Type,
		BusinessGroups:     master.BusinessGroupNames(),
		BusinessGroupCount: len(master.BusinessGroups),
		CommitteeStatus:    master.CommitteeStatus,
		AwaitingCommittee:  master.AwaitingCommittee(),
		CommitteeNote:      master.CommitteeNote,
		PIC:                master.PIC,
		PICNote:            master.PICNote,
	}
}

func toMasterDTOs(masters []inboxxol.MasterXOL) []MasterDTO {
	// Senarai KOSONG, bukan nil: JSON `null` memaksa setiap layar memeriksa dua bentuk
	// "tidak ada", dan yang satu mudah terlupakan.
	result := make([]MasterDTO, 0, len(masters))
	for _, master := range masters {
		result = append(result, toMasterDTO(master))
	}
	return result
}

// MasterListResponse adalah jawaban daftar perjanjian XOL.
type MasterListResponse struct {
	Masters []MasterDTO `json:"perjanjian"`
}

// ClaimSummaryDTO adalah satu baris grid "DATA XOL BASED ON DOL AND COL".
//
// `id_master` ikut dikirim karena grid menggabungkan hasil SELURUH perjanjian. Tanpa
// penanda per baris, layar tidak dapat membuka rincian di baliknya — rincian itu butuh
// tahun, kurs, dan group business perjanjian ASAL baris tersebut.
type ClaimSummaryDTO struct {
	MasterID         string  `json:"id_master"`
	LossDate         string  `json:"tanggal_kejadian"`
	CauseOfLoss      string  `json:"sebab_kerugian"`
	BusinessGroup    string  `json:"group_business"`
	OutstandingValue float64 `json:"nilai_outstanding"`
	AcceptedValue    float64 `json:"nilai_akseptasi"`
}

// ClaimSummaryResponse adalah isi grid utama beserta perjanjian yang menjadi dasarnya.
type ClaimSummaryResponse struct {
	Master MasterDTO         `json:"perjanjian"`
	Rows   []ClaimSummaryDTO `json:"baris"`
}

func toClaimSummaryResponse(overview usecase.ClaimOverview) ClaimSummaryResponse {
	rows := make([]ClaimSummaryDTO, 0, len(overview.Rows))
	for _, row := range overview.Rows {
		rows = append(rows, ClaimSummaryDTO{
			MasterID:         row.MasterID,
			LossDate:         row.LossDate,
			CauseOfLoss:      row.CauseOfLoss,
			BusinessGroup:    row.BusinessGroup,
			OutstandingValue: row.OutstandingValue,
			AcceptedValue:    row.AcceptedValue,
		})
	}
	return ClaimSummaryResponse{Master: toMasterDTO(overview.Master), Rows: rows}
}

// BreakdownDTO adalah satu baris grid rincian.
type BreakdownDTO struct {
	BusinessGroup    string  `json:"group_business"`
	BusinessGroupID  string  `json:"kode_group_business"`
	ClaimCount       int     `json:"jumlah_klaim"`
	OutstandingValue float64 `json:"nilai_outstanding"`
	AcceptedValue    float64 `json:"nilai_akseptasi"`

	// Source membedakan klaim sendiri dari treaty inward. Layar memakainya untuk menandai
	// baris terakhir, yang nilainya berasal dari tabel lain dan sudah dikonversi di
	// sumbernya.
	Source string `json:"sumber"`

	// RateMissing menyatakan kurs mata uang baris ini tidak ditemukan, sehingga nilainya
	// TIDAK dapat dipercaya.
	//
	// Ia dikirim supaya layar dapat menyatakannya. Di sistem lama keadaan ini tidak
	// pernah terlihat: `GETCURRENCYSTANDARD` mengembalikan `1`, dan nilai valuta asing
	// diperlakukan satu banding satu terhadap rupiah tanpa satu pun tanda (`D-49` #5).
	RateMissing bool `json:"kurs_tidak_tersedia"`
}

// BreakdownResponse adalah isi grid rincian.
type BreakdownResponse struct {
	Rows []BreakdownDTO `json:"baris"`
}

func toBreakdownResponse(rows []inboxxol.BusinessBreakdown) BreakdownResponse {
	result := make([]BreakdownDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, BreakdownDTO{
			BusinessGroup:    row.BusinessGroup,
			BusinessGroupID:  row.BusinessGroupID,
			ClaimCount:       row.ClaimCount,
			OutstandingValue: row.OutstandingValue,
			AcceptedValue:    row.AcceptedValue,
			Source:           string(row.Source),
			RateMissing:      row.RateMissing,
		})
	}
	return BreakdownResponse{Rows: result}
}

// AdviceDTO adalah satu baris grid PLA/DLA.
type AdviceDTO struct {
	// Number sudah dirakit beserta revisinya — "nomor / revisi" — persis yang dibaca
	// pengguna di kolom "NO PLA / DLA".
	Number string `json:"nomor"`

	// RawNumber adalah nomor tanpa revisi. Dikirim terpisah karena ia yang dipakai
	// mencocokkan ke sistem lain, sedangkan Number hanya untuk dibaca.
	RawNumber string `json:"nomor_asli"`

	Revision      string  `json:"revisi"`
	ReinsurerName string  `json:"nama_insurance"`
	LayerName     string  `json:"nama_layer"`
	Year          string  `json:"tahun"`
	CauseOfLoss   string  `json:"sebab_kerugian"`
	ExchangeRate  float64 `json:"kurs"`

	// SharePercent adalah angka, bukan teks bersatuan. Tanda persennya ditambahkan saat
	// ditampilkan — nilai bersatuan tidak dapat dijumlahkan maupun diurutkan.
	SharePercent float64 `json:"share_percent"`

	Email  string `json:"email"`
	Remark string `json:"remark"`

	ApprovalStatus string `json:"status_persetujuan"`
	Approved       bool   `json:"sudah_disetujui"`
	ApprovalNote   string `json:"catatan_persetujuan"`
	PICNote        string `json:"catatan_pic"`
	LayerLimit     string `json:"batas_layer"`
	Country        string `json:"negara"`
	IssuedOn       string `json:"tanggal_terbit"`
	Type           string `json:"tipe"`
}

// approvalStatusApproved adalah nilai STATUSAPPROVE yang berarti sudah disetujui.
//
// Kebalikan dari `'0'`, yang menjadi penyaring antrean komite. Tidak ada daftar nilai
// yang sah di mana pun — DDL-nya belum ada (`R-08`) — sehingga yang dapat dinyatakan
// hanyalah "bukan menunggu".
const approvalStatusPending = "0"

func toAdviceDTOs(advices []inboxxol.Advice) []AdviceDTO {
	result := make([]AdviceDTO, 0, len(advices))
	for _, advice := range advices {
		result = append(result, AdviceDTO{
			Number:         advice.DisplayNumber(),
			RawNumber:      advice.Number,
			Revision:       advice.Revision,
			ReinsurerName:  advice.ReinsurerName,
			LayerName:      advice.LayerName,
			Year:           advice.Year,
			CauseOfLoss:    advice.CauseOfLoss,
			ExchangeRate:   advice.ExchangeRate,
			SharePercent:   advice.SharePercent,
			Email:          advice.Email,
			Remark:         advice.Remark,
			ApprovalStatus: advice.ApprovalStatus,
			Approved:       advice.ApprovalStatus != approvalStatusPending,
			ApprovalNote:   advice.ApprovalNote,
			PICNote:        advice.PICNote,
			LayerLimit:     advice.Limit,
			Country:        advice.Country,
			IssuedOn:       advice.IssuedOn,
			Type:           string(advice.Type),
		})
	}
	return result
}

// AdviceListResponse adalah jawaban pencarian PLA/DLA.
type AdviceListResponse struct {
	Advices []AdviceDTO `json:"pemberitahuan"`
}

// ApprovalItemDTO adalah satu baris antrean persetujuan pemberitahuan.
type ApprovalItemDTO struct {
	// Year adalah tahun perjanjian. Kolomnya di grid lama berjudul "Date Of Loss", dan
	// judul itu menyesatkan — isinya tahun, bukan tanggal. Judul lama direplikasi di
	// layar (`P-5`); nama field-nya di sini dibetulkan.
	Year string `json:"tahun"`

	CauseOfLoss    string `json:"sebab_kerugian"`
	Type           string `json:"tipe"`
	LastInsertedAt string `json:"tanggal_insert"`
}

// ApprovalResponse adalah isi tab "Inbox XOL Komite".
type ApprovalResponse struct {
	Advices []ApprovalItemDTO `json:"pemberitahuan"`
	Masters []MasterDTO       `json:"perjanjian"`
}

func toApprovalResponse(queue usecase.ApprovalQueue) ApprovalResponse {
	advices := make([]ApprovalItemDTO, 0, len(queue.Advices))
	for _, item := range queue.Advices {
		advices = append(advices, ApprovalItemDTO{
			Year:           item.Year,
			CauseOfLoss:    item.CauseOfLoss,
			Type:           string(item.Type),
			LastInsertedAt: item.LastInsertedAt,
		})
	}
	return ApprovalResponse{Advices: advices, Masters: toMasterDTOs(queue.Masters)}
}

// CauseOfLossDTO adalah satu pilihan Penyebab Kerugian.
type CauseOfLossDTO struct {
	ID string `json:"id"`

	// Description adalah yang tersimpan di kolom CAUSEOFLOSS pada tabel klaim XOL —
	// bukan ID-nya. Layar karena itu mengirimkan DESKRIPSI saat menyaring, bukan kode.
	Description string `json:"deskripsi"`
}

// CauseOfLossResponse adalah isi dropdown Penyebab Kerugian.
type CauseOfLossResponse struct {
	Causes []CauseOfLossDTO `json:"sebab_kerugian"`
}

func toCauseOfLossResponse(causes []inboxxol.CauseOfLoss) CauseOfLossResponse {
	result := make([]CauseOfLossDTO, 0, len(causes))
	for _, cause := range causes {
		result = append(result, CauseOfLossDTO{ID: cause.ID, Description: cause.Description})
	}
	return CauseOfLossResponse{Causes: result}
}

// ViolationDTO adalah satu pelanggaran validasi beserta isian yang melanggarnya.
//
// Bentuknya `{field, pesan}` mengikuti modul masterstatus, bukan `{kolom, pesan}` maupun
// peta `kolom → pesan`. Ketiganya hidup berdampingan hari ini, dan penyeragamannya adalah
// `TKT-F1-004` yang masih terhalang — dicatat di `api/client.ts`.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dibaca klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// SummaryBusinessDTO adalah satu baris grid "Summary Data XOL".
type SummaryBusinessDTO struct {
	BusinessGroupID   string `json:"kode_group_business"`
	BusinessGroupName string `json:"group_business"`
}

// SummaryBusinessResponse membungkus grid "Summary Data XOL".
type SummaryBusinessResponse struct {
	Rows []SummaryBusinessDTO `json:"baris"`
}

func toSummaryBusinessResponse(rows []inboxxol.SummaryBusiness) SummaryBusinessResponse {
	// Senarai KOSONG, bukan nil — alasannya sama dengan toMasterDTOs.
	result := make([]SummaryBusinessDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, SummaryBusinessDTO{
			BusinessGroupID:   row.BusinessGroupID,
			BusinessGroupName: row.BusinessGroupName,
		})
	}
	return SummaryBusinessResponse{Rows: result}
}

// ClaimListItemDTO adalah satu baris grid "No Klaim" pada layar rincian.
type ClaimListItemDTO struct {
	ClaimNo      string `json:"no_klaim"`
	CurrencyName string `json:"mata_uang"`
	Source       string `json:"sumber"`

	OutstandingValue float64 `json:"nilai_outstanding"`
	AcceptedValue    float64 `json:"nilai_akseptasi"`

	// RateMissing menandai baris yang kursnya tidak ditemukan. Layar menolak menggambar
	// angkanya, bukan menggambar nol (`D-48`).
	RateMissing bool `json:"kurs_tidak_tersedia"`
}

// ClaimListResponse membungkus grid "No Klaim".
type ClaimListResponse struct {
	Rows []ClaimListItemDTO `json:"baris"`
}

func toClaimListResponse(rows []inboxxol.ClaimListItem) ClaimListResponse {
	result := make([]ClaimListItemDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, ClaimListItemDTO{
			ClaimNo:          row.ClaimNo,
			CurrencyName:     row.CurrencyName,
			Source:           string(row.Source),
			OutstandingValue: row.OutstandingValue,
			AcceptedValue:    row.AcceptedValue,
			RateMissing:      row.RateMissing,
		})
	}
	return ClaimListResponse{Rows: result}
}

// UploadResultResponse adalah jawaban "Upload MBU Salvage".
type UploadResultResponse struct {
	JumlahBaris     int                 `json:"jumlah_baris"`
	JumlahTersimpan int                 `json:"jumlah_tersimpan"`
	Ditolak         []UploadRejectedDTO `json:"ditolak"`
}

// UploadRejectedDTO adalah satu baris berkas yang tidak tersimpan.
//
// Nomor barisnya ikut karena yang diperbaiki pengguna adalah BERKASNYA. Pada berkas
// ratusan baris, "ada yang gagal" saja tidak dapat ditindaklanjuti.
type UploadRejectedDTO struct {
	Baris   int    `json:"baris"`
	NoKlaim string `json:"no_klaim"`
	Alasan  string `json:"alasan"`
}

// uploadResultDTO mengubah hasil domain menjadi bentuk kontrak.
func uploadResultDTO(result inboxxol.SalvageUploadResult) UploadResultResponse {
	rejected := make([]UploadRejectedDTO, 0, len(result.Rejected))
	for _, row := range result.Rejected {
		rejected = append(rejected, UploadRejectedDTO{
			Baris:   row.LineNumber,
			NoKlaim: row.ClaimNo,
			Alasan:  row.Message,
		})
	}
	return UploadResultResponse{
		JumlahBaris:     result.Rows,
		JumlahTersimpan: result.Inserted,
		Ditolak:         rejected,
	}
}

// InsertDolColRequest adalah badan permintaan `POST /inbox-xol/dol-col`.
//
// Nama fieldnya SAMA dengan nama field yang sudah dipakai modul ini di tempat lain —
// `id_master`, `tanggal_kejadian`, `sebab_kerugian` — dan itu bukan kebetulan: nilainya
// pun berarti hal yang sama, termasuk bentuk tanggalnya (`DD/MM/YYYY`, bentuk kolom
// `DOL`). Satu nama yang berarti dua hal berbeda di satu modul adalah persis cacat yang
// `D-19` perintahkan tidak dibawa.
//
// Nilainya WAJIB cocok dengan konstanta Field* di `internal/inboxxol/errors.go`; bila
// tidak, pesan validasinya tetap sampai ke layar tetapi tidak menempel pada isian mana
// pun.
type InsertDolColRequest struct {
	MasterID    string `json:"id_master"`
	LossDate    string `json:"tanggal_kejadian"`
	CauseOfLoss string `json:"sebab_kerugian"`
}

// toDomain mengubah badan permintaan menjadi bentuk domain.
func (r InsertDolColRequest) toDomain() inboxxol.DolColRequest {
	return inboxxol.DolColRequest{
		MasterID:    r.MasterID,
		LossDate:    r.LossDate,
		CauseOfLoss: r.CauseOfLoss,
	}
}

// InsertDolColResponse adalah jawaban simpan yang berhasil.
//
// Ia membawa JUMLAH BARIS, bukan sekadar status: satu simpan menghasilkan satu baris per
// group business perjanjian, dan angka itulah satu-satunya cara pengguna mengetahui
// berapa banyak yang benar-benar ditulis atas namanya.
type InsertDolColResponse struct {
	JumlahBaris int `json:"jumlah_baris"`
}
