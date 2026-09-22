package masterpenolakanhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/masterpenolakan"
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
//
// Satu fungsi untuk kedua tab, bukan satu per tab: keduanya hidup di satu layar, dan dua
// tafsiran atas galat yang sama akan membuat pengguna melihat dua bentuk pesan pada layar
// yang sama.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterpenolakan.ValidationError

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

	case errors.Is(err, masterpenolakan.ErrParentNotFound):
		// Diperiksa SEBELUM ErrNotFound. Keduanya berarti "tidak ditemukan", tetapi yang
		// ini menunjuk ISIAN yang dapat diperbaiki pengguna — Status Penolakan 1 yang
		// dipilihnya sudah tidak ada — sehingga jawabannya 422 dengan keterangan menempel
		// di isian itu, bukan 404 yang membuat layar tampak kehilangan barisnya sendiri.
		//
		// Nama isiannya `id_status_1`, sama persis dengan field JSON yang dikirim layar,
		// supaya keterangan galat menempel di tempat yang benar tanpa penerjemahan.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail: []ViolationDTO{{
				Field:   "id_status_1",
				Message: "Status Penolakan 1 yang dipilih sudah tidak ada. Muat ulang halaman, lalu pilih kembali.",
			}},
		}, true

	case errors.Is(err, masterpenolakan.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Status penolakan yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterpenolakan.ErrKomiteNotFound):
		// Pesannya menyebut master yang benar. Kedua tab hidup di satu layar, dan
		// "tidak ditemukan" tanpa keterangan akan membuat petugas mencari di tab yang
		// salah.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Penolakan komite yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterpenolakan.ErrIDTaken):
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
