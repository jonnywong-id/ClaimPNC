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
		perPortal.Get("/inbox-xol/klaim/summary", h.SummarizeBusiness)
		perPortal.Get("/inbox-xol/klaim/daftar", h.ListClaims)
		perPortal.Get("/inbox-xol/klaim/rincian/unduh", h.ExportClaimDetail)
		perPortal.Get("/inbox-xol/pla-dla", h.SearchAdvice)
		perPortal.Get("/inbox-xol/pla-dla/unduh", h.DownloadAdvice)
		perPortal.Get("/inbox-xol/persetujuan", h.ListApprovals)
		perPortal.Get("/inbox-xol/sebab-kerugian", h.ListCauseOfLoss)
		perPortal.Get("/inbox-xol/format-unggah", h.DownloadUploadTemplate)

		// Aksi tulis sistem lama, dan baru DUA yang dapat dikerjakan.
		//
		//	unggah/mbu-salvage → POOLDATA.T_SALVAGE_MBU       ✓ berjalan
		//	dol-col            → POOLDATA.XOL_TABLE_ALL_KLAIM  ✓ berjalan
		//	dol-col/hapus      → POOLDATA.XOL_TABLE_ALL_KLAIM (DELETE)
		//	persetujuan        → POOLDATA.T_PLA_XOL, POOLDATA.T_DLA_XOL
		//	pengajuan          → POOLDATA.MST_XOL_PNC
		//	unggah/inward      → POOLDATA.T_CLAIM_INWARD_XOL
		//	generate/pla       → penerbitan PLA XOL
		//	generate/dla       → penerbitan DLA XOL
		//
		// Sisanya dijawab penolakan beserta sebabnya, bukan "halaman tidak ditemukan" —
		// lihat Handler.RejectWrite. Sebabnya berbeda-beda:
		//
		//	dol-col/hapus  ia MENGHAPUS baris yang sudah ada, dan jalur Pega yang
		//	               menjalankan DELETE itu belum tertelusuri: tombol "Remove All
		//	               Data" di section memanggil ToFlaggingDataXOLByRequest, yang
		//	               hanya Property-Set dan Page-New — tidak menghapus apa pun.
		//	               Memindahkan penghapusan atas dasar tebakan berarti membuang
		//	               baris produksi tanpa tahu aturan aslinya.
		//
		//	unggah/inward  satu activity TIDAK ada di export —
		//	               `ConvertDataCsvInwardToPage`, yang hanya dirujuk flow
		//	               action-nya sendiri. Pemetaan kolom berkasnya karena itu
		//	               belum pernah dibaca siapa pun, dan menebaknya berarti
		//	               menulis tabel dengan isi yang dikarang.
		//
		//	generate/*     rule-nya LENGKAP di export, tetapi belum dianalisis:
		//	               `GenerateXOLByType` (257 KB) memanggil `GenerateDLAXOL_`
		//	               (543 KB) dan `PerhitunganxolUntukWillisDanSimasre` (524 KB),
		//	               ditambah delapan rule SQL. Tombolnya digambar lebih dulu
		//	               supaya layarnya utuh; isinya menyusul.
		//
		// `unggah/mbu-salvage` dapat dikerjakan karena seluruh rantainya ada:
		// flow action → `ConvertDataCsvSalvageMBUToPage` → `InsertDataSalvageMBU`.
		perPortal.Post("/inbox-xol/dol-col", h.InsertDolCol)
		perPortal.Post("/inbox-xol/dol-col/hapus", h.RejectWrite)
		perPortal.Post("/inbox-xol/persetujuan", h.RejectWrite)
		perPortal.Post("/inbox-xol/pengajuan-komite", h.RejectWrite)
		perPortal.Post("/inbox-xol/unggah/mbu-salvage", h.UploadSalvageMBU)
		perPortal.Post("/inbox-xol/unggah/inward", h.RejectWrite)
		perPortal.Post("/inbox-xol/generate/pla", h.RejectWrite)
		perPortal.Post("/inbox-xol/generate/dla", h.RejectWrite)
	})
}
