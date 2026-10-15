package masterpasalhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/masterpasal"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan
// menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterpasal.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422
		// berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja.
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, violation := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: violation.Field, Message: violation.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterpasal.ErrNotFound):
		// Satu pesan untuk tiga jalur — memuat, menyimpan, dan menghapus — karena
		// penyebabnya satu dan sama: barisnya sudah tidak ada lagi. Pesannya menyebut
		// kemungkinan penyebabnya, karena di modul ini penyebab itu NYATA: layar ini
		// punya tombol Hapus yang membuang barisnya secara permanen, sehingga dua petugas
		// yang bekerja bersamaan memang dapat saling mendahului.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Pasal kerugian yang dimaksud tidak ditemukan. Mungkin sudah dihapus atau diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterpasal.ErrIDTaken):
		// 409, bukan 422: pengguna tidak melakukan kesalahan apa pun, dan tidak ada isian
		// yang dapat ia perbaiki. Yang terjadi adalah dua penambahan bersamaan.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Nomor baru sudah dipakai petugas lain. Coba simpan sekali lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
