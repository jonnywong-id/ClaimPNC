package inboxanalystdoctorhttp

import (
	"log/slog"
	"time"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
//
// Itulah yang dicocokkan ke `PC_ASSIGN_WORKLIST.PXASSIGNEDOPERATORID`. Memakai NIK di
// sini akan membuat antrean tampak kosong bagi setiap pengguna.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul Inbox Analyst Doctor.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	clock      inboxanalystdoctor.Clock
	location   *time.Location
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan diimpor
	// dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa menariknya
	// serta.
	GetCaller CallerReader

	// Clock adalah seam waktu (`F-5`). Wajib: kolom "Lama Waktu Klaim" dihitung darinya.
	Clock inboxanalystdoctor.Clock

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Analyst Doctor.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		clock:      o.Clock,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}
