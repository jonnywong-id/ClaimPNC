package laporanhasilai

import (
	"claim-pnc/internal/platform/validation"
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama sifatnya dengan nama field JSON (`D-80`).
//
// Keduanya mengikuti label isian di layar lama: "Tgl Input Dari" dan "Tgl Input Sampai"
// (`D-13`). Label itu menyesatkan — yang disaring adalah TANGGALKOMITE — tetapi Work Owner
// memutuskan pada 2026-09-26 untuk memakainya apa adanya. Lihat doc Filter.
const (
	FieldDateFrom = "dari"
	FieldDateTo   = "sampai"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation = validation.Violation

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`,
// `11-CROSSCUTTING.md` §1.1). Di layar ini ia terasa langsung: pengguna yang menekan
// "Cari Data" dengan kedua isian kosong melanggar dua hal, dan diberi tahu keduanya dalam
// satu kali jalan alih-alih satu lalu satu lagi.
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
		return "laporanhasilai: isian tidak sah"
	}
	return validation.Format(e.Violations, "laporanhasilai: ", ": ", "; ", "")
}
