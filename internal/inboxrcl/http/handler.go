package inboxrclhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
	now        func() time.Time
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
//
// Now hanya dipakai keputusan dokter RCL (waktu kirim PUCL, tugas baru). Kosong berarti
// time.Now.
type Options struct {
	Service   *usecase.Service
	GetCaller CallerReader

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	Now func() time.Time

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

	now := o.Now
	if now == nil {
		now = time.Now
	}

	return &Handler{
		now:        now,
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

// Detail menangani GET /api/inbox-rcl/klaim/{nomor} — isi layar kerja `RCLDokter`.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
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

	detail, err := h.service.Detail(r.Context(), active.Alias, caller, chi.URLParam(r, "nomor"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailResponse(detail, active.Alias, h.location))
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
