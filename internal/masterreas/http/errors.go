package masterreashttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/masterreas"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
)

// Kode galat modul ini.
//
// # Modul ini TIDAK punya galat domain sendiri, dan itu bukan kelalaian
//
// Modul master lain memetakan galatnya sendiri — validasi gagal, kunci bentrok, isian
// terkunci — karena ketiganya lahir dari jalur TULIS. Modul ini hanya membaca (lihat banner
// paket masterreas), sehingga tidak ada satu pun keadaan yang dapat ditolak atas dasar
// aturan bisnis.
//
// Yang tersisa hanyalah dua kelompok, dan keduanya sudah punya pemiliknya masing-masing:
//
//	galat PORTAL     dipetakan portalhttp.WithPortalError, satu pemetaan untuk seluruh modul
//	galat TEKNIS     diserahkan ke penulis galat bersama; 500 dengan pesan umum
//
// Satu-satunya kode yang didefinisikan di sini adalah CodeMalformedRequest, dan ia ada
// supaya penyaring kueri yang cacat tidak jatuh menjadi 500 — kegagalan klien tidak boleh
// terbaca sebagai kegagalan server.
const (
	CodeMalformedRequest = "permintaan_cacat"

	// CodeValidationFailed: isian tidak lolos pemeriksaan — 422.
	CodeValidationFailed = "validasi_gagal"

	// CodeNotFound: baris yang hendak diubah tidak ada — 404.
	//
	// Pada modul ini ia lebih mungkin terjadi daripada di modul master lain: alur PLA/DLA
	// menulis ke tabel yang sama lewat `Database/UPDATEREAS.prc`, sehingga baris dapat
	// berubah di belakang layar antara saat daftar dimuat dan saat Simpan ditekan.
	CodeNotFound = "tidak_ditemukan"
)

// mapError memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.WithPortalError yang
// membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai seluruh
// modul bisnis, bukan satu tafsiran per modul.
func mapError(err error) (int, ErrorResponse, bool) {
	var validationError *masterreas.ValidationError

	switch {
	case errors.As(err, &validationError):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan bisnis.
		// Frontend menanganinya berbeda — 400 berarti ada cacat di frontend, 422 berarti
		// pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja (`P-5`).
		detail := make([]ViolationDTO, 0, len(validationError.Violation))
		for _, p := range validationError.Violation {
			detail = append(detail, ViolationDTO{Field: p.Field, Message: p.Message})
		}
		return http.StatusUnprocessableEntity, ErrorResponse{
			Code:    CodeValidationFailed,
			Message: "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail:  detail,
		}, true

	case errors.Is(err, masterreas.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{
			Code: CodeNotFound,
			Message: "Baris member reas ini sudah tidak ada. Muat ulang daftarnya, " +
				"lalu coba lagi.",
		}, true
	}

	return 0, ErrorResponse{}, false
}

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// writeModuleError meneruskan galat ke penulis bersama, dan mencatat yang perlu dicatat.
//
// Tidak ada pemetaan di sini; lihat catatan pada blok konstanta di atas.
//
// # Galat portal TIDAK dicatat sebagai kegagalan
//
// Permintaan tanpa header portal, atau menyebut portal yang tidak dikenal, adalah kesalahan
// KLIEN — dijawab 400 oleh portalhttp.WithPortalError. Mencatatnya sebagai Error akan
// menenggelamkan kegagalan sungguhan di antara permintaan yang hanya dibuka sebelum portal
// dipilih, dan itu terjadi setiap kali seseorang membuka aplikasi.
//
// `ErrNotReady` DIKECUALIKAN dari pengecualian itu: ia dijawab 503, dan penyebabnya memang
// pekerjaan administrator — kredensial basis data entitas yang belum diisi. Ia harus
// terlihat.
//
// Rincian galat internal TIDAK pernah dikirim ke peramban; ia hanya masuk log.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	// Galat yang dikenali modul ini dijawab di sini; sisanya diteruskan ke penulis bersama,
	// yang menjawab 500 dengan pesan umum dan menaruh rinciannya di log saja.
	if status, body, known := mapError(err); known {
		h.writeResponse(w, r, status, body)
		return
	}

	clientMistake := errors.Is(err, portal.ErrNotStated) || errors.Is(err, portal.ErrNotFound)
	if !clientMistake {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeError(w, r, err)
}
