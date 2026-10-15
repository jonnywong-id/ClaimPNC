package inboxservicecenterhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/inboxservicecenter/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Itulah yang dicocokkan ke kolom `PIC` pada `T_KLAIM_PORTAL_REKANAN`. Memakai NIK di
	// sini akan membuat keempat tab tampak kosong bagi setiap pengguna.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Service Center.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan diimpor
	// dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa menariknya
	// serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Service Center.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-service-center/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab dan kolomnya adalah bentuk layar, bukan data
// entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-service-center.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxservicecenter.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxservicecenter.QueryInput{
			Tab:     query.Get("tab"),
			Keyword: query.Get("cari"),
		},
		inboxservicecenter.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// Detail menangani GET /api/inbox-service-center/{id}.
//
// Klaim yang tidak ada dan klaim milik petugas lain dijawab sama — 404. Membedakannya
// memberi tahu penanya bahwa sebuah ID nyata, dan ID di tabel ini berurutan.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxservicecenter.ErrCallerUnknown)
		return
	}

	detailed, err := h.service.Detail(r.Context(), active.Alias, caller, chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailResponse(detailed, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxservicecenter.Caller, bool) {
	if h.caller == nil {
		return inboxservicecenter.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxservicecenter.Caller{}, false
	}
	return inboxservicecenter.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
