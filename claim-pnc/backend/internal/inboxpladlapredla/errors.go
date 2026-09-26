package inboxpladlapredla

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox PLA, DLA, Pre DLA.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia TIDAK menyaring daftar di layar ini — ketiga daftarnya bersama. Yang
	// membutuhkannya adalah jejak: barisnya memuat nama tertanggung dan nomor polis, dan
	// `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang selama pemeriksaan
	// peran belum ada.
	ErrCallerUnknown = errors.New("inboxpladlapredla: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta yang belum dibangun.
	//
	// Ia ADA supaya tombol "Send", "Upload File Penunjang", dan "Print Pre DLA" dapat
	// dijawab dengan ALASAN alih-alih dengan "halaman tidak ditemukan". Keduanya terlihat
	// sangat berbeda bagi pengguna, dan hanya yang pertama yang memberi tahu apa yang
	// harus dilakukannya.
	ErrWriteNotAvailable = errors.New("inboxpladlapredla: tindakan ini belum tersedia")

	// ErrDocumentsNotOnTab berarti grid rincian diminta pada tab yang tidak punya.
	//
	// Tab Pre DLA tidak menggambar grid rincian di Pega. Permintaannya ditolak dengan
	// pesan yang menyebut alasannya, bukan dijawab daftar kosong — daftar kosong akan
	// terbaca sebagai "klaim ini belum punya Pre-DLA", padahal setiap baris di tab itu
	// PASTI punya.
	ErrDocumentsNotOnTab = errors.New("inboxpladlapredla: tab ini tidak punya grid rincian")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab    = "daftar"
	FieldSearch = "cari"
	FieldFrom   = "dari"
	FieldTo     = "sampai"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`). Di layar ini ia
// terasa pada panel pencarian: rentang tanggal terbalik DAN kata kunci terlalu panjang
// harus disampaikan bersamaan, bukan satu, lalu satu lagi.
type ValidationError struct {
	Violations []Violation
}

// NewValidationError membentuk galat validasi dari daftar pelanggaran.
//
// Ia mengembalikan nil bila tidak ada pelanggaran, sehingga pemanggil dapat menyerahkan
// hasil pengumpulannya apa adanya tanpa memeriksa panjangnya lebih dulu.
func NewValidationError(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: violations}
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Violations, bukan
// ini.
func (e *ValidationError) Error() string {
	if len(e.Violations) == 0 {
		return "inboxpladlapredla: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxpladlapredla: " + strings.Join(parts, "; ")
}
