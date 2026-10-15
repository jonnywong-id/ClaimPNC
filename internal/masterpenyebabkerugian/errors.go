package masterpenyebabkerugian

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound  → ID yang diminta tidak ada di master.
//   - ErrIDTaken   → ID yang hendak disisipkan sudah dipakai. Seharusnya mustahil karena
//     ID dibentuk urutan basis data; ia ada supaya kemustahilan itu
//     TERLIHAT bila terjadi, bukan menimpa baris yang sudah ada diam-diam.
//   - ErrNoSite    → tabel situs pada basis data entitas itu tidak memuat baris aktif,
//     sehingga ID tidak dapat dibentuk sama sekali. Procedure lama
//     menjawab keadaan ini dengan kalimat di `ErrMsg` lalu berhenti seolah
//     tidak terjadi apa-apa (`PEGA_M_CAUSE_OF_LOSS.prc:14-16`); di sini ia
//     menjadi galat yang benar-benar galat.
//
// Tidak ada ErrDescriptionTaken di sini, dan itu disengaja: deskripsi ganda DITERIMA di
// modul ini (keputusan Work Owner 2026-09-20 — lihat CheckDescription). Menambahkannya
// akan menjanjikan penegakan yang tidak ada di basis data.
var (
	ErrNotFound = errors.New("masterpenyebabkerugian: penyebab kerugian tidak ditemukan")
	ErrIDTaken  = errors.New("masterpenyebabkerugian: ID penyebab kerugian sudah dipakai")
	ErrNoSite   = errors.New("masterpenyebabkerugian: tabel situs tidak memuat baris aktif")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom yang
// salah — bukan sekadar menampilkan satu pesan di atas form.
// Nilainya `deskripsi`, mengikuti label yang BENAR-BENAR terpasang di layar Pega —
// `pyCaption` "Deskripsi Kerugian" pada `Section/BrowseCauseOfLoss-Section.xml`. Layar
// memakainya untuk menandai kolom yang salah, sehingga ia harus menunjuk isian yang sama
// dengan yang dilihat pengguna.
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
	return "masterpenyebabkerugian: validasi gagal — " + strings.Join(message, "; ")
}

// NewValidationError membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang terbungkus
// interface — supaya `if err != nil` di pemanggil berperilaku seperti yang terbaca.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violation: violations}
}
