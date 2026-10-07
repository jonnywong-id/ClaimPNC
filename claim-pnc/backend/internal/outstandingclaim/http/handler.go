package outstandingclaimhttp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/outstandingclaim"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Layout menangani GET /api/outstanding-claim/tata-letak.
//
// Ia GET dan tidak mengubah apa pun: kelompok isian, judul, dan susunan grid adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Layout(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toLayoutResponse(h.Service.Layout(), active.Alias))
}

// Detail menangani GET /api/outstanding-claim/{no_klaim}.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, outstandingclaim.ErrCallerUnknown)
		return
	}

	detail, err := h.Service.Detail(
		r.Context(),
		active.Alias,
		caller,
		chi.URLParam(r, "no_klaim"),
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDetailResponse(detail, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (outstandingclaim.Caller, bool) {
	if h.Caller == nil {
		return outstandingclaim.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return outstandingclaim.Caller{}, false
	}
	return outstandingclaim.Caller{Login: caller.Login}, true
}
