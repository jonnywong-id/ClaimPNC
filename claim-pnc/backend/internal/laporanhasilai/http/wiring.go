package laporanhasilaihttp

import (
	"errors"
	"log/slog"

	"claim-pnc/internal/laporanhasilai/usecase"
)

// Handler melayani permintaan Laporan Hasil AI.
//
// # Tidak ada Caller di sini, dan itu bukan kelalaian
//
// Modul yang menulis menerima identitas pemanggil untuk mengisi kolom pencatat siapa.
// Modul ini **tidak menulis apa pun** — layar lamanya baca-saja, dan kedua tombolnya
// memanggil activity yang sama tanpa satu pun langkah tulis.
type Handler struct {
	service    *usecase.Service
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// Service melayani seluruh perkara modul ini. Wajib.
	Service *usecase.Service

	Logger *slog.Logger

	// WriteJSON disuntikkan dari cmd, bukan diimpor dari modul auth. Modul tidak saling
	// mengimpor lapisan transport-nya — itulah yang membuat modul dapat dipindahkan tanpa
	// menariknya serta.
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Laporan Hasil AI.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("laporanhasilai/http: Service wajib diisi")
	}
	if o.WriteJSON == nil {
		return nil, errors.New("laporanhasilai/http: WriteJSON wajib diisi")
	}
	return &Handler{
		service:    o.Service,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}, nil
}
