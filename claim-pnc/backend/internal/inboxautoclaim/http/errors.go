package inboxautoclaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/platform/logging"
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
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500 dengan
		// pesan umum dan menaruh rinciannya di log saja. Rincian galat internal tidak
		// pernah dikirim ke peramban.
		h.writeError(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeResponse(w, r, status, body)
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
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, p := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada baris yang belum benar. Perbaiki berkasnya lalu unggah lagi.",
			Detail:  detail,
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
