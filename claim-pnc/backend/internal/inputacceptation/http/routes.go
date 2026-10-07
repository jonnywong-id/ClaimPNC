package inputacceptationhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Acceptation Claim.
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
// Co, nilai klaim, pembagian reasuransi, dan Nomor Akseptasi — seluruhnya milik satu badan
// hukum, bukan milik badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan.
// Pada modul ini akibatnya dua kali lipat dibanding modul yang hanya membaca: jatuh ke koneksi
// bawaan berarti MENULISI akseptasi milik badan hukum yang salah (`R-20`, `TKT-F6-002`).
//
// # Kenapa tidak ada rute `/tata-letak` tersendiri
//
// Karena layar ini selalu dibuka untuk satu klaim tertentu — tidak ada keadaan di mana bentuk
// layarnya dibutuhkan tanpa isinya. Modul Outstanding Claim memisahkannya; di sini keduanya
// dikirim bersama, sehingga satu layar = satu perjalanan dan keduanya tidak dapat menjawab
// keadaan yang berbeda.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran, dan di layar ini
// ketiadaannya berakibat nyata.
//
// Rincian TIDAK disaring menurut pemanggil — di Pega pun tidak, karena di sana layar ini hanya
// dapat dicapai lewat assignment yang sudah terbuka. Pada API yang dapat dipanggil langsung,
// jaminan itu hilang: nomor klaim non-prop berurutan, sehingga siapa pun yang sudah masuk
// dapat menelusuri seluruh akseptasi klaim treaty non-prop di portalnya satu per satu.
//
// Yang meredamnya bukan modul ini melainkan `TKT-F3-005`, yang belum ada. Sampai itu ada,
// setiap pembukaan dan setiap Submit DICATAT beserta pelakunya — dan pencatatan bukan kendali,
// hanya jejak.
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

		perPortal.Get("/input-acceptation/{no_klaim}", h.Detail)
		perPortal.Post("/input-acceptation/{no_klaim}", h.Submit)
	})
}
