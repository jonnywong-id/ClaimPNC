package masterxolhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterxol"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	ErrCodeNotFound         = "xol_tidak_ditemukan"
	ErrCodeLayerNotFound    = "layer_xol_tidak_ditemukan"
	ErrCodeValidationFailed = "validasi_gagal"
	ErrCodeIDTaken          = "id_xol_sudah_dipakai"
	ErrCodeBadRequest       = "permintaan_cacat"
	ErrCodeInternal         = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke penulis cadangan — yang di cmd diisi
// rantai penulis galat sadar-portal, sehingga galat portal dan galat sesi tetap dijawab
// dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya pun
// tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log —
// rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallbackWriter ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, recognized := mapError(err)
		if !recognized {
			if fallbackWriter != nil {
				fallbackWriter(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, ErrorResponse{
				Code:    ErrCodeInternal,
				Message: "Terjadi kesalahan pada sistem.",
			}
		}
		if status >= http.StatusInternalServerError {
			logging.From(r.Context(), logger).Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
		}
		writeJSON(w, r, status, body)
	}
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *masterxol.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di bagiannya
		// (docs/Steering/10-API-STRATEGY.md §5).
		detail := make([]ViolationDTO, 0, len(validation.Violation))
		for _, v := range validation.Violation {
			detail = append(detail, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    ErrCodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterxol.ErrIDTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan saat ini. Di modul ini sebabnya khas — nomor dibentuk `max+1`, dan
		// dua penyimpanan yang benar-benar bersamaan dapat memperebutkan nomor yang sama.
		// Menyimpan sekali lagi hampir pasti berhasil.
		return http.StatusConflict, ErrorResponse{
			Code:    ErrCodeIDTaken,
			Message: "Nomor yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, masterxol.ErrLayerNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    ErrCodeLayerNotFound,
			Message: "Layer XOL tidak ditemukan pada master ini.",
		}, true

	case errors.Is(err, masterxol.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    ErrCodeNotFound,
			Message: "Master XOL tidak ditemukan.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
