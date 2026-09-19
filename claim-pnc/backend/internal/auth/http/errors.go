package authhttp

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan.
const (
	CodeWrongCredential  = "kredensial_salah"
	CodeUserInactive     = "pengguna_tidak_aktif"
	CodeIdentityDown     = "sistem_identitas_tidak_terhubung"
	CodeInvalidSession   = "sesi_tidak_sah"
	CodeSessionExpired   = "sesi_kedaluwarsa"
	CodeMalformedRequest = "permintaan_cacat"
	CodeInternalError    = "galat_internal"
)

// wrongCredentialMessage sengaja sama untuk pengguna yang tidak ada dan kata sandi yang
// salah. Membedakan keduanya memberi tahu siapa saja yang punya akun di sistem ini.
const wrongCredentialMessage = "Nama pengguna atau kata sandi salah."

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat menjadi respons HTTP.
//
// Galat yang tidak dikenali dijawab 500 dengan pesan umum, dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body := mapError(err)
		if status >= http.StatusInternalServerError {
			logging.From(r.Context(), logger).Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		WriteJSON(w, r, status, body, logger)
	}
}

func mapError(err error) (int, ErrorResponse) {
	var incompleteProfile *auth.IncompleteProfileError

	switch {
	case errors.Is(err, auth.ErrWrongCredential):
		return http.StatusUnauthorized, ErrorResponse{
			Code:    CodeWrongCredential,
			Message: wrongCredentialMessage,
		}

	case errors.Is(err, auth.ErrUserInactive):
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeUserInactive,
			Message: "Akun Anda tidak aktif. Hubungi administrator Claim PNC.",
		}

	case errors.Is(err, auth.ErrIdentitySystemUnreachable):
		// 503, bukan 401: ini bukan kesalahan pengguna, dan mencoba berulang kali
		// justru membanjiri sistem yang sedang bermasalah.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeIdentityDown,
			Message: "Sistem identitas sedang tidak dapat dihubungi. Coba beberapa saat lagi.",
		}

	case errors.As(err, &incompleteProfile):
		// Sistem identitas menjawab dengan profil yang tidak lengkap. Meneruskannya
		// berarti pengguna masuk tetapi tidak dikenali data klaimnya sendiri.
		return http.StatusBadGateway, ErrorResponse{
			Code:    CodeIdentityDown,
			Message: "Profile pengguna dari sistem identitas tidak lengkap. Hubungi administrator Claim PNC.",
		}

	case errors.Is(err, auth.ErrSessionExpired):
		// Dibedakan dari sesi tidak sah supaya frontend dapat menyelamatkan isian yang
		// belum tersimpan, bukan sekadar melempar pengguna ke layar masuk.
		return http.StatusUnauthorized, ErrorResponse{
			Code:    CodeSessionExpired,
			Message: "Sesi Anda sudah berakhir. Silakan masuk kembali.",
		}

	case errors.Is(err, auth.ErrSessionNotFound), errors.Is(err, auth.ErrSessionRevoked):
		return http.StatusUnauthorized, ErrorResponse{
			Code:    CodeInvalidSession,
			Message: "Sesi tidak sah. Silakan masuk kembali.",
		}

	default:
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeInternalError,
			Message: "Terjadi kesalahan pada sistem.",
		}
	}
}

// WriteJSON menuliskan badan respons.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, body any, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Respons autentikasi tidak boleh disinggahi cache mana pun di jalur.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logging.From(r.Context(), logger).Error("gagal menulis respons",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
}
