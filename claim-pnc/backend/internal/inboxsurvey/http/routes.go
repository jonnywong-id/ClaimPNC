package inboxsurveyhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul My Work (MENU_ID 50).
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah cmd/claimpnc.
//
// Tuntutan itu LEBIH KERAS di modul ini daripada di kebanyakan inbox. Antreannya tidak
// sekadar disaring identitas — identitas itu diterjemahkan lebih dulu menjadi NAMA SURVEYOR
// lewat `POOLDATA.MST_LOGIN_SURVEYOR`, dan nama itulah yang menentukan baris siapa yang
// tampil. Sesi yang tidak terbaca karena itu bukan menghasilkan layar yang kurang lengkap,
// melainkan layar yang tidak punya isi sama sekali.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Termasuk jembatan identitasnya: modul
// ini membaca `MST_LOGIN_SURVEYOR` dari portal yang sedang aktif, sehingga portal yang salah
// akan memetakan pemanggil ke surveyor bernama sama di entitas lain — lalu SELURUH antrean
// yang muncul sesudahnya milik entitas itu, tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan.
//
// Rute keterangan layar ikut di balik pemeriksaan itu meski isinya sama di seluruh entitas.
// Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias portal, dan
// mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal akan membuat
// layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi dan disaring identitas surveyor. Yang BELUM ada adalah pemeriksaan
// peran: "apakah peran pemanggil memiliki menu ini" adalah `TKT-F3-005`, yang bergantung pada
// tabel peran `TKT-F3-004` — dan tabel itu dapat dibangun tetapi belum dapat diisi, karena
// penugasan operator ke peran tidak ada di basis data maupun di export (`11-SECURITY.md`
// §3.1).
//
// Di sistem lama, butir menu ini dijaga `POOLDATA.M_OTORISASI_PNC`, yang pada data contoh
// hanya memuat satu baris untuk MENU_ID 50 — grup `IT`. Aturan itu belum ditegakkan di sini.
// Yang meredam akibatnya untuk sementara adalah jembatan identitas: pengguna yang tidak
// terdaftar sebagai surveyor menerima 403 yang MENYEBUTKAN sebabnya, bukan antrean kosong.
// Itu peredam, bukan kendali — dan perbedaannya penting.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Kenapa empat rute, bukan satu
//
//	/keterangan   bentuk layar        tidak menyentuh basis data sama sekali
//	/jumlah-tab   bilah tab           tujuh penjumlahan, tidak berubah saat halaman berpindah
//	/kpi          tab KPI             tabel LAIN (`DETAIL_KPI_ADJUSTER`)
//	(akar)        daftar antrean      satu halaman satu tab
//
// Menyatukannya akan membuat setiap penekanan tombol halaman ikut menjalankan tujuh
// penjumlahan tab dan satu ringkasan KPI — tiga pekerjaan untuk satu yang diminta.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-survey/keterangan", h.Metadata)
		perPortal.Get("/inbox-survey/jumlah-tab", h.Counts)
		perPortal.Get("/inbox-survey/kpi", h.KPI)
		perPortal.Get("/inbox-survey", h.List)
	})
}
