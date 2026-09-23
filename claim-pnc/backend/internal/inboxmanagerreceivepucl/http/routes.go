package inboxmanagerreceivepuclhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Manager Receive / PUCL.
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
// # Kewenangan — dan di layar ini ketiadaannya paling berat akibatnya
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran, dan modul ini yang
// paling terdampak di antara seluruh modul inbox yang sudah dibangun.
//
// Alasannya: TIDAK SATU PUN tabnya menyaring menurut pemanggil. Modul inbox lain setidaknya
// punya satu tab "milik saya" yang menyaring `PXASSIGNEDOPERATORID` menurut login; di sini
// penyaring itu memang tidak pernah ada — Report Definition-nya menyaring unit organisasi,
// dan parameternya tidak pernah diisi. Layar ini memang dirancang sebagai pandangan
// PENYELIA.
//
// Akibatnya, sampai `TKT-F3-004` selesai, setiap pengguna yang dapat masuk melihat nomor
// polis dan nama tertanggung seluruh berkas penerimaan dokumen dan seluruh klaim di antrean
// RCL/PUCL pada portalnya. Dua hal yang membatasi taruhannya, dan keduanya perlu disebut
// apa adanya:
//
//   - Modul ini MEMBACA saja. Tidak ada satu pun aksi yang mengubah data.
//   - Setiap pembukaan DICATAT — bukan hanya yang mencurigakan. Lihat usecase.List.
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

		perPortal.Get("/inbox-manager-receive-pucl/tab", h.Metadata)
		perPortal.Get("/inbox-manager-receive-pucl", h.List)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — termasuk dibuka ulang dari
		// riwayat peramban dengan penyaring yang sama.
		perPortal.Get("/inbox-manager-receive-pucl/ekspor", h.Export)

		// Aksi tulis sistem lama. Rutenya ADA supaya tindakan di layar menjawab dengan
		// alasan, bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Yang paling nyata di layar ini adalah pencetakan surat PUCL/RCL, yang mengisi
		// `TANGGALCETAKDOKUMENPUCL_1` pada objek kerja klaim — tabel yang selama masa
		// paralel masih dimiliki Pega (`P-1`).
		perPortal.Post("/inbox-manager-receive-pucl/tindakan", h.RejectWrite)
	})
}
