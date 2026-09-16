// Package spa menyematkan hasil build antarmuka ke dalam binary.
//
// SPA adalah singkatan dari **Single Page Application** — aplikasi satu halaman, yaitu
// bentuk antarmuka yang ditetapkan ADR-0002: React + TypeScript + Vite yang dikompilasi
// menjadi berkas statis.
//
// # Isi paket ini
//
// Satu berkas Go dan satu folder `dist/` berisi hasil `npm run build`. **Tidak ada satu
// baris pun kode React di sini** — seluruh kode sumber antarmuka ada di folder
// `frontend/` di luar modul Go ini.
//
// # Kenapa letaknya di backend, bukan di frontend
//
// Direktif `go:embed` tidak dapat menjangkau ke luar direktori paketnya: pola yang
// memuat `../` ditolak kompilator sebagai `invalid pattern syntax`. Padahal ADR-0002
// menuntut produksi menjalankan **satu proses saja** — binary Go ini — tanpa runtime
// Node.js. Agar berkas statis ikut masuk ke dalam binary, folder hasil build-nya harus
// berada di dalam modul Go. Karena itu `npm run build` di `frontend/` menulis keluaran
// ke sini, bukan ke folder frontend sendiri.
//
// Alurnya: `frontend/src` → `npm run build` → `backend/spa/dist` → `go build` → satu
// binary.
package spa

import (
	"embed"
	"errors"
	"io/fs"
)

//go:embed all:dist
var tersemat embed.FS

// ErrBelumDibangun menandai binary yang dikompilasi tanpa hasil build antarmuka.
//
// Ini bukan kegagalan fatal: aplikasi tetap dapat melayani API. Yang tidak tersedia
// hanyalah halamannya, dan pemanggil memilih sendiri apa yang dilakukan terhadap itu.
var ErrBelumDibangun = errors.New("spa: hasil build antarmuka tidak ditemukan; jalankan npm run build di folder frontend")

// Berkas mengembalikan berkas antarmuka siap sajikan, berakar di dist.
func Berkas() (fs.FS, error) {
	akar, err := fs.Sub(tersemat, "dist")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(akar, "index.html"); err != nil {
		return nil, ErrBelumDibangun
	}
	return akar, nil
}
