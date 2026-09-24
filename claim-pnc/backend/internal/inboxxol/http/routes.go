package inboxxolhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox XOL.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Perjanjian XOL, nilai klaim, dan
// daftar reasuradur satu badan hukum bukan milik badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan nilai klaim dan nama reasuradur
// satu badan hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — dan di layar ini
// ketiadaannya berakibat nyata, bukan sekadar kurang rapi: `Section/InboxClaimXOL-Section.xml`
// menjaga kedua tabnya dengan access group yang berbeda,
//
//	Inbox XOL        GCNMFW:PncPICTeknik
//	Inbox XOL Komite GCNMFW:CaseManager
//
// dan pembedaan itu TIDAK dapat ditegakkan hari ini. Tabel peran adalah `TKT-F3-004`,
// yang dapat dibangun tetapi belum dapat diisi: penugasan operator ke peran tidak ada di
// basis data maupun di export (`11-SECURITY.md` §3.1). Sampai itu ada, setiap pengguna
// yang dapat masuk melihat kedua tab.
//
// Taruhannya terbatas selama modul ini membaca saja — antrean komite yang terlihat bukan
// antrean komite yang dapat disetujui. Begitu aksi persetujuan dipindahkan ke sini,
// pemeriksaan peran menjadi prasyarat, bukan pelengkap.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-xol/perjanjian", h.ListMasters)
		perPortal.Get("/inbox-xol/klaim", h.SummarizeClaims)
		perPortal.Get("/inbox-xol/klaim/rincian", h.Breakdown)
		perPortal.Get("/inbox-xol/pla-dla", h.SearchAdvice)
		perPortal.Get("/inbox-xol/pla-dla/unduh", h.DownloadAdvice)
		perPortal.Get("/inbox-xol/persetujuan", h.ListApprovals)
		perPortal.Get("/inbox-xol/sebab-kerugian", h.ListCauseOfLoss)

		// Ketiga aksi tulis sistem lama. Rutenya ADA supaya tombolnya menjawab dengan
		// alasan, bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Ketiganya menulis tabel yang selama masa paralel masih dimiliki Pega (`P-1`):
		//
		//	dol-col     → POOLDATA.XOL_TABLE_ALL_KLAIM
		//	persetujuan → POOLDATA.T_PLA_XOL, POOLDATA.T_DLA_XOL
		//	pengajuan   → POOLDATA.MST_XOL_PNC
		perPortal.Post("/inbox-xol/dol-col", h.RejectWrite)
		perPortal.Post("/inbox-xol/persetujuan", h.RejectWrite)
		perPortal.Post("/inbox-xol/pengajuan-komite", h.RejectWrite)
	})
}
