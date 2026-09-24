package inboxlaporanklaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Laporan Klaim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Berbeda dari modul master — yang sebagian rutenya melayani daftar milik aplikasi dan
// karena itu berada di luar pemeriksaan portal — setiap rute di sini menyentuh basis data
// entitas. Bahkan daftar pilihan dropdown pun: isi "Pilih Kanwil" dibaca dari
// POOLDATA.BRANCH milik entitas yang bersangkutan, dan cabang satu badan hukum bukan
// cabang badan hukum lain.
//
// Permintaan tanpa header portal karena itu DITOLAK, tidak pernah dilayani portal utama
// sebagai cadangan — jatuh ke koneksi bawaan berarti membaca berkas laporan satu badan
// hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Urutan pendaftaran
//
// `/ekspor` dan `/pilihan` didaftarkan SEBELUM `/{id}`. Tanpa itu keduanya terbaca
// sebagai nomor berkas, dan tombol Export akan menjawab "laporan tidak ditemukan".
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox/laporan-klaim", h.List)
		perPortal.Post("/inbox/laporan-klaim", h.Create)
		perPortal.Get("/inbox/laporan-klaim/pilihan", h.Options)
		perPortal.Get("/inbox/laporan-klaim/ekspor", h.Export)
		perPortal.Get("/inbox/laporan-klaim/{id}", h.Get)
		perPortal.Put("/inbox/laporan-klaim/{id}", h.Save)
	})
}
