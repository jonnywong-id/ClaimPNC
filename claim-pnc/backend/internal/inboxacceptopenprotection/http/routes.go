package inboxacceptopenprotectionhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Accept Open Protection.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini.
//
// # Kenapa jalurnya "inbox-accept-open-protection"
//
// Itu judul yang tertulis di dalam layarnya sendiri
// (`Section/InputProtection_Section-Section.xml:338`). Butir menunya bernama "Inbox Open
// Protection", tetapi nama itu bertabrakan dengan judul layar modul `inputreqprotection` —
// lihat komentar paket inboxacceptopenprotection.
//
// # Tiga kolom yang ditulis, dan tidak lebih
//
// `PUT /{nomor}/akseptasi` menuliskan `STATUS_AKSEPTASI`, `TANGGAL_AKSEPTASI`, dan
// `DIAKSEP_OLEH`. Tidak ada rute yang menyunting polis, klaim, tipe, maupun keterangan —
// keempatnya dimiliki modul `inputreqprotection`, dan `P-1` menetapkan satu kolom ditulis
// satu pemilik.
//
// Rute yang tidak didaftarkan dijawab chi dengan 405, sehingga penambahan rute tulis kelak
// menjadi keputusan sadar — bukan sesuatu yang lolos review.
//
// # Kewenangan
//
// Layar lama dibatasi `When/IsOpenProtectionPNC-When.xml` pada lima access group:
// `PncCollection`, `CaseManager`, `PncOPCGeneral`, `PNCKomite`, dan `Administrators`. Di
// dalamnya, antrean PREMI hanya tampil bagi `PncCollection`.
//
// Pembatasan itu BELUM ditegakkan di sini. Ia `TKT-F3-005`, yang bergantung pada tabel peran
// `TKT-F3-004`; tabel itu dapat dibangun tetapi belum dapat diisi karena penugasan operator
// ke peran tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Taruhannya di sini lebih besar daripada di layar baca: akseptasi adalah PERSETUJUAN atas
// pembukaan proteksi, dan `D-59` menetapkan tidak ada pemisahan tugas formal. Sampai peran
// tersedia, yang tersisa sebagai kontrol hanyalah jejak `DIAKSEP_OLEH`. Dicatat terbuka di
// `docs/keputusan-implementasi.md`.
//
// # Seluruh rutenya menuntut portal aktif
//
// Tabelnya ada di basis data setiap entitas (`ADR-0030`). Permintaan tanpa header
// `X-Portal` DITOLAK middleware — tidak pernah dilayani portal utama sebagai cadangan
// (`R-20`, `TKT-F6-002`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/inbox-accept-open-protection", func(accept chi.Router) {
		accept.Use(portalhttp.ActivePortal(portalDeps))

		accept.Get("/", h.List)
		accept.Get("/{nomor}", h.Get)

		// Akseptasi diberi jalurnya sendiri, bukan PUT atas sumber daya proteksi.
		//
		// `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai PERISTIWA, bukan
		// pembaruan field: akseptasi punya invarian sendiri, mengubah kepemilikan tahap, dan
		// wajib tercatat. `PATCH` dengan `{"status": "1"}` akan melewatkan ketiganya.
		accept.Put("/{nomor}/akseptasi", h.Decide)
	})
}
