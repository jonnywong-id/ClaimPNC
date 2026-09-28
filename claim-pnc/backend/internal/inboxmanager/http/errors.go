package inboxmanagerhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxmanager"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya dengan
// nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeTabNotAllowed  = "tab_bukan_hak_anda"
	CodeNotDecidable   = "antrean_tidak_dapat_diputuskan"
	CodeApproveBlocked = "persetujuan_sedang_ditahan"
	CodeSourceDown     = "sumber_antrean_tidak_terbaca"
	CodeInternalError  = "galat_internal"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis galat
// portal dan auth. Bila tidak ada cadangan, atau cadangannya pun tidak mengenalinya,
// jawabannya 500 dengan pesan umum dan rinciannya hanya masuk log.
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
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxmanager.ValidationError

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
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxmanager.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Layar ini menuliskan keputusan atas " +
				"nama seseorang, sehingga ia wajib tercatat. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxmanager.ErrTabNotAllowed):
		return http.StatusForbidden, ErrorResponse{
			Code: CodeTabNotAllowed,
			Message: "Tab itu ada, tetapi bukan hak lini bisnis Anda. Satu tab pada layar " +
				"ini dibatasi lini bisnis, mengikuti layar lamanya.",
		}, true

	case errors.Is(err, inboxmanager.ErrQueueNotDecidable):
		// 409: permintaannya sah bentuknya, tetapi ditujukan ke tab yang memang bukan
		// antrean. Ia hampir selalu berarti permintaan disusun tangan, bukan datang dari
		// layar — layar tidak menggambar tombol keputusan di tab dashboard.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNotDecidable,
			Message: "Tab itu bukan antrean persetujuan, jadi tidak ada yang dapat diputuskan.",
		}, true

	case errors.Is(err, inboxmanager.ErrApproveBlocked):
		// 409 pula, tetapi dengan kode dan pesan yang BERBEDA — dan pembedaan itu yang
		// membuat layar dapat menampilkan alasannya alih-alih menyebut permintaan salah.
		return http.StatusConflict, ErrorResponse{
			Code: CodeApproveBlocked,
			Message: "Persetujuan pada antrean itu sedang ditahan. Penolakan tetap dapat " +
				"dikerjakan. Alasan lengkapnya ada di keterangan antrean tersebut.",
		}, true

	case errors.Is(err, inboxmanager.ErrSourceUnavailable):
		// 503, bukan 500: aplikasinya tidak rusak — objek yang dibacanya di basis data
		// yang sedang tidak sah. Statusnya berbeda supaya pemantauan tidak mencampurnya
		// dengan kegagalan tak terduga, dan supaya layar dapat membedakan "menunggu
		// perbaikan" dari "rusak".
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeSourceDown,
			Message: "Antrean itu belum dapat ditampilkan: objek sumbernya di basis data " +
				"sedang tidak dapat dibaca. Ini BUKAN kerusakan aplikasi dan tidak dapat " +
				"diperbaiki dengan mencoba lagi. Laporkan ke tim teknis — nama objeknya " +
				"tercatat di log server.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
