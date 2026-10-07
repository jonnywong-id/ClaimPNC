package inboxacceptopenprotectionhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/usecase"
	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah bagian usecase yang dipakai handler ini.
type Service interface {
	List(ctx context.Context, q usecase.ListQuery) (inboxacceptopenprotection.Page, error)
	Queues(ctx context.Context, login string) ([]inboxacceptopenprotection.Queue, error)
	Get(ctx context.Context, portalAlias, number, login string) (inboxacceptopenprotection.Protection, error)
	Decide(ctx context.Context, cmd usecase.DecideCommand) (inboxacceptopenprotection.Protection, error)
}

// GetCaller membaca identitas pengguna dari konteks permintaan.
type GetCaller func(ctx context.Context) (Caller, bool)

// List melayani antrean akseptasi.
//
// Antrean dipilih lewat parameter `antrean`; kosong berarti NON PREMI, yang di layar lama
// adalah grid yang tampil bagi peran selain penagihan premi.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	queue, ok := h.readQueue(w, r)
	if !ok {
		return
	}
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
	caller, ok := h.requireCaller(w, r)
	if !ok {
		return
	}

	page, err := h.Service.List(r.Context(), usecase.ListQuery{
		PortalAlias: alias,
		Queue:       queue,
		Login:       caller.Login,
		Search:      strings.TrimSpace(r.URL.Query().Get("cari")),
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	rows := make([]protectionDTO, 0, len(page.Protections))
	for _, p := range page.Protections {
		rows = append(rows, toProtectionDTO(p, h.Location))
	}

	h.WriteJSON(w, r, http.StatusOK, listResponse{
		Proteksi: rows,
		Total:    page.Total,
		Antrean:  string(queue),
	})
}

// Get melayani pembacaan satu proteksi untuk form akseptasi.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if number == "" {
		writeBadRequest(h.WriteJSON, w, r, "Nomor proteksi wajib disebutkan.")
		return
	}
	alias, ok := h.requirePortal(w, r)
	if !ok {
		return
	}
	caller, ok := h.requireCaller(w, r)
	if !ok {
		return
	}

	p, err := h.Service.Get(r.Context(), alias, number, caller.Login)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDetailDTO(p, h.Location))
}

// Queues melayani daftar antrean yang boleh dibuka pemanggil.
//
// Layar memanggilnya untuk menggambar tab. Tanpa rute ini, layar hanya punya dua pilihan
// yang sama-sama buruk: menggambar kedua tab lalu membiarkan salah satunya dijawab 403, atau
// menebak kewenangan dari sesuatu yang ada di peramban.
//
// Rute ini TIDAK menuntut portal aktif — kewenangan seseorang sama di keempat portal
// (`D-78`), dan tabelnya pun tinggal di basis data utama.
func (h *Handler) Queues(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCaller(w, r)
	if !ok {
		return
	}

	queues, err := h.Service.Queues(r.Context(), caller.Login)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	daftar := make([]string, 0, len(queues))
	for _, q := range queues {
		daftar = append(daftar, string(q))
	}

	h.WriteJSON(w, r, http.StatusOK, queuesResponse{Antrean: daftar})
}

// Decide melayani keputusan akseptasi: setuju atau tolak.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	number := strings.TrimSpace(chi.URLParam(r, "nomor"))
	if number == "" {
		writeBadRequest(h.WriteJSON, w, r, "Nomor proteksi wajib disebutkan.")
		return
	}

	var req decideRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeBadRequest(h.WriteJSON, w, r, "Badan permintaan tidak dapat dibaca.")
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

	saved, err := h.Service.Decide(r.Context(), usecase.DecideCommand{
		PortalAlias: alias,
		Number:      number,
		Decision:    inboxacceptopenprotection.Decision(strings.TrimSpace(req.Keputusan)),
		By:          caller.Login,
	})
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDetailDTO(saved, h.Location))
}

// ── Pembacaan permintaan ─────────────────────────────────────────────────────────

// readQueue membaca antrean yang diminta.
//
// Nilai yang tidak dikenali DITOLAK, bukan dijatuhkan ke antrean bawaan. Menjatuhkannya
// diam-diam akan membuat salah ketik pada frontend menampilkan antrean yang salah tanpa satu
// pun gejala — dan pada layar ini, antrean yang salah berarti petugas melihat pekerjaan
// yang bukan miliknya.
func (h *Handler) readQueue(w http.ResponseWriter, r *http.Request) (inboxacceptopenprotection.Queue, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("antrean"))
	if raw == "" {
		return inboxacceptopenprotection.QueueNonPremium, true
	}

	queue := inboxacceptopenprotection.Queue(raw)
	if !queue.Valid() {
		writeBadRequest(h.WriteJSON, w, r, `Parameter antrean harus "premi" atau "non-premi".`)
		return "", false
	}
	return queue, true
}

// requirePortal mengambil alias portal yang sedang dibuka.
//
// Ketiadaannya DITOLAK, tidak pernah dilayani portal utama sebagai cadangan — jatuh ke
// koneksi default berarti seseorang mengakseptasi proteksi milik badan hukum lain (`R-20`).
func (h *Handler) requirePortal(w http.ResponseWriter, r *http.Request) (string, bool) {
	active, found := portalhttp.ActivePortalFrom(r.Context())
	if !found || strings.TrimSpace(active.Alias) == "" {
		h.WriteError(w, r, errors.New("inboxacceptopenprotection/http: portal aktif tidak dikenali"))
		return "", false
	}
	return active.Alias, true
}

// requireCaller mengambil identitas pemanggil dari sesi.
func (h *Handler) requireCaller(w http.ResponseWriter, r *http.Request) (Caller, bool) {
	if h.Caller == nil {
		h.WriteError(w, r, errors.New("inboxacceptopenprotection/http: pembaca identitas tidak dipasang"))
		return Caller{}, false
	}

	caller, ok := h.Caller(r.Context())
	if !ok || strings.TrimSpace(caller.Login) == "" {
		h.WriteError(w, r, errors.New("inboxacceptopenprotection/http: identitas pemanggil tidak tersedia di konteks"))
		return Caller{}, false
	}
	return caller, true
}

// readLimit membaca batas jumlah baris. Permintaan di atas MaxLimit DITOLAK, bukan dipangkas
// diam-diam (`10-API-STRATEGY.md` §4).
func (h *Handler) readLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("batas"))
	if raw == "" {
		return inboxacceptopenprotection.DefaultLimit, true
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		writeBadRequest(h.WriteJSON, w, r, "Parameter batas harus berupa angka lebih besar dari nol.")
		return 0, false
	}
	if limit > inboxacceptopenprotection.MaxLimit {
		writeBadRequest(h.WriteJSON, w, r,
			"Parameter batas melebihi "+strconv.Itoa(inboxacceptopenprotection.MaxLimit)+".")
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
		writeBadRequest(h.WriteJSON, w, r, "Parameter lewati harus berupa angka nol atau lebih.")
		return 0, false
	}
	return offset, true
}
