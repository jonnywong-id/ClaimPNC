package pelaporanklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeNotFound       = "laporan_klaim_tidak_ditemukan"
	CodeValidationFail = "validasi_gagal"
	CodeAlreadyMoved   = "laporan_sudah_ditransfer"
	CodeAlreadyRegd    = "laporan_sudah_diregistrasi"
	CodeNumberTaken    = "nomor_laporan_sudah_dipakai"
	CodeBadRequest     = "permintaan_cacat"
	CodeInternalError  = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
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
	var validation *pelaporanklaim.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (docs/Steering/10-API-STRATEGY.md §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Details: details,
		}, true

	case errors.Is(err, pelaporanklaim.ErrAlreadyTransferred):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan — biasanya karena orang lain sudah mentransfernya lebih dulu.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeAlreadyMoved,
			Message: "Laporan ini sudah ditransfer ke ASM pusat.",
		}, true

	case errors.Is(err, pelaporanklaim.ErrAlreadyRegistered):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeAlreadyRegd,
			Message: "Laporan ini sudah diregistrasi menjadi klaim, sehingga tidak dapat diubah lagi.",
		}, true

	case errors.Is(err, pelaporanklaim.ErrNumberTaken):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNumberTaken,
			Message: "Nomor laporan yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, pelaporanklaim.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Laporan klaim tidak ditemukan.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
