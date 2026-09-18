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
	ErrTidakDitemukan          = errors.New("masterpicteknik: PIC teknik tidak ditemukan")
	ErrSudahAda                = errors.New("masterpicteknik: ID operator sudah terdaftar")
	ErrOperatorTidakDikenal    = errors.New("masterpicteknik: ID operator tidak terdaftar di direktori operator")
	ErrDirektoriTidakTerhubung = errors.New("masterpicteknik: direktori operator tidak dapat dihubungi")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// kolom yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldIDOperator = "id_operator"
	FieldEmail      = "email"
	FieldLiniBisnis = "lini_bisnis"
	FieldGrup       = "grup"
	FieldAtasan     = "atasan"
	FieldKuota      = "kuota"
	FieldKuotaLuar  = "kuota_luar"
)

// Pelanggaran adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Pelanggaran struct {
	Field string
	Pesan string
}

// GalatValidasi memuat SELURUH pelanggaran sekaligus.
//
// Ia sengaja bukan daftar string: transport perlu tahu field mana yang salah untuk
// menandainya di layar, dan informasi itu hilang bila pesannya dirangkai menjadi satu
// kalimat.
type GalatValidasi struct {
	Pelanggaran []Pelanggaran
}

func (g *GalatValidasi) Error() string {
	pesan := make([]string, 0, len(g.Pelanggaran))
	for _, p := range g.Pelanggaran {
		pesan = append(pesan, p.Field+": "+p.Pesan)
	}
	return "masterpicteknik: validasi gagal — " + strings.Join(pesan, "; ")
}

// GalatValidasiBaru membentuk galat validasi, atau nil bila tidak ada pelanggaran.
//
// Mengembalikan nil bertipe error yang benar-benar nil — bukan pointer nil yang
// terbungkus interface — supaya `if err != nil` di pemanggil berperilaku seperti yang
// terbaca.
func GalatValidasiBaru(pelanggaran []Pelanggaran) error {
	if len(pelanggaran) == 0 {
		return nil
	}
	return &GalatValidasi{Pelanggaran: pelanggaran}
}
