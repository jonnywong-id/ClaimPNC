package inboxmanagerreceivepuclhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail    = "validasi_gagal"
	CodeCallerUnknown     = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotAvailable = "belum_tersedia"
	CodeDocumentNotFound  = "berkas_tidak_ditemukan"
	CodeReferenceRequired = "kunci_berkas_kosong"
	CodeInternalError     = "galat_internal"
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
	var validation *inboxmanagerreceivepucl.ValidationError

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

	case errors.Is(err, inboxmanagerreceivepucl.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Layar ini menampilkan pekerjaan " +
				"seluruh petugas, sehingga pembukaannya wajib tercatat atas nama " +
				"seseorang. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxmanagerreceivepucl.ErrReferenceRequired):
		// 422, bukan 404: alamatnya ada, yang kurang adalah kunci berkas di dalamnya.
		// Menjawab 404 akan menyatakan berkasnya tidak ada — padahal belum ada berkas yang
		// dicari sama sekali.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeReferenceRequired,
			Message: "Berkas penerimaan dokumen yang dibuka tidak disebutkan.",
		}, true

	case errors.Is(err, inboxmanagerreceivepucl.ErrDocumentNotFound):
		// 404, dan ia BENAR-BENAR 404: yang diminta adalah satu sumber daya bernama, dan
		// sumber daya itu tidak ada.
		//
		// Pesannya menyebut kemungkinan sebabnya, bukan hanya menyatakan tidak ada. Berkas
		// yang penugasannya sudah selesai tetap dapat dibuka di sini — kueri layar kerja
		// memang tidak menggabung tabel penugasan — sehingga "tidak ditemukan" hampir selalu
		// berarti kuncinya yang keliru, atau berkasnya milik portal lain.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeDocumentNotFound,
			Message: "Berkas penerimaan dokumen ini tidak ditemukan. Periksa apakah " +
				"Anda sudah berada di portal yang benar — setiap entitas punya basis " +
				"datanya sendiri, dan berkas milik entitas lain tidak terbaca dari sini.",
		}, true

	case errors.Is(err, inboxmanagerreceivepucl.ErrWriteNotAvailable):
		// 501, bukan 403 maupun 404.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 404 akan menyatakan alamatnya tidak ada,
		// sehingga tombolnya terbaca sebagai kerusakan. 501 menyatakan yang sebenarnya:
		// alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		return http.StatusNotImplemented, ErrorResponse{
			Code: CodeWriteNotAvailable,
			Message: "Tindakan atas berkas dan klaim di layar ini belum tersedia di " +
				"sistem baru. Selama Pega dan sistem baru berjalan berdampingan, tabel " +
				"objek kerja dan tabel penugasan hanya boleh ditulis satu sistem, dan " +
				"keduanya masih dimiliki Pega. Kerjakan lewat Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
