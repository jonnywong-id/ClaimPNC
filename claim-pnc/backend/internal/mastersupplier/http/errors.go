package mastersupplierhttp

import (
	"errors"
	"net/http"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/platform/apierror"
)

// Kode galat modul ini.
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
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"

	// CodeNameTaken dibedakan dari CodeValidationFailed dengan sengaja.
	//
	// Ia konflik KEADAAN, bukan isian yang cacat: namanya sah, dan yang salah hanyalah
	// bahwa nama itu sudah dipakai supplier lain. Perbaikannya bukan "betulkan isian"
	// melainkan "buka baris yang sudah ada, atau pakai nama lain", dan layar menanganinya
	// berbeda.
	CodeNameTaken = "nama_supplier_sudah_ada"

	// CodeNameLocked muncul bila nama diubah pada baris yang sudah tersimpan.
	//
	// Ia juga konflik keadaan, bukan isian cacat — dan pesannya harus menjelaskan bahwa
	// yang salah bukan isinya melainkan bahwa isian itu memang tidak boleh berubah.
	CodeNameLocked = "nama_supplier_terkunci"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	apierror.Write(w, r, err, mapError, h.Logger, h.WriteResponse, h.WriteError)
}

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *mastersupplier.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		// Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422 berarti
		// pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (P-5). Pada form
		// berisi lima belas isian wajib, mengirimnya satu per satu berarti lima belas kali
		// bolak-balik untuk satu form kosong.
		detail := apierror.ColumnErrors(validationError.Violation)
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, mastersupplier.ErrNameTaken):
		// 409, bukan 422. Pesannya mengikuti gaya pesan padanannya di Master Bengkel,
		// karena sistem lama tidak punya pesan sendiri untuk keadaan ini — pemeriksaannya
		// memang tidak ada di sana.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNameTaken,
			Message: "Nama supplier tersebut telah digunakan. Silakan ganti dengan nama yang lain.",
			Detail: []ViolationDTO{{
				Field:   "nama",
				Message: "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.",
			}},
		}, true

	case errors.Is(err, mastersupplier.ErrNameLocked):
		// 409 juga, dan pesannya menjelaskan SEBABNYA — bukan sekadar menolak.
		//
		// Petugas yang menghadapinya tidak melakukan kesalahan mengetik: layar mengunci
		// isian itu, sehingga satu-satunya cara sampai ke sini adalah permintaan yang tidak
		// melewati layar. Pesannya tetap ditulis untuk manusia, karena yang membacanya
		// tetap manusia.
		return http.StatusConflict, ErrorResponse{
			Code:    CodeNameLocked,
			Message: "Nama supplier tidak dapat diubah setelah tersimpan.",
			Detail: []ViolationDTO{{
				Field:   "nama",
				Message: "Nama supplier tidak dapat diubah setelah tersimpan. Batalkan, lalu muat ulang barisnya.",
			}},
		}, true

	case errors.Is(err, mastersupplier.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Supplier yang dimaksud tidak ditemukan. Mungkin sudah diubah petugas lain — muat ulang daftarnya.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
