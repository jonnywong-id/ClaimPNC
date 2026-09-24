package inboxcompliancehttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Compliance.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
//
// # GetCaller hanya dipakai jalur TULIS
//
// Kedua jalur baca tidak membutuhkannya: antreannya WORKBASKET, yakni antrean bersama yang
// isinya sama bagi setiap petugas Compliance.
//
// Yang membutuhkannya adalah pengiriman ke Post Audit — bukan untuk menentukan apa yang
// boleh dikirim, melainkan untuk MENCATAT siapa yang mengirim. Tabelnya tidak punya kolom
// pengirim, sehingga log adalah satu-satunya tempat identitas itu tersimpan. Lihat catatan
// di usecase.Caller.
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

// NewHandler membentuk handler modul Inbox Compliance.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-compliance/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab dan kolomnya adalah bentuk layar, bukan data
// entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-compliance.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		inboxcompliance.QueryInput{Tab: query.Get("tab")},
		inboxcompliance.Pagination{
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

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan inboxcompliance.Pagination.Normalize
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

// SendPostAudit menangani POST /api/inbox-compliance/post-audit.
//
// POST, bukan PUT: ia MEMBUAT baris baru, dan pemanggilan kedua dengan badan yang sama
// membuat baris kedua — bukan menimpa yang pertama. Menandainya PUT akan menjanjikan
// idempotensi yang tidak ada (lihat catatan di usecase.SendToPostAudit).
func (h *Handler) SendPostAudit(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxcompliance.ErrCallerUnknown)
		return
	}

	var body SendPostAuditRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// 400, bukan 422: badan yang tidak dapat diurai berarti klien salah membentuk
		// permintaan, bukan pengguna salah mengisi (`10-API-STRATEGY.md` §5).
		h.writeError(w, r, errMalformedBody)
		return
	}

	sent, err := h.service.SendToPostAudit(
		r.Context(),
		active.Alias,
		usecase.Caller{Login: caller.Login},
		inboxcompliance.PostAuditInput{
			Reference: strings.TrimSpace(body.Reference),
			Remarks:   body.Remarks,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// 201 beserta barisnya, bukan 204: layar membutuhkan nomor yang terbit untuk
	// menampilkannya pada pesan berhasil, dan mengambilnya lewat permintaan kedua berarti
	// menebak baris mana yang baru saja dibuat.
	h.writeJSON(w, r, http.StatusCreated, toSendPostAuditResponse(sent, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Hanya jalur TULIS yang memerlukannya. Kedua jalur baca tidak — antreannya workbasket,
// yang isinya sama bagi setiap petugas.
func (h *Handler) readCaller(r *http.Request) (Caller, bool) {
	if h.caller == nil {
		return Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return Caller{}, false
	}
	return caller, true
}
