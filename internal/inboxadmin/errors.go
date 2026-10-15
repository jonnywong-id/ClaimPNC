package inboxadmin

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox Admin.
//
// Keduanya tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP,
// dan domain tidak boleh tahu apa pun tentang HTTP
// (`11-CROSSCUTTING.md` §1.2 — kesalahan domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan: tiga dari delapan tab menyaring menurut login
	// pemanggil, sehingga tanpa identitas layar tidak dapat membedakan "antrean saya"
	// dari "antrean semua orang".
	ErrCallerUnknown = errors.New("inboxadmin: identitas pemanggil tidak terbaca")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab      = "tab"
	FieldBusiness = "bisnis"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`), dan pada layar
// dengan beberapa penyaring sekaligus ia yang membedakan "perbaiki dua hal" dari
// "perbaiki satu, kirim, lalu diberi tahu ada yang kedua".
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
		return "inboxadmin: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxadmin: " + strings.Join(parts, "; ")
}
