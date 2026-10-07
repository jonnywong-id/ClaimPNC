package inboxacceptopenprotectionhttp

import (
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pengguna yang sedang masuk, sejauh yang dibutuhkan modul ini.
type Caller struct {
	Login string
}

// Handler melayani permintaan Inbox Accept Open Protection.
type Handler struct {
	httpkit.Timed[Service, GetCaller]
}

// Options adalah bahan pembentuk Handler.
//
// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
type Options = httpkit.TimedOptions[Service, GetCaller]

// NewHandler membentuk handler modul Inbox Accept Open Protection.
func NewHandler(o Options) *Handler { return &Handler{Timed: httpkit.NewTimed(o, WriteError)} }
