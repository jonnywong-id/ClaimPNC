package inboxlaporanklaimhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/platform/logging"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih terhalang
// keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth, dan
// menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan sisanya
// ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap sama sehingga
// klien tidak menghadapi dua bentuk galat yang berbeda.
const (
	CodeValidationFailed = "validasi_gagal"
	CodeNotFound         = "tidak_ditemukan"
	CodeMalformedRequest = "permintaan_cacat"
	CodeCallerIncomplete = "profil_pemanggil_tidak_lengkap"

	// CodeReadOnly: berkas ada, tetapi penulisnya sistem lain.
	//
	// Ia BUKAN validasi_gagal, meski keduanya menolak penyimpanan. Tidak ada satu pun
	// isian yang dapat diperbaiki pengguna, dan klien membedakan jenis galat lewat
	// `kode` — memakai kode validasi akan membuat layar menunggu `detail` yang tidak
	// pernah datang, lalu menampilkan form yang tampak dapat diperbaiki padahal tidak.
	CodeReadOnly = "laporan_hanya_baca"

	// CodeBranchUnknown: cabang klaim petugas tidak dapat ditentukan dari login-nya.
	CodeBranchUnknown = "cabang_tidak_dikenali"

	// CodeBranchUnreadable: sumber data cabang sedang tidak dapat dibaca.
	//
	// Dipisahkan dari CodeBranchUnknown meski layar menutup pada keduanya. Yang satu
	// menimpa satu orang dan dibereskan di data pegawai; yang lain menimpa SEMUA orang
	// dan dibereskan di infrastruktur. Satu kode untuk keduanya akan membuat gangguan
	// yang menimpa seluruh kantor terbaca sebagai masalah satu pengguna.
	CodeBranchUnreadable = "sumber_cabang_tidak_terbaca"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan
		// jaringan, cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500
		// dengan pesan umum dan menaruh rinciannya di log saja. Rincian galat internal
		// tidak pernah dikirim ke peramban.
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

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *inboxlaporanklaim.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend,
		// 422 berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, v := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: v.Field, Message: v.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrUnknownCategory):
		// 422, bukan 404: yang tidak dikenal adalah ISIAN pada permintaan, bukan alamat
		// sumber daya. Menjawab 404 akan terbaca sebagai "layar ini tidak ada".
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Tab yang diminta tidak dikenal.",
			Detail:  []ViolationDTO{{Field: "kategori", Message: "Tab tidak dikenal."}},
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrUnknownBusinessLine):
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Pilihan bisnis tidak dikenal.",
			Detail:  []ViolationDTO{{Field: "bisnis", Message: "Pilihan bisnis tidak dikenal."}},
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code:    CodeNotFound,
			Message: "Laporan klaim yang dimaksud tidak ditemukan. Muat ulang daftarnya.",
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrCallerUnknown):
		// 409, bukan 422: tidak ada satu pun isian yang dapat diperbaiki pengguna. Yang
		// kurang adalah identitasnya sendiri, dan itu harus dibereskan di tempat lain.
		//
		// Ia BUKAN galat sesi — sesi sudah diperiksa middleware jauh sebelum sampai ke
		// sini; yang ini berarti jembatan ke konteks pemanggil tidak terpasang.
		return http.StatusConflict, ErrorResponse{
			Code: CodeCallerIncomplete,
			Message: "Identitas Anda belum terbaca lengkap, sehingga laporan baru tidak " +
				"dapat dibuat. Hubungi pengelola aplikasi.",
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrBranchUnknown):
		// 403, bukan 404 dan bukan 422: pemanggilnya sudah masuk, alamatnya benar, dan
		// tidak ada satu pun isian yang dapat diperbaiki. Yang tidak dapat ditetapkan
		// adalah BATAS DATA-nya, dan tanpa batas itu permintaannya tidak boleh dilayani
		// (`10-API-STRATEGY.md` §5: "sudah login tapi tidak berwenang").
		return http.StatusForbidden, ErrorResponse{
			Code: CodeBranchUnknown,
			Message: "Cabang klaim Anda tidak dapat ditentukan, sehingga daftar laporan tidak " +
				"dapat ditampilkan. Hubungi pengelola aplikasi agar login Anda didaftarkan " +
				"pada cabangnya.",
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrBranchUnreadable):
		// 503, bukan 403: yang gagal bukan kewenangan pemanggil melainkan sumber datanya
		// — dan ia menimpa seluruh petugas sekaligus. 503 juga menyatakan keadaannya
		// SEMENTARA, sehingga mencoba lagi memang masuk akal.
		//
		// Ia tetap dicatat sebagai galat di log oleh writeModuleError, karena statusnya
		// di atas 500. Itu disengaja: DB link yang mati harus terlihat operator.
		return http.StatusServiceUnavailable, ErrorResponse{
			Code: CodeBranchUnreadable,
			Message: "Data cabang sedang tidak dapat dibaca, sehingga daftar laporan belum " +
				"dapat ditampilkan. Coba lagi beberapa saat, dan laporkan bila berulang.",
		}, true

	case errors.Is(err, inboxlaporanklaim.ErrReadOnlyOrigin):
		// 409, bukan 403: yang menolak bukan kewenangan pengguna melainkan KEADAAN
		// berkasnya — selama masa paralel, penulisnya masih Pega (`ADR-0004`, `P-1`).
		// Pengguna yang sama dapat menyimpan berkas lain tanpa masalah.
		return http.StatusConflict, ErrorResponse{
			Code: CodeReadOnly,
			Message: "Laporan ini masih dikelola sistem lama dan hanya dapat dibaca dari sini. " +
				"Perubahannya dilakukan di Pega.",
		}, true

	default:
		return 0, ErrorResponse{}, false
	}
}
