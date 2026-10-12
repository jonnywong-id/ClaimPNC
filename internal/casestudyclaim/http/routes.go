package casestudyclaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Case Study Claim.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc
// ia dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa jalurnya "case-study-claim"
//
// Itu nama BUTIR MENU-nya di `POOLDATA.M_MENU_APLIKASI_PNC` — MENU_ID 74, "Case Study
// Claim" — dan `D-81` menetapkan rute mengikuti nama butir menu, bukan nama harness.
//
// # Seluruh rutenya menuntut portal aktif
//
// Keempat tabel yang dibaca modul ini ada di basis data setiap entitas (`ADR-0030`),
// sehingga setiap permintaan harus menyebut entitas mana yang dibacanya lewat header
// `X-Portal`. Permintaan yang tidak menyebutkannya DITOLAK oleh middleware — tidak pernah
// dilayani portal utama sebagai cadangan.
//
// Taruhannya di layar ini lebih besar daripada rata-rata: barisnya adalah klaim di atas
// Rp 5 miliar beserta nama tertanggung, nomor polis, dan kronologinya (`R-20`,
// `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya — keadaan yang sama dengan
// seluruh rute lain hari ini. Penegakan "apakah peran pemanggil memiliki menu ini" adalah
// `TKT-F3-005`, yang bergantung pada tabel peran `TKT-F3-004`; tabel itu dapat dibangun
// tetapi belum dapat diisi karena penugasan operator ke peran tidak ada di basis data
// maupun di export (`11-SECURITY.md` §3.1).
//
// Di Pega, butir menunya dijaga `IsGCNMInternalUser` pada
// `Navigation/pyCaseWorkerNavigation-Navigation.xml`. Aturan itu tidak dibawa; yang
// menentukan siapa melihat butirnya sekarang adalah `M_OTORISASI_PNC`.
//
// Yang perlu disadari: rute TULIS di bawah tidak dijaga peran apa pun hari ini, dan
// catatan telaah yang ditulisnya masuk ke `POOLDATA.T_CLAIM_PNC` — tabel klaim yang
// sebenarnya. Yang tersisa sebagai kontrol hanyalah jejak di sisi peladen (`D-59`).
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Route("/case-study-claim", func(study chi.Router) {
		study.Use(portalhttp.ActivePortal(portalDeps))

		// Keterangan layar dipisahkan dari daftar, bukan disisipkan ke dalam responsnya.
		//
		// Keduanya berubah pada irama yang sangat berbeda: isi dropdown dan susunan kolom
		// dibaca dari KODE dan tidak berubah selama aplikasi berjalan, sedangkan daftarnya
		// ditembak ulang setiap kali pengguna menekan "Lihat Data" atau berpindah halaman.
		// Menyatukannya berarti mengirim ulang 24 definisi kolom pada setiap penekanan
		// tombol paginasi.
		study.Get("/penyaring", h.Metadata)

		study.Get("/", h.List)

		// Unduhan dipisahkan menjadi jalurnya sendiri, bukan parameter `format=csv` pada
		// daftar. Keduanya berbeda sifat: yang satu dipaginasi dan dibaca layar, yang lain
		// mengalir sampai habis dan diterima sebagai berkas. Menyatukannya membuat satu
		// endpoint punya dua bentuk respons dan dua batas ukuran.
		study.Get("/unduh", h.Export)

		// Satu-satunya rute yang MENULIS.
		//
		// PUT, bukan POST: ia menimpa satu kolom dengan nilai yang diberikan, dan
		// mengirimnya dua kali menghasilkan keadaan yang persis sama. Itu definisi
		// idempoten, dan PUT-lah yang menyatakannya kepada klien maupun kepada proksi mana
		// pun di antaranya.
		study.Put("/{nomor}/catatan", h.SaveRemark)
	})
}
