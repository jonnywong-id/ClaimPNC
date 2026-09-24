package inboxkomunikasicabanghttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Komunikasi Cabang.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// Tuntutan itu lebih keras di layar ini daripada di modul inbox lain: batas datanya
// DITURUNKAN dari login pemanggil, sehingga rute yang lolos tanpa sesi bukan sekadar
// kehilangan jejak — ia tidak punya batas sama sekali.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Isi percakapan menyebut klaim yang
// sedang berjalan, dan itu milik satu badan hukum, bukan milik badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan percakapan satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu meski isinya sama di seluruh
// entitas. Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias
// portal, dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal
// akan membuat layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi, DAN disaring cabang. Yang BELUM ada adalah pemeriksaan peran
// (`TKT-F3-004`) — butir menunya di `M_OTORISASI_PNC` yang menentukan siapa melihatnya
// sekarang.
//
// Yang membatasi taruhannya, dan keduanya perlu disebut apa adanya:
//
//   - Modul ini MEMBACA saja. Tidak ada satu pun aksi yang mengubah data.
//   - Batas cabang ditegakkan di SETIAP rute yang menyentuh data, termasuk layar detail dan
//     ekspor — bukan hanya di daftar. Batas yang berlaku pada daftar tetapi tidak pada
//     detail bukan batas sama sekali; ia hanya menyulitkan orang yang patuh.
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

		perPortal.Get("/inbox-komunikasi-cabang/tab", h.Metadata)
		perPortal.Get("/inbox-komunikasi-cabang", h.List)

		// Layar "Detail Komunikasi" — yang di Pega terbuka lewat flow action
		// `DETAILKOMUNIKASICABANG_11`.
		//
		// Nomornya di JALUR, bukan parameter query: ia mengidentifikasi sumber daya, bukan
		// menyaringnya (`10-API-STRATEGY.md` §2). Bersarang satu tingkat, sesuai batas dua
		// tingkat pada aturan yang sama.
		perPortal.Get("/inbox-komunikasi-cabang/komunikasi/{komunikasi}", h.Detail)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — termasuk dibuka ulang dari riwayat
		// peramban.
		//
		// Ia menempuh List yang sama dengan tabel, sehingga batas cabangnya tidak dapat
		// terlewat. Satu uji mengunci itu.
		perPortal.Get("/inbox-komunikasi-cabang/ekspor", h.Export)

		// Aksi tulis sistem lama. Rutenya ADA supaya tindakan di layar menjawab dengan
		// alasan, bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Empat yang nyata di layar ini: "Kirim Pesan", "Balas", "Selesai Komunikasi" yang
		// mengubah kanal percakapan sehingga barisnya HILANG dari kedua tab, dan "Tambah".
		// Seluruhnya menyentuh tabel yang selama masa paralel masih dimiliki Pega (`P-1`).
		perPortal.Post("/inbox-komunikasi-cabang/tindakan", h.RejectWrite)
	})
}
