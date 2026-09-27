package inputreqprotectionhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Input Req Protection.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc ia
// dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa jalurnya "input-req-protection"
//
// Itu nama butir menu yang membuka layarnya di portal Pega
// (`Navigation/pyCaseWorkerNavigation-Navigation.xml`), dan `D-81` menetapkan nama modul
// mengikuti nama yang dipakai Work Owner.
//
// # Kenapa ada rute tulis, padahal modul inbox lain hanya membaca
//
// Layar ini memuat tombol **"Input Open Protection"**
// (`Section/InboxReqProtection_Section-Section.xml`), dan alur `CreateProtection_Flow`
// menempatkan pembuatan proteksi sebagai langkah PERTAMA yang dimiliki modul ini. Jadi
// menulis memang bagian dari tugasnya.
//
// Yang TIDAK didaftarkan di sini: akseptasi. Kolom `STATUS_AKSEPTASI`,
// `TANGGAL_AKSEPTASI`, dan `DIAKSEP_OLEH` dimiliki modul `inboxacceptopenprotection`.
// Pembagian itu yang menjaga `P-1` tetap berlaku meski dua modul menyentuh satu tabel.
//
// Tidak ada rute hapus, dan ketiadaannya disengaja: `D-66` menetapkan tidak ada penghapusan
// fisik pada data bernilai bisnis. Rute yang tidak didaftarkan dijawab chi dengan 405,
// sehingga penambahannya kelak menjadi keputusan sadar — bukan sesuatu yang lolos review.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya — keadaan yang sama dengan
// seluruh rute lain hari ini. Layar lama dibatasi `When/IsReqProtection-When.xml` pada empat
// access group: `PncAdmin`, `PncPICTeknik`, `PNCKomiteTeknik`, dan `Administrators`.
//
// Penegakannya adalah `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`; tabel itu
// dapat dibangun tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di
// basis data maupun di export (`11-SECURITY.md` §3.1). Sampai itu tersedia, setiap pengguna
// yang dapat masuk dapat membuat permintaan proteksi. Dicatat terbuka di
// `docs/keputusan-implementasi.md`.
//
// # Seluruh rutenya menuntut portal aktif
//
// `POOLDATA.T_CLAIM_OPENPROTECTION` ada di basis data setiap entitas (`ADR-0030`), sehingga
// setiap permintaan harus menyebut entitas mana yang dibacanya lewat header `X-Portal`.
// Permintaan yang tidak menyebutkannya DITOLAK middleware — tidak pernah dilayani portal
// utama sebagai cadangan (`R-20`, `TKT-F6-002`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/input-req-protection", func(protection chi.Router) {
		protection.Use(portalhttp.ActivePortal(portalDeps))

		protection.Get("/", h.List)
		protection.Post("/", h.Create)

		// Master tipe proteksi, untuk pilihan pada form.
		//
		// Didaftarkan SEBELUM rute `/{nomor}` dengan sengaja. chi mencocokkan pola statis
		// lebih dulu, sehingga urutannya sebenarnya tidak menentukan — tetapi menaruhnya di
		// atas membuat hubungan itu terbaca oleh yang menambah rute berikutnya.
		//
		// Hanya GET. Pengelolaan masternya belum punya layar, dan rute tulis yang tidak
		// didaftarkan dijawab chi dengan 405 — sehingga penambahannya kelak menjadi
		// keputusan sadar.
		protection.Get("/tipe", h.ListTypes)

		// Pencarian klaim, menggantikan tombol CARI pada form Pega.
		//
		// `Activity/OpenProtection-Act.xml` — yang di Pega dipicu field No Klaim — memuat
		// klaimnya lalu mengisi No Polis, Nama Tertanggung, Object Name, Branch Name,
		// Current Date Of Loss, dan Cause Of Loss Dipilih. Keenamnya tidak pernah diketik.
		//
		// Hanya GET, dan hanya MEMBACA. Tabel klaim dimiliki Pega selama masa paralel;
		// `P-1` melarang menulisnya, tidak melarang membacanya.
		protection.Get("/klaim/{nomor}", h.FindClaim)

		// Nomor proteksi dipakai sebagai kunci jalur, bukan ID teknis.
		//
		// Ia yang dilihat dan disebut pengguna, dan ia pula yang muncul di tautan yang
		// mereka simpan. ID teknis tidak pernah keluar dari penyimpanan — memakainya di URL
		// akan membuat tautan berubah bila baris dipindahkan.
		protection.Get("/{nomor}", h.Get)
		protection.Put("/{nomor}", h.Update)
	})
}
