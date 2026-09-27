package inputreqprotectionhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inputreqprotection"
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
	CodeNotFound      = "tidak_ditemukan"
	CodeValidation    = "validasi_gagal"
	CodeConflict      = "konflik"
	CodeInternalError = "galat_internal"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Details memuat pelanggaran per field, dan hanya terisi pada CodeValidation.
	//
	// Ia ada supaya layar dapat menempelkan pesan ke kolom yang tepat. Tanpa itu, seluruh
	// pesan menumpuk di satu tempat dan pengguna harus mencocokkannya sendiri ke kolom mana
	// — pada form berisi delapan field wajib, itu pekerjaan yang tidak perlu ada.
	//
	// # Kenapa kuncinya `detail`, dan bukan nama lain
	//
	// Klien bersama (`frontend/src/api/client.ts`) membaca `detail` berisi senarai
	// `{field, pesan}`, dan bentuk itu sudah dipakai modul `masterstatus`. Memakai nama
	// lain akan membuat pelanggaran validasi TIDAK PERNAH sampai ke kolomnya — pesannya
	// hilang tanpa galat, dan pengguna hanya melihat "Isian belum lengkap" tanpa tahu yang
	// mana.
	//
	// Ada dua bentuk lain yang hidup berdampingan hari ini (`masterstatusprogres` memakai
	// `kolom`, `masterrekening` memakai peta `field`). Penyeragamannya adalah
	// `TKT-F1-004`, yang masih terhalang; sampai itu selesai, yang benar adalah mengikuti
	// bentuk yang sudah dibaca klien — bukan menambah bentuk keempat.
	Details []FieldErrorResponse `json:"detail,omitempty"`
}

// FieldErrorResponse adalah satu pelanggaran pada satu field.
type FieldErrorResponse struct {
	Field   string `json:"field"`
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
// # Pembedaan 422 dari 409, dan kenapa itu penting bagi layar
//
// `10-API-STRATEGY.md` §5 menetapkan `422` untuk validasi bisnis yang gagal dan `409` untuk
// pelanggaran aturan atau konflik keadaan. Layar menanganinya berbeda: yang pertama
// menempelkan pesan ke kolom dan membiarkan pengguna memperbaikinya, yang kedua menutup
// form karena tidak ada yang dapat diperbaiki pengguna — proteksinya memang sudah tertaut
// atau sudah diakseptasi.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban; ia hanya masuk log. Membocorkan
// struktur basis data atau jejak tumpukan ke klien adalah celah keamanan
// (`11-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var validation *inputreqprotection.ValidationError
		if errors.As(err, &validation) {
			details := make([]FieldErrorResponse, 0, len(validation.Errors))
			for _, fe := range validation.Errors {
				details = append(details, FieldErrorResponse{Field: fe.Field, Message: fe.Message})
			}

			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:    CodeValidation,
				Message: "Isian belum lengkap atau belum benar.",
				Details: details,
			})
			return
		}

		switch {
		case errors.Is(err, inputreqprotection.ErrNotFound):
			writeJSON(w, r, http.StatusNotFound, ErrorResponse{
				Code:    CodeNotFound,
				Message: "Permintaan proteksi tidak ditemukan.",
			})
			return

		case errors.Is(err, inputreqprotection.ErrLocked):
			writeJSON(w, r, http.StatusConflict, ErrorResponse{
				Code:    CodeConflict,
				Message: "Permintaan proteksi sudah tertaut ke klaim dan tidak dapat diubah lagi.",
			})
			return

		case errors.Is(err, inputreqprotection.ErrAccepted):
			writeJSON(w, r, http.StatusConflict, ErrorResponse{
				Code:    CodeConflict,
				Message: "Permintaan proteksi sudah diakseptasi dan tidak dapat diubah lagi.",
			})
			return
		}

		if fallback != nil {
			// Galat yang tidak dikenali modul ini diteruskan ke cadangan — di cmd diisi
			// penulis galat auth, sehingga galat sesi yang lolos dari middleware tetap
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
