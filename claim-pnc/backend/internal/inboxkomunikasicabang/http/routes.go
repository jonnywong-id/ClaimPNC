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
// Sejak 2026-09-24 taruhannya NAIK: modul ini tidak lagi membaca saja. Dua rutenya menulis,
// dan salah satunya tidak dapat dibatalkan. Yang membatasinya tinggal satu hal, dan ia harus
// disebut apa adanya:
//
//   - Batas cabang ditegakkan di SETIAP rute yang menyentuh data — daftar, detail, ekspor,
//     DAN kedua rute tulis. Pada rute tulis ia diselesaikan ULANG, bukan dipercaya dari
//     permintaan sebelumnya, dan ikut sebagai penyaring pada pernyataan SQL-nya sendiri.
//     Batas yang berlaku pada pembacaan tetapi tidak pada penulisan bukan batas sama sekali:
//     balasan yang telanjur tersimpan di percakapan cabang lain tidak dapat ditarik kembali
//     lewat layar mana pun.
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

		// DUA aksi yang MENULIS, ditambahkan 2026-09-24.
		//
		// Keduanya bersarang di bawah nomor percakapannya, bukan berdiri sebagai satu
		// endpoint "tindakan" bersama. Alasannya bukan kerapian: satu endpoint yang menerima
		// nama tindakan sebagai isian akan membuat "balas" dan "tutup percakapan" berbagi satu
		// bentuk permintaan, satu bentuk jawaban, dan satu baris di log akses — padahal yang
		// satu dapat diulang dan yang satu tidak dapat dibatalkan.
		//
		// POST, bukan PUT maupun DELETE. Keduanya peristiwa yang ditambahkan pada percakapan,
		// bukan penggantian maupun penghapusan sumber daya (`10-API-STRATEGY.md` §2); dan
		// `D-66` melarang penghapusan fisik, sehingga DELETE akan menjanjikan hal yang memang
		// tidak terjadi.
		perPortal.Post("/inbox-komunikasi-cabang/komunikasi/{komunikasi}/balas", h.Reply)
		perPortal.Post("/inbox-komunikasi-cabang/komunikasi/{komunikasi}/selesai", h.Finish)

		// Pembuatan percakapan BARU — tombol "Kirim Pesan" pada form yang dibuka "Tambah".
		//
		// Ia TIDAK bersarang di bawah nomor percakapan, dan itu konsekuensi langsung dari apa
		// yang dilakukannya: nomornya belum ada sampai permintaan ini selesai.
		perPortal.Post("/inbox-komunikasi-cabang/pesan", h.SendMessage)

		// Daftar cabang untuk pemilih tujuan pada form itu.
		//
		// GET, dan TIDAK disaring menurut cabang pemanggil — yang dibatasi adalah percakapan,
		// bukan daftar cabang. Menyaringnya akan mengosongkan pemilihnya bagi setiap petugas
		// cabang, sehingga tidak seorang pun dapat mengirim pesan ke mana pun.
		perPortal.Get("/inbox-komunikasi-cabang/cabang", h.Branches)

		// CATATAN. Rute `POST /inbox-komunikasi-cabang/tindakan` DIHAPUS pada 2026-09-24.
		//
		// Ia menjawab keempat tindakan tulis layar lama dengan alasan, dan keempatnya kini
		// benar-benar bekerja. Yang terakhir menyusul — "Tambah" ternyata tombol yang hanya
		// MEMBUKA form, tidak menyentuh peladen sama sekali.
	})
}
