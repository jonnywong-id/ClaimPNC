package inboxbandinghargasalvage

import (
	"errors"

	"claim-pnc/internal/platform/validation"
)

// Galat domain modul Inbox Banding Harga Salvage.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan domain
// adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan: KEDUA tab menyaring `NAMAKOMITE` menurut login
	// pemanggil, sehingga tanpa identitas layar ini tidak punya satu baris pun untuk
	// ditampilkan — dan menampilkan seluruh banding sebagai gantinya berarti membuka
	// antrean komite lain, lengkap dengan angka uang yang sedang mereka putuskan.
	ErrCallerUnknown = errors.New(
		"inboxbandinghargasalvage: identitas pemanggil tidak terbaca")

	// ErrAlreadyDecided berarti tidak ada baris yang dapat diputuskan.
	//
	// Dua sebab, dan keduanya sengaja TIDAK dibedakan: barangnya sudah diputus lebih dulu,
	// atau ia bukan milik komite ini. Membedakannya memberi tahu penanya bahwa sebuah
	// IDDETAILSALVAGE nyata dan sedang ditangani orang lain — dan id itu berurutan.
	//
	// Ia juga yang menjawab klik ganda: penyaring `TGLAPPROVE IS NULL` membuat penekanan
	// kedua tidak mengubah apa pun, dan pengguna berhak tahu itu alih-alih melihat pesan
	// "berhasil" yang kedua kalinya tidak berarti apa-apa.
	ErrAlreadyDecided = errors.New(
		"inboxbandinghargasalvage: banding ini sudah diputus atau bukan milik Anda")

	// ErrWriteNotAvailable berarti aksi tulis diminta yang belum dibangun.
	//
	// Ia sengaja BUKAN "tidak ditemukan". Layar lama punya kolom "Action" berisi tombol
	// Approve dan Reject; tindakan yang dijawab "halaman tidak ditemukan" terbaca sebagai
	// kerusakan, sementara yang dibutuhkan pengguna adalah tahu mengapa dan ke mana ia
	// harus pergi.
	ErrWriteNotAvailable = errors.New(
		"inboxbandinghargasalvage: tindakan ini belum tersedia")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian mana
// yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab    = "tab"
	FieldSearch = "cari"

	// FieldSalvageID dipakai tombol Approve/Reject. Nama isian lainnya sudah terdaftar di
	// tab.go sebagai nama kolom grid, dan didaftarkan ulang di sini berarti dua konstanta
	// yang dapat berselisih tanpa ketahuan.
	FieldSalvageID = "id_salvage"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation = validation.Violation

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`).
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membentuk galat validasi dari daftar pelanggaran.
func NewValidationError(violations []Violation) *ValidationError {
	return &ValidationError{Violations: violations}
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Violations, bukan ini.
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "inboxbandinghargasalvage: isian tidak sah"
	}
	return validation.Format(e.Violations, "inboxbandinghargasalvage: ", ": ", "; ", "")
}
