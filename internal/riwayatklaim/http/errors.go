package riwayatklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/riwayatklaim"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeNotRegistered  = "proteksi_belum_terdaftar"
	CodeQuotaExhausted = "jatah_pencarian_habis"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
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
// galat portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari
// middleware tetap dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada
// cadangan, atau cadangannya pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log — rincian galat internal tidak pernah dikirim ke peramban.
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
			logger.Error(
				"permintaan gagal",
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
	var validation *riwayatklaim.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di isiannya
		// (`10-API-STRATEGY.md` §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))

		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{
				Field:   v.Field,
				Message: v.Message,
			})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu cari lagi.",
			Details: details,
		}, true

	case errors.Is(err, riwayatklaim.ErrNotRegistered):
		// 403, bukan 404: layarnya ada, pemanggilnya yang belum berhak membukanya.
		// Pesannya menyebut apa yang harus dilakukan, mengikuti sistem lama yang
		// berbunyi "Input Data Proteksi Terlebih Dahulu".
		return http.StatusForbidden, ErrorResponse{
			Code: CodeNotRegistered,
			Message: "Anda belum terdaftar di Master Proteksi Data untuk layar ini. " +
				"Hubungi pengelola proteksi data sebelum membukanya.",
		}, true

	case errors.Is(err, riwayatklaim.ErrQuotaExhausted):
		// 409, bukan 403: haknya ada, jatahnya yang habis — dan jatah dapat ditambah
		// tanpa mengubah hak. Membedakannya membuat layar dapat menyarankan tindakan
		// yang berbeda.
		return http.StatusConflict, ErrorResponse{
			Code: CodeQuotaExhausted,
			Message: "Jatah pencarian data Anda sudah habis. " +
				"Minta penambahan di Master Proteksi Data.",
		}, true

	case errors.Is(err, riwayatklaim.ErrCallerUnknown):
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga gerbang proteksi data " +
				"tidak dapat diperiksa. Masuk ulang lalu coba lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
