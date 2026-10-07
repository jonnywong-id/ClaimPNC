package outstandingclaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Outstanding Claim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena keduanya menyentuh basis data entitas. Isinya memuat nama tertanggung, nama Ceding
// Co, nilai klaim, dan pembagian reasuransi — seluruhnya milik satu badan hukum, bukan milik
// badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan
// — jatuh ke koneksi bawaan berarti menampilkan rincian klaim satu badan hukum kepada petugas
// badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute susunan layar (`/tata-letak`) ikut di balik pemeriksaan itu meski isinya sama di
// seluruh entitas. Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat
// alias portal, dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut
// portal akan membuat layar mengira ia sudah berada di portal yang benar.
//
// # Kenapa `/tata-letak` didaftarkan SEBELUM `/{no_klaim}`
//
// Karena `tata-letak` akan tertangkap sebagai sebuah nomor klaim bila urutannya terbalik, dan
// jawabannya bukan galat melainkan "klaim tidak ditemukan" — kegagalan yang terbaca seperti
// data hilang, bukan seperti rute yang bertabrakan.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran, dan di layar ini
// ketiadaannya berakibat nyata.
//
// Rincian TIDAK disaring menurut pemanggil — di Pega pun tidak, karena di sana layar ini
// hanya dapat dicapai lewat assignment yang sudah terbuka. Pada API yang dapat dipanggil
// langsung, jaminan itu hilang: nomor klaim treaty berurutan, sehingga siapa pun yang sudah
// masuk dapat menelusuri seluruh klaim treaty di portalnya satu per satu.
//
// Yang meredamnya bukan modul ini melainkan `TKT-F3-005`, yang belum ada. Sampai itu ada,
// setiap pembukaan DICATAT beserta pelakunya — dan pencatatan bukan kendali, hanya jejak.
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

		perPortal.Get("/outstanding-claim/tata-letak", h.Layout)
		perPortal.Get("/outstanding-claim/{no_klaim}", h.Detail)
	})
}
