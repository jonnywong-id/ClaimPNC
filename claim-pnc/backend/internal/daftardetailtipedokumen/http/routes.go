package daftardetailtipedokumenhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Daftar Detail Tipe Dokumen ke router yang
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
// modul ini: setiap rutenya menyentuh basis data entitas — termasuk rute daftar pilihan,
// karena keempat master yang dibacanya pun hidup di basis data setiap entitas. Tidak ada
// satu pun rute yang boleh berada di luarnya.
//
// # Kenapa jalurnya `detail-tipe-dokumen`, bukan nama modulnya utuh
//
// `D-81` menetapkan nama MODUL mengikuti nama yang disebut Work Owner — "Daftar Detail
// Tipe Dokumen", sama dengan MENU_DESC pada MENU_ID 41 — dan itulah nama foldernya. Nama
// JALUR mengikuti dua saudaranya yang sudah ada, supaya hubungan ketiganya terbaca dari
// URL-nya:
//
//	/master/tipe-dokumen           MENU_ID 40  induk
//	/master/detail-tipe-dokumen    MENU_ID 41  paket ini
//	/master/objek-dokumen          MENU_ID 43  master yang dirujuknya
//
// Awalan `master` menyatakan golongan data, bukan sekadar merapikan URL: seluruh master
// berbagi satu menu dan satu kewenangan, sehingga penegakan izin per menu (`D-59`) dapat
// dipasang pada satu tempat, bukan ditambahkan satu per satu ke 29 rute yang sudah
// telanjur tersebar.
//
// # Kenapa /master/bisnis TIDAK didaftarkan di sini
//
// Karena sudah dimiliki modul Master COL Simas Online, dan chi akan PANIK saat start bila
// dua modul mendaftarkan jalur yang sama. Daftar bisnis modul ini ikut di dalam
// `/pilihan` bersama ketiga master lain — bukan menembak `/master/bisnis` — supaya form
// cukup satu permintaan dan tidak menghadapi empat keadaan setengah-siap.
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

		perPortal.Route("/master/detail-tipe-dokumen", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)

			// Didaftarkan SEBELUM `/{id}` supaya maksudnya terbaca dari urutannya,
			// meski chi memang mendahulukan jalur tetap di atas jalur berparameter.
			// Tanpa kebiasaan itu, satu rute tetap yang kelak ditambahkan di bawah
			// `/{id}` akan tampak aman padahal bergantung pada perilaku pustaka.
			master.Get("/pilihan", h.References)

			master.Get("/{id}", h.Get)
			master.Put("/{id}", h.Update)
		})
	})
}
