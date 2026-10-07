package dashboardclaimhttp

import (
	"errors"
	"log/slog"
	"time"
)

// Handler melayani rute Dashboard Claim.
type Handler struct {
	service  Service
	logger   *slog.Logger
	location *time.Location

	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service
	Logger  *slog.Logger

	// Location adalah zona waktu tampilan. Kosong berarti Asia/Jakarta.
	//
	// Ia disuntik, tidak dibaca dari lingkungan di sini, supaya uji dapat menetapkannya dan
	// hasilnya tidak berubah menurut mesin yang menjalankannya.
	//
	// Bawaannya WIB, bukan UTC: waktu disimpan UTC dan DITAMPILKAN WIB (`F-5`). Bawaan UTC
	// akan membuat tanggal bergeser satu hari pada kejadian menjelang tengah malam — dan
	// pergeseran itu tidak terlihat sebagai galat, hanya sebagai tanggal yang salah.
	Location *time.Location

	// WriteResponse dan WriteError disuntik dari cmd supaya bentuk respons seragam di
	// seluruh modul, bukan disusun ulang di setiap handler.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler dan menolak Options yang tidak lengkap.
//
// Penolakan terjadi saat aplikasi START, bukan saat pengguna membuka layar.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("dashboardclaim/http: Service wajib diisi")
	}
	if o.WriteResponse == nil {
		return nil, errors.New("dashboardclaim/http: WriteResponse wajib diisi")
	}
	if o.WriteError == nil {
		return nil, errors.New("dashboardclaim/http: WriteError wajib diisi")
	}
	location := o.Location
	if location == nil {
		location = jakarta()
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		location:      location,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}
