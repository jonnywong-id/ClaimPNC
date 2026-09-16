// Package httpserver menyusun server HTTP aplikasi: middleware lintas modul, tempat
// modul memasang rutenya, dan penyajian berkas statis SPA.
//
// Paket ini TIDAK tahu apa pun tentang isi modul mana pun. Modul memasang rutenya
// sendiri lewat PasangAPI — itulah yang membuat modul dapat ditambah tanpa menyentuh
// berkas ini.
package httpserver

import (
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/middleware"
)

// Bahan adalah yang dibutuhkan server untuk berdiri.
type Bahan struct {
	Logger *slog.Logger

	// PasangAPI dipanggil dengan router yang sudah berada di bawah /api. Di sinilah
	// setiap modul mendaftarkan rutenya.
	PasangAPI func(chi.Router)

	// BerkasSPA adalah hasil build SPA. Bila nil, aplikasi hanya melayani API.
	BerkasSPA fs.FS
}

// Router menyusun seluruh rute: API di bawah /api, sisanya milik SPA.
func Router(b Bahan) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.IDPermintaan)
	r.Use(middleware.Pulih(b.Logger))
	r.Use(middleware.Log(b.Logger))

	if b.PasangAPI != nil {
		r.Route("/api", b.PasangAPI)
	}
	if b.BerkasSPA != nil {
		r.NotFound(spa(b.BerkasSPA))
	}
	return r
}

// spa melayani berkas statis hasil build React.
//
// Jalur yang tidak cocok dengan berkas mana pun dijawab dengan index.html, bukan 404:
// rute dalam seperti /klaim/123/estimasi adalah milik router di peramban, dan memuat
// ulang halaman pada rute itu harus tetap menampilkan halaman yang benar.
func spa(berkas fs.FS) http.HandlerFunc {
	pelayan := http.FileServer(http.FS(berkas))

	return func(w http.ResponseWriter, r *http.Request) {
		bersih := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if bersih == "." || bersih == "/" {
			bersih = "index.html"
		}
		if _, err := fs.Stat(berkas, bersih); err != nil {
			indeks, err := fs.ReadFile(berkas, "index.html")
			if err != nil {
				http.Error(w, "halaman tidak tersedia", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// Kerangka halaman tidak boleh di-cache: satu rilis baru harus langsung
			// terpakai tanpa pengguna menekan muat ulang paksa.
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(indeks)
			return
		}
		pelayan.ServeHTTP(w, r)
	}
}
