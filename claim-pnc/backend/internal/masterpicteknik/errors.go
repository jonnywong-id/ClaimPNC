package masterpicteknik

import (
	"errors"
	"strings"
)

// Empat kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound              → ID operator yang diminta tidak ada di master.
//   - ErrAlreadyExists         → ID operator itu sudah terdaftar. Ini KONFLIK, bukan
//     isian cacat: isiannya sah, tetapi bentrok dengan isi
//     penyimpanan saat ini.
//   - ErrEmployeeUnknown       → ID operator tidak terdaftar di direktori pegawai,
//     sehingga namanya tidak dapat ditemukan. Inilah penolakan
//     yang di sistem lama berbunyi "set error kalau tidak
//     ditemukan di service".
//   - ErrDirectoryUnreachable  → direktori pegawai tidak dapat dihubungi. Dibedakan dari
//     yang di atas karena tindak lanjutnya berbeda: yang satu
//     memperbaiki isian, yang lain menunggu.
var (
	ErrNotFound             = errors.New("masterpicteknik: PIC teknik tidak ditemukan")
	ErrAlreadyExists        = errors.New("masterpicteknik: ID operator sudah terdaftar")
	ErrEmployeeUnknown      = errors.New("masterpicteknik: ID operator tidak terdaftar di direktori pegawai")
	ErrDirectoryUnreachable = errors.New("masterpicteknik: direktori pegawai tidak dapat dihubungi")
)

// ErrDirectoryNotConfigured dikembalikan bila alamat layanan direktori belum terdaftar
// untuk portal yang diminta.
//
// Ia DIBEDAKAN dari ErrDirectoryUnreachable dengan sengaja. Keduanya sama-sama berarti
// "nama tidak dapat dicari sekarang", tetapi tindak lanjutnya berlawanan: yang satu
// menunggu sampai jaringan pulih, yang ini tidak akan pulih sendiri — barisnya harus
// ditambahkan ke POOLDATA.GCNM_CONNECT_REST lebih dulu. Menyamarkannya sebagai gangguan
// jaringan akan membuat administrator menunggu sesuatu yang tidak akan terjadi.
var ErrDirectoryNotConfigured = errors.New("masterpicteknik: alamat layanan direktori pegawai belum terdaftar untuk portal ini")

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom yang
// salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldOperatorID    = "id_operator"
	FieldEmail         = "email"
	FieldBusinessLine  = "lini_bisnis"
	FieldGroup         = "grup"
	FieldSupervisor    = "atasan"
	FieldQuota         = "kuota"
	FieldExternalQuota = "kuota_luar"
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
	return "masterpicteknik: validasi gagal — " + strings.Join(message, "; ")
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
