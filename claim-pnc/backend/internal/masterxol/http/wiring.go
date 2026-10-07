package masterxolhttp

import (
	"context"
	"errors"
	"log/slog"
)

// Caller adalah identitas pemanggil yang sedang bekerja.
//
// Ia dinyatakan di sini sebagai tipe sempit, bukan diimpor dari modul auth, supaya kedua
// modul tetap tidak saling mengimpor. Yang menjembatani keduanya hanyalah berkas
// perakitan di cmd/claimpnc.
type Caller struct {
	Identity string
	Name     string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Master XOL.
type Handler struct {
	service       Service
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service

	// Caller mengisi kolom PIC saat induk diajukan ke komite. WAJIB: `D-59` menetapkan
	// tidak ada pemisahan tugas formal, sehingga jejak siapa-mengajukan-apa adalah satu-
	// satunya kontrol pengimbang yang tersisa.
	Caller CallerReader

	Logger *slog.Logger

	// WriteResponse dan FallbackErrorWriter dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat portal dengan cara yang sama.
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Master XOL.
func NewHandler(o Options) (*Handler, error) {
	switch {
	case o.Service == nil:
		return nil, errors.New("masterxol/http: Service wajib diisi")
	case o.Caller == nil:
		return nil, errors.New("masterxol/http: Caller wajib diisi")
	case o.WriteResponse == nil:
		return nil, errors.New("masterxol/http: WriteResponse wajib diisi")
	}

	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}, nil
}
