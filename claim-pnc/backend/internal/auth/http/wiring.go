package authhttp

import (
	"log/slog"
)

// Handler memuat handler masuk, keluar, identitas pemanggil, dan perpanjangan sesi.
type Handler struct {
	service    Service
	logger     *slog.Logger
	writeError ErrorWriter
}

// NewHandler membentuk handler modul auth.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger, writeError: WriteError(logger)}
}
