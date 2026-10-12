package archivedokumenklaimhttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan rute modul Archive Dokumen Klaim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa SELURUH rute di balik pemeriksaan portal
//
// Karena setiap satunya menyentuh basis data entitas. Berkas arsip satu badan hukum bukan
// berkas badan hukum lain, dan mengirimkannya ke sistem Arsip atas nama entitas yang salah
// adalah kesalahan yang tidak meninggalkan satu pun galat — hanya berkas yang tercatat di
// tempat yang keliru.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kewenangan
//
// Rutenya terlindungi sesi. Yang BELUM ada adalah pemeriksaan peran: "apakah peran
// pemanggil memiliki menu ini" adalah `TKT-F3-005`, yang bergantung pada tabel peran
// `TKT-F3-004` — dan tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan
// operator ke peran tidak ada di basis data maupun di export (`11-SECURITY.md` §3.1).
//
// Satu kendali yang SUDAH ada dan tidak menunggu itu: daftar kirim ke cabang disaring
// menurut jabatan pemanggil DI SERVER, bukan dengan menyembunyikan baris di layar.
// Menyembunyikan di layar bukan kendali (`D-59`).
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

		// Isi dropdown dan cakupan lini bisnis pemanggil. GET, karena membukanya tidak
		// mengubah apa pun.
		perPortal.Get("/arsip-dokumen/buka", h.Open)

		// Grid ARCHIVE FILE KLAIM.
		perPortal.Get("/arsip-dokumen", h.Search)

		// Tombol "Export To Excel". Ia didaftarkan sebagai sub-jalur, bukan sebagai
		// parameter pada jalur di atasnya, karena yang dikembalikan BUKAN JSON melainkan
		// berkas — dan pembedaannya harus terbaca dari alamatnya.
		perPortal.Get("/arsip-dokumen/ekspor", h.Export)

		// Calon klaim pada bagian Input Data Archive. Ia sub-jalur, bukan parameter pada
		// jalur di atasnya, karena yang dicari memang benda yang BERBEDA: klaim, bukan
		// berkas arsip.
		perPortal.Get("/arsip-dokumen/klaim", h.SearchClaims)

		// Isi pemilih "Pilih Kode".
		perPortal.Get("/arsip-dokumen/kode-filling", h.FillingCodes)

		// Simpan berkas arsip — menyisipkan bila `id` nol, mengubah bila tidak.
		//
		// POST untuk keduanya, bukan POST dan PUT terpisah. Layar lama punya SATU tombol
		// "Save To Archive" yang melayani keduanya, dan yang menentukan bukan tombolnya
		// melainkan apakah sebuah baris sedang dibuka. Memisahkannya menjadi dua endpoint
		// memaksa layar menebak lebih dulu — dan tebakan yang salah menyisipkan baris
		// ganda alih-alih mengubah yang ada.
		perPortal.Post("/arsip-dokumen", h.Save)

		// Daftar berkas yang belum dikirim ke sistem Arsip.
		perPortal.Get("/arsip-dokumen/kirim-cabang", h.Pending)

		// Kirim satu berkas ke sistem Arsip.
		//
		// POST, dan ia TIDAK idempoten — sistem Arsip menerima berkas yang sama dua kali
		// bila dipanggil dua kali. Yang menahannya adalah pemeriksaan CABANGSTATUS di
		// usecase, bukan sifat metodenya. Kunci idempotensi yang diminta
		// `10-API-STRATEGY.md` §7 belum dapat dipasang: kontrak sistem Arsip tidak
		// menyediakan tempat untuk membawanya.
		perPortal.Post("/arsip-dokumen/{id}/kirim-cabang", h.Send)
	})
}
