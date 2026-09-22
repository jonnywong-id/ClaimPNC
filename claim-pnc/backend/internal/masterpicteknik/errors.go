package masterpicteknik

import (
	"errors"
	"strings"
)

// Empat kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrTidakDitemukan        → ID operator yang diminta tidak ada di master.
//   - ErrSudahAda              → ID operator itu sudah terdaftar. Ini KONFLIK, bukan
//     isian cacat: isiannya sah, tetapi bentrok dengan isi
//     penyimpanan saat ini.
//   - ErrOperatorTidakDikenal  → ID operator tidak terdaftar di direktori operator,
//     sehingga namanya tidak dapat ditemukan. Inilah
//     penolakan yang di sistem lama berbunyi "set error kalau
//     tidak ditemukan di service".
//   - ErrDirektoriTidakTerhubung → direktori operator tidak dapat dihubungi. Dibedakan
//     dari yang di atas karena tindak lanjutnya berbeda:
//     yang satu memperbaiki isian, yang lain menunggu.
var (
	ErrNotFound          = errors.New("masterpicteknik: PIC teknik tidak ditemukan")
	ErrAlreadyExists                = errors.New("masterpicteknik: ID operator sudah terdaftar")
	ErrUnknownOperator    = errors.New("masterpicteknik: ID operator tidak terdaftar di direktori operator")
	ErrDirectoryUnreachable = errors.New("masterpicteknik: direktori operator tidak dapat dihubungi")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// kolom yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldOperatorID = "id_operator"
	FieldEmail      = "email"
	FieldBusinessLine = "lini_bisnis"
	FieldGroup       = "grup"
	FieldSupervisor     = "atasan"
	FieldQuota      = "kuota"
	FieldExternalQuota  = "kuota_luar"
)

// Pelanggaran adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field string
	Message string
}

// GalatValidasi memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu field mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat.
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	message := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		message = append(message, p.Field+": "+p.Message)
	}
	return "masterpicteknik: validasi gagal — " + strings.Join(message, "; ")
}

// GalatValidasiBaru membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang
// terbungkus interface — supaya `if err != nil` di pemanggil berperilaku seperti yang
// terbaca.
func NewValidationError(violation []Violation) error {
	if len(violation) == 0 {
		return nil
	}
	return &ValidationError{Violation: violation}
}
