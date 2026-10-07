package casestudyclaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan — teks
// dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend (`D-80`); yang
// berbahasa Inggris hanyalah nama konstantanya.
const (
	CodeBadRequest    = "permintaan_cacat"
	CodeValidation    = "validasi_gagal"
	CodeNotFound      = "tidak_ditemukan"
	CodeInternalError = "galat_internal"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Field menyebut isian mana yang salah, bila galatnya menyangkut satu isian.
	//
	// Kosong berarti galatnya tidak menunjuk isian tertentu. Layar memakainya untuk
	// menyorot isian yang bersangkutan alih-alih menampilkan satu pesan di atas formulir
	// yang harus dicocokkan sendiri oleh pengguna.
	Field string `json:"isian,omitempty"`
}

// JSONWriter menuliskan badan respons.
//
// Dipasok dari luar supaya seluruh modul menulis respons dengan cara yang sama, termasuk
// header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat menjadi respons HTTP.
//
// # Pembedaan 422 dan 404 disengaja
//
// `10-API-STRATEGY.md` §5 menetapkan `422` untuk permintaan yang bentuknya benar tetapi
// melanggar aturan bisnis, dan `404` untuk yang tidak ditemukan. Keduanya ditangani
// pengguna dengan cara yang berbeda: yang pertama menuntut ia memperbaiki isian, yang
// kedua menuntut ia memuat ulang daftar.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban; ia hanya masuk log. Membocorkan
// struktur basis data atau jejak tumpukan ke klien adalah celah keamanan
// (`11-CROSSCUTTING.md` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		switch {
		case errors.Is(err, casestudyclaim.ErrPeriodRequired):
			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:  CodeValidation,
				Field: "dari",
				Message: "Periode Awal dan Akhir wajib diisi. " +
					"Tanpa keduanya, layar lama pun tidak menampilkan satu baris pun.",
			})

		case errors.Is(err, casestudyclaim.ErrPeriodReversed):
			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:    CodeValidation,
				Field:   "sampai",
				Message: "Tahun pada Akhir tidak boleh lebih awal daripada tahun pada Awal.",
			})

		case errors.Is(err, casestudyclaim.ErrRemarkTooLong):
			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:  CodeValidation,
				Field: "catatan",
				Message: "Catatan terlalu panjang. " +
					"Batasnya 2.000 karakter.",
			})

		case errors.Is(err, casestudyclaim.ErrRemarkRejectedByColumn):
			// Basis data menolaknya meski lolos pemeriksaan aplikasi — kolomnya lebih
			// sempit daripada dugaan kami (`R-08`). Pengguna tetap diberi tahu bahwa yang
			// salah adalah PANJANGNYA, bukan sistemnya; rincian galat Oracle-nya hanya
			// masuk log.
			logging.From(r.Context(), logger).Warn(
				"catatan telaah ditolak basis data karena melebihi lebar kolom",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:  CodeValidation,
				Field: "catatan",
				Message: "Catatan terlalu panjang untuk kolomnya. " +
					"Persingkat, lalu simpan lagi.",
			})

		case errors.Is(err, casestudyclaim.ErrClaimRequired):
			writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeBadRequest,
				Field:   "nomor_klaim",
				Message: "Nomor klaim tidak disebutkan.",
			})

		case errors.Is(err, casestudyclaim.ErrClaimNotFound):
			// 404, dan pesannya menyuruh memuat ulang — bukan menghubungi tim teknis.
			// Baris memang dapat hilang di antara saat daftar dibaca dan saat Save
			// ditekan, dan itu keadaan yang sah.
			writeJSON(w, r, http.StatusNotFound, ErrorResponse{
				Code: CodeNotFound,
				Message: "Klaim ini sudah tidak ada di daftar. " +
					"Muat ulang daftarnya, lalu coba lagi.",
			})

		case errors.Is(err, casestudyclaim.ErrUnknownBusiness),
			errors.Is(err, casestudyclaim.ErrUnknownStatus):
			writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
				Code:    CodeBadRequest,
				Message: "Pilihan penyaring tidak dikenal.",
			})

		default:
			if fallback != nil {
				// Galat yang bukan milik modul ini diteruskan — di cmd diisi penulis galat
				// yang mengenali galat sesi dan galat portal, sehingga keduanya tetap
				// dijawab dengan kode yang sudah dikenal frontend.
				fallback(w, r, err)
				return
			}
			logging.From(r.Context(), logger).Error("permintaan gagal",
				slog.String("jalur", r.URL.Path),
				slog.String("galat", err.Error()),
			)
			writeJSON(w, r, http.StatusInternalServerError, ErrorResponse{
				Code:    CodeInternalError,
				Message: "Terjadi kesalahan pada sistem.",
			})
		}
	}
}

// writeBadRequest menjawab permintaan yang cacat bentuknya.
//
// Dipisahkan dari WriteError karena ia bukan kegagalan sistem melainkan kesalahan klien,
// dan pesannya boleh menyebutkan apa yang salah — tidak ada rincian internal di dalamnya.
func writeBadRequest(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request, field, message string) {
	writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Field:   field,
		Message: message,
	})
}
