package inboxmanagerhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Manager.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas — dan satu di antaranya MENULISINYA.
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan: jatuh ke koneksi bawaan berarti menuliskan keputusan satu badan hukum ke basis
// data badan hukum lain, tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Pada modul baca-saja, akibat terburuk kekeliruan itu adalah data yang salah TERLIHAT. Di
// sini akibatnya data yang salah TERUBAH, dan itu tidak dapat dibatalkan dengan memuat ulang
// halaman.
//
// # Kewenangan
//
// Dua lapis, dan keduanya perlu dibedakan.
//
// Yang SUDAH ada: pembatasan satu tab menurut lini bisnis pemanggil, ditegakkan di server pada
// setiap permintaan lewat `inboxmanager.NewQuery` dan `inboxmanager.NewDecision` — bukan hanya
// dengan menyembunyikan tab di layar. Sistem lama hanya menyembunyikan kontainernya lewat
// `pyContainerVisibleWhen`, dan `11-SECURITY.md` §3.1 menyebut penyembunyian seperti itu
// sebagai kenyamanan tampilan, bukan kendali.
//
// Yang BELUM ada: pemeriksaan peran — "apakah peran pemanggil memiliki menu ini". Itu
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`, dan tabel itu dapat dibangun
// tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di basis data maupun
// di export.
//
// Akibat yang perlu disebut apa adanya: SIAPA PUN yang dapat membuka layar ini dapat
// menyetujui dan menolak. `D-59` menetapkan satuan izin adalah menu dan tidak ada pemisahan
// tugas formal, sehingga tidak ada kontrol teknis yang mencegah seseorang menyetujui
// pengajuan yang diajukannya sendiri.
//
// Yang mengimbanginya hanya satu, dan itu memang yang `D-59` tetapkan: SETIAP keputusan
// dicatat — siapa, portal apa, antrean apa, keputusan apa, dan kunci baris mana. Lihat
// usecase.logDecision.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-manager/tab", h.Metadata)
		perPortal.Get("/inbox-manager/ringkasan", h.Counters)
		perPortal.Get("/inbox-manager", h.List)

		// Ekspor adalah GET: ia tidak mengubah apa pun.
		perPortal.Get("/inbox-manager/ekspor", h.Export)

		// SATU-SATUNYA rute yang menulis di seluruh modul inbox.
		perPortal.Post("/inbox-manager/keputusan", h.Decide)
	})
}
