package masterxolhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Master XOL ke router yang diberikan.
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
// # Kenapa "master/xol"
//
// Awalan master menyatakan golongan data, mengikuti rute master yang sudah ada —
// `master/status-klaim`, `master/rekening`, `master/recovery`. Butir menunya pun berada
// di bawah kelompok MASTER (`MENU_ID 19`, induk `MENU_ID 1`).
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
// (`docs/Steering/11-SECURITY.md` §3.1). Keadaan ini sama dengan seluruh rute lain yang
// sudah ada hari ini, dan dicatat terbuka di docs/keputusan-implementasi.md.
//
// Ia perlu disebut khusus di modul ini karena menyimpan di sini **mengajukan struktur
// treaty ke komite** — sebuah tindakan yang di sistem lama pun tidak dibatasi apa pun
// selain penyembunyian menu.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/xol", func(xol chi.Router) {
			// Kedua rute statis didaftarkan lebih dulu supaya terbaca sebagai apa adanya,
			// bukan sebagai nilai {id}. chi memang mendahulukan ruas statis atas ruas
			// berparameter, tetapi urutannya ditulis begini agar terbaca manusia juga.
			xol.Get("/form", h.Form)
			xol.Get("/bisnis", h.BusinessGroup)

			xol.Get("/", h.List)
			xol.Post("/", h.Create)

			xol.Get("/{id}", h.Get)
			xol.Put("/{id}", h.Update)

			// DELETE tingkat induk BERKASKADE — bisnis, lapisan, dan reas ikut terhapus.
			// Sistem lama tidak berkaskade dan meninggalkan baris yatim di produksi;
			// perbaikannya adalah keputusan Work Owner 2026-09-20.
			xol.Delete("/{id}", h.DeleteMaster)

			// Ketiga rute hapus di bawah meniru tombol Hapus per baris pada grid, yang di
			// sistem lama menghapus SEKETIKA lewat `DeleteFromTabelMst` — bukan menunggu
			// tombol Simpan. Perilaku itu dipertahankan: Simpan bersifat upsert dan tidak
			// pernah menghapus baris yang tidak disebut.
			xol.Delete("/{id}/bisnis/{idBisnis}", h.DeleteBusiness)
			xol.Delete("/{id}/layer/{idLayer}", h.DeleteLayer)
			xol.Delete("/{id}/layer/{idLayer}/reas/{idReas}", h.DeleteReinsurer)
		})
	})
}
