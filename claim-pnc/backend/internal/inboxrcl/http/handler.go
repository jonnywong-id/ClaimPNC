package inboxrclhttp

import (
	"net/http"
	"time"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-rcl/keterangan.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-rcl.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxrcl.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxrcl.Filter{
			Search: query.Get("cari"),
			Offset: nonNegativeNumber(query.Get("lewati")),
			Limit:  nonNegativeNumber(query.Get("batas")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias, h.location))
}

func (h *Handler) readCaller(r *http.Request) (inboxrcl.Caller, bool) {
	if h.caller == nil {
		return inboxrcl.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxrcl.Caller{}, false
	}
	return inboxrcl.Caller{Login: caller.Login}, true
}

// nonNegativeNumber membaca angka dari parameter query; nilai yang tidak terbaca menjadi 0
// dan Filter.Normalize membetulkannya menjadi nilai bawaan.
func nonNegativeNumber(raw string) int { return httpquery.NonNegative(raw) }

// jakarta mengembalikan zona WIB; offset tetap +07:00 bila basis data zona waktu tidak ada.
func jakarta() *time.Location { return clock.Jakarta() }
