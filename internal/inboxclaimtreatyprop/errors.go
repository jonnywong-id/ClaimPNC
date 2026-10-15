package inboxclaimtreatyprop

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox Claim Treaty Prop.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan: tab pertama menyaring menurut login pemanggil,
	// sehingga tanpa identitas layar tidak dapat membedakan "antrean saya" dari "antrean
	// semua orang" — dan yang kedua memperlihatkan nama tertanggung milik pekerjaan orang
	// lain.
	ErrCallerUnknown = errors.New("inboxclaimtreatyprop: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta pada modul yang hanya membaca.
	//
	// Ia ADA supaya tombol "Create Claim Treaty Prop" dapat dijawab dengan alasan alih-alih
	// dengan "halaman tidak ditemukan" — dua hal yang terlihat sangat berbeda bagi
	// pengguna, dan hanya yang pertama yang memberi tahu apa yang harus dilakukannya.
	ErrWriteNotAvailable = errors.New("inboxclaimtreatyprop: modul ini belum menulis apa pun")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const FieldTab = "tab"

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`).
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
		return "inboxclaimtreatyprop: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxclaimtreatyprop: " + strings.Join(parts, "; ")
}
