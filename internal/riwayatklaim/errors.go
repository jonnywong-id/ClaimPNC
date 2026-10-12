package riwayatklaim

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotRegistered → pemanggil tidak punya baris proteksi data untuk layar ini,
//     sehingga layar tidak boleh dibuka sama sekali. Di sistem lama
//     pesannya "Input Data Proteksi Terlebih Dahulu".
//   - ErrQuotaExhausted → pemanggil terdaftar, tetapi jatah pencariannya habis. Ini
//     KONFLIK keadaan, bukan isian cacat: permintaannya sah, jatahnya
//     yang sudah tidak ada.
//   - ErrCallerUnknown → identitas pemanggil tidak terbaca. Gerbang proteksi bertumpu
//     pada nama login, dan tanpa itu tidak ada yang dapat diperiksa
//     maupun dicatat.
var (
	ErrNotRegistered  = errors.New("riwayatklaim: pengguna belum terdaftar di master proteksi data")
	ErrQuotaExhausted = errors.New("riwayatklaim: jatah pencarian data sudah habis")
	ErrCallerUnknown  = errors.New("riwayatklaim: identitas pemanggil tidak terbaca")
)

// Field yang dapat membawa pelanggaran validasi.
//
// NAMA konstantanya berbahasa Inggris (`D-80`); NILAINYA berbahasa Indonesia karena ia
// nama field JSON — kontrak yang dibaca klien, dan termasuk pengecualian `D-80`.
//
// Nilainya dipakai apa adanya oleh lapisan transport sebagai penunjuk isian di layar,
// sehingga antarmuka dapat menandai isian yang salah — bukan sekadar menampilkan satu
// pesan di atas formulir.
//
// Nilainya SAMA PERSIS dengan nama field JSON pada dto. Bila keduanya berbeda, pesannya
// tetap sampai ke layar tetapi tidak menempel pada isian mana pun.
const (
	FieldSearchType  = "tipe_pencarian"
	FieldSearchValue = "nilai_pencarian"
	FieldSearchDate  = "tanggal_pencarian"
	FieldBirthDate   = "tanggal_lahir"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu isian mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat.
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "riwayatklaim: validasi gagal — " + strings.Join(messages, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}
