package masterautoclaimhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterautoclaim/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat sepuluh isian pendek; 64 KiB sudah jauh lebih dari cukup.
// Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori, bukan
// setelah.
const maxRequestBody = 64 << 10

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja — lihat usecase.Actor untuk alasan kenapa yang dipakai LOGIN dan
// bukan NIK.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak
// mengimpor lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah
// tanpa menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master auto claim.
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

// NewHandler membentuk handler modul master auto claim.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterautoclaim/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterautoclaim/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterautoclaim/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/auto-claim.
//
// # Penyaringnya
//
//	?status=0|1|2     wajib dikenal; bila tidak disebut dianggap "1"
//	?komite_saya=true hanya baris yang KOMITE-nya adalah pemanggil
//
// Keduanya cerminan langsung dari kedua parameter Activity/BrowseAutoKlaim_act. Keempat
// tab layar lama menjadi empat kombinasi keduanya:
//
//	Master Auto Klaim   status=1
//	Komite Approval     status=0&komite_saya=true
//	Waiting Approval    status=0
//	Reject              status=2
//
// Nilai KOMITE yang dibandingkan TIDAK diterima dari permintaan; ia diambil dari sesi
// pemanggil di lapisan aplikasi. Menerimanya dari layar berarti siapa pun dapat melihat
// antrean persetujuan komite lain hanya dengan mengganti satu nilai.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya
		// cacat perakitan gagal keras, bukan diam-diam menyajikan antrean komite
		// kepada pemanggil tanpa identitas.
		h.writeError(w, r, errors.New("masterautoclaim/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	status := masterautoclaim.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = masterautoclaim.ApprovalStatus(raw)
	}
	committeeOnly := r.URL.Query().Get("komite_saya") == "true"

	list, err := h.service.List(r.Context(), active.Alias, status, committeeOnly, usecase.Actor{Login: by.Login})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		AutoClaim: toListDTO(list),
		Status:    string(status),
		Portal:    active.Alias,
	})
}

// Get menangani GET /master/auto-claim/{inisial}.
//
// Padanan Activity/UpdateMstAutoClaim_act1, yang memuat satu baris ke form penyuntingan.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	initial := chi.URLParam(r, "inisial")
	if initial == "" {
		h.writeModuleError(w, r, masterautoclaim.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, initial)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		AutoClaim: toDTO(found),
		Portal:    active.Alias,
	})
}

// Create menangani POST /master/auto-claim.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterautoclaim/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request CreateRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, masterautoclaim.Input{
		Initial:         request.Initial,
		ReceiverName:    request.ReceiverName,
		BankName:        request.BankName,
		AccountNumber:   request.AccountNumber,
		MaxPercent:      request.MaxPercent,
		ReporterPIC:     request.ReporterPIC,
		ReporterEmail:   request.ReporterEmail,
		ReceiverAddress: request.ReceiverAddress,
		ClientID:        request.ClientID,
		ClientName:      request.ClientName,
	}, usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk KOMITE dan
	// APPROVAL yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		AutoClaim: toDTO(saved),
		Portal:    active.Alias,
	})
}

// Save menangani PUT /master/auto-claim/{inisial}.
//
// Ia melayani TIGA tombol sekaligus — Simpan, Approve, dan Reject — dibedakan oleh
// `status` di dalam badan permintaan. Bentuk itu bukan penyederhanaan kami melainkan
// bentuk sistem lama: satu activity, `UpdateMstAutoClaim_act`, dengan satu parameter
// `stsapprove`. Keputusan Work Owner 2026-09-19 mempertahankannya.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterautoclaim/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	initial := chi.URLParam(r, "inisial")
	if initial == "" {
		h.writeModuleError(w, r, masterautoclaim.ErrNotFound)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	// Inisial dan nama penerima sengaja TIDAK diisi di sini: keduanya tidak datang dari
	// badan permintaan, dan lapisan aplikasi memeriksanya dengan CheckEditable yang
	// memang tidak menuntut keduanya. Mengisinya dengan nilai palsu hanya supaya
	// pemeriksaan lolos akan menyembunyikan aturan yang sebenarnya berlaku.
	saved, err := h.service.Save(r.Context(), active.Alias, initial, masterautoclaim.Input{
		BankName:        request.BankName,
		AccountNumber:   request.AccountNumber,
		MaxPercent:      request.MaxPercent,
		ReporterPIC:     request.ReporterPIC,
		ReporterEmail:   request.ReporterEmail,
		ReceiverAddress: request.ReceiverAddress,
		ClientID:        request.ClientID,
		ClientName:      request.ClientName,
		Status:          masterautoclaim.ApprovalStatus(request.Status),
	}, usecase.Actor{Login: by.Login})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		AutoClaim: toDTO(saved),
		Portal:    active.Alias,
	})
}

// BusinessSources menangani GET /master/auto-claim/sumber-bisnis?cari=...
func (h *Handler) BusinessSources(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.SearchBusinessSources(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, BusinessSourceListResponse{
		BusinessSource: toBusinessSourceListDTO(list),
		Portal:         active.Alias,
	})
}

// Clients menangani GET /master/auto-claim/client?cari=...
func (h *Handler) Clients(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.SearchClients(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ClientListResponse{
		Client: toClientListDTO(list),
		Portal: active.Alias,
	})
}

// Banks menangani GET /master/auto-claim/bank.
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
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama
	// field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa
	// satu pun tanda bahwa ada yang salah.
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

// Mount mendaftarkan rute modul master auto claim.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif. Tidak ada satu pun yang isinya milik aplikasi:
// master, ketiga lookup-nya, dan penyetuju komitenya semuanya dibaca dari basis data
// entitas. Dua entitas punya sumber bisnis dan daftar client yang berbeda, dan
// menyajikan daftar satu entitas kepada entitas lain adalah kebocoran yang justru
// dicegah R-20.
//
// # Urutan pendaftaran
//
// Ketiga rute lookup didaftarkan LEBIH DULU daripada rute ber-parameter `{inisial}`
// pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak
// perlu mengetahui hal itu untuk yakin bahwa `/bank` tidak pernah terbaca sebagai
// sebuah inisial.
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
// melarang penghapusan fisik data bernilai bisnis — baris yang tidak lagi dipakai
// ditolak komite, bukan dibuang.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/auto-claim/sumber-bisnis", h.BusinessSources)
		perPortal.Get("/master/auto-claim/client", h.Clients)
		perPortal.Get("/master/auto-claim/bank", h.Banks)

		perPortal.Get("/master/auto-claim", h.List)
		perPortal.Post("/master/auto-claim", h.Create)
		perPortal.Get("/master/auto-claim/{inisial}", h.Get)
		perPortal.Put("/master/auto-claim/{inisial}", h.Save)
	})
}
