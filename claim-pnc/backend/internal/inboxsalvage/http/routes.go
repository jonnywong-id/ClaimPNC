package inboxsalvagehttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Salvage.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Barisnya memuat nomor klaim dan NILAI
// UANG, dan keduanya milik satu badan hukum, bukan milik badan hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan pengajuan satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Di modul ini taruhannya lebih besar daripada modul inbox lain, dan alasannya satu:
// modul ini MENULIS. Permintaan simpan yang jatuh ke koneksi bawaan tidak sekadar
// menampilkan data entitas lain — ia menyisipkan baris ke dalamnya.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — `When/IsInboxSalvage`
// membatasi menu ini, dan pembatasan itu belum dapat ditegakkan sampai `TKT-F3-004`
// selesai.
//
// Yang membatasi taruhannya, dan keduanya perlu disebut apa adanya:
//
//   - Satu daftar menyaring menurut pemanggil — "Request Balai Lelang" hanya menampilkan
//     pengajuan milik PIC yang membukanya. Kedua belas daftar lain bersama.
//   - Setiap pembukaan DAN setiap penyimpanan DICATAT. Lihat usecase.List dan
//     usecase.Create.
//
// Yang kedua bukan pengganti kewenangan; ia hanya membuat perbuatannya dapat ditelusuri
// setelah terjadi. `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru
// untuk keadaan seperti ini — dan di modul yang menulis nilai uang, itu bukan formalitas.
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

		perPortal.Get("/inbox-salvage/daftar", h.Metadata)
		perPortal.Get("/inbox-salvage", h.List)

		// Tabel ringkas "Status Salvage / Jumlah".
		//
		// Rute TERSENDIRI, bukan bagian jawaban daftar, karena isinya tidak berubah saat
		// pengguna berpindah daftar — dan menggabungkannya akan menjalankan keempat belas
		// hitungannya setiap kali tab dibuka.
		perPortal.Get("/inbox-salvage/ringkas", h.Counts)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa.
		perPortal.Get("/inbox-salvage/ekspor", h.Export)

		// Tombol "Tambah" beserta "Submit"-nya.
		//
		// SATU rute untuk membuat dan mengubah, dibedakan isian `mode` di dalam badan
		// permintaan — bukan dua rute. Alasannya bukan kerapian: di Pega pun keduanya
		// menempuh activity yang SAMA (`SetStsSalvagePNC_act`), dan yang membedakannya
		// adalah terisi-tidaknya ID salvage. Memisahkannya akan membuat dua jalur yang
		// harus dijaga tetap setara.
		perPortal.Post("/inbox-salvage", h.Create)

		// Tombol "Upload Detail Salvage".
		//
		// Ia TIDAK menyimpan apa pun — hanya membaca berkas CSV dan mengembalikan barisnya
		// supaya layar dapat mengisi tabel Detail Item Salvage. Lihat Handler.Upload.
		perPortal.Post("/inbox-salvage/unggah-detail", h.Upload)

		// Aksi tulis yang belum dibangun. Rutenya ADA supaya tombol di layar menjawab
		// dengan alasan, bukan dengan "halaman tidak ditemukan" — lihat
		// Handler.RejectWrite.
		//
		// Tiga yang nyata di layar ini: Approve dan Reject pada grid Checker, dan Send To
		// BalaiLelang. Ketiganya mengubah `STSTRANSFER`, dan dua di antaranya menembak
		// sistem di luar aplikasi ini.
		perPortal.Post("/inbox-salvage/tindakan", h.RejectWrite)
	})
}
