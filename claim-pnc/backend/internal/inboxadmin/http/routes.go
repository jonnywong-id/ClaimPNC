package inboxadminhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Admin.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Antrean kerja satu badan hukum bukan
// antrean badan hukum lain, dan baris yang muncul di sana memuat nama tertanggung.
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
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran: "apakah peran pemanggil
// memiliki menu ini" adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` —
// dan tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan operator ke peran
// tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Di sistem lama, layar ini membedakan perilaku bagi `GCNMFW:CaseManager` dan
// `GCNMFW:PncManagerAdmin` — keduanya melewati penyaring cabang dan memperoleh pemilih
// korwil. Pembedaan itu belum dibawa, dan tidak dapat dibawa sampai sumber peran ada.
// Akibatnya untuk sekarang SELURUH pengguna berperilaku seperti manajer: tanpa penyaring
// cabang. Itu sejalan dengan keputusan Work Owner 2026-09-20 yang menunda penyaring cabang
// sampai API pengganti DB Link HRD tersedia (`R-03`).
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

		perPortal.Get("/inbox-admin/tab", h.Metadata)
		perPortal.Get("/inbox-admin", h.List)
	})
}
