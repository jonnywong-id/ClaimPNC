package inboxservicecenter

import (
	"errors"

	"claim-pnc/internal/platform/validation"
)

// Galat domain modul Inbox Service Center.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan domain
// adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan: KEEMPAT tab menyaring `PIC` menurut login
	// pemanggil, sehingga tanpa identitas layar ini tidak punya satu baris pun untuk
	// ditampilkan — dan menampilkan seluruh klaim portal rekanan sebagai gantinya berarti
	// membocorkan antrean orang lain.
	ErrCallerUnknown = errors.New("inboxservicecenter: identitas pemanggil tidak terbaca")

	// ErrNotFound berarti klaim itu tidak ada, ATAU ada tetapi bukan milik pemanggil.
	//
	// Keduanya sengaja tidak dibedakan. Menjawab "ada, tetapi bukan milik Anda" memberi tahu
	// penanya bahwa ID itu nyata — dan ID di tabel ini berurutan, sehingga jawaban itu cukup
	// untuk memetakan seluruh isi tabel tanpa pernah membaca satu barisnya.
	ErrNotFound = errors.New("inboxservicecenter: klaim tidak ditemukan")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian mana
// yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab    = "tab"
	FieldSearch = "cari"
)

// Isian `id` tidak didaftarkan ulang di sini: ia sudah ada di tab.go sebagai FieldID, yaitu
// nama kolom pertama grid. Nilainya sama, dan mendaftarkannya dua kali berarti dua konstanta
// yang dapat berselisih tanpa ketahuan.

// Violation adalah satu pelanggaran pada satu isian.
type Violation = validation.Violation

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
		return "inboxservicecenter: isian tidak sah"
	}
	return validation.Format(e.Violations, "inboxservicecenter: ", ": ", "; ", "")
}
