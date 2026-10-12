package inboxosclaimpercabanghttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox OS Claim per Cabang (`MENU_ID 69`).
//
// # Yang dituntut pemanggil
//
// Kedua rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah cmd/claimpnc.
//
// # Kenapa keduanya di balik pemeriksaan portal
//
// Karena keduanya menyentuh basis data entitas, dan barisnya memuat nomor polis, nama
// tertanggung, serta nilai estimasi klaim — seluruhnya milik satu badan hukum.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan.
// Jatuh ke koneksi bawaan berarti menampilkan klaim satu badan hukum kepada petugas badan
// hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi, dan batas datanya CABANG — bukan peran. Yang belum ada adalah
// pemeriksaan peran (`TKT-F3-004`), sehingga setiap pengguna yang dapat masuk dan punya cabang
// melihat klaim outstanding cabangnya.
//
// Di layar lama pun begitu: tidak ada satu pun pemeriksaan privilege pada jalur ini —
// `pyPrivilegeName` terisi hanya pada 1 dari 902 activity di seluruh export, dan yang satu itu
// privilege bawaan Pega untuk ekspor ruleset. Yang menentukan siapa melihat butir menunya
// sekarang adalah `M_OTORISASI_PNC`.
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

		perPortal.Get("/inbox-os-claim-per-cabang", h.List)

		// Ekspor adalah GET, bukan POST, meski di sistem lama ia tombol yang menjalankan
		// activity. Ia tidak mengubah apa pun, dan menjadikannya GET membuat unduhannya
		// dapat dipicu tautan biasa — termasuk dibuka ulang dari riwayat peramban.
		perPortal.Get("/inbox-os-claim-per-cabang/ekspor", h.Export)

		// Panel ringkasan. Rute terpisah dari daftar karena keduanya berubah pada irama yang
		// berbeda — daftar berganti halaman, panel tidak — dan karena kegagalan salah satunya
		// tidak boleh mengosongkan yang lain.
		//
		// Didaftarkan SEBELUM `/{nomor}` supaya jalur harfiahnya tidak tertelan parameter.
		perPortal.Get("/inbox-os-claim-per-cabang/ringkasan", h.Summary)

		// Popup Detail. Ia didaftarkan SETELAH `/ekspor` dengan sengaja: chi mencocokkan
		// jalur harfiah lebih dulu daripada parameter, tetapi menaruhnya berurutan begini
		// membuat urutan itu terbaca oleh siapa pun yang menyuntingnya kemudian.
		//
		// Nomor klaim ada di JALUR karena ia menentukan sumber daya mana yang diminta, bukan
		// cara menyaringnya (`10-API-STRATEGY.md` §2). Bersarangnya satu tingkat — batas yang
		// ditetapkan bab itu dua.
		perPortal.Get("/inbox-os-claim-per-cabang/{nomor}", h.Detail)
	})
}
