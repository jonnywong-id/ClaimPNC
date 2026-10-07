package masterreashttp

import (
	"claim-pnc/internal/masterreas/usecase"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpkit"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// Handler melayani permintaan master reas.
//
// TANPA CallerReader, berbeda dari modul master lain. Dua sebabnya, dan keduanya berasal
// dari modul ini yang hanya membaca: tidak ada baris yang diturunkan dari identitas
// pemanggil, dan tidak ada perubahan yang perlu dicatat pelakunya.
//
// Kewenangan membuka layar ini tetap ditegakkan — lewat middleware Autentikasi yang dipasang
// cmd, dan kelak lewat pemeriksaan peran `TKT-F3-005` yang belum ada.
type Handler struct {
	httpkit.Basic[*usecase.Service]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul master reas.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("masterreas/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Basic: base}, nil
}
