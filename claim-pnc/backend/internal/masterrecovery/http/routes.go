package masterrecoveryhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Master Recovery ke router yang diberikan.
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
// Rute unduh format pun ikut di balik pemeriksaan portal, meski isinya tidak bergantung
// pada entitas mana pun. Itu disengaja: satu rute yang lolos tanpa portal akan menjadi
// contoh yang ditiru rute berikutnya, dan pengecualian yang sekali diberikan sulit
// ditarik kembali.
//
// # Kenapa "master/recovery"
//
// Awalan master menyatakan golongan data, mengikuti rute master yang sudah ada —
// `master/status-klaim`, `master/rekening`, `master/tipe-surveyor`, `master/pic-teknik`.
// Butir menunya pun berada di bawah kelompok MASTER (`MENU_ID 16`, induk `MENU_ID 1`).
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
//
// Ia perlu disebut khusus di modul ini karena rutenya MENERBITKAN REKENING VIRTUAL dan
// MENCATAT NILAI UANG — dua hal yang, tanpa pemeriksaan peran, terbuka bagi setiap
// pengguna yang berhasil masuk.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/recovery", func(recovery chi.Router) {
			// Bekal awal layar: nomor batch perkiraan dan pilihan tahun, dalam satu
			// permintaan. Satu layar sebaiknya dilayani satu permintaan
			// (`10-API-STRATEGY.md` §1).
			recovery.Get("/form", h.Form)

			recovery.Get("/principal", h.Principals)
			recovery.Get("/polis/{nomor}", h.Policy)
			recovery.Get("/format-unggahan", h.Template)

			// POST, bukan GET, meski ia "mencari": penerbitan VA MENIMBULKAN AKIBAT di luar
			// sistem — rekening nyata terbentuk di bank. GET wajib aman diulang, dan
			// peramban maupun proxy bebas mengulangnya sendiri.
			recovery.Post("/virtual-account", h.IssueVirtualAccount)

			recovery.Post("/bukti-bayar", h.UploadDocument)

			// Pembacaan CSV memakai POST karena berkasnya dikirim di badan permintaan,
			// meski ia tidak menyimpan apa pun.
			recovery.Post("/baris-klaim", h.ReadClaimLine)

			recovery.Post("/", h.Save)

			// GET "/" sengaja TIDAK didaftarkan.
			//
			// Tidak ada satu pun kueri di export Pega yang membaca
			// POOLDATA.MST_RECOVERY_ASM_PENJAMINAN — layarnya form entri, bukan daftar.
			// Keputusan Work Owner 2026-09-19 menetapkan itu ditiru apa adanya.
			//
			// PUT dan DELETE juga tidak ada, dengan alasan yang sama: procedure lamanya
			// hanya mengenal INSERT. Rute yang tidak ada tidak dapat dipanggil kode yang
			// ditulis kemudian tanpa keputusan sadar.
		})
	})
}
