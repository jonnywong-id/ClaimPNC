package masterbengkelhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat tiga puluh tiga isian pendek — lebih banyak daripada modul
// master lain, tetapi tidak satu pun yang panjang. 64 KiB sudah jauh lebih dari cukup,
// dan batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori.
const maxRequestBody = 64 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya: `SetApprovalAllMaster` menelusuri seluruh baris
// bercentang berapa pun jumlahnya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani
// produksi pada tabel yang sama (D-21).
const maxDecisionRows = 200

// Branches menangani GET /master/bengkel/cabang.
func (h *Handler) Branches(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.ListBranches(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, BranchListResponse{
		Branch: toBranchListDTO(list),
		Portal: active.Alias,
	})
}

// Cities menangani GET /master/bengkel/kota?cari=...
func (h *Handler) Cities(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.SearchCities(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, CityListResponse{
		City:   toCityListDTO(list),
		Portal: active.Alias,
	})
}

// Banks menangani GET /master/bengkel/bank.
func (h *Handler) Banks(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.ListBanks(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, BankListResponse{
		Bank:   toBankListDTO(list),
		Portal: active.Alias,
	})
}

// Mount mendaftarkan rute modul master bengkel.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif. Tidak ada satu pun yang isinya milik aplikasi:
// master dan ketiga lookup-nya dibaca dari basis data entitas, dan dua entitas punya
// daftar cabang serta bengkel yang berbeda. Menyajikan daftar satu entitas kepada
// entitas lain adalah kebocoran yang justru dicegah R-20.
//
// # Urutan pendaftaran
//
// Keempat rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter
// `{id}` pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga
// urutannya sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca
// tidak perlu mengetahui hal itu untuk yakin bahwa `/kota` tidak pernah terbaca sebagai
// sebuah ID bengkel.
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
// Tidak ada DELETE. Sistem lama tidak punya satu pun terhadap tabel ini, dan `D-66`
// melarang penghapusan fisik data bernilai bisnis — bengkel yang tidak lagi dipakai
// ditolak atau ditandai lewat status bengkel, bukan dibuang.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/bengkel/cabang", h.Branches)
		perPortal.Get("/master/bengkel/kota", h.Cities)
		perPortal.Get("/master/bengkel/bank", h.Banks)
		perPortal.Post("/master/bengkel/keputusan", h.Decide)

		perPortal.Get("/master/bengkel", h.List)
		perPortal.Post("/master/bengkel", h.Create)
		perPortal.Get("/master/bengkel/{id}", h.Get)
		perPortal.Put("/master/bengkel/{id}", h.Save)

		// Dokumen lampiran. Jalurnya bersarang di bawah bengkelnya karena ia memang milik
		// satu baris — dan `10-API-STRATEGY.md` §2 membatasi sarang pada dua tingkat,
		// yang masih terpenuhi.
		perPortal.Post("/master/bengkel/{id}/dokumen", h.UploadDocument)
		perPortal.Get("/master/bengkel/{id}/dokumen", h.Document)
		perPortal.Get("/master/bengkel/{id}/dokumen/berkas", h.DocumentFile)
	})
}
