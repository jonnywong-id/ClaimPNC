// Package portalhttp adalah lapisan transport modul portal.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola yang sama dengan
// auth/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `portalhttp` supaya tidak menutupi `net/http`.
package portalhttp

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"
)

// PortalDTO adalah bentuk portal yang dikirim ke peramban.
//
// Terpisah dari portal.Portal supaya perubahan internal tidak bocor ke klien.
//
// Nama field JSON tetap berbahasa Indonesia: kontrak API sengaja tidak diusik saat
// penamaan kode dialihkan ke bahasa Inggris (keputusan Work Owner 2026-09-18).
type PortalDTO struct {
	ID    string `json:"id"`
	Name  string `json:"nama"`
	Alias string `json:"alias"`

	// Ready menyatakan koneksi basis data portal ini sudah dapat dipakai. Portal yang
	// belum siap tetap ditampilkan — tetapi ditandai, bukan disembunyikan.
	Ready bool `json:"siap"`
}

// ListResponse adalah jawaban GET /api/portal.
type ListResponse struct {
	Portals []PortalDTO `json:"portal"`

	// Primary adalah alias portal yang basis datanya melayani sesi dan lookup pra-login.
	// Dikirim supaya antarmuka dapat memilihnya sebagai portal awal tanpa menebak.
	Primary string `json:"utama"`
}

// Handler melayani permintaan daftar portal.
type Handler struct {
	repo          portal.Repo
	readyAliases  func() []string
	primaryAlias  string
	logger        *slog.Logger
	writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
	writeError    func(w http.ResponseWriter, r *http.Request, err error)
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Repo portal.Repo

	// ReadyAliases menyebut portal yang koneksinya hidup. Ia fungsi, bukan slice, supaya
	// perubahan ketersediaan terbaca pada saat permintaan datang.
	ReadyAliases func() []string

	PrimaryAlias  string
	Logger        *slog.Logger
	WriteResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
	WriteError    func(w http.ResponseWriter, r *http.Request, err error)
}

// NewHandler membentuk handler modul portal.
func NewHandler(o Options) *Handler {
	return &Handler{
		repo:          o.Repo,
		readyAliases:  o.ReadyAliases,
		primaryAlias:  o.PrimaryAlias,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}
}

// List menangani GET /api/portal.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.List(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	var ready []string
	if h.readyAliases != nil {
		ready = h.readyAliases()
	}

	marked := portal.MarkAvailable(list, ready)
	body := make([]PortalDTO, 0, len(marked))
	for _, p := range marked {
		body = append(body, PortalDTO{ID: p.ID, Name: p.Name, Alias: p.Alias, Ready: p.Ready})
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{Portals: body, Primary: h.primaryAlias})
}

// Mount mendaftarkan rute modul portal.
//
// Rutenya TERLINDUNGI: daftar portal baru dibutuhkan setelah pengguna masuk, karena
// pemilih portal berada di dalam aplikasi — bukan di layar masuk (keputusan Work Owner
// 2026-09-16, sejalan dengan ADR-0030 yang menetapkan berpindah portal tidak menuntut
// login ulang).
func Mount(r chi.Router, h *Handler) {
	r.Get("/portal", h.List)
}
