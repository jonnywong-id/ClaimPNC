package inboxsalvagehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxsalvage"
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
	CodeRowNotFound       = "pengajuan_tidak_ditemukan"
	CodeUploadInvalid     = "berkas_tidak_terbaca"
	CodeInternalError     = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis galat
// portal dan auth, sehingga galat sesi maupun galat portal yang lolos dari middleware tetap
// dijawab dengan kode yang sudah dikenal frontend. Bila tidak ada cadangan, atau cadangannya
// pun tidak mengenalinya, jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log —
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

		if status >= http.StatusInternalServerError && logger != nil {
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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya pemanggil
// dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang lain" — dua
// hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxsalvage.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		//
		// Di layar ini ia menyampaikan SELURUH isian form Tambah yang belum benar
		// sekaligus — bukan satu, lalu satu lagi (`P-5`). Form-nya punya tujuh belas
		// isian, dan menyampaikannya satu per satu akan menyiksa pengguna.
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxsalvage.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		//
		// Di layar ini identitas bukan sekadar soal jejak: daftar "Request Balai Lelang"
		// menyaring menurut PIC, dan tanpa identitas ia akan menampilkan pengajuan milik
		// petugas lain.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Daftar \"Request Balai Lelang\" " +
				"menampilkan pengajuan milik Anda sendiri, sehingga tanpa identitas " +
				"daftarnya tidak dapat disusun. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxsalvage.ErrRowNotFound):
		// 404, dan pesannya menyebut PORTAL.
		//
		// Penyebab paling mungkin bukan pengajuan yang benar-benar tidak ada, melainkan
		// ID yang benar dibuka pada portal yang salah — dan itu keadaan yang tidak
		// menghasilkan satu pun tanda lain (`R-20`).
		return http.StatusNotFound, ErrorResponse{
			Code: CodeRowNotFound,
			Message: "Pengajuan salvage tidak ditemukan pada entitas yang sedang " +
				"dipilih. Periksa pilihan portal di bilah atas — pengajuan milik " +
				"entitas lain tidak dapat dibuka dari sini.",
		}, true

	case errors.Is(err, inboxsalvage.ErrUploadEmpty),
		errors.Is(err, inboxsalvage.ErrUploadColumnMissing),
		errors.Is(err, inboxsalvage.ErrUploadTooManyRows):
		// 422, dan pesannya menyebut apa yang harus diperbaiki di BERKASNYA.
		//
		// Ketiganya dipisah dari galat validasi isian karena yang salah bukan isian di
		// layar melainkan berkas yang diunggah, dan yang harus diperbaiki pengguna ada di
		// luar layar ini.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUploadInvalid,
			Message: uploadMessage(err),
		}, true

	case errors.Is(err, inboxsalvage.ErrWriteNotAvailable):
		// 501, bukan 403 maupun 404.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 404 akan menyatakan alamatnya tidak ada,
		// sehingga tombolnya terbaca sebagai kerusakan. 501 menyatakan yang sebenarnya:
		// alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		return http.StatusNotImplemented, ErrorResponse{
			Code: CodeWriteNotAvailable,
			Message: "Tindakan \"Approve\", \"Reject\", dan \"Send To BalaiLelang\" " +
				"belum tersedia di sistem baru. Ketiganya mengubah status pengajuan, " +
				"dan dua di antaranya menembak balai lelang serta penyimpanan berkas " +
				"di luar aplikasi ini. Kerjakan lewat Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}

// uploadMessage menyusun kalimat yang menyebut apa yang salah pada berkasnya.
//
// Ketiganya dipisah karena tindakan pengguna berbeda: yang pertama menuntut ia memeriksa
// berkasnya tidak kosong, yang kedua memeriksa judul kolomnya, yang ketiga memecah
// berkasnya. Pesan yang sama untuk ketiganya akan membuat pengguna menebak.
func uploadMessage(err error) string {
	switch {
	case errors.Is(err, inboxsalvage.ErrUploadEmpty):
		return "Berkas yang diunggah kosong. Baris pertamanya harus berisi judul kolom."

	case errors.Is(err, inboxsalvage.ErrUploadColumnMissing):
		return "Berkas yang diunggah tidak memuat kolom \"Item\". Pastikan baris " +
			"pertamanya berisi judul kolom, dan pemisah antarkolomnya KOMA — berkas " +
			"yang disimpan Excel dengan setelan Indonesia memakai titik koma."

	default:
		return "Berkas yang diunggah memuat terlalu banyak baris untuk satu pengajuan. " +
			"Pecah menjadi beberapa berkas."
	}
}
