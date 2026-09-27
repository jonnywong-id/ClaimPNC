package inboxpladlapredlahttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox PLA, DLA, Pre DLA.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Setiap baris memuat NAMA
// TERTANGGUNG dan NOMOR POLIS, dan keduanya milik satu badan hukum, bukan milik badan
// hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan klaim satu badan hukum kepada
// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — dan di layar ini
// tidak ada pembatas lain sama sekali: ketiga daftarnya bersama, dan tidak satu pun
// kuerinya menyebut pemanggil. Itu perilaku Pega, tetapi di Pega menunya masih dibatasi
// `M_OTORISASI_PNC`, dan pembatasan berbasis peran baru dapat ditegakkan setelah
// `TKT-F3-004` selesai.
//
// Yang mengimbanginya sekarang hanya satu, dan ia perlu disebut apa adanya: setiap
// pembukaan DICATAT (lihat usecase.List dan usecase.Documents). Itu bukan pengganti
// kewenangan; ia hanya membuat perbuatannya dapat ditelusuri setelah terjadi. `D-59`
// menjadikan jejak audit satu-satunya kontrol pengimbang justru untuk keadaan seperti ini.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox-pla-dla-pre-dla/daftar", h.Metadata)
		perPortal.Get("/inbox-pla-dla-pre-dla", h.List)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — sekaligus membuat penyaring yang
		// sedang aktif terbawa apa adanya di alamatnya.
		perPortal.Get("/inbox-pla-dla-pre-dla/ekspor", h.Export)

		// Grid "Detail PLA List" / "Detail DLA List" satu klaim.
		//
		// Jalurnya memuat SEGMEN `klaim` supaya ia tidak bertabrakan dengan rute tetap di
		// atasnya. Tanpa segmen itu, `/inbox-pla-dla-pre-dla/{kunci}` akan menangkap
		// `daftar` dan `ekspor` sebagai kunci klaim pada urutan pendaftaran tertentu —
		// kelas kerusakan yang hanya muncul ketika rutenya bertambah.
		//
		// Kuncinya memuat SPASI (`ASM-FW-GCNMFW-WORK PNC-xxxx`) dan karena itu wajib
		// terkodekan oleh layar. Lihat Handler.Documents.
		perPortal.Get("/inbox-pla-dla-pre-dla/klaim/{kunci}", h.Documents)

		// Panel "Print Pre DLA" satu klaim (`Flow Action/PNCInboxPrintPreDLA-FA.xml`).
		//
		// Ia GET, dan itu bukan kelalaian meski tombolnya bernama "Print": panelnya hanya
		// MEMBACA. Yang menulis adalah tombol "Kirim Pre DLA" DI DALAM panelnya, dan
		// tombol itu menjawab alasan lewat rute tindakan di bawah.
		perPortal.Get("/inbox-pla-dla-pre-dla/cetak/{kunci}", h.Print)

		// Ketiga aksi tulis yang belum dibangun. Rutenya ADA supaya tombol di layar
		// menjawab dengan alasan, bukan dengan "halaman tidak ditemukan" — lihat
		// Handler.RejectWrite.
		//
		// Yang ditembak masing-masing berbeda: "Send" mengirim email beserta lampirannya,
		// "Upload File Penunjang" menulis ke penyimpanan dokumen (`D-16`), "Kirim Pre DLA"
		// menulis ke `T_PREDLALIST` yang masih dimiliki Pega (`P-1`), dan unduh lampiran
		// membaca penyimpanan dokumen yang sama dengan unggah.
		perPortal.Post("/inbox-pla-dla-pre-dla/tindakan", h.RejectWrite)
	})
}
