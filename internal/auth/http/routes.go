package authhttp

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
)

// Mount mendaftarkan seluruh rute modul auth ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Itulah yang membuat modul dapat ditambah tanpa menyentuh
// server, dan kelak dipisahkan tanpa membongkarnya.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di
// cmd/claimpnc ia dipasang di bawah /api.
func Mount(r chi.Router, h *Handler, pemeriksa SessionChecker, logger *slog.Logger) {
	writeError := WriteError(logger)

	// Dua rute terbuka: masuk belum punya sesi, dan keluar harus tetap bekerja walau
	// sesinya sudah tidak sah — menolak permintaan keluar tidak menolong siapa pun.
	r.Post("/masuk", h.Login)
	r.Post("/keluar", h.Logout)

	r.Group(func(protected chi.Router) {
		protected.Use(Authenticate(pemeriksa, writeError))
		protected.Get("/saya", h.Me)
		protected.Post("/sesi/perpanjang", h.Renew)
	})
}
