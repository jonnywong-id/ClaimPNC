package registrasihttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Mount mendaftarkan rute modul registrasi.
//
// Seluruh rutenya TERLINDUNGI: tidak satu pun bagian alur klaim boleh dijangkau tanpa
// sesi. Pemasangannya dilakukan pemanggil di dalam grup yang sudah memakai middleware
// autentikasi — sama seperti modul portal.
//
// # Yang BELUM ada di sini, dan konsekuensinya
//
// Pemeriksaan kewenangan per menu (`ADR-0023`, `TKT-F3-005`) belum dikerjakan karena
// tabel peran `TKT-F3-004` masih terhalang artefak. Akibatnya setiap pengguna yang punya
// sesi dapat memanggil seluruh rute di bawah ini. Ini utang yang disadari, bukan
// kelalaian: `ADR-0023` menuntut pemeriksaan di SETIAP endpoint, dan tempatnya sudah
// disiapkan — satu middleware di grup ini.
func Mount(r chi.Router, h *Handler) {
	r.Route("/registrasi", func(sub chi.Router) {
		// Definisi alur dibaca layar untuk menggambar jalur tahap. Ia tidak memuat data
		// klaim mana pun.
		sub.Get("/alur", h.Flow)

		sub.Get("/inbox", h.Inbox)

		sub.Post("/klaim", h.Start)
		sub.Get("/klaim/{klaimID}", func(w http.ResponseWriter, r *http.Request) {
			h.ViewClaim(w, r, chi.URLParam(r, "klaimID"))
		})

		// Tahap Input Register punya jalurnya sendiri karena ia membawa isian dan
		// gerbang validasi. Tahap lain ditutup lewat /tugas/{id}/selesai.
		sub.Post("/register", h.SaveRegister)

		sub.Post("/tugas/{tugasID}/ambil", func(w http.ResponseWriter, r *http.Request) {
			h.ClaimTask(w, r, chi.URLParam(r, "tugasID"))
		})
		sub.Post("/tugas/{tugasID}/selesai", func(w http.ResponseWriter, r *http.Request) {
			h.CompleteTask(w, r, chi.URLParam(r, "tugasID"))
		})
	})
}
