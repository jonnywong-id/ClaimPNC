package dokumenpenunjanghttp

import (
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pengguna yang sedang masuk.
type Caller struct {
	Login string
}

// Handler melayani permintaan dokumen penunjang.
type Handler struct {
	httpkit.Timed[Service, GetCaller]
}

// Options adalah bahan pembentuk Handler.
//
// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
type Options = httpkit.TimedOptions[Service, GetCaller]

// NewHandler membentuk handler modul dokumen penunjang.
func NewHandler(o Options) *Handler { return &Handler{Timed: httpkit.NewTimed(o, WriteError)} }
