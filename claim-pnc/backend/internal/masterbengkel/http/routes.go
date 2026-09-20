package masterbengkelhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/usecase"
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

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja. Ia TIDAK pernah tersimpan ke basis data — `POOLDATA.BENGKEL_HE` tidak
// punya kolom pencatat pelaku maupun waktu — dan hanya dipakai untuk log.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master bengkel.
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

// NewHandler membentuk handler modul master bengkel.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterbengkel/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterbengkel/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterbengkel/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/bengkel.
//
// # Penyaringnya
//
//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
//	?cari=...      mempersempit pada nama bengkel, kota, atau cabang
//
// Yang pertama adalah cerminan ketiga tab `Section/BrowseMasterHE-Section.xml`:
//
//	APPROVE           status=1
//	WAITING APPROVAL  status=0
//	REJECT            status=2
//
// Yang kedua DITAMBAHKAN. Report definition lama membatasi hasilnya di 500 baris
// (`pyMaxRecords=500`) tanpa satu pun cara mempersempitnya dari layar; pencarian di sini
// yang menggantikan pemotongan itu.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	status := masterbengkel.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = masterbengkel.ApprovalStatus(raw)
	}

	list, err := h.service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Bengkel: toListDTO(list),
		Status:  string(status),
		Portal:  active.Alias,
	})
}

// Get menangani GET /master/bengkel/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterbengkel.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Bengkel: toDTO(found),
		Portal:  active.Alias,
	})
}

// Create menangani POST /master/bengkel.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam menulis master tanpa identitas pemanggil
		// di log.
		h.writeError(w, r, errors.New("masterbengkel/http: identitas pemanggil tidak ada di konteks"))
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

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID_BENGKEL
	// dan APPROVAL yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Bengkel: toDTO(saved),
		Portal:  active.Alias,
	})
}

// Save menangani PUT /master/bengkel/{id}.
//
// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
// persis seperti `Activity/UpdateBengkelHE_act` step 7 yang menetapkan `APPROVAL := "0"`
// tanpa syarat apa pun. Keputusan komite menempuh Decide.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterbengkel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterbengkel.ErrNotFound)
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
		Bengkel: toDTO(saved),
		Portal:  active.Alias,
	})
}

// Decide menangani POST /master/bengkel/keputusan.
//
// Ia padanan `Activity/SetApprovalAllMaster`: daftar baris bercentang beserta satu
// status, ditetapkan sekaligus.
//
// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
//
// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai
// peristiwa, bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan
// membuat keputusan yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterbengkel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, masterbengkel.OneViolation("id_bengkel",
			"Terlalu banyak bengkel dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := masterbengkel.ApprovalStatus(request.Status)
	changed, err := h.service.Decide(r.Context(), active.Alias, request.ID, status,
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, DecisionResponse{
		Changed:     changed,
		Status:      string(status),
		StatusLabel: status.Label(),
		Portal:      active.Alias,
	})
}

// Branches menangani GET /master/bengkel/cabang.
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

// Cities menangani GET /master/bengkel/kota?cari=...
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

// Banks menangani GET /master/bengkel/bank.
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

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya
// sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Pada form berisi tiga puluh tiga isian, itu kelas cacat
	// yang paling mudah lolos.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
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
	})
}
