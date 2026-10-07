package masterpicteknikhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/masterpicteknik"
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
	CodeNotFound              = "pic_teknik_tidak_ditemukan"
	CodeValidationFailed      = "validasi_gagal"
	CodeAlreadyExists         = "id_operator_sudah_terdaftar"
	CodeDirectoryUnreachable  = "direktori_pegawai_tidak_terhubung"
	CodeDirectoryUnconfigured = "direktori_pegawai_belum_terdaftar"
	CodeMalformedRequest      = "permintaan_cacat"
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
	var validationError *masterpicteknik.ValidationError

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

	case errors.Is(err, masterpicteknik.ErrAlreadyExists):
		// 409, bukan 422: isian penggunanya sah, tetapi bentrok dengan keadaan penyimpanan
		// saat ini — mungkin karena petugas lain baru saja mendaftarkannya.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeAlreadyExists,
			Message: "ID operator itu sudah terdaftar. Buka datanya lalu ubah di sana.",
		}, true

	case errors.Is(err, masterpicteknik.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "PIC teknik tidak ditemukan. Mungkin baru saja diubah petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, masterpicteknik.ErrDirectoryNotConfigured):
		// 503 dan bukan 500, tetapi pesannya sengaja menyebut apa yang harus dilengkapi.
		// Ini bukan kesalahan pengguna dan TIDAK akan pulih dengan mencoba ulang: barisnya
		// harus ditambahkan ke katalog layanan lebih dulu. Membedakannya dari gangguan
		// jaringan di bawah mencegah administrator menunggu sesuatu yang tidak akan
		// terjadi.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeDirectoryUnconfigured,
			Message: "Layanan pencarian pegawai belum terdaftar untuk entitas ini. Mengulang tidak akan menolong — hubungi administrator Claim PNC.",
		}, true

	case errors.Is(err, masterpicteknik.ErrDirectoryUnreachable):
		// 503, bukan 500: ini bukan cacat aplikasi melainkan sumber luar yang sedang tidak
		// dapat dihubungi, dan tindak lanjutnya menunggu — bukan melapor.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code:    CodeDirectoryUnreachable,
			Message: "Direktori pegawai sedang tidak dapat dihubungi. Coba beberapa saat lagi.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
