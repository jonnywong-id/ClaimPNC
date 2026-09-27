package inboxacceptopenprotectionhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan.
const (
	CodeBadRequest    = "permintaan_cacat"
	CodeNotFound      = "tidak_ditemukan"
	CodeConflict      = "konflik"
	CodeInternalError = "galat_internal"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat menjadi respons HTTP.
//
// # Kenapa "sudah diakseptasi" dijawab 409, dan pesannya berbunyi demikian
//
// Layar ini antrean BERSAMA (`Flow/CreateProtection_Flow.xml` menempatkannya di workbasket
// `ProtectionPNC`). Petugas kedua yang menekan tombol atas baris yang sama TIDAK sedang
// melakukan kesalahan — ia hanya kalah cepat. Pesan yang menyatakan keputusan sudah diambil
// membuatnya menutup form; pesan galat teknis membuatnya mencoba lagi.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban (`11-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		switch {
		case errors.Is(err, inboxacceptopenprotection.ErrNotFound):
			writeJSON(w, r, http.StatusNotFound, ErrorResponse{
				Code:    CodeNotFound,
				Message: "Permintaan proteksi tidak ditemukan.",
			})
			return

		case errors.Is(err, inboxacceptopenprotection.ErrAlreadyDecided):
			writeJSON(w, r, http.StatusConflict, ErrorResponse{
				Code:    CodeConflict,
				Message: "Permintaan proteksi ini sudah diakseptasi. Muat ulang daftar untuk melihat keputusannya.",
			})
			return

		case errors.Is(err, inboxacceptopenprotection.ErrIncomplete):
			writeJSON(w, r, http.StatusConflict, ErrorResponse{
				Code:    CodeConflict,
				Message: "Permintaan proteksi belum tertaut ke klaim, sehingga belum dapat diakseptasi.",
			})
			return

		case errors.Is(err, inboxacceptopenprotection.ErrUnknownDecision):
			// 400, bukan 409: bentuk permintaannya yang salah, bukan keadaan datanya.
			writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeBadRequest,
				Message: `Keputusan harus "setuju" atau "tolak".`,
			})
			return
		}

		if fallback != nil {
			fallback(w, r, err)
			return
		}

		logging.From(r.Context(), logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
		writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		})
	}
}

// writeBadRequest menjawab permintaan yang cacat bentuknya.
func writeBadRequest(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request, message string) {
	writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Message: message,
	})
}
