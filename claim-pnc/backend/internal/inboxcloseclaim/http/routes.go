package inboxcloseclaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Close Claim.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc ia
// dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa jalurnya "inbox-close-claim"
//
// Itu nama butir menunya di `Database/m_menu_aplikasi_pnc.csv:59`, dan judul yang tertulis
// di dalam layar rujukannya sendiri (`Section/InboxManagerReopen1_Sec-Section.xml`).
//
// # Empat rute, dan hanya SATU yang menulis
//
//	GET  /                 daftar klaim tutup
//	GET  /penyaring        isi ketiga dropdown — bentuk layar, bukan data
//	GET  /unduh            CSV
//	POST /permintaan       ReOpen dan Copy Klaim
//
// Yang menulis hanya `POST /permintaan`, dan yang ditulisnya BUKAN klaim melainkan
// `POOLDATA.CPNC_PERMINTAAN_KLAIM` — tabel milik aplikasi ini sendiri, dibuat migrasi
// `0006`. `P-1` menetapkan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` ditulis Pega selama masa
// paralel, dan Work Owner memutuskan 2026-09-23 aplikasi ini mencatat permintaannya saja.
//
// Rute yang tidak didaftarkan dijawab chi dengan 405, sehingga penambahan rute tulis kelak
// menjadi keputusan sadar — bukan sesuatu yang lolos review.
//
// # Dua tombol layar lama yang TIDAK menjadi rute
//
//	Cari Data / Clear Filter   penyaring, dilayani GET / dengan query string
//	.CaseIDView                membuka layar detail `ViewTempDetailClaim`, milik modul lain
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya — keadaan yang sama dengan
// seluruh rute lain hari ini. Penegakan "apakah peran pemanggil memiliki menu ini" adalah
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`.
//
// Taruhannya di sini lebih besar daripada di layar yang hanya membaca. Di Pega, butir menu
// ini dijaga `When/IsManagerPNC_CLOSE-When.xml`, yang membuka aksesnya bagi **empat access
// group** — Administrators, CaseManager, PncManagerAdmin, PNCKomiteTeknik — **ditambah tiga
// Operator ID perorangan** yang namanya tertanam di dalam rule. Yang terakhir itu persis
// jenis hardcode yang `D-15` hapus; di sistem baru penggantinya adalah `M_OTORISASI_PNC`,
// yang sudah menentukan siapa MELIHAT butir menunya tetapi belum menjaga endpoint-nya.
//
// Sampai `TKT-F3-005` ada, siapa pun yang punya sesi dapat memanggil `POST /permintaan`.
// Dicatat terbuka di docs/keputusan-implementasi.md.
//
// # Seluruh rutenya menuntut portal aktif
//
// Klaim ada di basis data setiap entitas (`ADR-0030`), sehingga setiap permintaan harus
// menyebut entitas mana yang dibacanya lewat header `X-Portal`. Permintaan yang tidak
// menyebutkannya DITOLAK oleh middleware — tidak pernah dilayani portal utama sebagai
// cadangan, karena jatuh ke koneksi default berarti menampilkan klaim satu badan hukum di
// layar badan hukum lain tanpa satu pun galat (`R-20`, `TKT-F6-002`).
//
// Pada modul ini akibatnya melampaui tampilan: portal yang salah berarti permintaan ReOpen
// tercatat di basis data badan hukum yang keliru.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/inbox-close-claim", func(closeClaim chi.Router) {
		closeClaim.Use(portalhttp.ActivePortal(portalDeps))

		closeClaim.Get("/", h.List)
		closeClaim.Get("/penyaring", h.Metadata)

		// Unduhan dipisahkan menjadi jalurnya sendiri, bukan parameter `format=csv` pada
		// daftar. Keduanya berbeda sifat: yang satu dipaginasi dan dibaca layar, yang lain
		// mengalir sampai habis dan diterima sebagai berkas. Menyatukannya membuat satu
		// endpoint punya dua bentuk respons dan dua batas ukuran.
		closeClaim.Get("/unduh", h.Export)

		closeClaim.Post("/permintaan", h.Request)
	})
}
