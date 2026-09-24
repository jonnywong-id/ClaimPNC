package inboxcompliancehttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Compliance.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Antrean kepatuhan satu badan hukum
// bukan antrean badan hukum lain, dan baris yang muncul di sana memuat nomor polis serta
// nama tertanggung.
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
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran, dan di modul ini
// ketiadaannya lebih berat akibatnya daripada di modul master mana pun:
//
// Antrean ini adalah antrean PEMERIKSAAN KEPATUHAN. Di sistem lama ia hanya dapat dibuka
// peran `GCNMFW:PncComplience`, dan barisnya memuat nomor polis beserta nama tertanggung
// setiap klaim yang sedang diperiksa. Tanpa pemeriksaan peran, setiap pengguna yang sudah
// masuk dapat membukanya.
//
// Penegakannya adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` — dan
// tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan operator ke peran
// tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1). Ia dicatat di sini
// sebagai utang yang disadari, bukan sebagai hal yang terlewat.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`,
// `/api/inbox-admin`). `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan
// memperkenalkannya di modul ini saja akan membuat dua gaya jalur hidup berdampingan.
// Penyeragamannya dicatat sebagai utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-compliance/tab", h.Metadata)
		perPortal.Get("/inbox-compliance", h.List)

		// Satu-satunya rute yang MENGUBAH data di modul ini. Ia menulis ke
		// `POOLDATA.T_CLAIM_COMPLIANCE_H`, tabel baru yang tidak dikenal Pega — bukan ke
		// tabel warisan mana pun, sehingga `P-1` tidak dilanggar.
		//
		// Ketiadaan pemeriksaan peran paling berat akibatnya di sini: rute ini menerbitkan
		// baris yang dibaca pemeriksa Post Audit, dan setiap pengguna yang sudah masuk
		// dapat memanggilnya. Itu `TKT-F3-005`, dan sampai ia ada, jejaknya hanya berupa
		// baris log yang menyebut pengirimnya.
		perPortal.Post("/inbox-compliance/post-audit", h.SendPostAudit)
	})
}
