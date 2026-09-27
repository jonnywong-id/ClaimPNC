package inboxpladlapredlahttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxpladlapredla"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang sudah dibaca frontend, sama
// halnya dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail    = "validasi_gagal"
	CodeCallerUnknown     = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotAvailable = "belum_tersedia"
	CodeRowNotFound       = "klaim_tidak_ditemukan"
	CodeNoDocumentGrid    = "rincian_tidak_ada_di_daftar_ini"
	CodeInternalError     = "galat_internal"
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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang
// lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var (
		validation   *inboxpladlapredla.ValidationError
		notAvailable *inboxpladlapredla.NotAvailableError
	)

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan.
		// Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah kesalahan
		// pengguna yang harus ditandai di isiannya (`10-API-STRATEGY.md` §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Penyaring belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxpladlapredla.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		//
		// Di layar ini identitas tidak menyaring apa pun; yang membutuhkannya adalah
		// jejak. Pesannya karena itu menyebut alasan yang sebenarnya, bukan alasan yang
		// terdengar masuk akal.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Setiap pembukaan antrean ini " +
				"dicatat — barisnya memuat nama tertanggung dan nomor polis — sehingga " +
				"tanpa identitas ia tidak dapat dibuka. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxpladlapredla.ErrRowNotFound):
		// 404, dan pesannya menyebut PORTAL.
		//
		// Penyebab paling mungkin bukan klaim yang benar-benar tidak ada, melainkan kunci
		// yang benar dibuka pada portal yang salah — dan itu keadaan yang tidak
		// menghasilkan satu pun tanda lain (`R-20`).
		return http.StatusNotFound, ErrorResponse{
			Code: CodeRowNotFound,
			Message: "Klaim tidak ditemukan pada entitas yang sedang dipilih. Periksa " +
				"pilihan portal di bilah atas — klaim milik entitas lain tidak dapat " +
				"dibuka dari sini.",
		}, true

	case errors.Is(err, inboxpladlapredla.ErrDocumentsNotOnTab):
		// 404, bukan 422.
		//
		// Yang salah bukan isian pengguna melainkan alamat yang diminta: daftar Pre DLA
		// memang tidak punya grid rincian. Pesannya menyebut apa yang ADA sebagai
		// gantinya, supaya pengguna tidak menyimpulkan klaimnya yang bermasalah.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNoDocumentGrid,
			Message: "Daftar Pre DLA tidak punya grid rincian — di Pega pun tidak. " +
				"Rincian Pre-DLA dibuka lewat \"Print Pre DLA\", yang belum tersedia " +
				"di sistem baru.",
		}, true

	case errors.As(err, &notAvailable):
		// 501, bukan 403 maupun 404.
		//
		// 403 akan menyatakan pengguna tidak berwenang — padahal ia berwenang, dan
		// wewenangnya bukan yang menghalangi. 404 akan menyatakan alamatnya tidak ada,
		// sehingga tombolnya terbaca sebagai kerusakan. 501 menyatakan yang sebenarnya:
		// alamatnya ada, permintaannya sah, kemampuannya yang belum dibangun.
		//
		// Pesannya diambil dari TINDAKANNYA, bukan satu kalimat untuk ketiga tombol.
		// Pengguna yang menekan "Print Pre DLA" tidak perlu membaca penjelasan tentang
		// email reasuradur.
		return http.StatusNotImplemented, ErrorResponse{
			Code:    CodeWriteNotAvailable,
			Message: notAvailable.Reason(),
		}, true

	case errors.Is(err, inboxpladlapredla.ErrWriteNotAvailable):
		// Sentinel tanpa tindakan — jaring pengaman bagi pemanggil yang mengembalikannya
		// langsung. Jalur di atas yang dipakai tombol.
		return http.StatusNotImplemented, ErrorResponse{
			Code:    CodeWriteNotAvailable,
			Message: "Tindakan ini belum tersedia di sistem baru. Kerjakan lewat Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
