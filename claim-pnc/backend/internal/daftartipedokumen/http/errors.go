package daftartipedokumenhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan menambah
// kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Daftarnya pendek, dan itu memang cerminan modulnya: tanpa validasi berarti tanpa
// `validasi_gagal`, dan tanpa keunikan berarti tanpa galat bentrok.
const (
	CodeNotFound         = "tipe_dokumen_tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
)

// JSONWriter menuliskan badan respons yang berhasil. Modul ini tidak membawa penulisnya
// sendiri supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// galat portal, cacat pemrograman — diserahkan ke penulis bersama, yang menjawab
		// dengan kode yang sudah dikenal frontend dan menaruh rinciannya di log saja.
		// Rincian galat internal tidak pernah dikirim ke peramban.
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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang
// lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat dari cmd — satu pemetaan yang dipakai seluruh modul bisnis,
// bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, daftartipedokumen.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Tipe dokumen yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
