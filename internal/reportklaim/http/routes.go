package reportklaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Report Klaim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Setiap rute di sini menyentuh basis data entitas — termasuk katalog, yang tampak
// seperti daftar tetap milik aplikasi. Ia memang tetap, tetapi membukanya tanpa portal
// berarti layar dapat dibuka sebelum entitasnya dipilih, lalu tombol Export-nya ditekan
// dan ditolak. Menolak lebih awal lebih jujur daripada menolak setelah pengguna mengisi
// rentang tanggal.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti mengunduh berkas berisi data nasabah satu
// badan hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Satu rute yang dicabut
//
// `/report-klaim/pilihan-bisnis` berdiri di sini sampai 2026-10-13, mengisi autocomplete
// "Bisnis" milik panel REPORT KLAIM PER BISNIS. Panelnya tidak ada di Pega produksi,
// sehingga rutenya ikut dicabut — bukan dibiarkan menjawab daftar kosong.
//
// Peringatan urutan yang dulu tertulis di sini ikut hilang bersamanya: ia menjaga agar
// rute satu segmen tidak terbaca sebagai kode laporan. Bila kelak ada rute satu segmen
// lagi, ia harus didaftarkan SEBELUM `/{kode}/ekspor` dengan alasan yang sama.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/report-klaim", h.Catalog)
		perPortal.Get("/report-klaim/{kode}/ekspor", h.Export)
	})
}
