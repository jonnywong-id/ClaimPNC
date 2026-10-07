package masterautoclaimhttp

import (
	"claim-pnc/internal/masterautoclaim/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja — lihat usecase.Actor untuk alasan kenapa yang dipakai LOGIN dan
// bukan NIK.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master auto claim.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master auto claim.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("masterautoclaim/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}
