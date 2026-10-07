package inboxmanager

import (
	"errors"

	"claim-pnc/internal/platform/validation"
)

// Galat domain modul Inbox Manager.
//
// Seluruhnya tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP,
// dan domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan
// domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan di layar ini: modul ini MENULIS, dan setiap
	// keputusan dicatat atas nama pemanggilnya. Tanpa identitas tidak ada satu pun
	// keputusan yang boleh ditulis.
	ErrCallerUnknown = errors.New("inboxmanager: identitas pemanggil tidak terbaca")

	// ErrTabNotAllowed berarti tab yang diminta ada, tetapi bukan hak pemanggil.
	//
	// Ia DIBEDAKAN dari "tab tidak dikenal": yang pertama menyatakan pengguna salah alamat,
	// yang kedua menyatakan modulnya rusak. Menjawab keduanya dengan pesan yang sama akan
	// membuat petugas melaporkan kerusakan yang tidak ada.
	ErrTabNotAllowed = errors.New("inboxmanager: tab bukan hak pemanggil")

	// ErrQueueNotDecidable berarti tab yang diminta bukan antrean yang dapat diputuskan.
	//
	// Ia menjaga rute keputusan dari dipanggil atas tab dashboard maupun tab ringkasan.
	ErrQueueNotDecidable = errors.New("inboxmanager: antrean ini tidak dapat diputuskan")

	// ErrApproveBlocked berarti jalur SETUJU pada antrean itu sedang ditahan.
	//
	// Ia bukan galat sistem melainkan keadaan yang diputuskan dan dijelaskan — lihat
	// Tab.Decision.ApproveBlockedReason, dan catatan panjang pada tab Payment Klaim
	// Akseptasi di tab.go.
	//
	// Ia dijadikan galat tersendiri supaya layar dapat menyampaikan ALASANNYA, bukan
	// menggambar tombol yang gagal tanpa keterangan.
	ErrApproveBlocked = errors.New("inboxmanager: jalur setuju sedang ditahan")

	// ErrSourceUnavailable berarti sumber sebuah antrean tidak dapat dibaca dari basis data.
	//
	// # Kenapa ia galat TERSENDIRI, bukan galat sistem biasa
	//
	// Karena ia keadaan yang SUDAH DIKETAHUI, berlangsung lama, dan punya pemilik yang
	// jelas — bukan kegagalan tak terduga.
	//
	// Saat diperiksa 2026-09-28, `POOLDATA.SPAREPART_HE` adalah VIEW berstatus INVALID:
	// ia membaca `JSON_VALUE(a.JSONDATA, …)` dari `POOLDATA.M_SPAREPART_HE`, sementara
	// kolom `JSONDATA` sudah tidak ada lagi di tabel itu. Setiap pembacaannya gagal dengan
	// ORA-04063.
	//
	// Menjawabnya dengan "Terjadi kesalahan pada sistem" membuat penyelia melaporkannya
	// sebagai kerusakan aplikasi, dan yang menerima laporan itu harus menelusuri log untuk
	// menemukan hal yang sudah kita ketahui sejak awal.
	ErrSourceUnavailable = errors.New("inboxmanager: sumber antrean tidak dapat dibaca")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian mana
// yang salah — sama halnya dengan nama field JSON (`D-80`).
const (
	FieldTab     = "tab"
	FieldVerdict = "keputusan"
	FieldKeys    = "baris"
	FieldReason  = "alasan"
	FieldPeriod  = "periode"
)

// Violation adalah satu pelanggaran pada satu isian.
type Violation = validation.Violation

// ValidationError mengumpulkan SELURUH pelanggaran, bukan yang pertama saja.
//
// Ini meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`), dan pada layar
// keputusan ia lebih dari peniruan: penyelia yang memilih baris lalu lupa mengisi alasan
// tidak perlu menemukan kesalahannya satu per satu.
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
		return "inboxmanager: isian tidak sah"
	}
	return validation.Format(e.Violations, "inboxmanager: ", ": ", "; ", "")
}
