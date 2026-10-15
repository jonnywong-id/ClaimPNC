package archivedokumenklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/archivedokumenklaim"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan
// dengan mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeNotFound       = "berkas_tidak_ditemukan"
	CodeAlreadySent    = "berkas_sudah_dikirim"
	CodeServiceFailed  = "layanan_arsip_gagal"
	CodeServiceAddress = "alamat_layanan_arsip_kosong"
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
	var validation *archivedokumenklaim.ValidationError

	switch {
	case errors.As(err, &validation):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di isiannya
		// (`10-API-STRATEGY.md` §5).
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, archivedokumenklaim.ErrCallerUnknown):
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga berkas tidak dapat dicatat " +
				"atas nama siapa pun. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, archivedokumenklaim.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Berkas arsip yang dimaksud sudah tidak ada. " +
				"Muat ulang daftarnya lalu coba lagi.",
		}, true

	case errors.Is(err, archivedokumenklaim.ErrAlreadySent):
		// 409, bukan 422: permintaannya sah, keadaannya yang sudah berubah. Berkas yang
		// sudah sampai ke sistem Arsip tidak boleh dikirim dua kali, dan pembedaan ini
		// yang membuat layar dapat menyarankan "muat ulang daftar" alih-alih "perbaiki
		// isian".
		return http.StatusConflict, ErrorResponse{
			Code: CodeAlreadySent,
			Message: "Berkas ini sudah pernah dikirim ke sistem Arsip. " +
				"Muat ulang daftarnya untuk melihat keadaan terbarunya.",
		}, true

	case errors.Is(err, archivedokumenklaim.ErrServiceAddress):
		// 503, bukan 500: sistemnya tidak rusak, satu prasyaratnya yang belum dipasang —
		// dan yang memasangnya DBA, bukan tim pengembang. Pesannya menyebutkan tepat apa
		// yang kurang supaya pelapornya tidak perlu menebak.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeServiceAddress,
			Message: "Alamat layanan Arsip belum terdaftar untuk portal ini. " +
				"Mintakan penambahan barisnya pada tabel alamat layanan kepada DBA.",
		}, true

	case errors.Is(err, archivedokumenklaim.ErrServiceFailed):
		return http.StatusBadGateway, ErrorResponse{
			Code: CodeServiceFailed,
			Message: "Sistem Arsip tidak dapat dihubungi atau menolak pengiriman. " +
				"Berkasnya TIDAK ditandai terkirim, jadi dapat dicoba lagi nanti.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
