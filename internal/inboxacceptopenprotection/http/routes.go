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
// `PUT /{nomor}/akseptasi` menuliskan `APPROVAL_STATUS`, `RESOLVED_DATETIME`, dan
// `RESOLVED_BY`. Tidak ada rute yang menyunting polis, klaim, tipe, maupun keterangan —
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
// **Pembatasan itu DITEGAKKAN sejak 2026-09-25**, setelah Work Owner menetapkan
// `POOLDATA.M_LOGIN_GROUP_PNC.GROUP_ID` berisi nama access group Pega tanpa awalan
// `GCNMFW:`. Penegakannya ada di `usecase`, bukan di sini — lihat `authorization.go`.
//
// Penegakannya terjadi di SERVER pada setiap permintaan, sehingga mengetahui alamat rute
// tidak lagi cukup untuk membukanya. Itu yang membedakannya dari sistem lama, yang hanya
// menyembunyikan menu (`pyPrivilegeName` terisi pada 1 dari 902 activity).
//
// # Yang masih berlaku dan tidak berubah
//
// `D-59` tetap menetapkan tidak ada pemisahan tugas formal: seorang yang berwenang atas
// layar ini berwenang atas SELURUH tindakan di dalamnya. Jejak `RESOLVED_BY` karena itu tetap
// satu-satunya kontrol pengimbang atas persetujuannya sendiri.
//
// Penegakan serupa untuk 31 modul lain belum dikerjakan — itu `TKT-F3-005`, dan Work Owner
// menetapkan lingkupnya modul ini dulu.
//
// # Seluruh rutenya menuntut portal aktif
//
// Tabelnya ada di basis data setiap entitas (`ADR-0030`). Permintaan tanpa header
// `X-Portal` DITOLAK middleware — tidak pernah dilayani portal utama sebagai cadangan
// (`R-20`, `TKT-F6-002`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/inbox-accept-open-protection", func(accept chi.Router) {
		accept.Use(portalhttp.ActivePortal(portalDeps))

		// Antrean yang boleh dibuka pemanggil. Didaftarkan SEBELUM "/{nomor}" supaya chi
		// tidak menganggap "antrean" sebagai nomor proteksi.
		accept.Get("/antrean", h.Queues)

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
