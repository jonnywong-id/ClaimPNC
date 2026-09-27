package inboxrclhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox RCL.
//
// # Yang dituntut pemanggil
//
// Seluruh rute WAJIB sudah berada di balik middleware Autentikasi; cmd/claimpnc yang
// merakit urutannya. Antreannya disaring dengan identitas lama pemanggil, sehingga sesi yang
// tidak terbaca berarti layar tanpa isi sama sekali.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena identitas lama DAN antreannya sama-sama tersimpan di basis data entitas. Barisnya
// adalah klaim Personal Accident, dan `FR-R2` membatasi akses data medis pada peran Analyst
// Doctor dan RCL Dokter — pembatasan yang tidak bermakna bila datanya datang dari entitas
// yang salah. Permintaan tanpa portal DITOLAK, tidak dilayani portal utama (`R-20`).
//
// # Kewenangan
//
// Di Pega, butir menunya dijaga `When/IsRCLPA-When.xml`:
//
//	(Administrators OR (PNCKomite AND pyPosition = "PA") OR (CaseManager AND pyPosition = "PA"))
//	AND NOT ViewClaimPNC
//
// Aturan itu BELUM ditegakkan di sini (`TKT-F3-005`). Yang meredam akibatnya untuk sementara
// adalah penyaring identitas lama: pengguna tanpa identitas lama pada grup yang diizinkan
// melihat keadaan "identitas tidak ditemukan", bukan antrean orang lain. Itu peredam, bukan
// kendali.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-rcl/keterangan", h.Metadata)
		perPortal.Get("/inbox-rcl", h.List)
	})
}
