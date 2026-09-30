package outstandingclaim

import (
	"errors"
	"strings"
)

// Galat domain modul Outstanding Claim.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan domain
// adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Rincian di modul ini tidak disaring menurut pemanggil (lihat NewQuery), sehingga
	// identitasnya tidak menentukan APA yang terlihat. Ia tetap diwajibkan karena setiap
	// pembukaan dicatat beserta pelakunya — dan catatan tanpa pelaku tidak menjelaskan apa
	// pun saat ditelusuri kemudian.
	ErrCallerUnknown = errors.New("outstandingclaim: identitas pemanggil tidak terbaca")

	// ErrNotFound berarti nomor klaimnya tidak ada di portal yang sedang dipilih.
	//
	// Ia DIBEDAKAN dari rincian kosong dengan sengaja. Rincian kosong terbaca di layar
	// sebagai "klaim ini memang belum diisi", sedangkan yang benar bisa jadi "klaim ini
	// milik entitas lain" — dua keadaan yang menuntut tindakan berbeda dari penggunanya.
	ErrNotFound = errors.New("outstandingclaim: klaim tidak ditemukan")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian mana
// yang salah — sama halnya dengan nama field JSON (`D-80`).
const FieldClaimID = "no_klaim"

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
		return "outstandingclaim: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "outstandingclaim: " + strings.Join(parts, "; ")
}
