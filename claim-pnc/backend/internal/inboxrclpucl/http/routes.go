package inboxrclpuclhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox RCL/PUCL.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Barisnya memuat nomor polis dan nama
// tertanggung, dan keduanya milik satu badan hukum, bukan milik badan hukum lain.
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
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — `When/IsRCLPUCL`
// membatasi menu ini pada `GCNMFW:PncRCLPUCL` dan `GCNMFW:Administrators`, dan
// mengecualikan `GCNMFW:ViewClaimPNC`. Pembatasan itu belum dapat ditegakkan sampai
// `TKT-F3-004` selesai.
//
// Yang membatasi taruhannya, dan keduanya perlu disebut apa adanya:
//
//   - Modul ini MEMBACA saja. Tidak ada satu pun aksi yang mengubah data.
//   - Setiap pembukaan DICATAT — bukan hanya yang mencurigakan. Lihat usecase.List, dan
//     usecase.DailyReport yang mencatat rentang tanggalnya pula.
//
// Yang kedua bukan pengganti kewenangan; ia hanya membuat pembukaannya dapat ditelusuri
// setelah terjadi. `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru
// untuk keadaan seperti ini.
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

		perPortal.Get("/inbox-rcl-pucl/tab", h.Metadata)
		perPortal.Get("/inbox-rcl-pucl", h.List)

		// Layar kerja satu klaim — yang di Pega terbuka lewat Open Assignment saat nomor
		// klaim diklik.
		//
		// Kuncinya di JALUR, bukan parameter query: ia mengidentifikasi sumber daya, bukan
		// menyaringnya (`10-API-STRATEGY.md` §2). Bersarang satu tingkat, sesuai batas dua
		// tingkat pada aturan yang sama.
		perPortal.Get("/inbox-rcl-pucl/klaim/{referensi}", h.Detail)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — termasuk dibuka ulang dari
		// riwayat peramban dengan rentang tanggal yang sama.
		//
		// SATU rute untuk dua perilaku: tab "Cetak Surat" menghasilkan laporan harian
		// berbasis rentang tanggal, dua tab lain menyalin grid-nya. Percabangannya ada di
		// Handler.Export dan ditentukan oleh sifat tab, bukan oleh jalur yang berbeda —
		// di Pega pun tombolnya satu dan sama, hanya activity di baliknya yang berbeda.
		perPortal.Get("/inbox-rcl-pucl/ekspor", h.Export)

		// Aksi tulis sistem lama. Rutenya ADA supaya tindakan di layar menjawab dengan
		// alasan, bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Dua yang nyata di layar ini: mencetak surat PUCL/RCL, yang mengisi
		// `TANGGALCETAKDOKUMENPUCL_1` sehingga klaimnya BERPINDAH tab, dan mengirim
		// Reminder PUCL. Keduanya menyentuh tabel yang selama masa paralel masih dimiliki
		// Pega (`P-1`).
		perPortal.Post("/inbox-rcl-pucl/tindakan", h.RejectWrite)
	})
}
