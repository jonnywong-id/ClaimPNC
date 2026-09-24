package mastersupplierhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua puluh tiga isian pendek, dengan satu yang dapat panjang
// (Keterangan, sampai 500 karakter). 64 KiB sudah jauh lebih dari cukup, dan batasnya ada
// supaya permintaan bertubuh raksasa ditolak sebelum memakan memori.
const maxRequestBody = 64 << 10

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Berbeda dari Master Bengkel, identitas ini BENAR-BENAR TERSIMPAN: ia menjadi kunci
// `USERKLAIMID` di dalam dokumen supplier dan kolom `USER_REQ` pada baris permintaan
// persetujuan. Keduanya ditulis sistem lama juga.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master supplier.
type Handler struct {
	service       *usecase.Service
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerReader
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul master supplier.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("mastersupplier/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("mastersupplier/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("mastersupplier/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/supplier.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada nama, kota, atau contact person
//
// Ia DITAMBAHKAN. Layar lama tidak punya penyaring apa pun — `Section/InboxMasterSupplier`
// hanya punya tombol New Supplier, Edit, dan Refresh — dan gridnya membaca page list yang
// tidak ada satu pun rule di export yang mengisinya (`R-16`). Tanpa kueri lamanya, tidak
// ada yang dapat ditiru; yang dapat dilakukan adalah membuat daftar yang dapat
// dipersempit.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.List(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Supplier: toListDTO(list),
		Portal:   active.Alias,
	})
}

// Get menangani GET /master/supplier/{id}.
//
// Ia padanan `Activity/GetDataSupplier_pre`, termasuk penurunan JENIS_STATUS dari
// SUPPLIER_HE yang dikerjakan adapter saat membaca.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastersupplier.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Supplier: toDTO(found),
		Portal:   active.Alias,
	})
}

// Create menangani POST /master/supplier.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam menulis master dengan kunci USERKLAIMID
		// kosong — kolom yang justru menjadi satu-satunya jejak pelaku pada tabel ini.
		h.writeError(w, r, errors.New("mastersupplier/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID, SUPPLIER_HE,
	// dan STS_AKTIF yang ketiganya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
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

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("mastersupplier/http: identitas pemanggil tidak ada di konteks"))
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

	saved, err := h.service.Save(r.Context(), active.Alias, id, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
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

	list, err := h.service.ListBranches(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, BranchListResponse{
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

	list, err := h.service.SearchCities(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, CityListResponse{
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

	list, err := h.service.ListCountries(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, CountryListResponse{
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

	list, err := h.service.ListBanks(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, BankListResponse{
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

	set, err := h.service.ListCodes(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, CodeListResponse{
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
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Pada form berisi dua puluh tiga isian yang lima belas di
	// antaranya wajib, itu kelas cacat yang paling mudah lolos.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	return true
}

// Mount mendaftarkan rute modul master supplier.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
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
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
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
