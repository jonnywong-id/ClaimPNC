package inputacceptationhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeNotFound       = "tidak_ditemukan"
	CodeWriteNotOwned  = "belum_dapat_disimpan"
	CodeInternalError  = "galat_internal"
)

// JSONWriter menuliskan badan respons.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis galat
// portal dan auth. Bila tidak ada cadangan, atau cadangannya pun tidak mengenalinya, jawabannya
// 500 dengan pesan umum dan rinciannya hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.Writer(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inputacceptation.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		details := apierror.FieldErrors(validation.Violations)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inputacceptation.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga pembukaan akseptasi tidak " +
				"dapat dicatat. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inputacceptation.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Klaim treaty non-proporsional dengan nomor itu tidak ditemukan di " +
				"entitas yang sedang dipilih.",
		}, true

	case errors.Is(err, inputacceptation.ErrWriteNotOwned):
		// 409, bukan 403 maupun 501.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 501 akan menyatakan kemampuannya belum
		// dibangun — padahal sudah, seluruh jalurnya ada dan muatannya tervalidasi. Yang
		// sebenarnya terjadi adalah KONFLIK kepemilikan: tabelnya masih ditulis sistem lain,
		// dan itulah arti 409.
		return http.StatusConflict, ErrorResponse{
			Code: CodeWriteNotOwned,
			Message: "Akseptasi belum dapat disimpan dari sistem baru. Selama Pega dan " +
				"sistem baru berjalan berdampingan, tabel klaim hanya boleh ditulis satu " +
				"sistem, dan tabel itu masih dimiliki Pega. Perubahan Anda TIDAK tersimpan. " +
				"Lakukan akseptasi lewat Pega sampai kepemilikan tabelnya berpindah.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
