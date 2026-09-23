package inboxcloseclaim

import (
	"errors"
	"strings"
)

// Kegagalan yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
var (
	// ErrRequestPending berarti permintaan sejenis atas klaim yang sama MASIH MENUNGGU
	// dijalankan.
	//
	// Ia menjaga sesuatu yang nyata, dan lebih nyata di layar ini daripada di mana pun:
	// permintaan yang diajukan TIDAK mengubah klaim apa pun secara langsung — barisnya
	// tetap tampil sebagai klaim tutup sampai Pega menjalankannya. Pengguna yang tidak
	// melihat perubahan akan menekan tombolnya lagi.
	//
	// Tanpa pemeriksaan ini, satu klaim dapat memiliki dua permintaan reopen yang menunggu,
	// dan dua permintaan salin yang menunggu berarti DUA KLAIM BARU dari satu tombol yang
	// ditekan dua kali.
	ErrRequestPending = errors.New("inboxcloseclaim: permintaan atas klaim ini masih menunggu dijalankan")

	// ErrRequestNotAllowed berarti pemanggil TIDAK BERWENANG mengajukan ReOpen maupun Copy
	// Klaim.
	//
	// Hari ini ia berlaku untuk SETIAP pengguna: penjaganya adalah When rule `IsGCNMUser`,
	// yang isinya `1 = 2` — selalu salah. Lihat authorization.go untuk asalnya dan untuk
	// cara menyalakannya kelak.
	//
	// Ia dibedakan tegas dari ErrClaimNotFound di bawahnya. Keduanya menolak, tetapi
	// sebabnya berbeda dan pengguna harus dapat membedakannya: yang satu berarti "klaimnya
	// salah", yang lain berarti "aksinya memang tidak tersedia". Menjawab keduanya sama akan
	// membuat pengguna mencari-cari klaim yang sebenarnya sudah benar.
	ErrRequestNotAllowed = errors.New("inboxcloseclaim: pemanggil tidak berwenang mengajukan permintaan")

	// ErrClaimNotFound berarti klaim itu tidak ada, atau tidak berada di layar ini.
	//
	// Keduanya SENGAJA disamakan. Klaim yang masih berjalan bukan milik layar ini, dan
	// membedakan "tidak ada" dari "belum tutup" akan mengubah endpoint ini menjadi alat
	// untuk menebak keberadaan klaim milik lini bisnis yang bukan hak pemanggil.
	ErrClaimNotFound = errors.New("inboxcloseclaim: klaim tutup tidak ditemukan")
)

// Field yang dapat membawa pelanggaran validasi.
//
// Nilainya dipakai apa adanya lapisan transport sebagai penunjuk isian di layar, sehingga
// antarmuka dapat menandai kolom yang salah — bukan sekadar menampilkan satu pesan di atas
// form.
//
// Nilainya berbahasa Indonesia karena ia **kontrak API**, bukan nama internal (`D-80`).
const (
	FieldKind        = "jenis"
	FieldClaimID     = "klaim_id"
	FieldActor       = "pemohon"
	FieldReason      = "alasan"
	FieldRequestedAt = "pada"

	// Isian penyaring layar.
	FieldBusiness = "lini"
	FieldTransfer = "status_transfer"
	FieldPayment  = "status_bayar"
	FieldLimit    = "batas"
	FieldOffset   = "lewati"
)

// Violation adalah satu aturan yang dilanggar, beserta isian yang melanggarnya.
type Violation struct {
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama menampilkan
// seluruh pesan validasi bersamaan, dan mengembalikannya satu per satu akan membuat
// pengguna menekan tombol berkali-kali untuk menemukan kesalahan berikutnya
// (`12-CROSSCUTTING.md` §1.2 butir 1).
type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	messages := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		messages = append(messages, v.Field+": "+v.Message)
	}
	return "inboxcloseclaim: validasi gagal — " + strings.Join(messages, "; ")
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
