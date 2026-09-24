package inboxoutstandinghttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Outstanding.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc
// ia dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa jalurnya "inbox-outstanding"
//
// Itu nama yang dibaca pengguna di menu portal Pega, dan judul yang tertulis di dalam
// layar rujukannya sendiri (`Section/InboxRegister_Section-Section.xml:2150`).
//
// # Hanya GET, dan itu bukan kelalaian
//
// Modul ini MEMBACA. Klaim dimiliki modul `registrasi`, dan `P-1` menetapkan satu tabel
// ditulis satu sistem. Rute yang tidak didaftarkan dijawab chi dengan 405, sehingga
// penambahan rute tulis kelak menjadi keputusan sadar — bukan sesuatu yang lolos review.
//
// Layar rujukan memang tidak punya aksi yang menulis: tombolnya hanya **Cari**,
// **Export To Excel**, dan **Input Claim** (`CreateInputKlaim`, `ExportDataDetailKlaim`,
// `ExportLostAdjuster`). "Transfer" dan "Change New User" ada di section lain — yang
// dipakai dashboard — dan bukan bagian modul ini.
//
// Tombol **Input Claim** belum dibangun: ia membuka alur registrasi, milik modul lain.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya — keadaan yang sama dengan
// seluruh rute lain hari ini. Penegakan "apakah peran pemanggil memiliki menu ini" adalah
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`; tabel itu dapat dibangun
// tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di basis data
// maupun di export (`11-SECURITY.md` §3.1).
//
// Taruhannya di sini nyata: layar ini menampilkan SELURUH klaim yang masih berjalan,
// beserta nama tertanggung dan nomor polisnya. Yang membatasi apa yang terlihat hanyalah
// batas data per lini bisnis — dan batas itu sendiri belum berlaku bagi pengguna yang
// kolom LINEBUSINESS-nya belum diisi. Dicatat terbuka di docs/keputusan-implementasi.md.
// # Seluruh rutenya menuntut portal aktif
//
// `T_CLAIMLIST_ADMIN` ada di basis data setiap entitas (`ADR-0030`), sehingga setiap permintaan
// harus menyebut entitas mana yang dibacanya lewat header `X-Portal`. Permintaan yang
// tidak menyebutkannya DITOLAK oleh middleware — tidak pernah dilayani portal utama
// sebagai cadangan, karena jatuh ke koneksi default berarti menampilkan klaim satu badan
// hukum di layar badan hukum lain tanpa satu pun galat (`R-20`, `TKT-F6-002`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/inbox-outstanding", func(outstanding chi.Router) {
		outstanding.Use(portalhttp.ActivePortal(portalDeps))

		outstanding.Get("/", h.List)

		// Unduhan dipisahkan menjadi jalurnya sendiri, bukan parameter `format=csv` pada
		// daftar. Keduanya berbeda sifat: yang satu dipaginasi dan dibaca layar, yang lain
		// mengalir sampai habis dan diterima sebagai berkas. Menyatukannya membuat satu
		// endpoint punya dua bentuk respons dan dua batas ukuran.
		outstanding.Get("/unduh", h.Export)
	})
}
