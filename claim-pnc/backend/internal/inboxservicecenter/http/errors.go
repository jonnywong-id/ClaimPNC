package inboxservicecenterhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya dengan
// nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeNotFound       = "klaim_tidak_ditemukan"
	CodeInternalError  = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari
// middleware tetap dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada
// cadangan, atau cadangannya pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log — rincian galat internal tidak pernah dikirim ke peramban.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.Writer(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxservicecenter.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		details := apierror.FieldErrors(validation.Violations)

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxservicecenter.ErrNotFound):
		// 404, dan pesannya sengaja TIDAK membedakan "tidak ada" dari "bukan milik Anda".
		// Membedakannya memberi tahu penanya bahwa sebuah ID nyata, dan ID di tabel ini
		// berurutan — cukup untuk memetakan isinya tanpa pernah membaca satu barisnya.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Klaim tidak ditemukan, atau bukan klaim yang Anda tangani. " +
				"Buka kembali dari daftar Inbox Service Center.",
		}, true

	case errors.Is(err, inboxservicecenter.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga klaim yang Anda tangani tidak " +
				"dapat dipisahkan dari milik petugas lain. Masuk ulang lalu coba lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
