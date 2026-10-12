package outstandingclaimhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/outstandingclaim"
	"claim-pnc/internal/outstandingclaim/usecase"
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
	// Di modul ini ia tidak menyaring apa pun — ia yang dicatat pada setiap pembukaan
	// rincian. Memakai NIK di sini akan membuat jejaknya tidak dapat dicocokkan dengan
	// jejak modul lain, yang seluruhnya mencatat login.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Outstanding Claim.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Outstanding Claim.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Layout menangani GET /api/outstanding-claim/tata-letak.
//
// Ia GET dan tidak mengubah apa pun: kelompok isian, judul, dan susunan grid adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Layout(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toLayoutResponse(h.service.Layout(), active.Alias))
}

// Detail menangani GET /api/outstanding-claim/{no_klaim}.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, outstandingclaim.ErrCallerUnknown)
		return
	}

	detail, err := h.service.Detail(
		r.Context(),
		active.Alias,
		caller,
		chi.URLParam(r, "no_klaim"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailResponse(detail, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (outstandingclaim.Caller, bool) {
	if h.caller == nil {
		return outstandingclaim.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return outstandingclaim.Caller{}, false
	}
	return outstandingclaim.Caller{Login: caller.Login}, true
}
