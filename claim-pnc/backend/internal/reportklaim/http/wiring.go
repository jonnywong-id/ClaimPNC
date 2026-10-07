package reportklaimhttp

import (
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/reportklaim/usecase"
)

// Handler melayani permintaan Report Klaim.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerLookup]
}

// Options adalah bahan pembentuk Handler.
type Options = httpkit.MasterOptions[*usecase.Service, CallerLookup]

// NewHandler membentuk handler modul Report Klaim.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("reportklaim/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}
