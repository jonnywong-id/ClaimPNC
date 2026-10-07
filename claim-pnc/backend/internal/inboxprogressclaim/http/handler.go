package inboxprogressclaimhttp

import (
	"net/http"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-progress-claim/bagian.
//
// Ia GET dan tidak mengubah apa pun: daftar region, kolom, dan isi dropdown adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-progress-claim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxprogressclaim.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxprogressclaim.QueryInput{
			View:     query.Get("bagian"),
			Keyword:  query.Get("cari"),
			Business: query.Get("bisnis"),
			From:     query.Get("dari"),
			To:       query.Get("sampai"),
		},
		inboxprogressclaim.Pagination{
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
func (h *Handler) readCaller(r *http.Request) (inboxprogressclaim.Caller, bool) {
	if h.Caller == nil {
		return inboxprogressclaim.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxprogressclaim.Caller{}, false
	}
	return inboxprogressclaim.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
