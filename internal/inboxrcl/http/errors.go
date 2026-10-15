package inboxrclhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxrcl"
)

// Kode galat yang dikenali klien — KONTRAK, sehingga berbahasa Indonesia (`D-80`). Nilainya
// sama dengan Inbox Analyst Doctor supaya frontend menangani keadaan yang sama dengan cara
// yang sama.
const (
	CodeCallerUnknown = "profil_pemanggil_tidak_lengkap"
	CodeInternalError = "galat_internal"
	CodeClaimNotFound = "klaim_tidak_ditemukan"

	CodeUnreadableBody      = "badan_tidak_terbaca"
	CodeUnknownDecision     = "keputusan_tidak_dikenal"
	CodeDecisionNotAllowed  = "keputusan_tidak_berlaku"
	CodeTechnicalPICUnknown = "pic_teknik_tidak_diketahui"
)

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP; galat yang bukan milik modul
// ini diteruskan ke fallback (galat sesi dan portal). Rincian galat internal hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, recognized := mapError(err)

		if !recognized {
			if fallback != nil {
				fallback(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, ErrorResponse{
				Code:    CodeInternalError,
				Message: "Terjadi kesalahan pada sistem.",
			}
		}

		if status >= http.StatusInternalServerError && logger != nil {
			logger.Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}

		writeJSON(w, r, status, body)
	}
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
	case errors.Is(err, inboxrcl.ErrClaimNotFound):
		// 404 — klaim tidak ada di antrean RCL pemanggil. Tidak dibedakan dari "milik orang
		// lain" supaya keberadaan klaim orang lain tidak bocor.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeClaimNotFound,
			Message: "Klaim ini tidak ada di antrean RCL Anda.",
		}, true
	case errors.Is(err, inboxrcl.ErrUnknownDecision):
		// 400 — bentuk permintaan salah (cacat klien), bukan pelanggaran aturan.
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeUnknownDecision,
			Message: "Keputusan yang dikirim tidak dikenal.",
		}, true
	case errors.Is(err, inboxrcl.ErrDecisionNotAllowed):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeDecisionNotAllowed,
			Message: "Tombol ini tidak berlaku untuk klaim ini. Muat ulang layar lalu coba lagi.",
		}, true
	case errors.Is(err, inboxrcl.ErrTechnicalPICUnknown):
		return http.StatusConflict, ErrorResponse{
			Code: CodeTechnicalPICUnknown,
			Message: "Klaim tidak dapat dikembalikan ke analis karena PIC Teknik klaim ini " +
				"belum tercatat.",
		}, true
	default:
		return 0, ErrorResponse{}, false
	}
}
