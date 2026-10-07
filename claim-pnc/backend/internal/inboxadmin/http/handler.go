package inboxadminhttp

import (
	"net/http"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-admin/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan isi dropdown adalah bentuk
// layar, bukan data entitas. Berbeda dengan modul View History Claim, membuka layar ini
// tidak memakai jatah apa pun — layar ini tidak punya gerbang proteksi data.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-admin.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxadmin.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
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
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxadmin.Caller, bool) {
	if h.Caller == nil {
		return inboxadmin.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
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
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
