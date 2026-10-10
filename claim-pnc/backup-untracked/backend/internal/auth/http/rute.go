package authhttp

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
)

// Pasang mendaftarkan seluruh rute modul auth ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Itulah yang membuat modul dapat ditambah tanpa menyentuh
// server, dan kelak dipisahkan tanpa membongkarnya.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di
// cmd/claimpnc ia dipasang di bawah /api.
func Pasang(r chi.Router, h *Handler, pemeriksa PemeriksaSesi, logger *slog.Logger) {
	tulisGalat := TulisGalat(logger)

	// Dua rute terbuka: masuk belum punya sesi, dan keluar harus tetap bekerja walau
	// sesinya sudah tidak sah — menolak permintaan keluar tidak menolong siapa pun.
	r.Post("/masuk", h.Masuk)
	r.Post("/keluar", h.Keluar)

	r.Group(func(terlindungi chi.Router) {
		terlindungi.Use(Autentikasi(pemeriksa, tulisGalat))
		terlindungi.Get("/saya", h.Saya)
		terlindungi.Post("/sesi/perpanjang", h.Perpanjang)
	})
}
