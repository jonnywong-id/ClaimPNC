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

// Action adalah tindakan yang diminta salah satu tombol yang belum dibangun.
//
// Ketiganya ADA di layar dan dapat ditekan. Yang dijawab bukan "halaman tidak ditemukan"
// melainkan alasan spesifik mengapa tindakan ITU belum dapat dijalankan — dan alasannya
// berbeda-beda, sehingga menjawabnya dengan satu kalimat yang sama untuk ketiganya akan
// menyembunyikan apa yang sebenarnya kurang.
type Action string

const (
	// ActionSend adalah tombol **"SEND"** di bawah grid rincian.
	ActionSend Action = "kirim"

	// ActionUpload adalah tombol **"Upload File Penunjang"** di atas grid rincian.
	ActionUpload Action = "unggah-penunjang"

	// ActionSendPreDLA adalah tombol **"Kirim Pre DLA"** DI DALAM panel "Print Pre DLA".
	//
	// Tombol "Print Pre DLA" sendiri BUKAN tindakan yang ditolak — ia membuka panel, dan
	// panelnya sudah dibangun. Yang ditolak adalah tombol di dalamnya.
	ActionSendPreDLA Action = "kirim-pre-dla"

	// ActionDownloadAttachment adalah tombol unduh lampiran di dalam panel yang sama
	// (`PNCDownloadFile` pada `Section/PrintPreDLA-Section.xml`).
	ActionDownloadAttachment Action = "unduh-lampiran"
)

// actionReasons memetakan tiap tindakan ke alasan yang dibaca pengguna.
//
// Kalimatnya menyebut APA yang belum ada, bukan sekadar "belum tersedia". Petugas yang
// membaca "belum tersedia" akan menunggu; petugas yang membaca "kerjakan lewat Pega"
// tahu apa yang harus dilakukannya hari ini.
var actionReasons = map[Action]string{
	ActionSend: "Tombol \"Send\" belum tersedia di sistem baru. Di Pega ia menjalankan " +
		"EMPAT hal berurutan: mengirim surat PLA/DLA beserta lampirannya ke " +
		"reasuradur lewat email, memperbarui master reasuransi, menyisipkan " +
		"dokumennya, lalu menandai dokumen sebagai terkirim. Hanya yang terakhir " +
		"dapat dikerjakan sekarang — dan mengerjakannya sendirian akan membuat " +
		"barisnya HILANG dari antrean padahal tidak satu pun surat sampai ke " +
		"reasuradur. Kerjakan lewat Pega.",

	ActionUpload: "Tombol \"Upload File Penunjang\" belum tersedia di sistem baru. Ia " +
		"menyimpan lampiran ke penyimpanan dokumen internal (`D-16`), yang belum " +
		"tersambung. Kerjakan lewat Pega.",

	ActionSendPreDLA: "Tombol \"Kirim Pre DLA\" belum tersedia di sistem baru. Ia " +
		"menandai Pre-DLA sebagai terkirim beserta tanggalnya, dan tabel Pre-DLA " +
		"masih DITULIS Pega selama masa paralel — hanya satu sistem yang boleh " +
		"menulis satu tabel. Kerjakan lewat Pega.",

	ActionDownloadAttachment: "Unduh lampiran belum tersedia di sistem baru. Berkasnya " +
		"ada di penyimpanan dokumen internal (`D-16`), yang belum tersambung. " +
		"Ambil lewat Pega.",
}

// NotAvailableError menyatakan sebuah tombol ditekan yang tindakannya belum dibangun.
//
// Ia membawa TINDAKANNYA, bukan hanya kenyataan bahwa ia belum ada, supaya lapisan
// transport dapat menjawab alasan yang tepat. Tanpa itu, ketiga tombol menjawab kalimat
// yang sama — dan pengguna yang menekan "Print Pre DLA" akan membaca penjelasan tentang
// email reasuradur.
type NotAvailableError struct {
	Action Action
}

// NewNotAvailable membentuk penolakan untuk satu tindakan.
//
// Tindakan yang TIDAK dikenal tetap menghasilkan penolakan, bukan galat lain: yang
// dituju pengguna memang tombol yang belum dibangun, dan nama tindakan yang salah ketik
// di alamat bukan sesuatu yang perlu dibedakan di layar. Alasannya menjadi kalimat umum.
func NewNotAvailable(raw string) *NotAvailableError {
	action := Action(strings.TrimSpace(raw))
	if _, known := actionReasons[action]; !known {
		return &NotAvailableError{}
	}
	return &NotAvailableError{Action: action}
}

// Reason adalah kalimat yang dibaca pengguna.
func (e *NotAvailableError) Reason() string {
	if reason, known := actionReasons[e.Action]; known {
		return reason
	}
	return "Tindakan ini belum tersedia di sistem baru. Kerjakan lewat Pega."
}

// Error menyusun pesan ringkas untuk log. Yang dibaca pengguna adalah Reason.
func (e *NotAvailableError) Error() string {
	if e.Action == "" {
		return "inboxpladlapredla: tindakan tidak dikenal belum tersedia"
	}
	return "inboxpladlapredla: tindakan " + string(e.Action) + " belum tersedia"
}

// Unwrap membuat errors.Is(err, ErrWriteNotAvailable) tetap benar.
//
// Pemanggil yang hanya ingin tahu "apakah ini penolakan tombol" tidak perlu mengenal
// tipe ini, dan kode yang sudah memeriksa sentinelnya tidak perlu diubah.
func (e *NotAvailableError) Unwrap() error { return ErrWriteNotAvailable }

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
