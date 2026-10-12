package mastersurveyorshttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini. Klien membedakan jenis galat lewat kode ini, bukan dengan
// mencocokkan teks pesan — teks dapat berubah kapan saja tanpa mengubah artinya.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan menambah
// kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
const (
	CodeNotFound              = "surveyor_tidak_ditemukan"
	CodeValidationFailed      = "validasi_gagal"
	CodeNameTaken             = "nama_surveyor_sudah_ada"
	CodeLoginTaken            = "login_aplikasi_sudah_dipakai"
	CodeAlreadyDecided        = "keputusan_komite_sudah_diambil"
	CodeNotAssignedCommittee  = "bukan_komite_yang_ditunjuk"
	CodeUnknownDecisionStatus = "status_keputusan_tidak_dikenal"
	CodeSiteMissing           = "id_tidak_dapat_dibentuk"
	CodeMalformedRequest      = "permintaan_cacat"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — galat portal, kegagalan basis data,
		// kegagalan jaringan, cacat pemrograman — diserahkan ke penulis bersama. Galat
		// portal dipetakan portalhttp.WithPortalError yang membungkusnya di cmd; sisanya
		// dijawab 500 dengan pesan umum, dan rinciannya hanya masuk log.
		h.writeError(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeResponse(w, r, status, body)
}

// mapError menerjemahkan galat domain menjadi status dan badan HTTP.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini. Ia dibutuhkan supaya
// pemanggil dapat membedakan "ini milik saya" dari "ini bukan milik saya, serahkan ke yang
// lain" — dua hal yang tidak dapat dibedakan hanya dari status 500.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *mastersurveyors.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (`docs/Steering/10-API-STRATEGY.md` §5).
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  violationsOf(validationError.Field),
		}, true

	case errors.Is(err, mastersurveyors.ErrNameTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan
		// saat ini — mungkin karena petugas lain baru saja memakai nama itu.
		//
		// Pesannya menyebutkan aturan pembandingnya, karena tanpa itu penolakan akan
		// terasa keliru: "Budi Santoso" ditolak oleh "BUDISANTOSO" yang tidak terlihat
		// sama di layar.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNameTaken,
			Message: "Surveyor dengan nama itu sudah terdaftar. Perbandingannya mengabaikan huruf besar-kecil dan spasi, sehingga \"Budi Santoso\" dan \"budisantoso\" dianggap sama.",
		}, true

	case errors.Is(err, mastersurveyors.ErrLoginTaken):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeLoginTaken,
			Message: "Login aplikasi itu sudah dipakai. Pakai nama login lain.",
		}, true

	case errors.Is(err, mastersurveyors.ErrAlreadyDecided):
		return http.StatusConflict, ErrorResponse{
			Code:    CodeAlreadyDecided,
			Message: "Keputusan komite atas surveyor ini sudah pernah diambil dan tidak dapat diubah dari layar ini.",
		}, true

	case errors.Is(err, mastersurveyors.ErrNotAssignedCommittee):
		// 403, bukan 404: barisnya ada dan pemanggilnya sudah dikenali — yang kurang
		// adalah kewenangannya atas baris itu.
		return http.StatusForbidden, ErrorResponse{
			Code:    CodeNotAssignedCommittee,
			Message: "Anda bukan komite yang ditunjuk untuk surveyor ini.",
		}, true

	case errors.Is(err, mastersurveyors.ErrUnknownStatus):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeUnknownDecisionStatus,
			Message: "Keputusan harus berupa setuju atau tolak.",
		}, true

	case errors.Is(err, mastersurveyors.ErrNoSite):
		// 500, dan pesannya sengaja menyebut apa yang harus diperbaiki. Ini bukan
		// kesalahan pengguna dan tidak dapat ditolong dengan mencoba ulang: tabel situs
		// pada basis data entitas itu tidak memuat baris aktif, sehingga ID tidak dapat
		// dibentuk sama sekali.
		//
		// Nama tabelnya disebut karena ia objek milik kita sendiri — bukan rincian galat
		// driver dan bukan data nasabah — dan tanpa itu administrator tidak punya petunjuk
		// apa pun untuk menindaklanjuti.
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeSiteMissing,
			Message: "ID surveyor tidak dapat dibentuk: tabel situs pada basis data entitas ini belum berisi baris aktif. Hubungi administrator Claim PNC.",
		}, true

	case errors.Is(err, mastersurveyors.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Surveyor tidak ditemukan. Mungkin baru saja diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
