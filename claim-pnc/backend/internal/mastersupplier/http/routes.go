package mastersupplierhttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua puluh tiga isian pendek, dengan satu yang dapat panjang
// (Keterangan, sampai 500 karakter). 64 KiB sudah jauh lebih dari cukup, dan batasnya ada
// supaya permintaan bertubuh raksasa ditolak sebelum memakan memori.
const maxRequestBody = 64 << 10

// Create menangani POST /master/supplier.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.Caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam menulis master dengan kunci USERKLAIMID
		// kosong — kolom yang justru menjadi satu-satunya jejak pelaku pada tabel ini.
		h.WriteError(w, r, errors.New("mastersupplier/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.Service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.Logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID, SUPPLIER_HE,
	// dan STS_AKTIF yang ketiganya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.WriteResponse(w, r, http.StatusCreated, SingleResponse{
		Supplier: toDTO(saved),
		Portal:   active.Alias,
	})
}

// Save menangani PUT /master/supplier/{id}.
//
// Nama yang dikirim wajib SAMA dengan yang tersimpan; lihat usecase.Service.Save.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.Caller(r.Context())
	if !known {
		h.WriteError(w, r, errors.New("mastersupplier/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastersupplier.ErrNotFound)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.Service.Save(r.Context(), active.Alias, id, request.toInput(),
		usecase.Actor{Login: by.Login}, h.Logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
		Supplier: toDTO(saved),
		Portal:   active.Alias,
	})
}

// Branches menangani GET /master/supplier/cabang.
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

// Cities menangani GET /master/supplier/kota?cari=...
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

// Countries menangani GET /master/supplier/negara.
func (h *Handler) Countries(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.ListCountries(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, CountryListResponse{
		Country: toCountryListDTO(list),
		Portal:  active.Alias,
	})
}

// Banks menangani GET /master/supplier/bank.
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

// Codes menangani GET /master/supplier/sandi.
//
// Kelima daftar dropdown bersandi dikirim sekaligus — lihat CodeListResponse dan
// mastersupplier.CodeOption untuk alasan daftarnya dibaca dari data.
func (h *Handler) Codes(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.Service.ListCodes(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, CodeListResponse{
		PartnerStatus: toCodeListDTO(set.PartnerStatus),
		SupplyType:    toCodeListDTO(set.SupplyType),
		SupplierType:  toCodeListDTO(set.SupplierType),
		Active:        toCodeListDTO(set.Active),
		AutoPayment:   toCodeListDTO(set.AutoPayment),
		Portal:        active.Alias,
	})
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	return httpjson.Decode(w, r, maxRequestBody, target, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
}

// Mount mendaftarkan rute modul master supplier.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif. Tidak ada satu pun yang isinya milik aplikasi:
// master dan kelima lookup-nya dibaca dari basis data entitas, dan dua entitas punya
// daftar cabang serta supplier yang berbeda. Menyajikan daftar satu entitas kepada entitas
// lain adalah kebocoran yang justru dicegah R-20.
//
// # Urutan pendaftaran
//
// Kelima rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}`
// pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu
// mengetahui hal itu untuk yakin bahwa `/kota` tidak pernah terbaca sebagai sebuah ID
// supplier.
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
// Tidak ada DELETE. Sistem lama tidak punya satu pun terhadap tabel ini — layarnya bahkan
// tidak punya tombolnya — dan `D-66` melarang penghapusan fisik data bernilai bisnis.
// Supplier yang tidak lagi dipakai ditandai lewat Status Aktif, bukan dibuang.
//
// Tidak ada pula endpoint keputusan persetujuan. Sisi pemutus antrean `proteksi_klaimmbu`
// TIDAK ADA di export sama sekali (`R-16`); membangunnya berarti mengarang aturan yang
// menentukan supplier mana yang boleh dipakai.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/supplier/cabang", h.Branches)
		perPortal.Get("/master/supplier/kota", h.Cities)
		perPortal.Get("/master/supplier/negara", h.Countries)
		perPortal.Get("/master/supplier/bank", h.Banks)
		perPortal.Get("/master/supplier/sandi", h.Codes)

		perPortal.Get("/master/supplier", h.List)
		perPortal.Post("/master/supplier", h.Create)
		perPortal.Get("/master/supplier/{id}", h.Get)
		perPortal.Put("/master/supplier/{id}", h.Save)
	})
}
