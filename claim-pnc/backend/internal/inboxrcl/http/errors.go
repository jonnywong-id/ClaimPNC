package inboxrclhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien — KONTRAK, sehingga berbahasa Indonesia (`D-80`). Nilainya
// sama dengan Inbox Analyst Doctor supaya frontend menangani keadaan yang sama dengan cara
// yang sama.
const (
	CodeCallerUnknown = "profil_pemanggil_tidak_lengkap"
	CodeInternalError = "galat_internal"
)

// JSONWriter menuliskan badan respons.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP; galat yang bukan milik modul
// ini diteruskan ke fallback (galat sesi dan portal). Rincian galat internal hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.Writer(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

func mapError(err error) (int, ErrorResponse, bool) {
	switch {
	case errors.Is(err, inboxrcl.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap profil di dalamnya. Menjawab 200
		// dengan daftar kosong akan terbaca sebagai "tidak ada pekerjaan".
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga antrean RCL milik Anda tidak " +
				"dapat dipisahkan dari antrean dokter lain. Masuk ulang lalu coba lagi.",
		}, true
	default:
		return 0, ErrorResponse{}, false
	}
}
