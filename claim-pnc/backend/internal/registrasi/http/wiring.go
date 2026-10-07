package registrasihttp

import (
	"log/slog"
	"net/http"

	"claim-pnc/internal/registrasi/usecase"
)

// Handler melayani permintaan modul registrasi.
type Handler struct {
	service *usecase.Service
	logger  *slog.Logger

	// caller menerjemahkan permintaan HTTP menjadi identitas pemanggil.
	//
	// Ia disuntikkan, bukan dibaca langsung dari modul auth, supaya lapisan transport
	// modul ini tidak bergantung pada lapisan transport modul lain — dan supaya
	// handler dapat diuji tanpa membentuk seluruh layanan autentikasi.
	caller func(r *http.Request) (usecase.Caller, bool)

	writeResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service       *usecase.Service
	Logger        *slog.Logger
	Caller        func(r *http.Request) (usecase.Caller, bool)
	WriteResponse func(w http.ResponseWriter, r *http.Request, status int, body any)
}

// NewHandler membentuk handler modul registrasi.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		caller:        o.Caller,
		writeResponse: o.WriteResponse,
	}
}
