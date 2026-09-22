package mastermaskinghttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/mastermasking"
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
	CodeNotFound         = "masking_tidak_ditemukan"
	CodeValidationFailed = "validasi_gagal"
	CodePairTaken        = "masking_pengguna_sudah_ada"
	CodeBranchUnknown    = "cabang_tidak_dikenal"
	CodeIDTaken          = "id_masking_sudah_dipakai"
	CodeMalformedRequest = "permintaan_cacat"
	// CodeStatusNotChosen menjawab pencarian menurut status yang statusnya belum dipilih —
	// keadaan yang di layar lama dijawab pesan "Pilih Status Aktif".
	CodeStatusNotChosen = "status_belum_dipilih"
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
	var validationError *mastermasking.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: permintaannya berbentuk benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 adalah bug frontend, 422 adalah
		// kesalahan pengguna yang harus ditandai di kolomnya
		// (`docs/Steering/10-API-STRATEGY.md` §5).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, v := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Isian belum benar. Perbaiki yang ditandai lalu simpan lagi.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastermasking.ErrPairTaken):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan
		// saat ini. Pesannya menyebut apa yang harus dilakukan — mengubah baris yang sudah
		// ada, bukan menambah baris kedua — karena dua baris untuk orang yang sama di satu
		// cabang berarti dua kewenangan yang keduanya berlaku, dan yang satu dapat luput
		// saat dicabut.
		return http.StatusConflict, ErrorResponse{
			Code:    CodePairTaken,
			Message: "Pengguna itu sudah punya data masking di cabang tersebut. Ubah data yang sudah ada, jangan menambah baris baru.",
		}, true

	case errors.Is(err, mastermasking.ErrBranchUnknown):
		// 422: isiannya berbentuk benar tetapi menunjuk cabang yang tidak ada. Ia ditandai
		// pada kolom cabang supaya pengguna tahu persis isian mana yang harus diperbaiki.
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeBranchUnknown,
			Message: "Cabang yang dipilih tidak dikenal. Pilih cabang dari daftar.",
			Detail: []ViolationDTO{{
				Field:   mastermasking.FieldBranchID,
				Message: "Cabang tidak ditemukan pada entitas ini.",
			}},
		}, true

	case errors.Is(err, mastermasking.ErrIDTaken):
		// 409 dan pesannya menyuruh mencoba lagi, karena percobaan ulang MEMANG menolong:
		// nomor dibentuk `MAX+1` tanpa indeks unik yang menjaganya, sehingga bentrokan
		// bersifat sesaat dan hilang pada percobaan berikutnya.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeIDTaken,
			Message: "Nomor data masking sedang dipakai permintaan lain. Coba simpan sekali lagi.",
		}, true

	case errors.Is(err, mastermasking.ErrNoSequence):
		return http.StatusInternalServerError, ErrorResponse{
			Code:    CodeIDTaken,
			Message: "Nomor data masking tidak dapat dibentuk pada entitas ini. Hubungi administrator Claim PNC.",
		}, true

	case errors.Is(err, mastermasking.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Data masking tidak ditemukan. Mungkin baru saja diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
