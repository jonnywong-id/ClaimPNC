package statusprogreshttp

import (
	"errors"
	"log/slog"
	"net/http"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/statusprogres"
)

// Kode galat modul ini.
//
// # Kenapa modul ini memetakan galatnya sendiri
//
// Kontrak galat yang mengikat seluruh aplikasi adalah TKT-F1-004, dan ia masih
// terhalang keputusan Work Owner. Yang ada sekarang hanyalah pemetaan milik modul auth,
// dan menambah kode ke sana berarti menyunting modul yang sudah dinyatakan selesai.
//
// Karena itu modul ini memetakan galat yang DIKENALINYA sendiri, lalu menyerahkan
// sisanya ke penulis galat yang disuntikkan dari cmd — bentuk `{kode, pesan}` tetap
// sama sehingga klien tidak menghadapi dua bentuk galat yang berbeda.
//
// Begitu TKT-F1-004 diputuskan, pemetaan ini pindah ke tempat bersama dan berkas ini
// tinggal memakainya. Utang itu dicatat di docs/keputusan-implementasi.md.
const (
	KodeValidasiGagal   = "validasi_gagal"
	KodeTidakDitemukan  = "tidak_ditemukan"
	KodePermintaanCacat = "permintaan_cacat"
)

// PenulisGalat menuliskan galat dalam bentuk respons HTTP.
type PenulisGalat func(w http.ResponseWriter, r *http.Request, err error)

// PenulisRespon menuliskan badan respons yang berhasil.
type PenulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any)

// penulisGalatModul memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) tulisGalatModul(w http.ResponseWriter, r *http.Request, err error) {
	status, badan, dikenali := petakanGalat(err)
	if !dikenali {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan
		// jaringan, cacat pemrograman — diserahkan ke penulis bersama, yang menjawab
		// 500 dengan pesan umum dan menaruh rinciannya di log saja. Rincian galat
		// internal tidak pernah dikirim ke peramban.
		h.tulisGalat(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.Dari(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.tulisRespon(w, r, status, badan)
}

// petakanGalat memetakan galat domain menjadi status dan badan respons.
//
// Nilai ketiga menyatakan apakah galatnya dikenali modul ini.
//
// Galat PORTAL sengaja tidak ada di sini. Ia dipetakan portalhttp.DenganGalatPortal
// yang membungkus penulis galat yang disuntikkan dari cmd — satu pemetaan yang dipakai
// seluruh modul bisnis, bukan satu tafsiran per modul.
func petakanGalat(err error) (int, ResponsGalat, bool) {
	var galatValidasi *statusprogres.GalatValidasi

	switch {
	case errors.As(err, &galatValidasi):
		// 422, bukan 400: bentuk permintaannya benar, isinya yang melanggar aturan
		// bisnis. Frontend menanganinya berbeda — 400 berarti ada cacat di frontend,
		// 422 berarti pengguna perlu memperbaiki isiannya (`10-API-STRATEGY.md` §5).
		//
		// SELURUH pelanggaran dikirim sekaligus, bukan yang pertama saja.
		detail := make([]DetailGalat, 0, len(galatValidasi.Pelanggaran))
		for _, p := range galatValidasi.Pelanggaran {
			detail = append(detail, DetailGalat{Kolom: p.Kolom, Pesan: p.Pesan})
		}
		return http.StatusUnprocessableEntity, ResponsGalat{
			Kode:   KodeValidasiGagal,
			Pesan:  "Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.",
			Detail: detail,
		}, true

	case errors.Is(err, statusprogres.ErrTidakDitemukan):
		return http.StatusNotFound, ResponsGalat{
			Kode:  KodeTidakDitemukan,
			Pesan: "Status progres yang dimaksud tidak ditemukan. Mungkin sudah dihapus petugas lain — muat ulang daftarnya.",
		}, true

	case errors.Is(err, statusprogres.ErrSudahAda):
		// 409, bukan 422: pengguna tidak melakukan kesalahan apa pun, dan tidak ada
		// isian yang dapat ia perbaiki. Yang terjadi adalah dua penambahan bersamaan.
		return http.StatusConflict, ResponsGalat{
			Kode:  KodeValidasiGagal,
			Pesan: "Nomor status progres baru sudah dipakai petugas lain. Coba simpan sekali lagi.",
		}, true

	default:
		return 0, ResponsGalat{}, false
	}
}
