// Package laporanhasilaihttp adalah lapisan transport modul Laporan Hasil AI.
//
// Ia yang mengubah permintaan HTTP menjadi pemanggilan layanan, dan galat domain menjadi
// kode status. Tidak ada satu pun aturan bisnis di sini.
package laporanhasilaihttp

import (
	"time"

	"claim-pnc/internal/laporanhasilai"
)

// dateLayout adalah bentuk tanggal pada kontrak API: YYYY-MM-DD.
//
// Jam sengaja tidak dikirim. Tak satu pun kolom di kedua grid menampilkan jam — kolom
// tanggal di layar lama digambar lewat `@substring` yang hanya mengambil delapan karakter
// pertama — dan mengirimkannya akan membuat layar harus memutuskan zona waktu mana yang
// dipakai menampilkannya (`F-5`).
const dateLayout = "2006-01-02"

// RowDTO adalah satu baris grid rincian di kawat.
//
// # Kenapa nama field-nya berbahasa Indonesia, sedangkan tipe Go-nya Inggris
//
// Karena ia KONTRAK, bukan nama internal (`D-80`). Mengubahnya adalah perubahan yang
// merusak klien, bukan penggantian nama.
//
// Namanya mengikuti **judul kolom di layar**, bukan nama kolom basis data maupun nama
// properti klipboard Pega. Ketiganya berbeda di modul ini:
//
//	layar                kolom basis data     properti Pega (warisan, menyesatkan)
//	No Klaim             NO_KLAIM             .ClaimID
//	Object Name          OBJECTNAME           .ObjectName
//	Komite Status        STATUSAPPROVE        .KomiteAccepted
//	Tanggal Komite       TANGGALKOMITE        .TanggalComitee
//	AI Status            RESULTAI             .ResultAI
//	Tanggal AI           TGLAI                .TanggalAI
//	Note AI Terima       NOTETERIMA           .Notes
//	Note AI Tolak        NOTETOLAK            .NoteAkseptasi
//	Coverage Final       COVERAGE_AI_FINAL    .COVERAGE_AI_FINAL
//	Kategori Kronologi   KATEGORI_KRONOLOGI   .KATEGORI_KRONOLOGI
//
// # SELURUH field selalu dikirim, termasuk yang kosong
//
// Lima di antaranya SELALU kosong — layar lamanya pun menggambarkannya kosong, dan Work
// Owner memutuskan pada 2026-09-26 untuk menirunya. Keduanya tetap dikirim supaya kolomnya
// tidak menghilang begitu saja dari layar, dan supaya keputusan itu dapat dibalik tanpa
// menyentuh kontrak.
type RowDTO struct {
	// ID adalah kunci baris untuk tabel di layar. TIDAK digambar sebagai kolom.
	ID string `json:"id"`

	ClaimNumber string `json:"no_klaim"`

	// ObjectName SELALU kosong. Lihat doc paket `laporanhasilai`.
	ObjectName string `json:"nama_object"`

	CommitteeStatus string `json:"komite_status"`

	// CommitteeStatusCode adalah nilai mentah STATUSAPPROVE. Dikirim tetapi TIDAK
	// digambar: tanpanya, "MENUNGGU" dan kode yang tidak dikenal terlihat sama di layar
	// sementara keduanya berarti hal yang berbeda saat ditelusuri.
	CommitteeStatusCode string `json:"kode_komite_status"`

	// CommitteeDate null bila kolomnya NULL.
	CommitteeDate *string `json:"tanggal_komite"`

	AIStatus string `json:"ai_status"`

	// AIDate null bila AI belum menilai baris ini.
	AIDate *string `json:"tanggal_ai"`

	// AcceptNote SELALU kosong. Lihat doc paket `laporanhasilai`.
	AcceptNote string `json:"note_ai_terima"`

	// RejectNote SELALU kosong. Lihat doc paket `laporanhasilai`.
	RejectNote string `json:"note_ai_tolak"`

	// CoverageFinal SELALU kosong. Lihat doc paket `laporanhasilai`.
	CoverageFinal string `json:"coverage_final"`

	// ChronologyCategory SELALU kosong. Lihat doc paket `laporanhasilai`.
	ChronologyCategory string `json:"kategori_kronologi"`
}

// TallyDTO adalah satu baris grid ringkasan.
type TallyDTO struct {
	// Subject adalah isi kolom "Keputusan" — "Komite" atau "AI".
	Subject string `json:"keputusan"`

	// Total adalah Diterima + Ditolak, BUKAN jumlah baris. Lihat doc `Tally.Total`.
	Total int `json:"total"`

	Accepted int `json:"diterima"`
	Rejected int `json:"ditolak"`

	// Pending TIDAK ADA di layar lama — penambahan yang diputuskan Work Owner pada
	// 2026-09-26 supaya selisih antara Total dan jumlah baris dapat dibaca.
	Pending int `json:"menunggu"`
}

