package inboxkomunikasicabang

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox Komunikasi Cabang.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// # Kenapa ia MEMBLOKIR di layar ini, bukan sekadar menghilangkan jejak
	//
	// Karena batas datanya diturunkan DARI login. Tanpa login, `BranchResolver` tidak punya
	// apa pun untuk diterjemahkan, dan satu-satunya jalan yang tersisa adalah menampilkan
	// percakapan siapa saja. Itu tidak pernah menjadi pilihan.
	ErrCallerUnknown = errors.New("inboxkomunikasicabang: identitas pemanggil tidak terbaca")

	// ErrBranchUnreadable berarti sumber kode cabang tidak dapat dibaca sama sekali.
	//
	// # Kenapa ia DIBEDAKAN dari cabang yang tidak ditemukan
	//
	// Karena keduanya dibereskan pihak yang berbeda, dan menimpa orang yang berbeda:
	//
	//	cabang tidak ditemukan  -> petugasnya belum terdaftar di HRD; menimpa SATU orang
	//	sumbernya tidak terbaca -> DB Link atau tabel BRANCH mati; menimpa SEMUA orang
	//
	// Yang pertama BUKAN galat di layar ini — `P-5` menetapkan petugasnya diperlakukan
	// sebagai kantor pusat, sama seperti Pega. Yang kedua adalah galat, dan ia sementara:
	// mencoba lagi memang masuk akal.
	//
	// Menyamakan keduanya berarti petugas melihat percakapan kantor pusat ketika yang
	// sebenarnya terjadi adalah DB Link sedang mati — daftar yang tampak wajar padahal
	// isinya salah, kegagalan paling mahal yang ada.
	ErrBranchUnreadable = errors.New(
		"inboxkomunikasicabang: sumber kode cabang tidak dapat dibaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta pada modul yang hanya membaca.
	//
	// Ia ADA supaya keempat tindakan yang di layar lama menulis — Kirim Pesan, Balas,
	// Selesai Komunikasi, dan Tambah — dapat dijawab dengan alasan alih-alih dengan
	// "halaman tidak ditemukan". Keduanya terlihat sangat berbeda bagi pengguna, dan hanya
	// yang pertama yang memberi tahu apa yang harus dilakukannya.
	ErrWriteNotAvailable = errors.New(
		"inboxkomunikasicabang: modul ini belum menulis apa pun")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab = "tab"

	// FieldConversation menunjuk nomor percakapan pada permintaan layar detail.
	FieldConversation = "komunikasi"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

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
		return "inboxkomunikasicabang: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxkomunikasicabang: " + strings.Join(parts, "; ")
}
