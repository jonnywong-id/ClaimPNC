package inboxcompliancehttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-compliance/tab.
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

// List menangani GET /api/inbox-compliance.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
		r.Context(),
		active.Alias,
		inboxcompliance.QueryInput{Tab: query.Get("tab")},
		inboxcompliance.Pagination{
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

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan inboxcompliance.Pagination.Normalize
// membetulkannya menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman
// pertama adalah jawaban yang selalu masuk akal.
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }

// SendPostAudit menangani POST /api/inbox-compliance/post-audit.
//
// POST, bukan PUT: ia MEMBUAT baris baru, dan pemanggilan kedua dengan badan yang sama
// membuat baris kedua — bukan menimpa yang pertama. Menandainya PUT akan menjanjikan
// idempotensi yang tidak ada (lihat catatan di usecase.SendToPostAudit).
func (h *Handler) SendPostAudit(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body SendPostAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// 400, bukan 422: badan yang tidak dapat diurai berarti klien salah membentuk
		// permintaan, bukan pengguna salah mengisi (`10-API-STRATEGY.md` §5).
		h.WriteError(w, r, errMalformedBody)
		return
	}

	sent, err := h.Service.SendToPostAudit(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		inboxcompliance.PostAuditInput{
			Reference: strings.TrimSpace(body.Reference),
			Remarks:   body.Remarks,
		},
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	// 201 beserta barisnya, bukan 204: layar membutuhkan nomor yang terbit untuk
	// menampilkannya pada pesan berhasil, dan mengambilnya lewat permintaan kedua berarti
	// menebak baris mana yang baru saja dibuat.
	h.WriteJSON(w, r, http.StatusCreated, toSendPostAuditResponse(sent, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Hanya jalur TULIS yang memerlukannya. Kedua jalur baca tidak — antreannya workbasket,
// yang isinya sama bagi setiap petugas.
func (h *Handler) readCaller(r *http.Request) (Caller, bool) {
	if h.Caller == nil {
		return Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || caller.Login == "" {
		return Caller{}, false
	}
	return caller, true
}
