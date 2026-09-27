package inboxcompliance

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox Compliance.
//
// Keduanya tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP,
// dan domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrTabNotReady berarti tabnya dikenal, tetapi belum dapat dilayani karena artefak
	// yang dibutuhkannya belum ada.
	//
	// Ia DIBEDAKAN dari "tab tidak dikenal", dan pembedaan itu yang penting: keduanya
	// sama-sama gagal, tetapi yang satu berarti salah ketik dan yang lain berarti ada
	// pihak lain yang sedang ditunggu. Pesan yang sama untuk keduanya akan membuat
	// pengguna melapor "modulnya rusak" alih-alih menagih artefaknya.
	ErrTabNotReady = errors.New("inboxcompliance: tab belum dapat dilayani")

	// ErrClaimNotInQueue berarti klaim yang hendak dikirim ke Post Audit tidak sedang
	// menunggu di antrean Compliance.
	//
	// Ia BUKAN "tidak ditemukan": klaimnya boleh jadi ada dan sehat, hanya tidak berada
	// di antrean ini — mungkin sudah dikirim petugas lain beberapa detik sebelumnya,
	// mungkin sudah selesai. Pesan "tidak ditemukan" akan membuat petugas mencari
	// klaimnya, padahal yang perlu dilakukan hanyalah menyegarkan daftar.
	ErrClaimNotInQueue = errors.New("inboxcompliance: klaim tidak ada di antrean Compliance")

	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia hanya menghalangi jalur TULIS. Kedua jalur baca tidak memerlukannya — antreannya
	// workbasket, yang isinya sama bagi setiap petugas.
	//
	// Pada jalur tulis ia tetap menghalangi meski identitasnya tidak menentukan apa yang
	// boleh dikirim, karena tanpanya pengiriman itu tidak tercatat siapa pelakunya. Tabel
	// Post Audit tidak punya kolom pengirim, sehingga log adalah satu-satunya tempat
	// identitas itu tersimpan — dan menulis baris tanpa pelaku yang terbaca sama sekali
	// bertentangan dengan `D-28`.
	ErrCallerUnknown = errors.New("inboxcompliance: identitas pemanggil tidak terbaca")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab       = "tab"
	FieldReference = "referensi"
	FieldRemarks   = "catatan"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`). Layar ini hanya
// punya satu isian yang dapat dilanggar, sehingga bentuk jamaknya belum terpakai hari ini —
// ia tetap dipakai supaya seragam dengan modul lain, dan supaya penambahan penyaring kelak
// tidak menuntut mengubah bentuk galatnya.
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
		return "inboxcompliance: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxcompliance: " + strings.Join(parts, "; ")
}

// TabNotReadyError menyebut tab mana yang belum dapat dilayani dan apa yang kurang.
//
// Ia membungkus ErrTabNotReady supaya errors.Is tetap mengenalinya, sekaligus membawa
// keterangan yang cukup untuk ditampilkan kepada pengguna tanpa lapisan transport perlu
// menyusun kalimatnya sendiri.
type TabNotReadyError struct {
	Tab     Tab
	Blocker string
}

// NewTabNotReadyError membentuk galat tab belum siap.
func NewTabNotReadyError(tab Tab) *TabNotReadyError {
	return &TabNotReadyError{Tab: tab, Blocker: tab.Blocker}
}

func (e *TabNotReadyError) Error() string {
	return "inboxcompliance: tab " + e.Tab.Code + " belum dapat dilayani: " + e.Blocker
}

// Unwrap membuat errors.Is(err, ErrTabNotReady) bernilai benar.
func (e *TabNotReadyError) Unwrap() error { return ErrTabNotReady }
