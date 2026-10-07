package daftartipedokumenhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/daftartipedokumen"
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
// Daftarnya pendek, dan itu memang cerminan modulnya: tanpa validasi berarti tanpa
// `validasi_gagal`, dan tanpa keunikan berarti tanpa galat bentrok.
const (
	CodeNotFound         = "tipe_dokumen_tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
)

// JSONWriter menuliskan badan respons yang berhasil. Modul ini tidak membawa penulisnya
// sendiri supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.Logger, h.WriteResponse, h.WriteError)
}

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
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
