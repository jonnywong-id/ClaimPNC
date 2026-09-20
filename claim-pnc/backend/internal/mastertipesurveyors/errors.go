package mastertipesurveyors

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound         → kode yang diminta tidak ada di master.
//   - ErrDescriptionTaken → deskripsi yang sama sudah dipakai tipe lain. Ini KONFLIK,
//     bukan isian cacat: isian pengguna sah, tetapi bentrok
//     dengan keadaan penyimpanan saat ini.
//   - ErrCodeTaken        → kode yang hendak disisipkan sudah dipakai. Seharusnya
//     mustahil karena kodenya dibuat urutan; ia ada supaya
//     kemustahilan itu terlihat bila terjadi, bukan menimpa baris
//     yang sudah ada diam-diam.
var (
	ErrNotFound         = errors.New("mastertipesurveyors: tipe surveyor tidak ditemukan")
	ErrDescriptionTaken = errors.New("mastertipesurveyors: tipe surveyor dengan nama itu sudah ada")
	ErrCodeTaken        = errors.New("mastertipesurveyors: kode tipe surveyor sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom yang
// salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldDescription = "deskripsi"
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
	return "mastertipesurveyors: validasi gagal — " + strings.Join(message, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violation []Violation) error {
	if len(violation) == 0 {
		return nil
	}
	return &ValidationError{Violation: violation}
}
