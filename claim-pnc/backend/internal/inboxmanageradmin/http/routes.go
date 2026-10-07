package inboxmanageradminhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Inbox Manager Admin.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
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
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu, dan di modul ini alasannya
// lebih kuat daripada di modul lain: jawabannya BERBEDA menurut pemanggil — yang dikirim
// hanyalah tab yang boleh ia lihat — sehingga ia bukan keterangan yang sama bagi semua
// orang.
//
// # Kewenangan
//
// Dua lapis, dan keduanya perlu dibedakan.
//
// Yang SUDAH ada: pemisahan antartab menurut jabatan pemanggil. Ia ditegakkan di server pada
// setiap permintaan lewat `inboxmanageradmin.NewQuery`, bukan hanya dengan menyembunyikan
// tab di layar. Sistem lama hanya menyembunyikan kontainernya lewat `pyContainerVisibleWhen`
// — dan `11-SECURITY.md` §3.1 menyebut penyembunyian seperti itu sebagai kenyamanan
// tampilan, bukan kendali. Di sini ia menjadi kendali.
//
// Yang BELUM ada: pemeriksaan peran — "apakah peran pemanggil memiliki menu ini". Itu
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`, dan tabel itu dapat dibangun
// tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di basis data maupun
// di export (`11-SECURITY.md` §3.1).
//
// Akibat yang perlu disebut apa adanya: TIDAK SATU PUN tab di sini menyaring menurut
// pemanggil. Layar ini memang dirancang sebagai pandangan PENYELIA — Report Definition-nya
// menyaring unit organisasi penugasan, bukan pemegangnya. Sampai `TKT-F3-004` selesai,
// setiap pengguna yang jabatannya cocok melihat nomor polis dan nama tertanggung SELURUH
// klaim pada unit organisasinya.
//
// Dua hal yang membatasi taruhannya, dan keduanya perlu disebut apa adanya:
//
//   - Modul ini MEMBACA saja. Tidak ada satu pun aksi yang mengubah data — bahkan tombol
//     pada kolom terakhir layar lama pun tidak menulis apa-apa (lihat kepala paket).
//   - Setiap pembukaan DICATAT, bukan hanya yang mencurigakan. Lihat usecase.List.
//
// Yang kedua bukan pengganti kewenangan; ia hanya membuat pembukaannya dapat ditelusuri
// setelah terjadi. `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru
// untuk keadaan seperti ini.
//
// # Kenapa TIDAK ada rute tulis, berbeda dari modul Inbox Manager Receive / PUCL
//
// Karena layar lama tidak punya satu pun aksi tulis. Modul PUCL menyediakan rute penolak
// (`/tindakan`) supaya tombol cetak surat di sana menjawab dengan alasan alih-alih dengan
// "halaman tidak ditemukan"; di sini tidak ada tombol seperti itu untuk dijawab.
//
// Menambahkan rute penolak yang tidak pernah dipanggil justru menyesatkan: ia menyiratkan
// ada kemampuan yang tertunda, padahal yang benar adalah kemampuan itu memang tidak pernah
// ada.
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

		perPortal.Get("/inbox-manager-admin/tab", h.Metadata)
		perPortal.Get("/inbox-manager-admin", h.List)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — termasuk dibuka ulang dari
		// riwayat peramban dengan tab yang sama.
		perPortal.Get("/inbox-manager-admin/ekspor", h.Export)
	})
}
