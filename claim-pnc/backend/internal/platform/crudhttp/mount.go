package crudhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Routes adalah keempat penangan layar master: daftar, tambah, buka satu baris, dan simpan.
type Routes interface {
	List(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
}

// Mount mendaftarkan keempat rute master di bawah path, dengan kunci baris bernama idParam,
// seluruhnya di balik pemeriksaan portal aktif:
//
//	GET  path/            daftar
//	POST path/            tambah
//	GET  path/{idParam}   buka satu baris
//	PUT  path/{idParam}   simpan — PUT, bukan PATCH: seluruh isi dikirim, idempoten
func Mount(r chi.Router, deps portalhttp.ActivePortalDeps, path, idParam string, h Routes) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(deps))
		perPortal.Route(path, func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{"+idParam+"}", h.Get)
			master.Put("/{"+idParam+"}", h.Update)
		})
	})
}
