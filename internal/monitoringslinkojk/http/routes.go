package monitoringslinkojkhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Monitoring SLINK OJK (`MENU_ID 78`).
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Kewajiban lapor SLIK adalah
// kewajiban SATU badan hukum kepada OJK, dan barisnya memuat nomor CIF debitur, tanggal
// lahir, alamat, dan nomor telepon.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan kewajiban lapor satu badan
// hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — tabel peran adalah
// `TKT-F3-004`, yang dapat dibangun tetapi belum dapat diisi: penugasan operator ke peran
// tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Taruhannya terbatas selama modul ini membaca saja. Begitu aksi tulis dipindahkan ke
// sini, pemeriksaan peran menjadi prasyarat — layar ini menyusun dan mengirim laporan ke
// regulator, dan `D-59` menetapkan satuan izin adalah MENU tanpa pemisahan tugas formal.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// ============================================================================
// TIGA TOMBOL YANG MENULIS — DIBANGUN 2026-09-26
// ============================================================================
//
// Ketiganya sempat dinyatakan tidak dapat dibangun karena TIDAK terhubung aktivitas apa
// pun di `Sec_SegmentD01_1` — keempat tombol lain punya `<pyActivity>`, ketiga ini tidak.
//
// Itu benar tentang SECTION-nya, tetapi keliru sebagai kesimpulan: logikanya ada, hanya
// tidak di sana. Pemeriksaan ulang atas permintaan Work Owner menemukannya utuh:
//
//	Tombol              Logikanya dibaca dari
//	------------------  --------------------------------------------------------------
//	Proses Data Klaim   RDB List/InsertDataSlikOJKF06-SQL.xml     INSERT 28 kolom
//	                    RDB List/GetCountTClaimSlikOJK-SQL.xml    penentu 'C' versus 'U'
//	                    RDB List/GetDataSlinkAllFOG-SQL.xml       data klaim sumbernya
//	                    Activity/InsertAdjustmentListKredit-Act.xml  jalur akseptasi
//
//	Upload Data Klaim   Activity/PNCUploadAutoClaimSlikOJK-Act.xml   pemetaan nilai,
//	                    bermuara pada INSERT yang sama
//
//	SLIK OJK            RDB List/QuerySLINKIndividu-SQL.xml         nomor urut + INSERT
//	                    RDB List/UpdateTransactionClaimSlinkIndividu-SQL.xml  id transaksi
//
// # Satu bagian yang TETAP tidak dapat dibangun
//
// Panggilan REST keluar pada "SLIK OJK" — `Rest_SendDataClientBasedDebitur` — **nol
// kemunculan** di direktori `Connect REST/` (`R-16`). Ia berada di balik seam
// `monitoringslinkojk.Sender`; tanpa konfigurasi, tombolnya menolak dengan sebab yang
// terbaca dan TIDAK mencatat baris pengiriman palsu.
//
// # Konsekuensi `P-1` yang diterima secara sadar
//
// `POOLDATA.T_CLAIM_SLIK_OJK` juga diisi jalur akseptasi sistem lama. Sejak ketiga rute
// ini hidup, dua sistem menulis satu tabel — dan yang mencegah baris ganda hanyalah
// disiplin pemakaian. Satu pengaman yang dapat dibangun sudah dipasang, dan ia dari Pega
// sendiri: pencacah menandai klaim berulang sebagai `operasidata = 'U'`.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		// Keterangan layar: kedua segmen beserta katalog kolomnya, dan isi dropdown
		// "Business Name". Dipanggil sekali saat layar dibuka.
		perPortal.Get("/monitoring-slink-ojk/keterangan", h.Describe)

		// "Cari Data" — satu halaman grid pada segmen yang dipilih.
		perPortal.Get("/monitoring-slink-ojk/data", h.Search)

		// "Export Data" — seluruh baris yang cocok, dialirkan sebagai CSV.
		perPortal.Get("/monitoring-slink-ojk/ekspor", h.Export)

		// "Format File" — berkas contoh unggahan.
		perPortal.Get("/monitoring-slink-ojk/format", h.Template)

		// "Proses Data Klaim" — menyusun laporan dari data klaim sumber, mengikuti
		// penyaring yang sedang dipakai layar. Penyaringnya dibaca dari query string,
		// sama seperti "Cari Data", supaya yang tersusun persis yang terlihat.
		perPortal.Post("/monitoring-slink-ojk/proses", h.Process)

		// "Upload Data Klaim" — menyusun laporan dari berkas CSV unggahan.
		perPortal.Post("/monitoring-slink-ojk/unggah", h.Upload)

		// "SLIK OJK" — mengirim data debitur satu klaim ke sistem SLIK.
		//
		// Tanpa seam pengirim yang terkonfigurasi, ia menolak dengan sebab yang terbaca
		// dan TIDAK meninggalkan baris pengiriman palsu.
		perPortal.Post("/monitoring-slink-ojk/kirim", h.Submit)
	})
}
