package komite

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
var (
	// ErrUnknownBusinessLine berarti lini yang diminta tidak punya SATU BARIS PUN di
	// master — bukan sekadar tidak punya jenjang yang cocok dengan nilainya.
	//
	// Keduanya sengaja dibedakan. "Lini tidak ada di master" adalah salah ketik atau
	// lini baru yang belum diisi; "ada tetapi tidak ada jenjang yang cocok" adalah
	// keadaan data yang harus dilihat Work Owner. Menjawab keduanya dengan galat yang
	// sama akan menyembunyikan yang kedua.
	ErrUnknownBusinessLine = errors.New("komite: lini bisnis tidak ada di master ambang")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom
// yang salah — bukan sekadar menampilkan satu pesan di atas form.
//
// Nilainya berbahasa Indonesia karena ia **kontrak API**, bukan nama internal (`D-80`).
const (
	FieldValue        = "nilai"
	FieldBusinessLine = "lini"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama
// menampilkan seluruh pesan validasi bersamaan, dan mengembalikannya satu per satu akan
// membuat pengguna menekan tombol berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "komite: validasi gagal — " + strings.Join(messages, "; ")
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
	return &ValidationError{Violations: violations}
}
