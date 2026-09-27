package daftartipedokumenbisnishttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Daftar Tipe Dokumen Bisnis.
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
// modul ini: setiap rutenya menyentuh basis data entitas. Tabelnya per entitas —
// `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:22` membentuk ID dari kode situs milik
// basis data tempat procedure itu berjalan — dan keempat master rujukannya pun hidup di
// basis data setiap entitas. Tidak ada satu pun rute yang boleh berada di luarnya.
//
// # Jalurnya
//
// `master/tipe-dokumen-bisnis` bukan pilihan baru: ia sudah DICADANGKAN sejak modul
// Daftar Tipe Dokumen dibangun, tertulis di
// `internal/daftartipedokumen/http/routes.go:39` sebagai jalur untuk
// LST_TYPE_DOC_BUSINESS. Memakainya menjaga ketiga nama — modul, folder, dan jalur API —
// tetap saling menunjuk.
//
// Awalan `master` menyatakan golongan data, bukan sekadar merapikan URL: seluruh master
// berbagi satu menu dan satu kewenangan, sehingga penegakan izin per menu (`D-59`) dapat
// dipasang pada satu tempat.
//
// Tanpa awalan `/v1`, mengikuti seluruh rute yang sudah ada. Penyeragamannya ke
// `10-API-STRATEGY.md` §2 dicatat sebagai utang teknis (TKT-F1-004), bukan diselesaikan
// sepihak di satu modul.
//
// # Kenapa daftar bisnis punya jalurnya sendiri, bukan memakai /master/bisnis
//
// `/master/bisnis` sudah ada dan dimiliki modul Master COL Simas Online. Mendaftarkannya
// kembali di sini akan membuat chi panik saat start — dua handler pada satu jalur — dan
// memanggil rute milik modul lain dari layar ini akan membuat satu modul bergantung pada
// rute modul lain yang dapat berubah kapan saja.
//
// Akhiran `-pilihan` mengikuti pola yang sudah dipakai
// `/master/dokumen-travel-pilihan` pada modul Daftar Detail Dokumen Travel, dan alasannya
// sama.
//
// # Kenapa keempat daftar pilihan berada DI LUAR sub-rute modul
//
// Karena keempatnya membaca tabel yang BUKAN milik modul ini — BUSINESS milik GISFW,
// V_LST_DOC_TYPE milik modul MENU_ID 40, V_LST_DET_TYPE_DOC milik MENU_ID 41, dan
// V_LST_DOC_OBJ milik MENU_ID 43. Menaruhnya di dalam `/master/tipe-dokumen-bisnis/...` akan
// menyiratkan kepemilikan yang justru sedang dijaga tidak terjadi (`P-1`).
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi dan portal, tetapi BELUM diperiksa perannya. Penegakan
// "apakah peran pemanggil memiliki menu Master Data" adalah TKT-F3-005, yang bergantung
// pada tabel peran TKT-F3-004 — dan tabel itu dapat dibangun tetapi belum dapat diisi,
// karena penugasan operator ke peran tidak ada di basis data maupun di export
// (`11-SECURITY.md` §3.1). Keadaan ini sama dengan seluruh rute lain yang sudah ada hari
// ini.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/bisnis-pilihan", h.BusinessChoices)
		perPortal.Get("/master/tipe-dokumen-pilihan", h.DocumentTypeChoices)
		perPortal.Get("/master/detail-dokumen-pilihan", h.DetailTypeDocChoices)
		// Ejaannya `objek`, bukan `object`, mengikuti modul Daftar Objek Dokumen yang
		// memiliki tabelnya dan sudah memasang `/master/objek-dokumen`. Label di layar
		// tetap berbunyi "Object Dokumen" seperti Pega (`D-13`) — yang diseragamkan di
		// sini jalur API-nya, bukan teks yang dibaca petugas.
		perPortal.Get("/master/objek-dokumen-pilihan", h.ObjectDocChoices)

		perPortal.Route("/master/tipe-dokumen-bisnis", func(master chi.Router) {
			master.Get("/", h.ListBusinesses)
			master.Post("/", h.Create)

			// Daftar per bisnis diletakkan di bawah awalan `bisnis` supaya ia tidak
			// bertabrakan dengan `/{id}`. Tanpa awalan itu, chi tidak dapat membedakan
			// "ambil aturan ber-ID ini" dari "ambil seluruh aturan milik bisnis ini" —
			// keduanya satu ruas jalur, dan yang terdaftar lebih dulu akan menelan
			// keduanya.
			master.Get("/bisnis/{businessID}", h.ListByBusiness)

			master.Get("/{id}", h.Get)
			master.Put("/{id}", h.Update)
			master.Post("/{id}/jenis-klaim", h.AddCoverage)
		})
	})
}