// PaginationDTO adalah keterangan halaman yang menyertai daftar.
type PaginationDTO struct {
	Page  int `json:"halaman"`
	Size  int `json:"ukuran"`
	Total int `json:"total"`

	// TotalPage minimal 1, termasuk saat tidak ada satu baris pun — layar selalu
	// menggambar paginator, dan "halaman 1 dari 0" tidak dapat dibaca siapa pun.
	TotalPage int `json:"total_halaman"`
}

// FilterDTO memantulkan penyaring yang BENAR-BENAR dipakai menjawab.
//
// Ia ada supaya layar dapat membuktikan bahwa yang tergambar adalah jawaban atas isian
// yang sedang terlihat — bukan sisa jawaban permintaan sebelumnya yang datang terlambat.
type FilterDTO struct {
	From string `json:"dari"`
	To   string `json:"sampai"`
}

// SearchResponse adalah jawaban GET /api/laporan-hasil-ai.
type SearchResponse struct {
	// Summary selalu memuat DUA baris, pada urutan yang tergambar: Komite lalu AI.
	Summary []TallyDTO `json:"ringkasan"`

	Rows       []RowDTO      `json:"baris"`
	Pagination PaginationDTO `json:"paginasi"`
	Filter     FilterDTO     `json:"filter"`
	Portal     string        `json:"portal"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan, detail}` — sehingga klien tidak
// menghadapi dua bentuk galat yang berbeda.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toRowDTO mengubah satu baris domain menjadi bentuk kawat.
func toRowDTO(one laporanhasilai.Row) RowDTO {
	return RowDTO{
		ID:                  one.ID,
		ClaimNumber:         one.ClaimNumber,
		ObjectName:          one.ObjectName,
		CommitteeStatus:     one.CommitteeStatus,
		CommitteeStatusCode: one.CommitteeStatusCode,
		CommitteeDate:       dateOrNil(one.CommitteeDate),
		AIStatus:            one.AIStatus,
		AIDate:              dateOrNil(one.AIDate),
		AcceptNote:          one.AcceptNote,
		RejectNote:          one.RejectNote,
		CoverageFinal:       one.CoverageFinal,
		ChronologyCategory:  one.ChronologyCategory,
	}
}

// toRowListDTO mengembalikan senarai KOSONG, bukan nil, supaya JSON-nya `[]` dan bukan
// `null` — klien tidak perlu membedakan keduanya.
func toRowListDTO(list []laporanhasilai.Row) []RowDTO {
	result := make([]RowDTO, 0, len(list))
	for _, one := range list {
		result = append(result, toRowDTO(one))
	}
	return result
}

// toSummaryDTO menyusun kedua baris ringkasan pada urutan yang tergambar.
func toSummaryDTO(summary laporanhasilai.Summary) []TallyDTO {
	tallies := summary.Tallies()
	result := make([]TallyDTO, 0, len(tallies))
	for _, one := range tallies {
		result = append(result, TallyDTO{
			Subject:  one.Subject,
			Total:    one.Total(),
			Accepted: one.Accepted,
			Rejected: one.Rejected,
			Pending:  one.Pending,
		})
	}
	return result
}

// toPaginationDTO menyusun keterangan paginasi.
func toPaginationDTO(page laporanhasilai.Pagination, total int) PaginationDTO {
	clean := page.Normalize()

	totalPage := total / clean.Size
	if total%clean.Size != 0 {
		totalPage++
	}
	if totalPage < 1 {
		totalPage = 1
	}

	return PaginationDTO{
		Page:      clean.Page,
		Size:      clean.Size,
		Total:     total,
		TotalPage: totalPage,
	}
}

// toFilterDTO memantulkan penyaring yang dipakai.
func toFilterDTO(filter laporanhasilai.Filter) FilterDTO {
	clean := filter.Clean()
	return FilterDTO{
		From: formatDate(clean.From),
		To:   formatDate(clean.To),
	}
}

// formatDate menuliskan tanggal sebagai YYYY-MM-DD; nol menjadi teks kosong.
func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(dateLayout)
}

// dateOrNil menuliskan tanggal sebagai YYYY-MM-DD; nol menjadi null.
//
// null, bukan teks kosong: "belum ada tanggal" dan "tanggalnya kosong" adalah hal yang
// sama di sini, dan null menyatakannya tanpa membuat layar menebak.
func dateOrNil(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	text := value.Format(dateLayout)
	return &text
}
