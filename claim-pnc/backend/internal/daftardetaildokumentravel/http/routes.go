package daftardetaildokumentravelhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Daftar Detail Dokumen Travel ke router yang
// diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Authenticate. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// Middleware ActivePortal dipasang DI SINI, bukan di cmd, karena ia melekat pada sifat
// modul ini: setiap rutenya menyentuh basis data entitas — termasuk kedua rute daftar
// pilihan, karena M_DOCTRAVEL dan M_PLANTRAVEL pun hidup di basis data setiap entitas.
// Tidak ada satu pun rute yang boleh berada di luarnya.
//
// # Kenapa jalurnya sepanjang itu
//
// `D-81` menetapkan nama modul mengikuti nama yang disebut Work Owner, dan nama itu
// adalah "Daftar Detail Dokumen Travel" — sama dengan MENU_DESC pada MENU_ID 39. Jalur
// yang lebih pendek akan membuat nama modul, nama folder, dan jalur API tidak lagi
// saling menunjuk, dan pada aplikasi yang akan memuat sekurang-kurangnya 29 master,
// ketiganya harus dapat ditebak dari satu sama lain.
//
// Awalan `master` menyatakan golongan data, bukan sekadar merapikan URL: seluruh master
// berbagi satu menu dan satu kewenangan, sehingga penegakan izin per menu (`D-59`) dapat
// dipasang pada satu tempat, bukan ditambahkan satu per satu ke 29 rute yang sudah
// telanjur tersebar.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi dan portal, tetapi BELUM diperiksa perannya. Penegakan
// "apakah peran pemanggil memiliki menu Master Data" adalah TKT-F3-005, yang bergantung
// pada tabel peran TKT-F3-004 — dan tabel itu dapat dibangun tetapi belum dapat diisi,
// karena penugasan operator ke peran tidak ada di basis data maupun di export
// (`11-SECURITY.md` §3.1). Keadaan ini sama dengan seluruh rute lain yang sudah ada hari
// ini, dan dicatat terbuka di docs/keputusan-implementasi.md.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		// Kedua daftar pilihan berada di luar sub-rute modul, sejajar dengan
		// `/master/bisnis` milik modul Master COL Simas Online. Sebabnya bukan estetika:
		// keduanya membaca tabel yang BUKAN milik modul ini, dan menaruhnya di dalam
		// `/master/daftar-detail-dokumen-travel/...` akan menyiratkan kepemilikan yang
		// justru sedang dijaga tidak terjadi (`P-1`).
		perPortal.Get("/master/dokumen-travel-pilihan", h.Documents)
		perPortal.Get("/master/plan-travel", h.Plans)

		perPortal.Route("/master/daftar-detail-dokumen-travel", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{id}", h.Get)
			master.Put("/{id}", h.Update)
		})
	})
}
