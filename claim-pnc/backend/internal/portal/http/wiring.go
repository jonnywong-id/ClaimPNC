package portalhttp

import (
	"log/slog"
	"net/http"

	"claim-pnc/internal/portal"
)

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
