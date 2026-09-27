package daftartipedokumenhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Daftar Tipe Dokumen ke router yang diberikan.
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
// modul ini: setiap rutenya menyentuh basis data entitas. Tidak ada satu pun rute yang
// boleh berada di luarnya — berbeda dari modul Master Status Progres, yang punya satu rute
// daftar posisi yang memang tidak membaca data entitas mana pun.
//
// # Kenapa "master/tipe-dokumen" dan bukan "master/daftar-tipe-dokumen"
//
// Jalur menyebut SUMBER DAYA-nya, bukan nama layarnya. "Daftar" pada nama menu menyatakan
// bentuk layar — sebuah daftar — dan bentuk itu sudah dinyatakan metode HTTP-nya. Awalan
// master menyatakan golongan data: sekurang-kurangnya 29 kelompok master akan menyusul
// (`05-DOMAIN-MODEL.md` §5), dan seluruhnya berbagi satu menu serta satu kewenangan.
// Mengelompokkannya sejak sekarang membuat penegakan izin per menu (`D-59`) dapat dipasang
// pada satu tempat, bukan ditambahkan satu per satu ke 29 rute yang sudah telanjur
// tersebar.
//
// JANGAN tertukar dengan dua master turunannya, yang kelak memakai jalurnya sendiri:
//
//	master/tipe-dokumen           <- modul ini, LST_DOC_TYPE
//	master/detail-tipe-dokumen        V_LST_DET_TYPE_DOC        (belum dibangun)
//	master/tipe-dokumen-bisnis        LST_TYPE_DOC_BUSINESS     (belum dibangun)
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi dan portal, tetapi BELUM diperiksa perannya. Penegakan "apakah
// peran pemanggil memiliki menu Master Data" adalah TKT-F3-005, yang bergantung pada tabel
// peran TKT-F3-004 — dan tabel itu dapat dibangun tetapi belum dapat diisi, karena
// penugasan operator ke peran tidak ada di basis data maupun di export (`11-SECURITY.md`
// §3.1). Keadaan ini sama dengan seluruh rute lain yang sudah ada hari ini, dan dicatat
// terbuka di docs/keputusan-implementasi.md.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/tipe-dokumen", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{id}", h.Get)
			master.Put("/{id}", h.Update)
		})
	})
}
