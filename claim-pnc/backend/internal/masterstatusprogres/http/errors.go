package masterstatusprogreshttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan
// menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama
// sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// penulisGalatModul memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.logger, h.writeResponse, h.writeError)
}

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.DenganGalatPortal
// yang membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai
// seluruh modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterstatusprogres.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend,
		// 422 berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja.
		detail := apierror.ColumnErrors(validationError.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterstatusprogres.ErrParentNotFound):
		// Diperiksa SEBELUM ErrNotFound. Keduanya berarti "tidak ditemukan", tetapi yang
		// ini menunjuk ISIAN yang dapat diperbaiki pengguna — induk yang dipilihnya sudah
		// tidak ada — sehingga jawabannya 422 dengan keterangan menempel di isian itu,
		// bukan 404 yang membuat layar tampak kehilangan barisnya sendiri.
		//
		// Nama isiannya `id_induk`, sama persis dengan field JSON yang dikirim layar,
		// supaya keterangan galat menempel di tempat yang benar tanpa penerjemahan.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail: []ViolationDTO{{
				Field:   "id_induk",
				Message: "Status progres 1 yang dipilih sudah tidak ada. Muat ulang halaman, lalu pilih kembali.",
			}},
		}, true

	case errors.Is(err, masterstatusprogres.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Status progres yang dimaksud tidak ditemukan. Mungkin sudah dihapus petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterstatusprogres.ErrIDTaken):
		// 409, bukan 422: pengguna tidak melakukan kesalahan apa pun, dan tidak ada
		// isian yang dapat ia perbaiki. Yang terjadi adalah dua penambahan bersamaan.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Nomor status progres baru sudah dipakai petugas lain. Coba simpan sekali lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
