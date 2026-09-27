package inboxkomunikasicabanghttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Kode galat yang dikenali klien. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya tetap berbahasa Indonesia karena ia KONTRAK yang dibaca frontend, sama halnya
// dengan nama field JSON (`D-80`); yang berbahasa Inggris hanya nama konstantanya.
//
// CATATAN. CodeWriteNotAvailable ("belum_tersedia") DIHAPUS pada 2026-09-24 bersama
// ErrWriteNotAvailable — tidak ada lagi tindakan di layar ini yang ditolak dengan alasan itu.
const (
	CodeValidationFail       = "validasi_gagal"
	CodeCallerUnknown        = "profil_pemanggil_tidak_lengkap"
	CodeBranchUnreadable     = "sumber_cabang_tidak_terbaca"
	CodeConversationNotFound = "komunikasi_tidak_ditemukan"
	CodeInternalError        = "galat_internal"
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
	var validation *inboxkomunikasicabang.ValidationError

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

	case errors.Is(err, inboxkomunikasicabang.ErrCallerUnknown):
		// 409, bukan 401: sesinya sah — yang tidak lengkap adalah profil di dalamnya.
		// Menjawab 401 akan membuat layar melempar pengguna ke halaman masuk, lalu
		// mengembalikannya ke galat yang sama.
		//
		// Di layar ini akibatnya lebih berat daripada kehilangan jejak: batas datanya
		// diturunkan DARI login, sehingga tanpa login tidak ada batas yang dapat dipakai.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerUnknown,
			Message: "Identitas Anda tidak terbaca. Layar ini menampilkan percakapan " +
				"menurut cabang Anda, dan cabang itu diturunkan dari identitas yang " +
				"dipakai masuk. Masuk ulang lalu coba lagi.",
		}, true

	case errors.Is(err, inboxkomunikasicabang.ErrBranchUnreadable):
		// 503, bukan 403.
		//
		// Yang gagal BUKAN kewenangan pemanggil melainkan sumber datanya, dan ia menimpa
		// semua orang sekaligus — bukan satu orang. `503` juga menyatakan keadaannya
		// SEMENTARA, sehingga mencoba lagi memang masuk akal.
		//
		// Pembedaannya dari "petugas tidak terdaftar" adalah inti penanganan galat modul
		// ini: yang kedua BUKAN galat sama sekali dan tetap dilayani sebagai kantor pusat
		// (`P-5`). Menyamakan keduanya berarti petugas melihat percakapan kantor pusat
		// ketika yang sebenarnya terjadi adalah DB Link sedang mati.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeBranchUnreadable,
			Message: "Sumber data cabang sedang tidak dapat dibaca, sehingga daftar tidak " +
				"dapat ditampilkan tanpa risiko menampilkan percakapan cabang lain. " +
				"Coba lagi beberapa saat lagi; bila berulang, laporkan ke tim infrastruktur.",
		}, true

	case errors.Is(err, inboxkomunikasicabang.ErrConversationNotFound):
		// 404, dan pesannya menyebut TIGA kemungkinan sebab.
		//
		// Dua yang pertama menghasilkan jawaban yang sama dengan sengaja — jawaban yang
		// membedakan "tidak ada" dari "milik cabang lain" akan menyatakan bahwa nomor itu ada
		// di tempat lain, dan itu keterangan yang tidak berhak diterima pemanggilnya.
		//
		// Portal ikut disebut karena kunci yang benar pada portal yang SALAH menghasilkan
		// keadaan yang sama, dan itu tidak menghasilkan satu pun tanda lain (`R-20`).
		//
		// Sebab KETIGA ditambahkan 2026-09-24 bersama aksi tulis: percakapan yang SUDAH
		// ditutup juga menjawab begini, karena kedua pernyataan tulisnya menyaring
		// `CASEID = 'CABANG'`. Ia bukan sebab yang sama, tetapi jawabannya tepat — baris yang
		// sudah ditutup memang hilang dari kedua tab, sehingga layar yang masih menampilkannya
		// sedang usang. Menyebutnya membuat pengguna tahu harus menyegarkan, bukan menduga
		// datanya rusak.
		return http.StatusNotFound, ErrorResponse{
			Code: CodeConversationNotFound,
			Message: "Percakapan tidak ditemukan. Ia mungkin sudah ditutup orang lain, milik " +
				"cabang lain, atau milik entitas lain — segarkan daftar, lalu periksa pilihan " +
				"portal di bilah atas.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
