package mastertipespareparthttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat DUA isian pendek pada jalur simpan, dan satu daftar kunci
// pada jalur keputusan. 32 KiB sudah jauh lebih dari cukup — angkanya disamakan dengan
// modul master lain alih-alih diperkecil, supaya tidak ada satu modul yang diam-diam
// menolak permintaan yang diterima modul tetangganya.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya: setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ratusan kunci baris menghalangi Pega yang sedang melayani produksi
// pada tabel yang sama (D-21).
//
// Dua ratus, sama dengan modul master lain. Pada tabel penggolongan yang isinya berorde
// puluhan sampai ratusan baris, batas ini praktis tidak akan tersentuh.
const maxDecisionRows = 200

// Choices menangani GET /master/tipe-sparepart/pilihan.
//
// Ia daftar kategori yang mengisi dropdown `---PILIH KATEGORI---` pada form.
//
// # Kenapa endpoint tersendiri, bukan disisipkan pada jawaban daftar
//
// Karena keduanya berubah pada irama yang berbeda: daftar tipe berubah setiap kali ada yang
// menyimpan, sedangkan daftar kategori nyaris tidak pernah berubah selama satu sesi.
// Menyisipkannya pada setiap jawaban daftar berarti mengirim ulang isi yang sama pada
// setiap perpindahan tab.
//
// Pola yang sama dipakai `/master/sparepart/pilihan`.
func (h *Handler) Choices(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.Service.Choices(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK,
		toOptionsDTO(set.Category, set.Truncated, active.Alias))
}

// Mount mendaftarkan rute modul master tipe sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// `POOLDATA.GCNM_M_SPAREPART_TYPE` ada di basis data SETIAP entitas dan isinya berbeda.
// Tidak ada satu pun rute di modul ini yang isinya milik aplikasi, sehingga tidak ada yang
// dikecualikan — termasuk `/pilihan`, yang isinya dibaca dari tabel kategori entitas itu.
//
// # Urutan pendaftaran
//
// Rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui
// hal itu untuk yakin bahwa `/pilihan` dan `/keputusan` tidak pernah terbaca sebagai sebuah
// ID tipe.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan
//
// **Tidak ada DELETE.** Sistem lama tidak punya satu pun terhadap tabel ini — kesembilan
// rule yang menyentuhnya tidak memuat satu pun pernyataan hapus — dan `D-66` melarang
// penghapusan fisik data bernilai bisnis. Tipe yang tidak lagi dipakai DITOLAK, bukan
// dibuang; dengan begitu sparepart lama yang sudah menunjuknya tetap dapat menampilkan
// namanya.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/tipe-sparepart/pilihan", h.Choices)
		perPortal.Post("/master/tipe-sparepart/keputusan", h.Decide)

		perPortal.Get("/master/tipe-sparepart", h.List)
		perPortal.Post("/master/tipe-sparepart", h.Create)
		perPortal.Get("/master/tipe-sparepart/{id}", h.Get)
		perPortal.Put("/master/tipe-sparepart/{id}", h.Save)
	})
}
