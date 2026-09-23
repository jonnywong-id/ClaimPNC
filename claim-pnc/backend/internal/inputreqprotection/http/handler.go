package inputreqprotectionhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inputreqprotection"
	"claim-pnc/internal/inputreqprotection/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
//
// Dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor sebagai tipe konkret,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, q usecase.ListQuery) (inputreqprotection.Page, error)
	Get(ctx context.Context, portalAlias, number string) (inputreqprotection.Protection, error)
	Create(ctx context.Context, cmd usecase.SaveCommand) (inputreqprotection.Protection, error)
	Update(ctx context.Context, cmd usecase.SaveCommand) (inputreqprotection.Protection, error)
}

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
//
// Hanya satu field: modul ini tidak perlu tahu apa pun tentang bentuk sesi, dan modul auth
// tidak perlu tahu modul ini ada. Jembatannya dipasang di cmd/claimpnc, satu-satunya berkas
// yang memang tahu keduanya.
type Caller struct {
	Login string
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Input Req Protection.
type Handler struct {
	service     Service
	getCaller   GetCaller
	logger      *slog.Logger
	writeJSON   JSONWriter
	writeErrorF ErrorWriter
	location    *time.Location
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   Service
	GetCaller GetCaller
	Logger    *slog.Logger

	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	//
	// Ia parameter, bukan konstanta, supaya uji dapat menetapkannya dan tidak bergantung
	// pada basis data zona waktu mesin yang menjalankan.
	Location *time.Location
}

// NewHandler membentuk handler modul Input Req Protection.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Handler{
		service:     o.Service,
		getCaller:   o.GetCaller,
		logger:      o.Logger,
		writeJSON:   o.WriteJSON,
		writeErrorF: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		location:    location,
	}
}

func jakarta() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// List melayani daftar permintaan proteksi yang belum diakseptasi.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, ok := h.readLimit(w, r)
	if !ok {
		return
	}
	offset, ok := h.readOffset(w, r)
	if !ok {
		return
	}

	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	page, err := h.service.List(r.Context(), usecase.ListQuery{
		PortalAlias: alias,
		Search:      strings.TrimSpace(r.URL.Query().Get("cari")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	rows := make([]protectionDTO, 0, len(page.Protections))
	for _, p := range page.Protections {
		rows = append(rows, toProtectionDTO(p, h.location))
	}

	h.writeJSON(w, r, http.StatusOK, listResponse{Proteksi: rows, Total: page.Total})
}

// Get melayani pembacaan satu permintaan proteksi beserta isian formnya.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if number == "" {
		writeBadRequest(h.writeJSON, w, r, "Nomor proteksi wajib disebutkan.")
		return
	}

	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	p, err := h.service.Get(r.Context(), alias, number)
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailDTO(p, h.location))
}

// Create melayani pembuatan permintaan proteksi baru.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := h.readSaveRequest(w, r)
	if !ok {
		return
	}
	caller, ok := h.requireCaller(w, r)
	if !ok {
		return
	}
	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	saved, err := h.service.Create(r.Context(), usecase.SaveCommand{
		PortalAlias: alias,
		Draft:       toDraft(req, h.location),
		By:          caller.Login,
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	// 201, bukan 200: sumber daya baru terbentuk, dan nomornya baru diketahui klien dari
	// respons ini. Layar memakainya untuk berpindah ke halaman detail.
	h.writeJSON(w, r, http.StatusCreated, toDetailDTO(saved, h.location))
}

// Update melayani penyuntingan permintaan proteksi yang belum tertaut klaim.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if number == "" {
		writeBadRequest(h.writeJSON, w, r, "Nomor proteksi wajib disebutkan.")
		return
	}

	req, ok := h.readSaveRequest(w, r)
	if !ok {
		return
	}
	caller, ok := h.requireCaller(w, r)
	if !ok {
		return
	}

	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}

	saved, err := h.service.Update(r.Context(), usecase.SaveCommand{
		PortalAlias: alias,
		Number:      number,
		Draft:       toDraft(req, h.location),
		By:          caller.Login,
	})
	if err != nil {
		h.writeErrorF(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDetailDTO(saved, h.location))
}

