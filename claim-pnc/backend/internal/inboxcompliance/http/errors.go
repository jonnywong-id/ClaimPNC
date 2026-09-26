package inboxcompliancehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxcompliance"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail  = "validasi_gagal"
	CodeTabNotReady     = "tab_belum_tersedia"
	CodeMalformedBody   = "permintaan_tidak_terbaca"
	CodeClaimNotInQueue = "klaim_tidak_di_antrean"
	CodeCallerUnknown   = "profil_pemanggil_tidak_lengkap"
	CodeInternalError   = "galat_internal"
)

// errMalformedBody berarti badan permintaan tidak dapat diurai sebagai JSON.
//
// Ia galat milik lapisan transport, bukan domain — domain tidak tahu apa pun tentang bentuk
// kawatnya — sehingga ia tinggal di berkas ini, bukan di errors.go modul.
var errMalformedBody = errors.New("inboxcompliancehttp: badan permintaan tidak dapat dibaca")

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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya pemanggil
// dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang lain" — dua
// hal yang tidak dapat dibedakan hanya dari status 500.
func mapError(err error) (int, ErrorResponse, bool) {
	var (
		validation *inboxcompliance.ValidationError
		notReady   *inboxcompliance.TabNotReadyError
	)

	switch {
	case errors.Is(err, errMalformedBody):
		return http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedBody,
			Message: "Permintaan tidak dapat dibaca. Muat ulang halaman lalu coba lagi.",
		}, true

	case errors.Is(err, inboxcompliance.ErrClaimNotInQueue):
		// 409, bukan 404. Klaimnya boleh jadi ada dan sehat — yang tidak ada adalah
		// posisinya di antrean ini, dan keadaan itu berubah tanpa pengguna melakukan
		// apa pun. Menjawab 404 akan membuat petugas mencari klaimnya, padahal yang perlu
		// dilakukan hanyalah menyegarkan daftar.
		return http.StatusConflict, ErrorResponse{
			Code: CodeClaimNotInQueue,
			Message: "Klaim ini sudah tidak ada di antrean Compliance. Mungkin sudah " +
				"dikirim petugas lain. Segarkan daftar lalu periksa kembali.",
		}, true

	case errors.Is(err, inboxcompliance.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga pengiriman ini tidak dapat " +
				"dicatat atas nama siapa pun. Masuk ulang lalu coba lagi.",
		}, true

	case errors.As(err, &notReady):
		// 503, bukan 404 maupun 422. Tabnya ADA dan permintaannya benar — yang belum ada
		// adalah artefak dari pihak lain, dan keadaan itu akan berubah tanpa pengguna
		// melakukan apa pun. 404 akan membuat pengguna mengira tabnya tidak pernah ada;
		// 422 akan membuatnya mencari isian yang salah pada layar tanpa isian.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeTabNotReady,
			Message: "Tab " + notReady.Tab.Name + " belum dapat menampilkan data. " +
				notReady.Blocker,
		}, true

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
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
