package mastersurveyorshttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Master Surveyors ke router yang diberikan.
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
// Middleware ActivePortal dipasang DI SINI, bukan di cmd, karena seluruh rute modul ini
// menyentuh basis data entitas — tidak ada satu pun yang boleh dilayani tanpa portal.
// Permintaan tanpa portal ditolak, TIDAK PERNAH dialihkan ke portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kenapa "master/surveyor" dan bukan "surveyor"
//
// Awalan master menyatakan golongan data, bukan sekadar merapikan URL: sekurang-kurangnya
// 29 kelompok master akan menyusul (`docs/Steering/05-DOMAIN-MODEL.md` §5), dan seluruhnya
// berbagi satu menu serta satu kewenangan. Mengelompokkannya sejak sekarang membuat
// penegakan izin per menu (`D-59`) dapat dipasang pada satu tempat.
//
// Bentuknya TUNGGAL (`surveyor`), bukan jamak, mengikuti rute master yang sudah ada —
// `master/status-klaim`, `master/rekening`, `master/tipe-surveyor`. Nama MODULNYA tetap
// "Master Surveyors" sesuai butir menu `MENU_ID 15`; yang diseragamkan hanya jalurnya.
//
// Perhatikan bedanya dari tetangganya: `master/tipe-surveyor` adalah TIPE-nya,
// `master/surveyor` adalah ORANG-nya. Keduanya hidup berdampingan dan tidak boleh
// tertukar.
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
// Rutenya TERLINDUNGI sesi dan portal, dan keputusan komite diperiksa terhadap kolom
// KOMITE di usecase. Yang BELUM ada adalah pemeriksaan peran — "apakah peran pemanggil
// memiliki menu Master Data" adalah TKT-F3-005, yang bergantung pada tabel peran
// TKT-F3-004; tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan operator
// ke peran tidak ada di basis data maupun di export (`docs/Steering/11-SECURITY.md` §3.1).
//
// Keadaan ini sama dengan seluruh rute lain yang sudah ada hari ini, dan dicatat terbuka
// di docs/keputusan-implementasi.md.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/surveyor", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{id}", h.Get)
			master.Put("/{id}", h.Update)

			// Keputusan komite adalah PERISTIWA, bukan pembaruan field — karena itu ia
			// sub-sumber daya tersendiri, bukan PUT ke induknya.
			master.Post("/{id}/keputusan", h.Decide)
		})
	})
}
