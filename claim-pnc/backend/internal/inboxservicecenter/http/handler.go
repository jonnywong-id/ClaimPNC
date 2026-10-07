package inboxservicecenterhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-service-center/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab dan kolomnya adalah bentuk layar, bukan data
// entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-service-center.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxservicecenter.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
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
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// Detail menangani GET /api/inbox-service-center/{id}.
//
// Klaim yang tidak ada dan klaim milik petugas lain dijawab sama — 404. Membedakannya
// memberi tahu penanya bahwa sebuah ID nyata, dan ID di tabel ini berurutan.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxservicecenter.ErrCallerUnknown)
		return
	}

	detailed, err := h.Service.Detail(r.Context(), active.Alias, caller, chi.URLParam(r, "id"))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDetailResponse(detailed, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxservicecenter.Caller, bool) {
	if h.Caller == nil {
		return inboxservicecenter.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
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
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
