package inboxrclhttp

import (
	"log/slog"
	"time"

	"claim-pnc/internal/inboxrcl/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini — tipe
// milik modul ini, bukan tipe modul auth; jembatannya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK — padanan
// `OperatorID.pyUserIdentifier`, kunci pencarian identitas lamanya.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul Inbox RCL.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	location   *time.Location
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
//
// Tidak ada Clock: berbeda dari Inbox Analyst Doctor, layar ini tidak punya kolom durasi —
// kelima judul kolom di harness tidak memuatnya.
type Options struct {
	Service   *usecase.Service
	GetCaller CallerReader

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	Logger              *slog.Logger
	WriteJSON           JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox RCL.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}
