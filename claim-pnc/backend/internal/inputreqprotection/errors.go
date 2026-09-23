package inputreqprotection

import (
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound dikembalikan Repo ketika proteksi yang diminta tidak ada.
//
// Ia galat tersendiri, bukan nilai kosong. Mengembalikan Protection kosong akan membuat
// layar menampilkan form berisi field kosong seolah proteksinya ada — dan pengguna baru
// menyadarinya setelah menyimpan.
var ErrNotFound = errors.New("inputreqprotection: proteksi tidak ditemukan")

// ErrLocked dikembalikan ketika proteksi yang hendak disunting sudah tertaut ke klaim.
//
// Aturannya dari `Section/InboxReqProtection_Section-Section.xml:8657`, yang menonaktifkan
// tautan baris ketika .CaseID tidak kosong. Lihat Protection.Editable.
var ErrLocked = errors.New("inputreqprotection: proteksi sudah tertaut klaim dan tidak dapat disunting")

// ErrAccepted dikembalikan ketika proteksi yang hendak disunting sudah diakseptasi.
//
// Dibedakan dari ErrLocked karena penyebabnya berbeda dan yang harus dilakukan pengguna
// juga berbeda: yang satu menunggu akseptasi, yang satu sudah selesai. Pesan yang sama
// untuk keduanya akan membuat pengguna menunggu sesuatu yang tidak akan datang.
var ErrAccepted = errors.New("inputreqprotection: proteksi sudah diakseptasi dan tidak dapat disunting")

// ── Kesalahan validasi ───────────────────────────────────────────────────────────

// FieldError adalah satu pelanggaran aturan bisnis pada satu field.
type FieldError struct {
	// Field adalah nama field pada kontrak API — berbahasa Indonesia, sama dengan yang
	// dikirim ke klien, supaya layar dapat menempelkan pesannya ke kolom yang tepat tanpa
	// tabel pemetaan.
	Field string

	// Message dibaca pengguna. Berbahasa Indonesia dan MENJELASKAN CARA MEMPERBAIKI,
	// mengikuti gaya pesan sistem lama yang sudah dikenal pengguna (`11-CROSSCUTTING` §1.2
	// butir 6).
	Message string
}

func (e FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationError mengumpulkan SELURUH pelanggaran sekaligus.
//
// # Kenapa dikumpulkan, bukan berhenti pada yang pertama
//
// `11-CROSSCUTTING` §1.2 butir 1 menetapkannya sebagai KESETARAAN PERILAKU, bukan
// preferensi: layar lama memeriksa seluruh aturannya lalu menampilkan semua pesan
// sekaligus. Form proteksi memuat sampai delapan field wajib; mengembalikan satu pesan per
// percobaan akan membuat pengguna menyimpan delapan kali untuk mengetahui apa saja yang
// kurang.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "inputreqprotection: validasi gagal"
	}

	parts := make([]string, 0, len(e.Errors))
	for _, fe := range e.Errors {
		parts = append(parts, fe.Error())
	}
	return "inputreqprotection: " + strings.Join(parts, "; ")
}

// Add mencatat satu pelanggaran.
func (e *ValidationError) Add(field, message string) {
	e.Errors = append(e.Errors, FieldError{Field: field, Message: message})
}

// Failed menyatakan ada pelanggaran yang tercatat.
func (e *ValidationError) Failed() bool {
	return len(e.Errors) > 0
}

// OrNil mengembalikan nil bila tidak ada pelanggaran.
//
// Ia ada supaya pemanggil dapat menulis `return v.OrNil()` tanpa memeriksa panjang slice —
// dan supaya tidak ada yang mengembalikan `*ValidationError` kosong yang bukan nil, cacat
// klasik Go yang membuat `err != nil` bernilai benar padahal tidak ada galat.
func (e *ValidationError) OrNil() error {
	if e.Failed() {
		return e
	}
	return nil
}
