package inboxrclhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini — tipe
// milik modul ini, bukan tipe modul auth; jembatannya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK — padanan
	// `OperatorID.pyUserIdentifier`, kunci pencarian identitas lamanya.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox RCL.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	location   *time.Location
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
//
// Tidak ada Clock: berbeda dari Inbox Analyst Doctor, layar ini tidak punya kolom durasi —
// kelima judul kolom di harness tidak memuatnya.
type Options struct {
	Service   *usecase.Service
	GetCaller CallerReader

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	Logger              *slog.Logger
	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox RCL.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

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
func nonNegativeNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// jakarta mengembalikan zona WIB; offset tetap +07:00 bila basis data zona waktu tidak ada.
func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}
