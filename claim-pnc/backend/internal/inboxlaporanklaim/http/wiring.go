package inboxlaporanklaimhttp

import (
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Inbox Laporan Klaim.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerLookup]
}

// Options adalah bahan pembentuk Handler.
type Options = httpkit.MasterOptions[*usecase.Service, CallerLookup]

// NewHandler membentuk handler modul Inbox Laporan Klaim.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("inboxlaporanklaim/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}
