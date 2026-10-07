package masterstatushttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
const (
	ErrCodeNotFound         = "status_klaim_tidak_ditemukan"
	ErrCodeValidationFailed = "validasi_gagal"
	ErrCodeLabelTaken       = "label_status_sudah_dipakai"
	ErrCodeCodeTaken        = "kode_status_sudah_dipakai"
	ErrCodeBadRequest       = "permintaan_cacat"
	ErrCodeInternal         = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri
// supaya seluruh modul menulis respons dengan cara yang sama, termasuk header
// Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke penulisCadangan — yang di cmd diisi
// penulis galat auth, sehingga galat sesi yang lolos dari middleware tetap dijawab
// dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya
// pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk
// log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallbackWriter ErrorWriter) ErrorWriter {
	return apierror.ContextWriter(logger, writeJSON, fallbackWriter, mapError, ErrorResponse{
		Code:    ErrCodeInternal,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validasi *masterstatus.ValidationError

	switch {
	case errors.As(err, &validasi):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (docs/Steering/10-API-STRATEGY.md §5).
		detail := apierror.FieldErrors(validasi.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    ErrCodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterstatus.ErrLabelTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan
		// penyimpanan saat ini — mungkin karena orang lain baru saja memakai nama itu.
		return http.StatusConflict, ErrorResponse{
			Code:    ErrCodeLabelTaken,
			Message: "Status dengan nama itu sudah ada. Pakai nama lain.",
		}, true

	case errors.Is(err, masterstatus.ErrCodeTaken):
		return http.StatusConflict, ErrorResponse{
			Code:    ErrCodeCodeTaken,
			Message: "Kode status yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, masterstatus.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    ErrCodeNotFound,
			Message: "Status klaim tidak ditemukan.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
