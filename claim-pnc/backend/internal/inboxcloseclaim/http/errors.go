package inboxcloseclaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat yang dikenali klien.
//
// Klien membedakan jenis galat lewat kode ini, bukan dengan mencocokkan teks pesan — teks
// dapat berubah kapan saja tanpa mengubah artinya.
//
// Nilainya berbahasa Indonesia karena ia KONTRAK yang dibaca frontend (`D-80`); yang
// berbahasa Inggris hanyalah nama konstantanya.
const (
	CodeBadRequest      = "permintaan_cacat"
	CodeValidation      = "validasi_gagal"
	CodeNotAllowed      = "tidak_berwenang"
	CodeClaimNotFound   = "klaim_tidak_ditemukan"
	CodeRequestPending  = "permintaan_masih_menunggu"
	CodeInternalError   = "galat_internal"
	CodeMethodNotAllows = "metode_tidak_didukung"
)

// ErrorResponse adalah bentuk galat yang dikirim ke klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []violationDTO `json:"detail,omitempty"`
}

// violationDTO adalah satu pelanggaran validasi.
//
// Nama kuncinya `field`, mengikuti bentuk yang dipakai modul `masterstatus` — salah satu
// dari tiga bentuk yang hidup berdampingan hari ini dan yang sudah ditampung
// `APIError.violations()` di frontend. Menyeragamkan ketiganya adalah `TKT-F1-004`, yang
// masih terhalang.
type violationDTO = apierror.FieldError

// JSONWriter menuliskan badan respons.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// WriteError memetakan galat menjadi respons HTTP.
//
// Rincian galat internal TIDAK PERNAH dikirim ke peramban; ia hanya masuk log. Membocorkan
// struktur basis data atau jejak tumpukan ke klien adalah celah keamanan
// (`12-CROSSCUTTING` §1.2 butir 5).
func WriteError(logger *slog.Logger, writeJSON JSONWriter, fallback ErrorWriter) ErrorWriter {
	return apierror.ContextWriter(logger, writeJSON, fallback, mapError, ErrorResponse{
		Code:    CodeInternalError,
		Message: "Terjadi kesalahan pada sistem.",
	})
}

// mapError memetakan galat yang dikenali modul ini menjadi status dan badan respons; nilai
// ketiga false bila galatnya bukan milik modul ini.
func mapError(err error) (int, ErrorResponse, bool) {
	var validation *inboxcloseclaim.ValidationError
	if errors.As(err, &validation) {
		details := apierror.FieldErrors(validation.Violations)
		// 422, bukan 400: permintaannya benar bentuknya tetapi melanggar aturan bisnis
		// (`10-API-STRATEGY.md` §5). Frontend menanganinya berbeda.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidation,
			Message: "Permintaan tidak dapat diproses.",
			Detail:  details,
		}, true
	}

	if errors.Is(err, inboxcloseclaim.ErrRequestNotAllowed) {
		// 403, bukan 401: pemanggilnya SUDAH masuk — yang tidak ada adalah
		// kewenangannya (`10-API-STRATEGY.md` §5). Menjawab 401 akan membuat layar
		// mengira sesinya habis lalu mengeluarkan pengguna dari aplikasi.
		//
		// Pesannya diambil dari domain, bukan ditulis ulang di sini: aturan dan
		// keterangannya tidak boleh hidup di dua tempat yang dapat menyimpang.
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeNotAllowed,
			Message: inboxcloseclaim.AlasanTidakBolehMengajukan,
		}, true
	}

	if errors.Is(err, inboxcloseclaim.ErrClaimNotFound) {
		return http.StatusNotFound, ErrorResponse{
			Code: CodeClaimNotFound,
			Message: "Klaim tidak ditemukan di daftar klaim tutup. " +
				"Klaim yang masih berjalan tidak dapat diajukan dari layar ini.",
		}, true
	}

	if errors.Is(err, inboxcloseclaim.ErrRequestPending) {
		// 409, bukan 422: bukan isiannya yang salah melainkan KEADAAN yang berkonflik —
		// permintaan sebelumnya belum dijalankan.
		return http.StatusConflict, ErrorResponse{
			Code: CodeRequestPending,
			Message: "Permintaan sejenis atas klaim ini masih menunggu dijalankan. " +
				"Tunggu sampai permintaan sebelumnya selesai.",
		}, true
	}
	return 0, ErrorResponse{}, false
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
