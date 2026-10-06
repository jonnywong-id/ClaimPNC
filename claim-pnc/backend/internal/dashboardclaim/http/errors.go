package dashboardclaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/dashboardclaim"
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
	CodeBadRequest          = "permintaan_cacat"
	CodeValidation          = "validasi_gagal"
	CodeTileNotFound        = "tile_tidak_dikenal"
	CodeInternalError       = "galat_internal"
	CodeTransferUnavailable = "transfer_belum_aktif"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []violationDTO `json:"detail,omitempty"`
}

// violationDTO adalah satu pelanggaran validasi.
//
// Nama kuncinya `field`, mengikuti bentuk yang dipakai `masterstatus` dan `inboxcloseclaim`
// — salah satu dari tiga bentuk yang hidup berdampingan hari ini dan yang sudah ditampung
// `APIError.violations()` di frontend. Menyeragamkan ketiganya adalah `TKT-F1-004`, yang
// masih terhalang.
type violationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// JSONWriter menuliskan badan respons.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// WriteError memetakan galat menjadi respons HTTP.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban; ia hanya masuk log. Membocorkan
// struktur basis data atau jejak tumpukan ke klien adalah celah keamanan
// (`12-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var validation *dashboardclaim.ValidationError
		if errors.As(err, &validation) {
			details := make([]violationDTO, 0, len(validation.Violations))
			for _, v := range validation.Violations {
				details = append(details, violationDTO{Field: v.Field, Message: v.Message})
			}
			// 422, bukan 400: permintaannya benar bentuknya tetapi melanggar aturan
			// (`10-API-STRATEGY.md` §5). Frontend menanganinya berbeda.
			writeJSON(w, r, http.StatusUnprocessableEntity, ErrorResponse{
				Code:    CodeValidation,
				Message: "Permintaan tidak dapat diproses.",
				Detail:  details,
			})
			return
		}

		if errors.Is(err, dashboardclaim.ErrTransferUnavailable) {
			writeTransferUnavailable(writeJSON, w, r)
			return
		}

		if errors.Is(err, dashboardclaim.ErrTileNotFound) {
			// 404, bukan 422: yang salah bukan isian melainkan JALUR-nya.
			writeJSON(w, r, http.StatusNotFound, ErrorResponse{
				Code:    CodeTileNotFound,
				Message: "Tile dashboard tidak dikenal.",
			})
			return
		}

		if fallback != nil {
			// Galat yang tidak dikenali modul ini diteruskan ke cadangan — di cmd diisi
			// penulis galat portal dan auth, sehingga galat portal tetap dijawab dengan
			// kode yang sudah dikenal frontend dan bukan 500.
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

// writeTransferUnavailable menjawab permintaan transfer yang belum dapat dilayani.
//
// 503, bukan 500: sistemnya tidak rusak — satu perubahan skema sedang ditunggu (`D-63`), dan
// permintaan yang sama akan berhasil begitu DBA menjalankannya. Pesannya menyebutkan itu,
// supaya pengguna tidak melaporkannya sebagai kerusakan.
func writeTransferUnavailable(writeJSON JSONWriter, w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusServiceUnavailable, ErrorResponse{
		Code: CodeTransferUnavailable,
		Message: "Pencatatan permintaan transfer belum aktif. " +
			"Tabelnya menunggu dijalankan DBA; pemindahan tugas sementara ini dikerjakan di Pega.",
	})
}
