package inboxpladlahttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox PLA DLA.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas — dan di modul ini taruhannya paling
// besar di antara seluruh modul yang sudah dibangun, karena dua hal bertumpuk:
//
//   - Barisnya memuat nama tertanggung dan nomor polis, milik satu badan hukum.
//   - Yang membacanya PIHAK LUAR. Permintaan yang jatuh ke koneksi bawaan tidak sekadar
//     menampilkan data entitas lain kepada petugas sendiri — ia menampilkannya kepada
//     mitra reasuransi (`R-20`).
//
// # Kewenangan — dan di sini ia SUDAH ADA, berbeda dari modul inbox lain
//
// Pembatasnya bukan peran melainkan KEPEMILIKAN: daftar disaring menurut kode reasuradur
// milik login pemanggil, diturunkan di sisi peladen dari `POOLDATA.T_REINSURER.LOGIN`.
// Login yang tidak terdaftar di sana ditolak dengan pesan yang menjelaskan sebabnya —
// bukan dilayani daftar kosong.
//
// Itu pembatas yang sungguh berlaku, bukan sekadar penyembunyian menu. Yang masih belum
// ada tetap sama dengan modul lain: pemeriksaan peran (`TKT-F3-004`), yang akan menentukan
// siapa boleh melihat BUTIR MENU-nya.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-pla-dla/daftar", h.Metadata)
		perPortal.Get("/inbox-pla-dla", h.List)

		// Tabel ringkas "Status / Jumlah".
		//
		// Rute TERSENDIRI, bukan bagian jawaban daftar, karena isinya tidak berubah saat
		// pengguna berpindah HALAMAN — dan menggabungkannya akan menjalankan hitungannya
		// setiap kali halaman berganti.
		perPortal.Get("/inbox-pla-dla/ringkas", h.Counts)

		// Grid "DATA PLA DLA XOL KLAIM".
		//
		// Rute tersendiri pula, dan alasannya lebih kuat lagi: isinya tidak disaring tab
		// maupun kata kunci sama sekali. Menggabungkannya akan menjalankan gabungan dua
		// tabel XOL setiap kali pengguna mengetik satu huruf di kotak pencarian.
		perPortal.Get("/inbox-pla-dla/xol", h.XOL)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun.
		perPortal.Get("/inbox-pla-dla/ekspor", h.Export)

		// Ketiga tombol yang belum dibangun — "Detail Claim", "Detail", dan "DLA".
		// Rutenya ADA supaya tombolnya menjawab dengan alasan, bukan dengan "halaman
		// tidak ditemukan".
		perPortal.Post("/inbox-pla-dla/tindakan", h.RejectWrite)
	})
}
