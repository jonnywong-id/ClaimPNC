package masterstatus

import (
	"errors"
	"strings"
)

// Tiga kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
//   - ErrTidakDitemukan  → kode yang diminta tidak ada di master.
//   - ErrLabelSudahAda   → label yang sama sudah dipakai status lain. Ini KONFLIK, bukan
//     isian cacat: isian pengguna sah, tetapi bentrok dengan
//     keadaan penyimpanan saat ini.
//   - ErrKodeSudahAda    → kode yang hendak disisipkan sudah dipakai. Seharusnya
//     mustahil karena kodenya dibuat urutan; ia ada supaya
//     kemustahilan itu terlihat bila terjadi, bukan menimpa baris
//     yang sudah ada diam-diam.
var (
	ErrTidakDitemukan = errors.New("masterstatus: status klaim tidak ditemukan")
	ErrLabelSudahAda  = errors.New("masterstatus: label status sudah dipakai")
	ErrKodeSudahAda   = errors.New("masterstatus: kode status sudah dipakai")
)

// Field yang dapat membawa pelanggaran validasi. Nilainya dipakai apa adanya oleh
// lapisan transport sebagai penunjuk isian di layar, sehingga antarmuka dapat menandai
// kolom yang salah — bukan sekadar menampilkan satu pesan di atas form.
const (
	FieldLabel = "label"
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
	return "masterstatus: validasi gagal — " + strings.Join(pesan, "; ")
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
