package masterpicteknikhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	CodeNotFound     = "pic_teknik_tidak_ditemukan"
	CodeValidationFailed      = "validasi_gagal"
	CodeAlreadyExists           = "id_operator_sudah_terdaftar"
	CodeDirectoryUnreachable  = "direktori_operator_tidak_terhubung"
	CodeMalformedRequest    = "permintaan_cacat"
	CodeInternalError      = "galat_internal"
)

// PenulisJSON menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri
// supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// PenulisGalat menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// TulisGalat memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke penulisCadangan — yang di cmd diisi
// penulis galat auth, sehingga galat sesi yang lolos dari middleware tetap dijawab
// dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya pun
// tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallbackWriter ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, recognized := mapError(err)
		if !recognized {
			if fallbackWriter != nil {
				fallbackWriter(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, ErrorResponse{
				Code:  CodeInternalError,
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

// petakanGalat menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini, supaya pemanggil dapat
// membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang lain".
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *masterpicteknik.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (docs/Steering/10-API-STRATEGY.md §5).
		detail := make([]ViolationDTO, 0, len(validation.Violation))
		for _, p := range validation.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:   CodeValidationFailed,
			Message:  "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail: detail,
		}, true

	case errors.Is(err, masterpicteknik.ErrAlreadyExists):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan saat ini — mungkin karena orang lain baru saja mendaftarkannya.
		return http.StatusConflict, ErrorResponse{
			Code:  CodeAlreadyExists,
			Message: "ID operator itu sudah terdaftar. Buka datanya lalu ubah di sana.",
		}, true

	case errors.Is(err, masterpicteknik.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:  CodeNotFound,
			Message: "PIC teknik tidak ditemukan.",
		}, true

	case errors.Is(err, masterpicteknik.ErrDirectoryUnreachable):
		// 503, bukan 500: ini bukan cacat aplikasi melainkan sumber luar yang sedang
		// tidak dapat dihubungi, dan tindak lanjutnya menunggu — bukan melapor.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:  CodeDirectoryUnreachable,
			Message: "Direktori operator sedang tidak dapat dihubungi. Coba beberapa saat lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
