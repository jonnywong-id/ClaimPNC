package inboxsalvage

import (
	"errors"
	"strings"
)

// Galat domain modul Inbox Salvage.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Di layar ini ia BUKAN sekadar soal jejak, berbeda dari modul inbox lain: daftar
	// "Request Balai Lelang" menyaring `PIC = <pemanggil>`, sehingga tanpa identitas
	// daftarnya tidak dapat disusun sama sekali — dan yang lebih berbahaya, tanpa
	// penyaring itu ia akan menampilkan pengajuan milik petugas lain.
	ErrCallerUnknown = errors.New("inboxsalvage: identitas pemanggil tidak terbaca")

	// ErrWriteNotAvailable berarti aksi tulis diminta yang belum dibangun.
	//
	// Ia ADA supaya tombol Approve, Reject, dan Send To BalaiLelang dapat dijawab dengan
	// alasan alih-alih dengan "halaman tidak ditemukan". Keduanya terlihat sangat berbeda
	// bagi pengguna, dan hanya yang pertama yang memberi tahu apa yang harus dilakukannya.
	ErrWriteNotAvailable = errors.New("inboxsalvage: tindakan ini belum tersedia")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab    = "tab"
	FieldSearch = "cari"

	// Isian form Tambah. Namanya mengikuti judul di
	// `Section/TambahData_Salvage-Section.xml` (`D-13`).
	FieldFormClaimNo      = "nomor_klaim"
	FieldFormObjectName   = "nama_object"
	FieldFormCoverageName = "nama_coverage"
	FieldFormInputDate    = "tanggal_input"
	FieldFormSalvageType  = "jenis_salvage"
	FieldFormStatus       = "status_salvage"
	FieldFormLocation     = "lokasi_salvage"
	FieldFormCurrency     = "mata_uang"
	FieldFormMinimum      = "minimum_salvage"
	FieldFormQuantity     = "quantity_salvage"
	FieldFormEmail        = "email"
	FieldFormItems        = "detail_item_salvage"
	FieldFormFile         = "berkas"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`). Di layar ini ia
// terasa pada form Tambah, yang punya tujuh belas isian: pengguna yang melewatkan tiga di
// antaranya harus diberi tahu ketiganya sekaligus, bukan satu, lalu satu lagi.
//
// Sistem lama pun mengumpulkannya — `Activity/GCNMNewSalvage_act-Act.xml` langkah 4 dan 5
// memasang dua pesan lewat `Page-Set-Messages` sebelum menyimpan.
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
		return "inboxsalvage: isian tidak sah"
	}

	parts := make([]string, 0, len(e.Violations))
	for _, v := range e.Violations {
		parts = append(parts, v.Field+": "+v.Message)
	}
	return "inboxsalvage: " + strings.Join(parts, "; ")
}
