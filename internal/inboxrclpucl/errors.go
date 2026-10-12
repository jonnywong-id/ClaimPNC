package inboxrclpucl

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox RCL/PUCL.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Di layar ini ia bukan soal penyaringan — antreannya BERSAMA, dan tidak satu pun
	// kueri menyaring menurut pemanggil. Ia soal jejak: seluruh baris memuat nomor polis
	// dan nama tertanggung, dan selama pemeriksaan peran belum ada (`TKT-F3-004`), catatan
	// siapa yang membukanya adalah satu-satunya kontrol yang tersisa (`D-59`).
	ErrCallerUnknown = errors.New("inboxrclpucl: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta pada modul yang hanya membaca.
	//
	// Ia ADA supaya dua tindakan yang di layar lama menulis — mencetak surat PUCL/RCL dan
	// mengirim Reminder PUCL — dapat dijawab dengan alasan alih-alih dengan "halaman tidak
	// ditemukan". Keduanya terlihat sangat berbeda bagi pengguna, dan hanya yang pertama
	// yang memberi tahu apa yang harus dilakukannya.
	ErrWriteNotAvailable = errors.New("inboxrclpucl: modul ini belum menulis apa pun")

	// ErrReportNotAvailable berarti laporan harian diminta pada tab yang tidak punya.
	//
	// Hanya tab "Cetak Surat" yang punya laporan berbasis rentang tanggal; kedua tab lain
	// mengekspor grid-nya sendiri. Memintanya di tab lain adalah cacat pemanggil, bukan
	// kesalahan pengguna — tetapi ia tetap dijawab dengan kalimat yang terbaca, karena
	// yang membacanya adalah orang yang sedang menelusuri keluhan.
	ErrReportNotAvailable = errors.New(
		"inboxrclpucl: tab ini tidak punya laporan rentang tanggal")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
//
// Kedua nama tanggal mengikuti judul isiannya di layar lama apa adanya: section menuliskan
// "FROM RCL/PUCL" dan "TO RCL/PUCL" (`D-13`).
const (
	FieldTab      = "tab"
	FieldDateFrom = "dari"
	FieldDateTo   = "sampai"

	// FieldReference menunjuk kunci klaim pada permintaan layar kerja.
	FieldReference = "referensi"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`). Di layar ini ia
// terasa pada laporan harian: kedua tanggalnya wajib, dan pengguna yang mengosongkan
// keduanya harus diberi tahu keduanya sekaligus — bukan satu, lalu satu lagi.
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membentuk galat validasi dari daftar pelanggaran.
func NewValidationError(violations []Violation) *ValidationError {
	return &ValidationError{Violations: violations}
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Violations, bukan ini.
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "inboxrclpucl: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxrclpucl: " + strings.Join(parts, "; ")
}
