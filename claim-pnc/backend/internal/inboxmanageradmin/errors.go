package inboxmanageradmin

import (
	"errors"

	"claim-pnc/internal/platform/validation"
)

// Galat domain modul Inbox Manager Admin.
//
// Ketiganya tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP,
// dan domain tidak boleh tahu apa pun tentang HTTP
// (`11-CROSSCUTTING.md` §1.2 — kesalahan domain adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Ia bukan sekadar ketidaknyamanan di layar ini: jabatan pemanggil MENENTUKAN tab mana
	// yang boleh ia lihat, sehingga tanpa identitas tidak ada satu pun tab yang dapat
	// dibuka secara sah.
	ErrCallerUnknown = errors.New("inboxmanageradmin: identitas pemanggil tidak terbaca")

	// ErrTabNotAllowed berarti tab yang diminta ada, tetapi bukan hak pemanggil.
	//
	// Ia DIBEDAKAN dari "tab tidak dikenal", dan pembedaan itu penting: yang pertama
	// menyatakan pengguna salah alamat, yang kedua menyatakan modulnya rusak. Menjawab
	// keduanya dengan pesan yang sama akan membuat petugas melaporkan kerusakan yang tidak
	// ada.
	//
	// Pemeriksaannya ditegakkan di SERVER, bukan hanya dengan menyembunyikan tab di layar.
	// Sistem lama hanya menyembunyikan kontainernya (`pyContainerVisibleWhen`), dan
	// `11-SECURITY.md` §3.1 menyebut penyembunyian menu sebagai kenyamanan tampilan —
	// bukan kendali.
	ErrTabNotAllowed = errors.New("inboxmanageradmin: tab bukan hak pemanggil")

	// ErrNoTabAllowed berarti pemanggil tidak berhak atas satu tab pun.
	//
	// Ia bukan galat sistem melainkan keadaan yang memang mungkin terjadi — lihat catatan
	// konsekuensi pada VisibleTabs. Ia dijadikan galat tersendiri supaya layar dapat
	// menjelaskan sebabnya beserta jabatan yang diharapkan, alih-alih menggambar tabel
	// kosong yang terbaca seperti antrean yang memang sepi.
	ErrNoTabAllowed = errors.New("inboxmanageradmin: tidak ada tab yang menjadi hak pemanggil")

	// ErrSourceColumnMissing berarti kolom yang dibutuhkan kueri belum ada di basis data.
	//
	// # Kenapa ia galat TERSENDIRI, bukan galat sistem biasa
	//
	// Karena ia keadaan yang SUDAH DIKETAHUI AKAN TERJADI, berlangsung lama, dan punya
	// tindakan yang jelas — bukan kegagalan tak terduga.
	//
	// Tiga kolom diminta ditambahkan saat modul ini pindah ke `T_CLAIMLIST_ADMIN`
	// (Work Owner 2026-09-27) dan menunggu `migrations/0005` tahap 1 dijalankan DBA.
	// Sampai itu terjadi, setiap pembukaan antrean gagal dengan ORA-00904.
	//
	// Menjawabnya dengan "Terjadi kesalahan pada sistem" membuat petugas melaporkannya
	// sebagai kerusakan aplikasi, dan yang menerima laporan itu harus menelusuri log untuk
	// menemukan hal yang sudah kita ketahui sejak awal. Galat ini menyebutkannya langsung.
	ErrSourceColumnMissing = errors.New(
		"inboxmanageradmin: kolom sumber belum ada di basis data")
)

// Nama isian yang dapat ditunjuk sebuah pelanggaran validasi.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca layar untuk menandai isian
// mana yang salah — sama halnya dengan nama field JSON (`D-80`).
const FieldTab = "tab"

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
		return "inboxmanageradmin: isian tidak sah"
	}
	return validation.Format(e.Violations, "inboxmanageradmin: ", ": ", "; ", "")
}