// ── Pembacaan permintaan ─────────────────────────────────────────────────────────

// readSaveRequest membaca badan permintaan simpan.
//
// Field yang tidak dikenal DITOLAK (`DisallowUnknownFields`). Itu bukan kerewelan: ia yang
// menangkap salah ketik nama field di frontend, yang jika dibiarkan akan tersimpan sebagai
// nilai kosong tanpa satu pun gejala.
func (h *Handler) readSaveRequest(w http.ResponseWriter, r *http.Request) (saveRequest, bool) {
	var req saveRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeBadRequest(h.writeJSON, w, r, "Badan permintaan tidak dapat dibaca.")
		return saveRequest{}, false
	}
	return req, true
}

// requireCaller mengambil identitas pemanggil dari sesi.
//
// Kegagalan di sini adalah cacat pemrograman, bukan kesalahan pengguna: rute sudah
// dilindungi middleware sesi. Ia dijawab sebagai galat internal, dan dicatat — menjawabnya
// sebagai 401 akan membuat cacat itu tampak seperti masalah pengguna dan tidak pernah
// diperbaiki.
func (h *Handler) requireCaller(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	if h.getCaller == nil {
		h.writeErrorF(w, r, errors.New("inputreqprotection/http: pembaca identitas tidak dipasang"))
		return Caller{}, false
	}

	caller, ok := h.getCaller(r.Context())
	if !ok || strings.TrimSpace(caller.Login) == "" {
		h.writeErrorF(w, r, errors.New("inputreqprotection/http: identitas pemanggil tidak tersedia di konteks"))
		return Caller{}, false
	}
	return caller, true
}

// requirePortal mengambil alias portal yang sedang dibuka.
//
// Portal aktif sudah diperiksa middleware; ketiadaannya di sini berarti rute dipasang DI
// LUAR middleware itu — cacat perakitan, bukan kesalahan pengguna.
//
// Ia DITOLAK, tidak pernah dilayani portal utama sebagai cadangan. Jatuh ke koneksi default
// berarti menampilkan proteksi satu badan hukum di layar badan hukum lain tanpa satu pun
// galat (`R-20`, `TKT-F6-002`).
func (h *Handler) requirePortal(w http.ResponseWriter, r *http.Request) (string, bool) {
	active, found := portalhttp.ActivePortalFrom(r.Context())
	if !found || strings.TrimSpace(active.Alias) == "" {
		h.writeErrorF(w, r, errors.New("inputreqprotection/http: portal aktif tidak dikenali"))
		return "", false
	}
	return active.Alias, true
}

// readLimit membaca batas jumlah baris.
//
// Permintaan di atas MaxLimit DITOLAK, bukan dipangkas diam-diam — `10-API-STRATEGY.md` §4.
// Memangkasnya akan membuat klien mengira menerima seluruh data padahal tidak.
func (h *Handler) readLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("batas"))
	if raw == "" {
		return inputreqprotection.DefaultLimit, true
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		writeBadRequest(h.writeJSON, w, r, "Parameter batas harus berupa angka lebih besar dari nol.")
		return 0, false
	}
	if limit > inputreqprotection.MaxLimit {
		writeBadRequest(h.writeJSON, w, r,
			"Parameter batas melebihi "+strconv.Itoa(inputreqprotection.MaxLimit)+".")
		return 0, false
	}
	return limit, true
}

func (h *Handler) readOffset(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("lewati"))
	if raw == "" {
		return 0, true
	}

	offset, err := strconv.Atoi(raw)
	if err != nil || offset < 0 {
		writeBadRequest(h.writeJSON, w, r, "Parameter lewati harus berupa angka nol atau lebih.")
		return 0, false
	}
	return offset, true
}
