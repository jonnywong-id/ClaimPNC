package komitehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/platform/money"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia kontrak API; yang berbahasa Inggris hanya nama
// konstantanya (`D-80`).
const (
	CodeValidationFailed = "validasi_gagal"
	CodeUnknownLine      = "lini_tidak_dikenal"
	CodeInvalidValue     = "nilai_klaim_cacat"
	CodeInternalError    = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri
// supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat auth, sehingga galat sesi yang lolos dari middleware tetap dijawab dengan kode
// yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya pun tidak
// mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log —
// rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, known := mapError(err)
		if !known {
			if fallback != nil {
				fallback(w, r, err)
				return
			}
			status, body = http.StatusInternalServerError, ErrorResponse{
				Code:    CodeInternalError,
				Message: "Terjadi kesalahan pada sistem.",
			}
		}
		if status >= http.StatusInternalServerError {
			logging.Dari(r.Context(), logger).Error("permintaan gagal",
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
	var validation *komite.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (`docs/Steering/10-API-STRATEGY.md` §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, komite.ErrUnknownBusinessLine):
		// 404, bukan 422: yang diminta memang tidak ada di master. Ia dibedakan dengan
		// sengaja dari "ada tetapi tidak ada jenjang yang cocok", yang justru dijawab
		// 200 beserta penanda NoApprovers — karena yang kedua adalah TEMUAN tentang
		// isi master, bukan kesalahan permintaan.
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeUnknownLine,
			Message: "Lini bisnis itu tidak ada di master ambang komite.",
		}, true

	case errors.Is(err, money.ErrFormat):
		// 400: bentuk permintaannya yang salah, bukan isinya. Nilai uang dikirim
		// sebagai teks desimal kanonik, dan yang bukan itu berarti klien membentuknya
		// dengan cara yang tidak disepakati.
		return http.StatusBadRequest, ErrorResponse{
			Code: CodeInvalidValue,
			Message: "Nilai klaim harus berupa angka, boleh dengan paling banyak dua " +
				"angka di belakang titik. Contoh: 50000001 atau 50000001.50.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
