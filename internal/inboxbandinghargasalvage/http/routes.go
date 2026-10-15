package inboxbandinghargasalvagehttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Banding Harga Salvage.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Barisnya memuat nomor klaim dan DUA
// angka harga yang sedang dipertentangkan, dan keduanya milik satu badan hukum.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan
// — jatuh ke koneksi bawaan berarti menampilkan banding satu badan hukum kepada komite badan
// hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu meski isinya sama di seluruh
// entitas. Alasannya bukan kerahasiaan melainkan kejujuran jawaban: respons memuat alias
// portal, dan mengembalikan alias portal utama untuk permintaan yang tidak menyebut portal
// akan membuat layar mengira ia sudah berada di portal yang benar.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran: "apakah peran pemanggil
// memiliki menu ini" adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004` — dan
// tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan operator ke peran tidak
// ada di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Di sistem lama, butir menunya dijaga `When/IsGCNMUser-When.xml` — dan rule itu berisi
// `compareTwoValues(1, "=", 2)`, yakni kondisi yang TIDAK PERNAH benar. Ia sakelar "jangan
// tampilkan ini", bukan pemeriksaan peran; hal yang sama sudah tercatat di
// `keputusan-implementasi.md` §41.15 dan menyebut butir menu ini namanya. Menirunya sebagai
// penjaga akan membuat layar ini tidak dapat dibuka siapa pun, sehingga yang menentukan siapa
// melihat butirnya di sini adalah `M_OTORISASI_PNC` — sama seperti butir lain.
//
// Yang menahan kebocoran sementara ini BUKAN ketiadaan pemeriksaan peran, melainkan penyaring
// `NAMAKOMITE = pemanggil` yang melekat pada kedua tab: petugas yang bukan komite sebuah
// banding tidak akan melihatnya, peran apa pun yang ia punya.
//
// Satu pengecualian atas kalimat itu ada dan disengaja: satu Operator ID melihat antrean
// komite lain, aturan bernama orang yang ditiru dari Pega atas keputusan Work Owner. Lihat
// inboxbandinghargasalvage/komite.go, dan perhatikan bahwa setiap pembukaannya DICATAT.
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

		perPortal.Get("/inbox-banding-harga-salvage/tab", h.Metadata)
		perPortal.Get("/inbox-banding-harga-salvage", h.List)

		// Tabel ringkas "Status Salvage / Jumlah" di kepala layar.
		perPortal.Get("/inbox-banding-harga-salvage/ringkas", h.Summary)

		// Panel rincian pada grid "History Cheker" — seluruh keputusan satu klaim.
		//
		// Segmen `riwayat` dipakai supaya jalurnya tidak bertabrakan dengan rute tetap di
		// atasnya. Tanpa segmen itu, `/{noKlaim}` akan menangkap `tab` dan `ringkas`
		// sebagai nomor klaim pada urutan pendaftaran tertentu — kelas kerusakan yang
		// hanya muncul ketika rutenya bertambah.
		perPortal.Get("/inbox-banding-harga-salvage/riwayat/{noKlaim}", h.Decisions)

		// Tombol Approve dan Reject pada kolom "Action" grid Request.
		//
		// SATU rute untuk keduanya, dibedakan isian `setujui` di dalam badan permintaan —
		// bukan dua rute. Di Pega pun keduanya memanggil activity yang SAMA.
		perPortal.Post("/inbox-banding-harga-salvage/keputusan", h.Decide)

		// Tombol "Lihat File" pada kolom "Action" grid Request.
		//
		// Kedua id dikirim sebagai parameter kueri, bukan sebagai ruas jalur, karena
		// `IDDETAILSALVAGE` memuat garis miring di dalamnya (`PNC-<n>/<m>`). Sebagai ruas
		// jalur ia akan terpecah menjadi dua segmen, dan router tidak akan pernah
		// mencocokkannya.
		perPortal.Get("/inbox-banding-harga-salvage/dokumen", h.Documents)

		// Unduhan satu dokumen. Ia GET, bukan POST: permintaannya hanya membaca, dan
		// peramban harus dapat mengarahkan jendela ke sana untuk memulai unduhan.
		perPortal.Get("/inbox-banding-harga-salvage/dokumen/{dokumen}", h.DocumentContent)
	})
}
