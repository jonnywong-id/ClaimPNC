package mastercolsimasonlinehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/mastercolsimasonline"
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
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama
// sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Nilainya sengaja SAMA PERSIS dengan modul Master Status Progres, bukan dikarang
// sendiri: klien sudah mengenali ketiganya, dan kode galat baru untuk arti yang sama
// hanya menambah cabang di frontend tanpa menambah keterangan apa pun.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan
		// jaringan, cacat pemrograman — diserahkan ke penulis bersama, yang menjawab
		// 500 dengan pesan umum dan menaruh rinciannya di log saja. Rincian galat
		// internal tidak pernah dikirim ke peramban.
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
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai
// seluruh modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *mastercolsimasonline.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend,
		// 422 berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja.
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, v := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastercolsimasonline.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Cause of loss yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
