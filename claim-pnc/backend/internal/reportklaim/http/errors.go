package reportklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/reportklaim"
	reportklaimsql "claim-pnc/internal/reportklaim/repo/sqlstore"
)

// Kode galat modul ini.
//
// Modul memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya ke penulis
// galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga klien
// tidak menghadapi dua bentuk galat yang berbeda. Alasannya sama dengan modul Inbox
// Laporan Klaim: kontrak galat aplikasi (`TKT-F1-004`) masih terhalang keputusan Work
// Owner, dan menambah kode ke modul auth berarti menyunting modul yang sudah selesai.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeCallerIncomplete = "profil_pemanggil_tidak_lengkap"

	// CodeReportNotReady: laporannya ADA di katalog tetapi belum dapat dijalankan.
	//
	// Ia BUKAN tidak_ditemukan. Yang pertama berarti kodenya salah — layar memperbaiki
	// pemanggilannya; yang ini berarti artefak Pega-nya belum ada, dan tidak ada yang
	// dapat diperbaiki dari sisi layar maupun sisi pengguna.
	CodeReportNotReady = "laporan_belum_tersedia"

	// CodeUnknownAction: tombol yang diminta tidak ada pada panel itu.
	CodeUnknownAction = "tombol_tidak_dikenal"

	// CodeQueryNotPorted: kuerinya belum selesai dipindahkan dari export Pega.
	//
	// Dipisahkan dari CodeReportNotReady meski keduanya berarti "belum dapat dijalankan".
	// Sebab dan pemiliknya berbeda:
	//
	//	laporan_belum_tersedia   artefak Pega tidak ada          → Tim Pega
	//	kueri_belum_dipindahkan  artefaknya ada, kami yang belum → tim pengembang
	//
	// Ia juga berlaku hanya pada SEBAGIAN lini bisnis, sehingga pesannya menyebutkan itu.
	CodeQueryNotPorted = "kueri_belum_dipindahkan"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// errorBody adalah bentuk galat yang dikirim ke klien.
type errorBody struct {
	Kode   string          `json:"kode"`
	Pesan  string          `json:"pesan"`
	Detail []violationBody `json:"detail,omitempty"`
}

type violationBody struct {
	Isian string `json:"isian"`
	Pesan string `json:"pesan"`
}

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, status, body)
}

// mapError memetakan galat domain ke kode HTTP beserta badannya.
func mapError(err error) (int, errorBody, bool) {
	var validation *reportklaim.ValidationError
	if errors.As(err, &validation) {
		detail := make([]violationBody, 0, len(validation.Violation))
		for _, v := range validation.Violation {
			detail = append(detail, violationBody{Isian: v.Field, Pesan: v.Message})
		}
		// 422, bukan 400: permintaannya berbentuk benar tetapi melanggar aturan bisnis.
		// Frontend menanganinya berbeda — yang satu bug frontend, yang lain kesalahan
		// pengisian (`10-API-STRATEGY.md` §5).
		return http.StatusUnprocessableEntity, errorBody{
			Kode:   CodeValidationFailed,
			Pesan:  "Ada isian yang belum benar.",
			Detail: detail,
		}, true
	}

	switch {
	case errors.Is(err, reportklaim.ErrUnknownReport):
		return http.StatusNotFound, errorBody{
			Kode:  CodeNotFound,
			Pesan: "Laporan yang diminta tidak dikenal.",
		}, true

	case errors.Is(err, reportklaim.ErrReportNotReady):
		// 409, bukan 404: laporannya ADA, keadaannya yang belum memungkinkan.
		return http.StatusConflict, errorBody{
			Kode:  CodeReportNotReady,
			Pesan: "Laporan ini belum dapat dijalankan. Keterangannya tertera pada kartunya.",
		}, true

	case errors.Is(err, reportklaim.ErrUnknownAction):
		return http.StatusBadRequest, errorBody{
			Kode:  CodeUnknownAction,
			Pesan: "Tombol yang diminta tidak ada pada laporan ini.",
		}, true

	case errors.Is(err, reportklaimsql.ErrQueryNotPorted):
		return http.StatusConflict, errorBody{
			Kode: CodeQueryNotPorted,
			Pesan: "Laporan ini belum tersedia untuk lini bisnis yang dipilih. " +
				"Lini bisnis lain pada laporan yang sama tetap dapat diunduh.",
		}, true

	case errors.Is(err, reportklaim.ErrCallerUnknown):
		return http.StatusInternalServerError, errorBody{
			Kode:  CodeCallerIncomplete,
			Pesan: "Identitas pemanggil tidak dapat dibaca.",
		}, true
	}
	return 0, errorBody{}, false
}

// logExportFailure mencatat kegagalan yang terjadi SETELAH header terkirim.
//
// Sesudah header terkirim, galat tidak dapat lagi dijawab sebagai JSON — yang sampai ke
// pengguna berupa berkas CSV separuh jadi tanpa satu pun keterangan. Satu-satunya tempat
// keadaan itu terbaca adalah log.
func (h *Handler) logExportFailure(r *http.Request, err error) {
	if h.logger == nil {
		return
	}
	h.logger.ErrorContext(r.Context(), "ekspor laporan terputus di tengah",
		slog.String("jalur", r.URL.Path),
		slog.String("id_permintaan", logging.RequestID(r.Context())),
		slog.String("sebab", err.Error()))
}
