package inboxpladlahttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend (`D-80`).
const (
	CodeValidationFail    = "validasi_gagal"
	CodeCallerUnknown     = "profil_pemanggil_tidak_lengkap"
	CodeNotAReinsurer     = "bukan_reasuradur_terdaftar"
	CodeWriteNotAvailable = "belum_tersedia"
	CodeInternalError     = "galat_internal"

	// Kode layar RINCIAN.
	CodeNotAClaimList        = "bukan_daftar_klaim"
	CodeAdviceKindUnknown    = "jenis_pemberitahuan_tidak_dikenal"
	CodeClaimNotFound        = "klaim_tidak_ditemukan"
	CodeDocumentNotFound     = "dokumen_tidak_ditemukan"
	CodeConversationNotFound = "percakapan_tidak_ditemukan"
	CodeConversationAnswered = "percakapan_sudah_dijawab"
)

// JSONWriter menuliskan badan respons.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat portal dan auth. Bila tidak ada cadangan, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.Writer(logger, writeJSON, fallback, mapAny, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapAny mencoba pemetaan galat umum lebih dulu, lalu pemetaan galat rincian.
func mapAny(err error) (int, ErrorResponse, bool) {
	if status, body, recognized := mapError(err); recognized {
		return status, body, true
	}
	return mapDetailError(err)
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxpladla.ValidationError

	switch {
	case errors.As(err, &validation):
		details := apierror.FieldErrors(validation.Violations)

		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFail,
			Message: "Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.",
			Details: details,
		}, true

	case errors.Is(err, inboxpladla.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		//
		// Di layar ini identitas BUKAN sekadar soal jejak: login pemanggil adalah
		// penyaring utama ketiga daftarnya, sehingga tanpa login tidak ada daftar yang
		// dapat disusun sama sekali.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Layar ini menampilkan klaim " +
				"menurut kode reasuradur Anda, sehingga tanpa identitas daftarnya " +
				"tidak dapat disusun. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxpladla.ErrCallerNotAReinsurer):
		// 403, dan pesannya menjelaskan APA yang terjadi — bukan daftar kosong.
		//
		// # Kenapa ini bukan 200 dengan daftar kosong
		//
		// Karena keduanya berarti hal yang berbeda, dan hanya satu yang dapat
		// ditindaklanjuti:
		//
		//	terdaftar, belum ada klaim   tunggu — memang belum ada pekerjaan
		//	tidak terdaftar              layar ini bukan untuk Anda, atau pendaftaran
		//	                             Anda belum lengkap — hubungi administrator
		//
		// Di Pega keduanya terlihat SAMA: layar kosong tanpa satu pun keterangan.
		// Petugas internal yang tersesat ke menu ini menyimpulkan sistemnya rusak, dan
		// reasuradur yang loginnya belum didaftarkan menunggu pekerjaan yang tidak akan
		// pernah muncul.
		//
		// 403, bukan 404: alamatnya ada dan permintaannya sah — pemanggilnya yang tidak
		// berhak atas isinya.
		return http.StatusForbidden, ErrorResponse{
			Code: CodeNotAReinsurer,
			Message: "Layar ini menampilkan klaim yang pemberitahuannya dikirimkan " +
				"kepada satu mitra reasuransi, dan login Anda belum terdaftar " +
				"sebagai mitra pada entitas ini. Bila Anda petugas internal, " +
				"pakailah menu \"Inbox PLA, DLA, Pre DLA\". Bila Anda mitra " +
				"reasuransi, hubungi administrator Claim PNC untuk melengkapi " +
				"pendaftaran login Anda.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
