package dashboardclaim

import (
	"errors"
	"fmt"
	"strings"
)

// Galat domain modul Dashboard Claim.
//
// Keduanya sentinel supaya transport dapat membedakannya dengan errors.Is dan memetakannya
// ke kode HTTP yang berbeda — 404 untuk tile yang tidak ada, 422 untuk isian yang salah.
var (
	// ErrTileNotFound berarti jalur menyebut tile yang tidak dikenal.
	//
	// Dijawab 404, bukan 422: yang salah bukan isian pengguna melainkan JALUR-nya, dan
	// jalur yang tidak ada memang tidak ada.
	ErrTileNotFound = errors.New("dashboardclaim: tile tidak dikenal")

	// ErrClosedClaimUnavailable berarti seam ClosedClaimReader tidak terpasang.
	//
	// Ia cacat perakitan, bukan keadaan yang dapat terjadi karena permintaan pengguna —
	// karena itu ia TIDAK dijawab dengan angka nol. Menjawab nol akan menampilkan "0 klaim
	// tutup" pada layar manajerial, dan tidak ada yang dapat membedakannya dari keadaan
	// benar-benar kosong.
	ErrClosedClaimUnavailable = errors.New("dashboardclaim: pembaca klaim tutup belum terpasang")
)

// Nama field yang dapat dilanggar, dipakai sebagai kunci pada respons validasi.
//
// Nilainya sama persis dengan nama parameter query yang dikirim layar, sehingga frontend
// dapat menempelkan pesannya ke isian yang benar tanpa tabel pemetaan.
const (
	FieldBusinessLine = "lini_bisnis"
	FieldPage         = "halaman"
	FieldSize         = "ukuran"

	// Kedua dropdown panel penyaring.
	FieldCashierStatus = "status_transfer"
	FieldPaymentStatus = "status_bayar"
)

// Violation adalah satu pelanggaran validasi pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan hanya yang pertama.
//
// Ini bukan preferensi gaya melainkan kesetaraan perilaku (`P-5`): layar lama menampilkan
// seluruh pesan sekaligus, dan mengembalikan satu pesan per percobaan akan membuat pengguna
// menyerah pada isian yang panjang (`12-CROSSCUTTING` §1.2 butir 1).
type ValidationError struct {
	Violations []Violation
}

// Error memenuhi antarmuka error.
func (e *ValidationError) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "dashboardclaim: permintaan tidak valid"
	}
	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, fmt.Sprintf("%s: %s", v.Field, v.Message))
	}
	return "dashboardclaim: " + strings.Join(parts, "; ")
}

// NewValidationError membungkus daftar pelanggaran menjadi galat.
//
// Mengembalikan nil bila tidak ada pelanggaran, sehingga pemanggil dapat menuliskannya
// sebagai satu baris `return NewValidationError(v)` tanpa memeriksa panjangnya lebih dulu —
// dan tidak ada jalur yang mengembalikan galat kosong yang tampak seperti kegagalan.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}

// ErrTransferUnavailable DICABUT 2026-10-06, bersama tabel permintaan transfer.
//
// Sebabnya dulu tabel `POOLDATA.CPNC_PERMINTAAN_TRANSFER` (migrasi `0014`) yang belum
// dijalankan DBA. Tabel itu tidak jadi dipakai — ia tidak punya pelaksana — sehingga yang
// ditunggu sekarang bukan tabel melainkan **hak UPDATE** pada satu kolom. Penggantinya
// `ErrAssignmentUnavailable` di bawah.

// ErrClaimNotFound dikembalikan saat klaim yang hendak dipindahkan tidak ada.
//
// Ia dibedakan dari galat teknis karena sebabnya berbeda dan jawabannya ke layar pun berbeda:
// klaim yang hilang berarti seseorang menekan Transfer pada baris yang sudah tidak ada —
// biasanya karena halamannya sudah usang — dan itu `404`, bukan `500`.
var ErrClaimNotFound = errors.New("dashboardclaim: klaim tidak ditemukan")

// ErrAssignmentUnavailable dikembalikan saat pemindahan PIC belum dapat dijalankan.
//
// Sebabnya satu: akun aplikasi belum memiliki `GRANT UPDATE (USERTEKNIS_1)` pada tabel kerja
// Pega — hak yang diminta bersama berkas `pindahpic.sql`. Dinyatakan sebagai galat
// tersendiri supaya layar dapat mengatakan APA yang kurang, bukan sekadar gagal.
var ErrAssignmentUnavailable = errors.New("dashboardclaim: pemindahan PIC belum terpasang")
