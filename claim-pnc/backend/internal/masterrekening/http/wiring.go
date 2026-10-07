package masterrekeninghttp

import (
	"context"
	"log/slog"
	"net/http"
)

// Caller adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah tiga field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Options.Caller, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
type Caller struct {
	Identity string
	Name     string
	Email    string
}

// Handler melayani permintaan master rekening.
type Handler struct {
	serviceSelector ServiceSelector
	caller          func(context.Context) (Caller, bool)
	logger          *slog.Logger

	writeError func(w http.ResponseWriter, r *http.Request, err error)
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// ServiceSelector memilih layanan milik satu portal entitas. Wajib.
	ServiceSelector ServiceSelector

	// Caller membaca identitas pemanggil dari context. Diisi saat perakitan
	// dengan pembaca konteks milik modul auth.
	Caller func(context.Context) (Caller, bool)

	Logger     *slog.Logger
	WriteError func(w http.ResponseWriter, r *http.Request, err error)
}

// NewHandler membentuk handler modul master rekening.
func NewHandler(o Options) *Handler {
	write := o.WriteError
	if write == nil {
		write = WriteError(o.Logger)
	}
	return &Handler{
		serviceSelector: o.ServiceSelector,
		caller:          o.Caller,
		logger:          o.Logger,
		writeError:      write,
	}
}
