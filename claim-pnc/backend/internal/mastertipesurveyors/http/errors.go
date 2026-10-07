package mastertipesurveyorshttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat modul ini. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan
// menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama
// sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
const (
	CodeNotFound         = "tipe_surveyor_tidak_ditemukan"
	CodeValidationFailed = "validasi_gagal"
	CodeDescriptionTaken = "tipe_surveyor_sudah_ada"
	CodeCodeTaken        = "kode_tipe_surveyor_sudah_dipakai"
	CodeSiteMissing      = "kode_tidak_dapat_dibentuk"
	CodeMalformedRequest = "permintaan_cacat"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.Logger, h.WriteResponse, h.WriteError)
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke
// yang lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *mastertipesurveyors.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (`docs/Steering/10-API-STRATEGY.md` §5).
		detail := apierror.FieldErrors(validationError.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastertipesurveyors.ErrDescriptionTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan
		// saat ini — mungkin karena petugas lain baru saja memakai nama itu.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeDescriptionTaken,
			Message: "Tipe surveyor dengan nama itu sudah ada. Pakai nama lain.",
		}, true

	case errors.Is(err, mastertipesurveyors.ErrCodeTaken):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeCodeTaken,
			Message: "Kode tipe surveyor yang dibuat sistem sudah dipakai. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, mastertipesurveyors.ErrNoSite):
		// 500, dan pesannya sengaja menyebut apa yang harus diperbaiki. Ini bukan
		// kesalahan pengguna dan tidak dapat ditolong dengan mencoba ulang: tabel situs
		// pada basis data entitas itu tidak memuat baris aktif, sehingga kode tidak dapat
		// dibentuk sama sekali.
		//
		// Nama tabelnya disebut karena ia objek milik kita sendiri — bukan rincian galat
		// driver dan bukan data nasabah — dan tanpa itu administrator tidak punya petunjuk
		// apa pun untuk menindaklanjuti.
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeSiteMissing,
			Message: "Kode tipe surveyor tidak dapat dibentuk: tabel situs pada basis data entitas ini belum berisi baris aktif. Hubungi administrator Claim PNC.",
		}, true

	case errors.Is(err, mastertipesurveyors.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Tipe surveyor tidak ditemukan. Mungkin baru saja diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
