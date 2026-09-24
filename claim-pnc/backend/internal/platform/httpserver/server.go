// Package httpserver menyusun server HTTP aplikasi: middleware lintas modul, tempat
// modul memasang rutenya, dan penyajian berkas statis SPA.
//
// Paket ini TIDAK tahu apa pun tentang isi modul mana pun. Modul memasang rutenya
// sendiri lewat PasangAPI — itulah yang membuat modul dapat ditambah tanpa menyentuh
// berkas ini.
package httpserver

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/middleware"
)

// Deps adalah yang dibutuhkan server untuk berdiri.
type Deps struct {
	Logger *slog.Logger

	// MountAPI dipanggil dengan router yang sudah berada di bawah /api. Di sinilah
	// setiap modul mendaftarkan rutenya.
	MountAPI func(chi.Router)

	// SPAFiles adalah hasil build SPA. Bila nil, aplikasi hanya melayani API.
	SPAFiles fs.FS
}

// Router menyusun seluruh rute: API di bawah /api, sisanya milik SPA.
func Router(b Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recover(b.Logger))
	r.Use(middleware.Log(b.Logger))

	if b.MountAPI != nil {
		r.Route("/api", func(api chi.Router) {
			// Jalur /api yang tidak cocok dijawab JSON 404 — BUKAN diteruskan ke SPA.
			//
			// Tanpa baris ini, `r.NotFound(spa(...))` di bawah ikut berlaku di dalam /api,
			// sehingga alamat API yang salah ketik atau belum terdaftar dijawab
			// `index.html` dengan status **200**. Bagi peramban itu jawaban yang berhasil;
			// bagi klien API ia badan yang tidak dapat diurai.
			//
			// Akibatnya bukan galat yang terbaca, melainkan LAYAR KOSONG: `callAPI`
			// mengembalikan `null` untuk 200 yang badannya bukan JSON, dan layar yang
			// menunggu datanya tidak menampilkan apa pun — bukan memuat, bukan galat,
			// bukan data. Itu pernah terjadi pada layar kerja RCL/PUCL, dan tidak ada satu
			// pun petunjuk yang dapat dilaporkan pengguna.
			api.NotFound(apiNotFound)
			api.MethodNotAllowed(apiMethodNotAllowed)

			b.MountAPI(api)
		})
	}
	if b.SPAFiles != nil {
		r.NotFound(spa(b.SPAFiles))
	}
	return r
}

// apiNotFound menjawab alamat /api yang tidak terdaftar.
//
// Bentuk badannya sama dengan galat modul mana pun — `kode` dan `pesan` — supaya klien
// memperlakukannya lewat jalur yang sama, tanpa perkecualian.
func apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, http.StatusNotFound, "rute_tidak_ditemukan",
		"Alamat API tidak dikenal: "+r.URL.Path)
}

// apiMethodNotAllowed menjawab alamat yang ada tetapi metodenya tidak dilayani.
//
// Dipisah dari apiNotFound karena keduanya menuntut perbaikan yang berbeda: yang satu
// alamatnya salah, yang satu metodenya. Menjawab keduanya "tidak ditemukan" membuat
// pemanggil mencari alamat yang sebenarnya sudah benar.
func apiMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeAPIError(w, http.StatusMethodNotAllowed, "metode_tidak_dilayani",
		"Metode "+r.Method+" tidak dilayani untuk "+r.URL.Path)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// Galat penulisan diabaikan dengan sengaja: badan respons sudah dimulai, sehingga tidak
	// ada lagi yang dapat dikirimkan ke pemanggil. Yang tersisa hanyalah sambungan yang
	// terputus, dan itu sudah tercatat middleware log.
	_ = json.NewEncoder(w).Encode(map[string]string{"kode": code, "pesan": message})
}

// spa melayani berkas statis hasil build React.
//
// Jalur yang tidak cocok dengan berkas mana pun dijawab dengan index.html, bukan 404:
// rute dalam seperti /klaim/123/estimasi adalah milik router di peramban, dan memuat
// ulang halaman pada rute itu harus tetap menampilkan halaman yang benar.
func spa(files fs.FS) http.HandlerFunc {
	pelayan := http.FileServer(http.FS(files))

	return func(w http.ResponseWriter, r *http.Request) {
		bersih := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if bersih == "." || bersih == "/" {
			bersih = "index.html"
		}
		// Kerangka halaman tidak boleh di-cache: satu rilis baru harus langsung terpakai
		// tanpa pengguna menekan muat ulang paksa.
		//
		// Header ini dipasang SEBELUM percabangan, bukan hanya di dalamnya. Sebelumnya ia
		// terpasang pada jalur cadangan saja — sedangkan permintaan ke "/" menemukan
		// index.html sebagai berkas nyata, sehingga jalur yang PALING SERING dipakai
		// justru satu-satunya yang melewatinya. Akibatnya peramban dapat menahan kerangka
		// halaman versi lama meski binary sudah dibangun ulang, dan fitur yang sudah
		// diperbaiki tampak masih rusak tanpa satu pun galat.
		//
		// Berkas aset TIDAK ikut: namanya sudah memuat sidik isi (`index-<hash>.js`),
		// sehingga rilis baru menghasilkan nama baru dan cache-nya tidak pernah basi.
		if bersih == "index.html" {
			w.Header().Set("Cache-Control", "no-store")
		}

		if _, err := fs.Stat(files, bersih); err != nil {
			index, err := fs.ReadFile(files, "index.html")
			if err != nil {
				http.Error(w, "halaman tidak tersedia", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(index)
			return
		}
		pelayan.ServeHTTP(w, r)
	}
}
