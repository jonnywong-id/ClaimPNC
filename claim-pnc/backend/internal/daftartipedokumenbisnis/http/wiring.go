package daftartipedokumenbisnishttp

import (
	"context"
	"errors"
	"log/slog"

	"claim-pnc/internal/daftartipedokumenbisnis/usecase"
	"claim-pnc/internal/platform/apierror"
)

// JSONWriter menulis jawaban berhasil. Disuntikkan cmd supaya bentuk amplopnya seragam di
// seluruh aplikasi tanpa modul saling mengimpor.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menulis galat yang BUKAN milik modul ini — terutama galat portal.
type ErrorWriter = apierror.ErrorWriter

// Caller adalah identitas pemanggil, sejauh yang dibutuhkan modul ini.
//
// Tipe milik modul, bukan impor dari modul auth: modul tidak saling mengimpor lapisan
// transport-nya. Yang menjembatani keduanya adalah cmd.
type Caller struct {
	Identity string

	// Position adalah pyPosition pemanggil.
	//
	// Dipakai HANYA untuk memutuskan apakah tombol "Pilih semua" ditampilkan, meniru
	// `pyVisible` layar lama. Ia bukan kewenangan — lihat Service.MayBulkSelect.
	Position string
}

// Handler melayani rute modul Daftar Tipe Dokumen Bisnis.
type Handler struct {
	service       *usecase.Service
	caller        func(context.Context) (Caller, bool)
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah ketergantungan Handler.
type Options struct {
	Service       *usecase.Service
	Logger        *slog.Logger
	Caller        func(context.Context) (Caller, bool)
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk Handler dan menolak ketergantungan yang belum diisi.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("daftartipedokumenbisnis/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("daftartipedokumenbisnis/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}
