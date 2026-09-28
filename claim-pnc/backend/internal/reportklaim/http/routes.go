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
// # Urutan pendaftaran
//
// `/pilihan-bisnis` didaftarkan SEBELUM `/{kode}/ekspor`. Keduanya tidak benar-benar
// bertabrakan karena bentuknya berbeda, tetapi urutan ini menjaga agar penambahan rute
// satu segmen berikutnya tidak diam-diam terbaca sebagai kode laporan.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/report-klaim", h.Catalog)
		perPortal.Get("/report-klaim/pilihan-bisnis", h.BusinessOptions)
		perPortal.Get("/report-klaim/{kode}/ekspor", h.Export)
	})
}
