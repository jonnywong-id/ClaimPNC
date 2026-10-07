// Package masterhttp adalah mesin handler HTTP bersama untuk master data berpersetujuan:
// daftar per status, ambil satu baris, tambah, simpan, dan putuskan (setujui/tolak) beberapa
// baris sekaligus.
//
// Sebelumnya setiap modul master menyalin kelima handler ini apa adanya; yang berbeda hanya
// tipe barisnya, nama field JSON pada respons, dan pesan galatnya. Paket ini memegang
// alurnya satu kali, dan setiap modul menyerahkan bagian yang memang miliknya lewat Spec.
//
// Urutan pemeriksaan pada setiap handler dipertahankan persis seperti sebelumnya:
//
//	portal aktif → identitas pemanggil (tulis) → kunci di jalur → badan permintaan → service
//
// karena urutan itu menentukan galat MANA yang dilihat klien bila beberapa syarat sekaligus
// tidak terpenuhi.
package masterhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Service adalah empat operasi baca-tulis yang dipakai mesin ini.
type Service[T, In any, S ~string, A any] interface {
	List(ctx context.Context, portalAlias string, status S, keyword string) ([]T, error)
	Get(ctx context.Context, portalAlias, id string) (T, error)
	Create(ctx context.Context, portalAlias string, input In, by A, logger *slog.Logger) (T, error)
	Save(ctx context.Context, portalAlias, id string, input In, by A, logger *slog.Logger) (T, error)
}

// Decider memutuskan status persetujuan beberapa baris sekaligus.
type Decider[S ~string, A any] interface {
	Decide(ctx context.Context, portalAlias string, id []string, status S, by A, logger *slog.Logger) (int, error)
}

// Spec adalah bagian yang dimiliki modul. T baris, In masukan simpan, S status persetujuan,
// A pelaku, Req badan simpan, Dec badan keputusan.
type Spec[T, In any, S ~string, A, Req, Dec any] struct {
	// Module adalah awalan pesan galat internal, misalnya "masterbengkel/http".
	Module  string
	Service Service[T, In, S, A]
	// Caller membaca login pemanggil dari konteks; false bila tidak ada.
	Caller func(ctx context.Context) (string, bool)
	Actor  func(login string) A
	Logger *slog.Logger

	WriteResponse apierror.JSONWriter
	// WriteError adalah penulis galat bersama dari cmd; dipakai untuk cacat perakitan.
	WriteError apierror.ErrorWriter
	// WriteModuleError memetakan galat lewat pemetaan modul.
	WriteModuleError apierror.ErrorWriter

	NotFound      error
	DefaultStatus S
	MaxBody       int64
	// Malformed adalah badan respons 400 untuk permintaan yang tidak dapat dibaca.
	Malformed any

	ToInput    func(Req) In
	ListBody   func(r *http.Request, rows []T, status S, portalAlias string) any
	SingleBody func(r *http.Request, row T, portalAlias string) any

	// Bagian keputusan; dibiarkan kosong oleh modul yang tidak memutuskan lewat mesin ini.
	Decider         Decider[S, A]
	DecisionIDs     func(Dec) []string
	DecisionStatus  func(Dec) string
	MaxDecisionRows int
	TooManyRows     error
	DecisionBody    func(changed int, status S, portalAlias string) any
}

// Handler menjalankan kelima handler untuk satu modul.
type Handler[T, In any, S ~string, A, Req, Dec any] struct {
	spec Spec[T, In, S, A, Req, Dec]
}

// New merakit mesin untuk satu modul. Kelengkapan Spec diperiksa modulnya sendiri di
// NewHandler, supaya pesan penolakan perakitannya tetap milik modul.
func New[T, In any, S ~string, A, Req, Dec any](spec Spec[T, In, S, A, Req, Dec]) *Handler[T, In, S, A, Req, Dec] {
	return &Handler[T, In, S, A, Req, Dec]{spec: spec}
}

// List melayani daftar baris per status; status kosong berarti status bawaan.
func (h *Handler[T, In, S, A, Req, Dec]) List(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	status := h.spec.DefaultStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = S(raw)
	}
	list, err := h.spec.Service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.spec.WriteModuleError(w, r, err)
		return
	}
	h.spec.WriteResponse(w, r, http.StatusOK, h.spec.ListBody(r, list, status, active.Alias))
}

