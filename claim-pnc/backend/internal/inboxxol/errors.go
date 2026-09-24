package inboxxol

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrMasterNotFound   → perjanjian XOL yang diminta tidak ada. Bukan isian cacat:
//     permintaannya sah, perjanjiannya yang tidak ada.
//   - ErrCallerUnknown    → identitas pemanggil tidak terbaca.
//   - ErrWriteNotAvailable → aksi tulis diminta, padahal modul ini belum menulis apa pun.
//
// ErrWriteNotAvailable ada meski tidak ada satu pun rute yang menulis, dan itu disengaja:
// ia yang dipakai menjawab pemanggil yang mencoba menembak aksi tulis, sehingga jawabannya
// menyebut SEBABNYA — kepemilikan tabel masih di Pega selama masa paralel (`P-1`) — bukan
// sekadar 404 yang terbaca seperti salah alamat.
var (
	ErrMasterNotFound    = errors.New("inboxxol: perjanjian XOL tidak ditemukan")
	ErrCallerUnknown     = errors.New("inboxxol: identitas pemanggil tidak terbaca")
	ErrWriteNotAvailable = errors.New("inboxxol: modul ini belum menulis apa pun")
)

// Field yang dapat membawa pelanggaran validasi.
//
// NAMA konstantanya berbahasa Inggris (`D-80`); NILAINYA berbahasa Indonesia karena ia
// nama field JSON — kontrak yang dibaca klien, dan termasuk pengecualian `D-80`.
//
// Nilainya SAMA PERSIS dengan nama parameter pada dto. Bila keduanya berbeda, pesannya
// tetap sampai ke layar tetapi tidak menempel pada isian mana pun.
const (
	FieldMasterID    = "id_master"
	FieldYear        = "tahun"
	FieldLossDate    = "tanggal_kejadian"
	FieldCauseOfLoss = "sebab_kerugian"
	FieldAdviceType  = "tipe"
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
// kalimat. Mengembalikan seluruhnya sekaligus meniru perilaku sistem lama yang
// menampilkan semua pesan bersamaan (`P-5`).
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "inboxxol: validasi gagal — " + strings.Join(messages, "; ")
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
