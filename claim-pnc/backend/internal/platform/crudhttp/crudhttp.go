// Package crudhttp adalah mesin handler HTTP bersama untuk master data sederhana: daftar,
// ambil satu baris, tambah, dan ubah — tanpa alur persetujuan.
//
// Setiap modul tetap memegang isi kerjanya sendiri lewat closure di Spec: memanggil service,
// lalu menyusun badan respons. Yang dipegang mesin ini hanyalah kerangka di sekelilingnya,
// yang sebelumnya tersalin di setiap modul:
//
//	portal aktif → kunci di jalur (Get/Update) → badan permintaan (Create/Update) → kerja modul
//
// Galat dari kerja modul ditulis lewat pemetaan galat modul; badan respons ditulis dengan
// status 200, atau 201 untuk Create.
package crudhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Spec adalah bagian yang dimiliki modul. Req adalah badan permintaan tambah/ubah. Closure
// yang dibiarkan nil berarti modul melayani handler itu sendiri.
type Spec[Req any] struct {
	WriteResponse    apierror.JSONWriter
	WriteModuleError apierror.ErrorWriter
	// NotFound ditulis bila kunci di jalur kosong.
	NotFound error
	// Read mengurai badan permintaan; false bila responsnya sudah ditulis.
	Read func(w http.ResponseWriter, r *http.Request) (Req, bool)

	List   func(r *http.Request, portalAlias string) (any, error)
	Get    func(r *http.Request, portalAlias, id string) (any, error)
	Create func(r *http.Request, portalAlias string, request Req) (any, error)
	Update func(r *http.Request, portalAlias, id string, request Req) (any, error)
}

// Handler menjalankan keempat handler untuk satu modul.
type Handler[Req any] struct {
	spec Spec[Req]
}

// New merakit mesin untuk satu modul.
func New[Req any](spec Spec[Req]) *Handler[Req] { return &Handler[Req]{spec: spec} }

// List melayani daftar baris.
func (h *Handler[Req]) List(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	h.respond(w, r, http.StatusOK)(h.spec.List(r, active.Alias))
}

// Get melayani satu baris menurut kuncinya di jalur.
func (h *Handler[Req]) Get(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	h.respond(w, r, http.StatusOK)(h.spec.Get(r, active.Alias, id))
}

// Create menambah satu baris dan menjawab 201.
func (h *Handler[Req]) Create(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	request, ok := h.spec.Read(w, r)
	if !ok {
		return
	}
	h.respond(w, r, http.StatusCreated)(h.spec.Create(r, active.Alias, request))
}

// Update mengubah satu baris menurut kuncinya di jalur.
func (h *Handler[Req]) Update(w http.ResponseWriter, r *http.Request) {
	active, ok := h.portal(w, r)
	if !ok {
		return
	}
	id, ok := h.id(w, r)
	if !ok {
		return
	}
	request, ok := h.spec.Read(w, r)
	if !ok {
		return
	}
	h.respond(w, r, http.StatusOK)(h.spec.Update(r, active.Alias, id, request))
}

func (h *Handler[Req]) respond(w http.ResponseWriter, r *http.Request, status int) func(any, error) {
	return func(body any, err error) {
		if err != nil {
			h.spec.WriteModuleError(w, r, err)
			return
		}
		h.spec.WriteResponse(w, r, status, body)
	}
}

func (h *Handler[Req]) portal(w http.ResponseWriter, r *http.Request) (portal.Portal, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.spec.WriteModuleError(w, r, portal.ErrNotStated)
	}
	return active, exists
}

func (h *Handler[Req]) id(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.spec.WriteModuleError(w, r, h.spec.NotFound)
		return "", false
	}
	return id, true
}
