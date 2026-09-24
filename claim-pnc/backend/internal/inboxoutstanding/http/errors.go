package inboxoutstandinghttp

import (
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan — teks
// dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend (`D-80`); yang
// berbahasa Inggris hanyalah nama konstantanya.
//
// Daftarnya pendek karena modul ini hanya MEMBACA: tidak ada isian yang dapat melanggar
// aturan bisnis, dan tidak ada keadaan penyimpanan yang dapat berkonflik.
const (
	CodeBadRequest    = "permintaan_cacat"
	CodeInternalError = "galat_internal"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// JSONWriter menuliskan badan respons.
//
// Dipasok dari luar supaya seluruh modul menulis respons dengan cara yang sama, termasuk
// header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat menjadi respons HTTP.
//
// Modul ini tidak punya galat domainnya sendiri, sehingga seluruh galat yang sampai ke
// sini adalah kegagalan teknis atau galat milik modul lain. Yang pertama dijawab 500
// dengan pesan umum; yang kedua diteruskan ke fallback — di cmd diisi penulis galat auth,
// sehingga galat sesi yang lolos dari middleware tetap dijawab dengan kode yang sudah
// dikenal frontend.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban; ia hanya masuk log. Membocorkan
// struktur basis data atau jejak tumpukan ke klien adalah celah keamanan
// (`11-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if fallback != nil {
			// Modul ini tidak mengenali galat apa pun sebagai miliknya, sehingga cadangan
			// selalu diberi kesempatan lebih dulu. Ia yang mengenali galat sesi.
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

// writeBadRequest menjawab permintaan yang cacat bentuknya.
//
// Dipisahkan dari WriteError karena ia bukan kegagalan sistem melainkan kesalahan klien,
// dan pesannya boleh menyebutkan apa yang salah — tidak ada rincian internal di dalamnya.
func writeBadRequest(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request, message string) {
	writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeBadRequest,
		Message: message,
	})
}
