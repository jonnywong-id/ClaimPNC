package inputacceptation

import "errors"

// Nama isian yang dapat dilanggar, dipakai menandai pelanggaran pada isian yang benar.
const (
	// FieldClaimID adalah nomor klaim pada alamat layar.
	FieldClaimID = "no_klaim"
)

// Galat modul yang dikenali lapisan transport.
var (
	// ErrCallerUnknown berarti sesi sah tetapi profil di dalamnya tidak memuat login.
	//
	// Ia BUKAN galat autentikasi: yang gagal bukan masuknya, melainkan kelengkapan profil.
	// Membedakannya penting supaya layar tidak melempar pengguna ke halaman masuk lalu
	// mengembalikannya ke galat yang sama.
	ErrCallerUnknown = errors.New("inputacceptation: profil pemanggil tidak lengkap")

	// ErrNotFound berarti klaimnya tidak ada di portal yang sedang dipilih.
	//
	// Ia dibedakan dari rincian kosong dengan sengaja: rincian kosong terbaca di layar
	// sebagai "klaim tanpa isi", padahal yang benar adalah "klaim tidak ditemukan".
	ErrNotFound = errors.New("inputacceptation: klaim tidak ditemukan")

	// ErrWriteNotOwned berarti Submit ditolak karena tabelnya masih dimiliki Pega.
	//
	// # Kenapa penolakan ini galat tersendiri, bukan sekadar 500
	//
	// Karena sebabnya BUKAN kerusakan melainkan keadaan yang diketahui dan punya jalan
	// keluar: selama masa paralel, setiap tabel hanya boleh ditulis satu sistem (`P-1`), dan
	// tabel objek kerja beserta POOLDATA.JSON_KLAIM masih ditulis Pega. Perpindahan
	// kepemilikannya menempuh `D-63` — permintaan tertulis, persetujuan Work Owner,
	// pelaksanaan DBA.
	//
	// Galat yang menyebut sebab dan jalan keluarnya dapat ditindaklanjuti pengguna; galat
	// 500 hanya dapat dilaporkan.
	ErrWriteNotOwned = errors.New(
		"inputacceptation: tabel akseptasi masih dimiliki Pega selama masa paralel")
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	// Field adalah nama isian yang dilanggar, memakai nama pada kontrak API.
	Field string

	// Message adalah kalimat yang dapat langsung ditampilkan ke pengguna.
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran sekaligus.
//
// Ia jamak, bukan tunggal, karena layar ini mengirim ~11 isian dan lima grid dalam satu
// Submit. Mengembalikan pelanggaran satu per satu memaksa pengguna menekan Submit berkali-kali
// untuk menemukan kesalahan berikutnya — perilaku yang `11-CROSSCUTTING.md` §1.2 larang, dan
// yang sistem lama pun tidak lakukan.
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membungkus sekumpulan pelanggaran.
func NewValidationError(violations []Violation) *ValidationError {
	return &ValidationError{Violations: violations}
}

func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "inputacceptation: permintaan tidak sah"
	}
	return "inputacceptation: " + e.Violations[0].Message
}

// UnknownFieldError berarti pembaca dokumen menghasilkan isian yang tidak ada di section.
//
// Ia galat PEMROGRAMAN, bukan galat pengguna: kuncinya ditetapkan section.go dan pembaca
// dokumen, keduanya di dalam kode. Ia dibuat tipe tersendiri supaya uji dapat menangkapnya
// dengan tepat, dan supaya pesannya menyebut kunci mana yang berselisih.
type UnknownFieldError struct {
	Field string
}

func (e *UnknownFieldError) Error() string {
	return "inputacceptation: isian " + e.Field + " tidak dikenal bentuk layar"
}

// UnknownGridError berarti pembaca dokumen menghasilkan grid yang tidak ada di section.
type UnknownGridError struct {
	Grid string
}

func (e *UnknownGridError) Error() string {
	return "inputacceptation: tabel " + e.Grid + " tidak dikenal bentuk layar"
}
