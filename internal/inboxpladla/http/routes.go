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

		// Tabel "Status / Jumlah" — satu baris per DAFTAR, dan inilah tabel yang
		// benar-benar ada di Pega. Rute tersendiri karena isinya tidak berubah saat
		// pengguna berpindah daftar: ia menyebut keenamnya sekaligus.
		perPortal.Get("/inbox-pla-dla/ringkas-daftar", h.ListCounts)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun.
		perPortal.Get("/inbox-pla-dla/ekspor", h.Export)

		// ====================================================================
		// LAYAR RINCIAN — tombol "Detail Claim"
		// ====================================================================
		//
		// Jalurnya memuat SEGMEN `klaim` supaya ia tidak bertabrakan dengan rute tetap
		// di atasnya. Tanpa segmen itu, `/inbox-pla-dla/{kunci}` akan menangkap
		// `daftar`, `ringkas`, `ringkas-daftar`, `xol`, dan `ekspor` sebagai kunci klaim pada
		// pendaftaran tertentu — kelas kerusakan yang hanya muncul ketika rutenya
		// bertambah. Presedennya ada di modul `inboxpladlapredla`.
		//
		// Kuncinya memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-xxxx`) dan wajib terkodekan
		// layar.
		perPortal.Get("/inbox-pla-dla/klaim/{kunci}", h.Detail)

		// Grid dokumen satu nomor pemberitahuan — tombol "Dokumen" pada baris PLA/DLA.
		//
		// Nomor dan jenisnya lewat parameter query, bukan segmen alamat: nomor
		// pemberitahuan dapat memuat karakter apa pun yang dipakai penerbitnya, dan
		// parameter query mengodekannya tanpa aturan tambahan.
		perPortal.Get("/inbox-pla-dla/klaim/{kunci}/dokumen", h.Documents)

		// ISI satu dokumen. Ia menulis BERKAS, bukan JSON.
		perPortal.Get("/inbox-pla-dla/klaim/{kunci}/dokumen/{dokumen}", h.DocumentContent)

		// **SATU-SATUNYA rute yang MENULIS di modul ini**, dan pelakunya PIHAK LUAR.
		//
		// Ia menyimpan balasan reasuradur atas satu percakapan
		// (`RDB List/ReplyKomunikasi-SQL.xml`), dan tulisannya masuk ke tabel yang
		// dibaca petugas internal lewat modul `inboxkomunikasicabang`.
		//
		// Nomor percakapannya berada di BADAN permintaan, bukan di alamat. Alasannya
		// bukan gaya: badan permintaan tidak tercatat di log peladen web maupun di
		// riwayat peramban, sementara segmen alamat tercatat di keduanya — dan nomor
		// percakapan adalah kunci yang dipakai memagari siapa boleh membalas apa.
		perPortal.Post("/inbox-pla-dla/klaim/{kunci}/komunikasi/balas", h.Reply)

		// Kedua tombol yang belum dibangun — "Download ALL PLA" dan "Download ALL DLA"
		// pada layar rincian. Rutenya ADA supaya tombolnya menjawab dengan alasan,
		// bukan dengan "halaman tidak ditemukan".
		perPortal.Post("/inbox-pla-dla/tindakan", h.RejectWrite)
	})
}
