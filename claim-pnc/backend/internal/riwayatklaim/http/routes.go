package riwayatklaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul View History Claim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Riwayat klaim satu badan hukum
// bukan riwayat badan hukum lain, dan jatah proteksi seorang pengguna di satu entitas
// bukan jatahnya di entitas lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti membaca nama tertanggung dan tanggal lahir
// satu badan hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi DAN gerbang proteksi data — yang kedua ditegakkan di usecase,
// bukan di middleware, karena ia bukan sekadar izin melainkan jatah yang berkurang.
//
// Yang BELUM ada adalah pemeriksaan peran: "apakah peran pemanggil memiliki menu ini"
// adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` — dan tabel itu
// dapat dibangun tetapi belum dapat diisi, karena penugasan operator ke peran tidak ada
// di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Taruhannya di layar ini termasuk yang tertinggi di aplikasi: satu pencarian dapat
// mengembalikan seluruh riwayat klaim seorang nasabah, lengkap dengan nama tertanggung
// dan tanggal lahir peserta. Itulah sebabnya gerbang proteksi data dibangun penuh
// (keputusan Work Owner 2026-09-20) alih-alih menunggu `TKT-F3-005`.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		// POST, karena membuka layar MEMAKAI satu jatah pencarian. Lihat Handler.Open.
		perPortal.Post("/riwayat-klaim/buka", h.Open)

		perPortal.Get("/riwayat-klaim", h.Search)
	})
}
