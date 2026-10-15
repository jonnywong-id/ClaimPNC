package masterstatus

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound  → kode yang diminta tidak ada di master.
//   - ErrLabelTaken   → label yang sama sudah dipakai status lain. Ini KONFLIK, bukan
//     isian cacat: isian pengguna sah, tetapi bentrok dengan
//     keadaan penyimpanan saat ini.
//   - ErrCodeTaken    → kode yang hendak disisipkan sudah dipakai. Seharusnya
//     mustahil karena kodenya dibuat urutan; ia ada supaya
//     kemustahilan itu terlihat bila terjadi, bukan menimpa baris
//     yang sudah ada diam-diam.
var (
	ErrNotFound   = errors.New("masterstatus: status klaim tidak ditemukan")
	ErrLabelTaken = errors.New("masterstatus: label status sudah dipakai")
	ErrCodeTaken  = errors.New("masterstatus: kode status sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// kolom yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldLabel = "label"
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

func (g *ValidationError) Error() string {
	message := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		message = append(message, p.Field+": "+p.Message)
	}
	return "masterstatus: validasi gagal — " + strings.Join(message, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang
// terbungkus interface — supaya `if err != nil` di pemanggil berperilaku seperti yang
// terbaca.
func NewValidationError(pelanggaran []Violation) error {
	if len(pelanggaran) == 0 {
		return nil
	}
	return &ValidationError{Violation: pelanggaran}
}
