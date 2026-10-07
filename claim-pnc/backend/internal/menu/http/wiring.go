package menuhttp

import (
	"context"

	"claim-pnc/internal/menu/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah bagian identitas pemanggil yang dibutuhkan modul ini.
//
// Hanya SATU field, dan itu disengaja: menu hanya perlu tahu login siapa yang bertanya.
// Menerima seluruh catatan pengguna akan membuat modul ini bergantung pada bentuk data
// modul auth.
type Caller struct {
	// Login adalah yang DIKETIK pengguna di layar masuk — itulah yang dicocokkan ke
	// M_LOGIN_GROUP_PNC.LOGIN_ID dan M_OTORISASI_PNC.LOGIN_ID_GROUP, sesuai aturan yang
	// ditetapkan Work Owner 2026-09-18.
	Login string
}

// Handler melayani permintaan menu.
type Handler struct {
	httpkit.Master[*usecase.Service, func(ctx context.Context) (Caller, bool)]
}

// Options adalah bahan pembentuk Handler.
//
// Caller adalah jembatan SATU ARAH dari modul auth ke modul ini. Ia disuntikkan
// dari cmd, bukan diimpor, supaya kedua modul tetap tidak saling mengimpor — yang
// tahu keduanya hanyalah berkas perakitan.
// WriteResponse dan WriteError disuntikkan dari cmd supaya seluruh modul menuliskan
// respons dan galat sesi dengan cara yang sama.
type Options = httpkit.MasterOptions[*usecase.Service, func(ctx context.Context) (Caller, bool)]

// NewHandler membentuk handler modul menu.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("menu/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}
