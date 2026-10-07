package masterspareparthttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua puluh isian pendek dan tidak punya daftar anak sama
// sekali. 32 KiB sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh
// raksasa ditolak sebelum memakan memori.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya: `SetApprovalAllMaster` menelusuri seluruh baris
// bercentang berapa pun jumlahnya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani
// produksi pada tabel yang sama (D-21).
const maxDecisionRows = 200

// Options menangani GET /master/sparepart/pilihan.
//
// Berbeda dari Master Panel yang daftar pilihannya konstanta di dalam kode, isi di sini
// DIBACA DARI BASIS DATA entitas yang sedang dibuka. Karena itu rutenya ikut dipasangi
// pemeriksaan portal — menyajikan daftar kategori satu entitas kepada entitas lain adalah
// kebocoran yang justru dicegah `R-20`.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.Service.Options(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, toOptionsDTO(set.Category, set.Type, active.Alias))
}

// nameIndexOf menyusun pemetaan kode acuan ke namanya untuk satu permintaan.
//
// Kegagalannya DITELAN dengan sengaja, dan dicatat sebagai peringatan: nama kategori dan
// tipe adalah hiasan pada daftar yang isinya tetap terbaca tanpa keduanya. Lihat catatan
// pada List.
func (h *Handler) nameIndexOf(r *http.Request, portalAlias string) nameIndex {
	set, err := h.Service.Options(r.Context(), portalAlias)
	if err != nil {
		logging.From(r.Context(), h.Logger).Warn(
			"nama kategori dan tipe sparepart tidak dapat dibaca; daftar tetap disajikan dengan kodenya",
			slog.String("portal", portalAlias),
			slog.String("galat", err.Error()))
		return newNameIndex(nil, nil)
	}
	return newNameIndex(set.Category, set.Type)
}

// Mount mendaftarkan rute modul master sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Termasuk `/pilihan`, dan itu berbeda dari Master Panel. Alasannya: daftar pilihan modul
// ini DIBACA DARI BASIS DATA entitas — `GCNM_M_SPAREPART_CATEGORY` dan
// `GCNM_M_SPAREPART_TYPE` ada di basis data setiap entitas, dan isinya berbeda. Pada Master
// Panel daftarnya konstanta yang ditanam di dalam activity Pega, sehingga di sana pemeriksaan
// portal memang tidak melindungi apa pun.
//
// # Urutan pendaftaran
//
// Kedua rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui
// hal itu untuk yakin bahwa `/keputusan` tidak pernah terbaca sebagai sebuah ID sparepart.
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
// Tidak ada DELETE terhadap sparepart. Sistem lama tidak punya satu pun terhadap tabel ini,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis — sparepart yang tidak lagi
// dipakai ditolak atau ditandai lewat STS_AKTIF, bukan dibuang.
//
// Tidak ada pula endpoint unggah. Kedua tombolnya di layar lama — "Upload Document" dan
// "Upload Data Master Sparepart" — memanggil local action yang TIDAK ADA di export (`R-16`),
// sehingga susunan kolom CSV-nya, validasinya, dan apakah baris hasil unggah masuk antrean
// persetujuan seluruhnya tidak diketahui. Tombolnya tetap digambar dalam keadaan mati; lihat
// SparepartPage.tsx.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/sparepart/pilihan", h.Options)
		perPortal.Post("/master/sparepart/keputusan", h.Decide)

		perPortal.Get("/master/sparepart", h.List)
		perPortal.Post("/master/sparepart", h.Create)
		perPortal.Get("/master/sparepart/{id}", h.Get)
		perPortal.Put("/master/sparepart/{id}", h.Save)
	})
}
