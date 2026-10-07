package inboxsurveyhttp

import (
	"log/slog"
	"time"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
//
// Itulah yang dicocokkan ke `POOLDATA.MST_LOGIN_SURVEYOR.LOGIN` — pemetaan yang
// `RDB List/GetLoginLeaderSurveyor-SQL.xml` pakai persis begitu. Memakai NIK di sini akan
// membuat setiap pengguna terbaca sebagai bukan surveyor.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul My Work.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	clock      inboxsurvey.Clock
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

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	Location *time.Location

	// Clock adalah seam ke waktu. Kosong berarti jam mesin.
	//
	// Ia dibutuhkan kolom Aging, yang DIHITUNG dari tanggal janji survei dicatat terhadap
	// tanggal WIB hari ini — bukan dibaca dari kolom.
	Clock inboxsurvey.Clock

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul My Work.
func NewHandler(o Options) *Handler {
	location := o.Location
	if location == nil {
		location = jakarta()
	}

	clock := o.Clock
	if clock == nil {
		clock = systemClock{}
	}

	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		clock:      clock,
		location:   location,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}