// Get melayani satu baris menurut kuncinya di jalur.
func (h *Handler[T, In, S, A, Req, Dec]) Get(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	found, err := h.spec.Service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.spec.WriteModuleError(w, r, err)
		return
	}
	h.spec.WriteResponse(w, r, http.StatusOK, h.spec.SingleBody(r, found, active.Alias))
}

// Create menambah satu baris dan menjawab 201.
func (h *Handler[T, In, S, A, Req, Dec]) Create(w http.ResponseWriter, r *http.Request) {
	active, by, ok := h.writer(w, r)
	if !ok {
		return
	}
	var request Req
	if !h.ReadRequest(w, r, &request) {
		return
	}
	saved, err := h.spec.Service.Create(r.Context(), active.Alias, h.spec.ToInput(request), by, h.spec.Logger)
	if err != nil {
		h.spec.WriteModuleError(w, r, err)
		return
	}
	h.spec.WriteResponse(w, r, http.StatusCreated, h.spec.SingleBody(r, saved, active.Alias))
}

// Save menyimpan perubahan satu baris.
func (h *Handler[T, In, S, A, Req, Dec]) Save(w http.ResponseWriter, r *http.Request) {
	active, by, ok := h.writer(w, r)
	if !ok {
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	var request Req
	if !h.ReadRequest(w, r, &request) {
		return
	}
	saved, err := h.spec.Service.Save(r.Context(), active.Alias, id, h.spec.ToInput(request), by, h.spec.Logger)
	if err != nil {
		h.spec.WriteModuleError(w, r, err)
		return
	}
	h.spec.WriteResponse(w, r, http.StatusOK, h.spec.SingleBody(r, saved, active.Alias))
}

// Decide memutuskan status beberapa baris sekaligus.
func (h *Handler[T, In, S, A, Req, Dec]) Decide(w http.ResponseWriter, r *http.Request) {
	active, by, ok := h.writer(w, r)
	if !ok {
		return
	}
	var request Dec
	if !h.ReadRequest(w, r, &request) {
		return
	}
	ids := h.spec.DecisionIDs(request)
	if len(ids) > h.spec.MaxDecisionRows {
		h.spec.WriteModuleError(w, r, h.spec.TooManyRows)
		return
	}
	status := S(h.spec.DecisionStatus(request))
	changed, err := h.spec.Decider.Decide(r.Context(), active.Alias, ids, status, by, h.spec.Logger)
	if err != nil {
		h.spec.WriteModuleError(w, r, err)
		return
	}
	h.spec.WriteResponse(w, r, http.StatusOK, h.spec.DecisionBody(changed, status, active.Alias))
}

// ReadRequest mengurai badan JSON ke target: ukurannya dibatasi, field tak dikenal ditolak,
// dan data sesudah objek pertama juga ditolak. Kegagalan dijawab 400 dengan badan Malformed.
func (h *Handler[T, In, S, A, Req, Dec]) ReadRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	return httpjson.Decode(w, r, h.spec.MaxBody, target, h.spec.WriteResponse, h.spec.Malformed)
}

func (h *Handler[T, In, S, A, Req, Dec]) portal(w http.ResponseWriter, r *http.Request) (portal.Portal, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.spec.WriteModuleError(w, r, portal.ErrNotStated)
	}
	return active, exists
}

// writer memeriksa portal lalu identitas pemanggil — syarat setiap handler yang menulis.
func (h *Handler[T, In, S, A, Req, Dec]) writer(w http.ResponseWriter, r *http.Request) (portal.Portal, A, bool) {
	var none A
	active, ok := h.portal(w, r)
	if !ok {
		return active, none, false
	}
	login, known := h.spec.Caller(r.Context())
	if !known {
		h.spec.WriteError(w, r, errors.New(h.spec.Module+": identitas pemanggil tidak ada di konteks"))
		return active, none, false
	}
	return active, h.spec.Actor(login), true
}

func (h *Handler[T, In, S, A, Req, Dec]) id(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.spec.WriteModuleError(w, r, h.spec.NotFound)
		return "", false
	}
	return id, true
}
