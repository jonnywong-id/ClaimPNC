package masterpicteknikhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Master PIC Teknik ke router yang diberikan.
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
// menyentuh entitas — termasuk pencarian direktori, yang alamat layanannya pun dibaca per
// entitas. Permintaan tanpa portal ditolak, TIDAK PERNAH dialihkan ke portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kenapa "master/pic-teknik"
//
// Awalan master menyatakan golongan data, bukan sekadar merapikan URL: sekurang-kurangnya
// 29 kelompok master akan menyusul (`docs/Steering/05-DOMAIN-MODEL.md` §5), dan seluruhnya
// berbagi satu menu serta satu kewenangan. Mengelompokkannya sejak sekarang membuat
// penegakan izin per menu (`D-59`) dapat dipasang pada satu tempat, bukan ditambahkan satu
// per satu ke 29 rute yang sudah telanjur tersebar.
//
// Bentuknya mengikuti rute master yang sudah ada — `master/status-klaim`,
// `master/rekening`, `master/tipe-surveyor`.
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
// penugasan operator ke peran tidak ada di basis data maupun di export
// (`docs/Steering/11-SECURITY.md` §3.1). Keadaan ini sama dengan seluruh rute lain yang
// sudah ada hari ini, dan dicatat terbuka di docs/keputusan-implementasi.md.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/pic-teknik", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)

			// Pencarian direktori didaftarkan SEBELUM rute berparameter dan berada di
			// segmen tersendiri, sehingga "direktori" tidak pernah tertangkap sebagai
			// sebuah ID operator.
			master.Get("/direktori/{id}", h.Lookup)

			master.Get("/{id}", h.Get)

			// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali,
			// sehingga permintaannya menggantikan dan idempoten. Mengirim permintaan yang
			// sama dua kali menghasilkan keadaan akhir yang sama.
			master.Put("/{id}", h.Update)

			// DELETE sengaja TIDAK didaftarkan. Layar Pega hanya punya Tambah dan Refresh,
			// procedure lamanya hanya mengenal INSERT dan UPDATE, dan menghapus satu
			// petugas akan membuat setiap klaim yang menyimpan OPERATOR_ID-nya kehilangan
			// rujukan penugasan. Petugas yang berhenti dinonaktifkan, bukan dihapus. Rute
			// yang tidak ada tidak dapat dipanggil kode yang ditulis kemudian tanpa
			// keputusan sadar.
		})
	})
}
