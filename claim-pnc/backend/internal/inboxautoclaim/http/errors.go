package inboxautoclaimhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/inboxautoclaim"
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
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk {kode, pesan} tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Ketiga kode pertama sengaja dinamai SAMA dengan milik masterstatusprogres: keadaan yang
// sama sebaiknya punya kode yang sama walau kontrak bersamanya belum ada.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeEmptyUpload dibedakan dari validasi biasa: tidak ada isian yang salah, dan
	// tidak ada satu pun baris yang dapat ditunjuk. Yang perlu diperbaiki pengguna
	// adalah berkasnya, bukan sebuah kolom.
	CodeEmptyUpload = "unggahan_kosong"

	// CodePremiumServiceDown: layanan total premi (Cek Premi) tidak dapat dihubungi.
	CodePremiumServiceDown = "layanan_premi_gagal"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.Logger, h.WriteResponse, h.WriteError)
}

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *inboxautoclaim.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend,
		// 422 berarti pengguna perlu memperbaiki isiannya (10-API-STRATEGY.md §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja. Pada berkas
		// berisi ratusan baris, satu galat per percobaan berarti mengunggah ulang
		// ratusan kali.
		detail := apierror.ColumnErrors(validationError.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada baris yang belum benar. Perbaiki berkasnya lalu unggah lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, inboxautoclaim.ErrPremiumServiceUnavailable):
		// 502: permintaannya benar, sistem di belakang kita yang gagal. Galatnya tetap
		// dicatat (status >= 500), pesannya tidak memuat alamat layanan.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodePremiumServiceDown,
			Message: "Layanan cek premi tidak dapat dihubungi. Coba lagi beberapa saat lagi.",
		}, true

	case errors.Is(err, inboxautoclaim.ErrBatchNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Batch yang dimaksud tidak ada pada portal entitas ini. Muat ulang daftarnya.",
		}, true

	case errors.Is(err, inboxautoclaim.ErrEmptyUpload):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeEmptyUpload,
			Message: "Berkas tidak memuat satu baris data pun di bawah baris judul.",
		}, true

	case errors.Is(err, inboxautoclaim.ErrUnknownResult):
		// 400, bukan 422: yang salah adalah bentuk permintaannya — nilai penyaring di
		// luar daftar yang dikenal. Pengguna tidak dapat memperbaikinya lewat isian mana
		// pun, dan penyebabnya hampir pasti tautan yang disusun tangan.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Jenis hasil yang diminta tidak dikenal. Pakai tombol Export Berhasil atau Export Gagal.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
