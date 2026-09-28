package reportkpi

import (
	"errors"
	"strings"
)

// Galat domain modul Report KPI PNC.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Di layar ini ia bukan soal penyaringan — laporannya tidak disaring per pengguna. Ia
	// soal jejak: barisnya memuat penilaian kinerja orang yang dapat dinamai, dan selama
	// pemeriksaan peran belum ada (`TKT-F3-004`), catatan siapa yang membukanya adalah
	// satu-satunya kontrol yang tersisa (`D-59`).
	ErrCallerUnknown = errors.New("reportkpi: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta pada modul yang hanya membaca.
	//
	// Ia ADA supaya tombol yang di layar lama MENGHITUNG ULANG dan MENYIMPAN dapat dijawab
	// dengan alasan alih-alih dengan "halaman tidak ditemukan". Lihat catatan penutup pada
	// doc paket: penilaiannya ditulis Pega, dan `P-1` melarang dua sistem menulis satu
	// tabel yang sama.
	ErrWriteNotAvailable = errors.New("reportkpi: modul ini belum menulis apa pun")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama sifatnya dengan nama field JSON (`D-80`).
//
// Nama ketiganya mengikuti judul isian di layar lama: "Pilih Tipe Report", "Pilih
// Adjuster", dan "Periode" beserta "Dari"/"Sampai" (`D-13`).
const (
	FieldReportType = "tipe_report"
	FieldAdjuster   = "adjuster"
	FieldDateFrom   = "dari"
	FieldDateTo     = "sampai"

	// FieldAdminGroup menunjuk dropdown "Pilih Data KPI" pada tab KPI Admin.
	FieldAdminGroup = "kelompok"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`,
// `11-CROSSCUTTING.md` §1.1). Di layar ini ia terasa langsung: pengguna yang menekan
// "Cari" dengan form kosong melanggar tiga hal sekaligus, dan diberi tahu ketiganya dalam
// satu kali jalan.
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membentuk galat validasi dari daftar pelanggaran.
func NewValidationError(violations []Violation) *ValidationError {
	return &ValidationError{Violations: violations}
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Violations.
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "reportkpi: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "reportkpi: " + strings.Join(parts, "; ")
}
