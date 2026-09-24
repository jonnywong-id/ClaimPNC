package mastermasking

import (
	"errors"
	"strings"
)

// Empat kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrNotFound      → baris yang diminta tidak ada di master.
//   - ErrPairTaken     → pasangan cabang+login sudah punya baris lain. Ini KONFLIK, bukan
//     isian cacat: isian pengguna sah, tetapi bentrok dengan keadaan
//     penyimpanan saat ini. Aturannya diambil apa adanya dari
//     `Database/UPDATE_LOG_PROTEKSI.prc:29-31`, yang menolak insert
//     dengan pesan "LOGIN … SUDAH ADA".
//   - ErrBranchUnknown → kode cabang tidak ada di POOLDATA.BRANCH. Dibedakan dari galat
//     validasi biasa karena ia hanya dapat diketahui setelah membaca
//     tabel lain.
//   - ErrIDTaken       → ID_MST yang hendak disisipkan sudah dipakai.
//
// ErrIDTaken bukan kemustahilan yang dicatat demi kelengkapan — ia BENAR-BENAR DAPAT
// TERJADI. ID dibentuk dengan `MAX(TO_NUMBER(ID_MST))+1` (keputusan Work Owner
// 2026-09-20, meniru procedure lama), dan tidak ada satu pun indeks unik pada tabel ini —
// diperiksa langsung ke ALL_INDEXES pada 2026-09-20, yang ada hanya satu indeks NONUNIQUE.
// Dua penyimpanan yang tiba bersamaan karena itu dapat memperebutkan nomor yang sama.
// Galat ini ada supaya kejadian itu TERLIHAT dan dapat dicoba ulang, bukan menimpa baris
// yang sudah ada diam-diam.
var (
	ErrNotFound      = errors.New("mastermasking: data masking tidak ditemukan")
	ErrPairTaken     = errors.New("mastermasking: pengguna itu sudah punya data masking di cabang tersebut")
	ErrBranchUnknown = errors.New("mastermasking: kode cabang tidak dikenal")
	ErrIDTaken       = errors.New("mastermasking: ID masking sudah dipakai")
)

// ErrStatusNotChosen dikembalikan bila pencarian menurut status dijalankan tanpa menyebut
// statusnya.
//
// Ia tinggal di domain, bukan di usecase, supaya transport dapat mengenalinya tanpa
// mengimpor usecase — lapisan transport hanya boleh mengenal domain dan antarmuka sempit
// yang ia deklarasikan sendiri.
//
// Layar lama menjawab keadaan yang sama dengan pesan "Pilih Status Aktif"
// (`Activity/SearchDataMasking-Act.xml:727`).
var ErrStatusNotChosen = errors.New("mastermasking: status aktif belum dipilih")

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh lapisan
// transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai kolom yang
// salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldBranchID    = "cabang"
	FieldLogin       = "login"
	FieldModule      = "modul"
	FieldSubModule   = "sub_modul"
	FieldSearchQuota = "maks_cari"
	FieldViewQuota   = "maks_lihat"
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
	return "mastermasking: validasi gagal — " + strings.Join(message, "; ")
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
