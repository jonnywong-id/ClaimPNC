package inboxadminhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/usecase"
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
	// Itulah yang dicocokkan ke `PXCREATEOPNAME` dan `PXASSIGNEDOPERATORID` pada tabel
	// Pega. Memakai NIK di sini akan membuat tiga tab tampak kosong bagi setiap pengguna.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Admin.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
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

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Admin.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-admin/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan isi dropdown adalah bentuk
// layar, bukan data entitas. Berbeda dengan modul View History Claim, membuka layar ini
// tidak memakai jatah apa pun — layar ini tidak punya gerbang proteksi data.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-admin.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxadmin.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxadmin.QueryInput{
			Tab:      query.Get("tab"),
			Business: query.Get("bisnis"),
			Keyword:  query.Get("cari"),
		},
		inboxadmin.Pagination{
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

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxadmin.Caller, bool) {
	if h.caller == nil {
		return inboxadmin.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxadmin.Caller{}, false
	}
	return inboxadmin.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan inboxadmin.Pagination.Normalize
// membetulkannya menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman
// pertama adalah jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
