package masterxol

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound      → induk XOL yang diminta tidak ada.
//   - ErrLayerNotFound → lapisan yang diminta tidak ada, atau bukan milik induk itu.
//   - ErrIDTaken       → nomor yang hendak disisipkan sudah dipakai. Ia ada karena
//     `Database/INSERT_UPDATE_MST_XOL.prc` membentuk nomor dengan
//     `max(to_number(id))+1` (`:16`, `:73`), dan pola itu BISA menghasilkan
//     nomor kembar bila dua penyimpanan berjalan bersamaan. Galat ini
//     membuat kemungkinan itu terlihat, bukan menimpa baris yang sudah ada.
//
// Tidak ada ErrNameTaken: nama induk ganda DITERIMA. Layar lama tidak memeriksanya, dan
// produksi membuktikannya — dua induk sama-sama bernama "Section 1".
var (
	ErrNotFound      = errors.New("masterxol: master XOL tidak ditemukan")
	ErrLayerNotFound = errors.New("masterxol: layer XOL tidak ditemukan")
	ErrIDTaken       = errors.New("masterxol: nomor master XOL sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// bagian yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldName         = "nama"
	FieldYear         = "tahun"
	FieldExchangeRate = "kurs"
	FieldType         = "tipe"
	FieldRemarkPIC    = "remark_pic"
	FieldBusiness     = "bisnis"
	FieldLayer        = "layer"
	FieldReinsurer    = "reas"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu bagian mana yang salah untuk
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
	return "masterxol: validasi gagal — " + strings.Join(message, "; ")
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
