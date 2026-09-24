package masterdominanfactor

import (
	"errors"
	"strings"
)

// Dua kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound → ID yang diminta tidak ada di master.
//   - ErrIDTaken  → ID yang hendak disisipkan sudah dipakai. Ia ada karena sistem lama
//     membentuk ID dengan `max(ID)+1` (`Database/PEGA_M_DOMINAN_FACTOR.prc:11`),
//     dan pola itu BISA menghasilkan nomor kembar bila dua penyimpanan
//     berjalan bersamaan. Galat ini membuat kemungkinan itu terlihat bila
//     terjadi, bukan menimpa baris yang sudah ada diam-diam.
//
// Tidak ada ErrNameTaken di sini, dan itu disengaja: nama ganda DITERIMA di modul ini
// (keputusan Work Owner 2026-09-20 — lihat CheckName).
var (
	ErrNotFound = errors.New("masterdominanfactor: faktor dominan tidak ditemukan")
	ErrIDTaken  = errors.New("masterdominanfactor: ID faktor dominan sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// kolom yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldName = "nama"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu field mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat.
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	message := make([]string, 0, len(e.Violation))
	for _, v := range e.Violation {
		message = append(message, v.Field+": "+v.Message)
	}
	return "masterdominanfactor: validasi gagal — " + strings.Join(message, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang
// terbungkus interface — supaya `if err != nil` di pemanggil berperilaku seperti yang
// terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violation: violations}
}
