package inboxanalystdoctorhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Analyst Doctor.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah cmd/claimpnc.
//
// Tuntutan itu LEBIH KERAS di modul ini daripada di inbox mana pun. Antreannya disaring
// dengan identitas pemanggil, sehingga sesi yang tidak terbaca bukan menghasilkan layar yang
// kurang lengkap melainkan layar yang tidak punya isi sama sekali.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Di layar ini taruhannya lebih besar
// daripada di inbox lain: antreannya berisi klaim Personal Accident, dan `FR-R2` membatasi
// akses data medis pada peran Analyst Doctor dan RCL Dokter — pembatasan yang menjadi tidak
// bermakna bila datanya datang dari entitas yang salah.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan klaim satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar ikut di balik pemeriksaan itu meski isinya sama di seluruh entitas.
// Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias portal, dan
// mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal akan membuat
// layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi dan disaring identitas pemanggil. Yang BELUM ada adalah
// pemeriksaan peran: "apakah peran pemanggil memiliki menu ini" adalah `TKT-F3-005`, yang
// bergantung pada tabel peran `TKT-F3-004` — dan tabel itu dapat dibangun tetapi belum dapat
// diisi, karena penugasan operator ke peran tidak ada di basis data maupun di export
// (`11-SECURITY.md` §3.1).
//
// Di Pega, butir menunya dijaga `When/IsAnalystDoctor-When.xml` berkelas `@baseclass`:
//
//	(AccessGroup = "GCNMFW:Administrators" OR AccessGroup = "GCNMFW:PncAnalystDoctor")
//	AND AccessGroup != "GCNMFW:ViewClaimPNC"
//
// Aturan itu BELUM ditegakkan di sini. Yang meredam akibatnya untuk sementara adalah
// penyaring identitas: pengguna yang tidak punya tugas Analyst Doctor melihat antrean kosong,
// bukan antrean orang lain. Itu peredam, bukan kendali — dan perbedaannya penting.
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

		perPortal.Get("/inbox-analyst-doctor/keterangan", h.Metadata)
		perPortal.Get("/inbox-analyst-doctor", h.List)
	})
}
