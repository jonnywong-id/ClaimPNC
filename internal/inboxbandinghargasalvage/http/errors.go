package inboxbandinghargasalvagehttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya dengan
// nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
const (
	CodeValidationFail = "validasi_gagal"
	CodeCallerUnknown  = "profil_pemanggil_tidak_lengkap"
	CodeWriteNotBuilt  = "tindakan_belum_tersedia"
	CodeBadRequest     = "permintaan_tidak_terbaca"
	CodeAlreadyDecided = "banding_sudah_diputus"
	CodeDocumentGone   = "dokumen_tidak_ditemukan"
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
	var validation *inboxbandinghargasalvage.ValidationError

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

	case errors.Is(err, inboxbandinghargasalvage.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca, sehingga banding yang Anda putuskan " +
				"tidak dapat dipisahkan dari milik komite lain. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxbandinghargasalvage.ErrAlreadyDecided):
		// 409, bukan 404. Barangnya ADA — yang tidak ada adalah kesempatan memutuskannya
		// lagi. Ia pula jawaban atas klik ganda: penekanan kedua memang tidak mengubah
		// apa pun, dan pengguna berhak tahu itu.
		return http.StatusConflict, ErrorResponse{
			Code: CodeAlreadyDecided,
			Message: "Banding ini sudah diputus, atau bukan banding yang Anda tangani. " +
				"Muat ulang daftarnya untuk melihat keadaan terbarunya.",
		}, true

	case errors.Is(err, inboxbandinghargasalvage.ErrDocumentNotFound):
		// 404, dan di sini ia memang tepat: yang diminta sebuah BERKAS, dan jawabannya
		// "tidak ada berkas itu untuk Anda". Ia menutup dua keadaan sekaligus — dokumennya
		// tidak ada, atau ia milik banding komite lain — dan keduanya sengaja tidak
		// dibedakan, karena membedakannya memberi tahu penanya bahwa sebuah id nyata.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeDocumentGone,
			Message: "Dokumen ini tidak dapat dibuka. Ia mungkin sudah dihapus, sudah " +
				"ditandai ditolak, atau bukan bagian dari banding yang Anda tangani. " +
				"Muat ulang daftarnya untuk melihat keadaan terbarunya.",
		}, true

	case errors.Is(err, inboxbandinghargasalvage.ErrWriteNotAvailable):
		// 501, bukan 404. Tindakannya memang ada di layar lama; yang tidak ada adalah
		// penulisnya pada pemasangan ini. Menjawab "tidak ditemukan" terbaca sebagai
		// kerusakan, sementara yang dibutuhkan pengguna adalah tahu MENGAPA.
		//
		// Sejak tombol Approve/Reject dibangun, galat ini menyempit maknanya: ia hanya
		// muncul pada pemasangan yang penyimpanannya memang tidak menyediakan penulis —
		// misalnya sambungan baca-saja. Ia bukan lagi jawaban atas "belum dibangun".
		return http.StatusNotImplemented, ErrorResponse{
			Code: CodeWriteNotBuilt,
			Message: "Keputusan banding harga tidak dapat disimpan pada sambungan ini, " +
				"karena penyimpanannya dipasang untuk membaca saja. Hubungi administrator " +
				"aplikasi; keputusan Anda belum tercatat.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
