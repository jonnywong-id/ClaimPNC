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

	// ErrCaseNotFound berarti nomor case komite itu tidak ada sama sekali.
	ErrCaseNotFound = errors.New("komite: kasus komite tidak ditemukan")

	// ErrNotAssigned berarti kasusnya ada, tetapi bukan milik pemanggil.
	//
	// Ia dibedakan dari ErrCaseNotFound di dalam domain, lalu SENGAJA DISAMAKAN di
	// lapisan transport — keduanya dijawab 404. Membedakannya di sana akan mengubah
	// endpoint ini menjadi alat untuk menebak nomor case: "404" berarti tidak ada,
	// "403" berarti ada tetapi milik orang lain, dan yang kedua membocorkan keberadaan
	// pekerjaan beserta nilainya kepada siapa pun yang punya sesi.
	//
	// Perbedaannya tetap berguna di sini: log dapat menyebut sebab yang sebenarnya,
	// sementara peramban tidak.
	ErrNotAssigned = errors.New("komite: kasus komite bukan milik pemanggil")

	// ErrDecisionClosed berarti komite pada kasus itu sudah selesai.
	//
	// Ia menjaga sesuatu yang nyata: dua anggota yang membuka layar bersamaan, lalu
	// keduanya menekan tombol. Tanpa pemeriksaan ini, keputusan kedua akan tercatat
	// sebagai jenjang yang sama dua kali — dan penjenjangan berhenti dapat dipercaya.
	ErrDecisionClosed = errors.New("komite: komite pada kasus ini sudah selesai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom
// yang salah — bukan sekadar menampilkan satu pesan di atas form.
//
// Nilainya berbahasa Indonesia karena ia **kontrak API**, bukan nama internal (`D-80`).
const (
	FieldValue        = "nilai"
	FieldBusinessLine = "lini"

	// Isian layar Inbox Komite.
	FieldInboxKind = "kotak"
	FieldDateTo    = "sampai"
	FieldDecision  = "keputusan"
	FieldNote      = "catatan"
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
