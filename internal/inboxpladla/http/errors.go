package inboxpladlahttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxpladla"
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
	CodeAdviceKindUnknown    = "jenis_pemberitahuan_tidak_dikenal"
	CodeClaimNotFound        = "klaim_tidak_ditemukan"
	CodeDocumentNotFound     = "dokumen_tidak_ditemukan"
	CodeConversationNotFound = "percakapan_tidak_ditemukan"
	CodeConversationAnswered = "percakapan_sudah_dijawab"
)

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat modul ini menjadi respons HTTP.
//
// Galat yang BUKAN milik modul ini diteruskan ke fallback — yang di cmd diisi penulis
// galat portal dan auth. Bila tidak ada cadangan, jawabannya 500 dengan pesan umum dan
// rinciannya hanya masuk log.
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status, body, recognized := mapError(err)

		// Galat layar rincian dicoba SESUDAHNYA, bukan digabung ke satu switch raksasa.
		//
		// Urutannya penting satu kali: `NotAvailableError` membungkus
		// ErrWriteNotAvailable lewat Unwrap, sehingga mapError akan menangkapnya lebih
		// dulu dan menjawab kalimat umum. Itulah sebabnya mapError TIDAK lagi memeriksa
		// sentinel itu — pemeriksaannya dipindahkan seluruhnya ke mapDetailError, yang
		// menjawab alasan per tombol.
		if !recognized {
			status, body, recognized = mapDetailError(err)
		}

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
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxpladla.ValidationError

	switch {
	case errors.As(err, &validation):
		details := make([]ViolationDTO, 0, len(validation.Violations))
		for _, v := range validation.Violations {
			details = append(details, ViolationDTO{Field: v.Field, Message: v.Message})
		}

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
