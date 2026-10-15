package inboxclaimtreatynonprophttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Claim Treaty Non Prop.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Barisnya memuat nama tertanggung dan
// nama Ceding Co — perusahaan asuransi yang mengalihkan risikonya — dan keduanya milik satu
// badan hukum, bukan milik badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan antrean satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu meski isinya sama di seluruh
// entitas. Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias
// portal, dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal
// akan membuat layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — dan di layar ini
// ketiadaannya berakibat nyata, bukan sekadar kurang rapi.
//
// Sistem lama memisahkan ketiga antrean ini dengan KEADAAN pemanggil, bukan dengan pilihan
// pengguna. `Activity/GetWorkCNP_Act-Act.xml` menyetel `InputData.CARI13 = "tampil"` — yang
// membuka bagian komite — lewat DUA jalan:
//
//	menjalankan report `FilterEmailKomiteWithLimit` atas
//	`ASM-FW-GCNMFW-Int-EMAILKOMITE`, yaitu memeriksa apakah pemanggil terdaftar sebagai
//	anggota komite di POOLDATA.EMAILKOMITE;
//
//	membandingkan `OperatorID.pyUserIdentifier` dengan SATU NAMA yang ditanam di dalam
//	rule.
//
// Jalan kedua adalah hardcode yang `D-15` tetapkan menjadi master data, dan ia TIDAK dibawa
// — namanya pun tidak disalin ke repo ini. Jalan pertama tidak dapat ditegakkan hari ini:
// tabel peran adalah `TKT-F3-004`, yang dapat dibangun tetapi belum dapat diisi karena
// penugasan operator ke peran tidak ada di basis data maupun di export
// (`11-SECURITY.md` §3.1).
//
// Sampai itu ada, setiap pengguna yang dapat masuk melihat ketiga tab. Taruhannya terbatas
// selama modul ini membaca saja — antrean teknik yang terlihat bukan antrean teknik yang
// dapat diambil, dan tab komite belum dapat diisi sama sekali. Begitu aksi pengambilan
// pekerjaan dipindahkan ke sini, pemeriksaan peran menjadi prasyarat, bukan pelengkap.
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

		perPortal.Get("/inbox-claim-treaty-non-prop/tab", h.Metadata)
		perPortal.Get("/inbox-claim-treaty-non-prop", h.List)

		// Ekspor adalah GET, bukan POST, meski di sistem lama ia tombol yang menjalankan
		// activity. Ia tidak mengubah apa pun, dan menjadikannya GET membuat unduhannya
		// dapat dipicu tautan biasa — termasuk dibuka ulang dari riwayat peramban dengan
		// penyaring yang sama.
		perPortal.Get("/inbox-claim-treaty-non-prop/ekspor", h.Export)

		// Aksi tulis sistem lama. Rutenya ADA supaya tombolnya menjawab dengan alasan,
		// bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Ia menulis objek kerja lewat `Activity/CreateClaimTNonProp_Act-Act.xml`, di
		// tabel yang selama masa paralel masih dimiliki Pega (`P-1`).
		perPortal.Post("/inbox-claim-treaty-non-prop/klaim", h.RejectWrite)
	})
}
