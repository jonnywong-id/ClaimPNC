package mastermaskinghttp

import (
	"context"
	"errors"
	"log/slog"
)

// Caller adalah identitas pemanggil yang sedang bekerja.
//
// Ia dinyatakan di sini sebagai tipe sempit, bukan diimpor dari modul auth, supaya kedua
// modul tetap tidak saling mengimpor. Yang menjembatani keduanya hanyalah berkas perakitan
// di cmd/claimpnc.
type Caller struct {
	Identity string
	Name     string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Master Masking.
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

	// Caller dipakai mengisi kolom USERINPUT.
	//
	// WAJIB, dan pada modul ini alasannya lebih keras daripada sekadar kerapian: yang
	// dicatat adalah siapa yang memberi atau mencabut kewenangan membuka data pribadi
	// nasabah. `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga jejak inilah
	// satu-satunya kontrol pengimbang yang tersisa.
	Caller CallerReader

	Logger *slog.Logger

	// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
	// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
	// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Masking.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan pertama
// datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	switch {
	case o.Service == nil:
		return nil, errors.New("mastermasking/http: Service wajib diisi")
	case o.Caller == nil:
		return nil, errors.New("mastermasking/http: Caller wajib diisi")
	case o.WriteResponse == nil || o.WriteError == nil:
		return nil, errors.New("mastermasking/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}
