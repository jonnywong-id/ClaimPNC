package monitoringslinkojkhttp

import (
	"log/slog"
	"time"

	"claim-pnc/internal/monitoringslinkojk/usecase"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpkit"
)

// JSONWriter menuliskan badan respons. Modul ini tidak membawa penulisnya sendiri supaya
// seluruh modul menulis respons dengan cara yang sama, termasuk header Cache-Control-nya.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
//
// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
// di sistem lama, bukan NIK.
type Caller = httpkit.Caller

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan modul Monitoring SLINK OJK.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter

	// clock membaca waktu sekarang, dipakai mengisi `bulanlapor`.
	//
	// Fungsi, bukan pemanggilan `time.Now()` langsung: bulan lapor menentukan periode
	// laporan ke OJK, dan ia harus dapat diuji tanpa bergantung pada jam mesin penjalan
	// (`F-5`).
	clock func() time.Time
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter

	// Now opsional; kosong berarti jam sistem. Lihat Handler.clock.
	Now func() time.Time
}

// NewHandler membentuk handler modul Monitoring SLINK OJK.
func NewHandler(o Options) *Handler {
	clock := o.Now
	if clock == nil {
		clock = time.Now
	}
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
		clock:      clock,
	}
}
