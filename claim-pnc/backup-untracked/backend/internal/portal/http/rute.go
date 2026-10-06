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
type PortalDTO struct {
	ID    string `json:"id"`
	Nama  string `json:"nama"`
	Alias string `json:"alias"`

	// Siap menyatakan koneksi basis data portal ini sudah dapat dipakai. Portal yang
	// belum siap tetap ditampilkan — tetapi ditandai, bukan disembunyikan.
	Siap bool `json:"siap"`
}

// ResponsDaftar adalah jawaban GET /api/portal.
type ResponsDaftar struct {
	Portal []PortalDTO `json:"portal"`

	// Utama adalah alias portal yang basis datanya melayani sesi dan lookup pra-login.
	// Dikirim supaya antarmuka dapat memilihnya sebagai portal awal tanpa menebak.
	Utama string `json:"utama"`
}

// Handler melayani permintaan daftar portal.
type Handler struct {
	repo        portal.Repo
	aliasSiap   func() []string
	aliasUtama  string
	logger      *slog.Logger
	tulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any)
	tulisGalat  func(w http.ResponseWriter, r *http.Request, err error)
}

// Opsi adalah bahan pembentuk Handler.
type Opsi struct {
	Repo portal.Repo

	// AliasSiap menyebut portal yang koneksinya hidup. Ia fungsi, bukan slice, supaya
	// perubahan ketersediaan terbaca pada saat permintaan datang.
	AliasSiap func() []string

	AliasUtama  string
	Logger      *slog.Logger
	TulisRespon func(w http.ResponseWriter, r *http.Request, status int, badan any)
	TulisGalat  func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerBaru membentuk handler modul portal.
func HandlerBaru(o Opsi) *Handler {
	return &Handler{
		repo:        o.Repo,
		aliasSiap:   o.AliasSiap,
		aliasUtama:  o.AliasUtama,
		logger:      o.Logger,
		tulisRespon: o.TulisRespon,
		tulisGalat:  o.TulisGalat,
	}
}

// Daftar menangani GET /api/portal.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.repo.Daftar(r.Context())
	if err != nil {
		h.tulisGalat(w, r, err)
		return
	}

	var siap []string
	if h.aliasSiap != nil {
		siap = h.aliasSiap()
	}

	ditandai := portal.TandaiTersedia(daftar, siap)
	isi := make([]PortalDTO, 0, len(ditandai))
	for _, p := range ditandai {
		isi = append(isi, PortalDTO{ID: p.ID, Nama: p.Nama, Alias: p.Alias, Siap: p.Siap})
	}

	h.tulisRespon(w, r, http.StatusOK, ResponsDaftar{Portal: isi, Utama: h.aliasUtama})
}

// Pasang mendaftarkan rute modul portal.
//
// Rutenya TERLINDUNGI: daftar portal baru dibutuhkan setelah pengguna masuk, karena
// pemilih portal berada di dalam aplikasi — bukan di layar masuk (keputusan Work Owner
// 2026-09-16, sejalan dengan ADR-0030 yang menetapkan berpindah portal tidak menuntut
// login ulang).
func Pasang(r chi.Router, h *Handler) {
	r.Get("/portal", h.Daftar)
}
