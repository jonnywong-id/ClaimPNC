package reportkpihttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Report KPI PNC.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Barisnya memuat penilaian kinerja
// adjuster yang bekerja untuk SATU badan hukum, dan penilaian itu bukan milik badan hukum
// lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan — jatuh ke koneksi bawaan berarti menampilkan penilaian satu badan hukum kepada
// penyelia badan hukum lain tanpa satu pun pesan galat (`R-20`, `TKT-F6-002`).
//
// Rute keterangan layar (`/tab`) ikut di balik pemeriksaan itu meski isinya sama di
// seluruh entitas. Alasannya bukan kerahasiaan melainkan keseragaman perilaku: layar
// memuat keterangan dan data pada saat yang sama, dan satu rute yang lolos tanpa portal
// akan membuat layar tergambar separuh sebelum penolakannya sampai.
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran — `TKT-F3-004` — dan
// itu terasa lebih tajam di sini daripada di layar daftar biasa: isinya PENILAIAN KINERJA
// orang yang dapat dinamai, dan siapa pun yang dapat masuk melihatnya.
//
// Yang membatasi taruhannya, dan keduanya perlu disebut apa adanya:
//
//   - Modul ini MEMBACA saja. Tidak ada satu pun aksi yang mengubah data.
//   - Setiap pembukaan DICATAT beserta penyaringnya — termasuk adjuster dan periode yang
//     dipilih. Lihat usecase.record.
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
//
// # Kenapa jalurnya menyebut `adjuster`
//
// Karena layarnya punya TIGA tab, dan dua di antaranya belum dibangun. Menaruh ringkasan
// di `/api/report-kpi` begitu saja akan memaksa jalur itu dipindahkan ketika tab KPI PIC
// Teknik dan KPI Admin menyusul — dan pemindahan jalur adalah perubahan yang merusak
// klien. Menyebut tabnya sejak awal membuat kedua tab berikutnya cukup menambah jalur
// sebelah.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/report-kpi/tab", h.Metadata)

		// Ringkasan didaftarkan SEBELUM rute rincian yang lebih pendek, dan urutan itu
		// tidak menentukan apa pun pada chi — ia dituliskan begini semata supaya
		// pembacanya melihat keduanya berpasangan.
		perPortal.Get("/report-kpi/adjuster/ringkasan", h.Summary)
		perPortal.Get("/report-kpi/adjuster/pilihan", h.Adjusters)

		// Ekspor adalah GET, bukan POST. Ia tidak mengubah apa pun, dan menjadikannya GET
		// membuat unduhannya dapat dipicu tautan biasa — termasuk dibuka ulang dari
		// riwayat peramban dengan penyaring yang sama.
		perPortal.Get("/report-kpi/adjuster/ekspor", h.Export)

		perPortal.Get("/report-kpi/adjuster", h.Detail)

		// Tab KPI Admin. Susunannya sengaja sejajar dengan tab Adjuster di atas —
		// kartu skor menggantikan ringkasan, rincian dan ekspor sama bentuknya.
		//
		// Ia TIDAK punya rute daftar pilihan: kelompoknya hanya dua dan keduanya tetap,
		// sehingga dikirim bersama metadata alih-alih lewat permintaan tersendiri.
		perPortal.Get("/report-kpi/admin/kartu-skor", h.Scorecard)
		perPortal.Get("/report-kpi/admin/ekspor", h.AdminExport)
		perPortal.Get("/report-kpi/admin", h.AdminDetail)

		// Tab KPI PIC Teknik. Ia hanya punya DUA rute — bukan tiga seperti kedua tab lain
		// — karena keluarannya satu: daftar kartu skor, satu kartu per PIC.
		//
		// Tidak ada rute rincian terpisah, dan itu bukan kekurangan: rinciannya adalah
		// keempat baris di dalam kartu itu sendiri.
		perPortal.Get("/report-kpi/pic-teknik/ekspor", h.PICTeknikExport)
		perPortal.Get("/report-kpi/pic-teknik", h.PICTeknik)

		// Aksi tulis sistem lama. Rutenya ADA supaya tombol di layar menjawab dengan
		// alasan, bukan dengan "halaman tidak ditemukan" — lihat Handler.RejectWrite.
		//
		// Yang nyata di layar ini satu: menekan "Cari" di Pega MENGHITUNG ULANG penilaian
		// setiap kasus survei lalu MENYIMPANNYA. Tabelnya masih dimiliki Pega (`P-1`).
		perPortal.Post("/report-kpi/tindakan", h.RejectWrite)
	})
}
