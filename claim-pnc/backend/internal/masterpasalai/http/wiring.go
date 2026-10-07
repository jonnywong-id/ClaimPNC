package masterpasalaihttp

import (
	"claim-pnc/internal/masterpasalai/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Master Pasal AI.
//
// # Tidak ada Caller di sini, dan itu bukan kelalaian
//
// Modul yang menulis menerima identitas pemanggil untuk mengisi kolom pencatat siapa. Modul
// ini **tidak menulis apa pun** — layar lamanya baca-saja — sehingga tidak ada yang perlu
// dicatat.
type Handler struct {
	httpkit.Basic[*usecase.Service]
}

// Options adalah bahan pembentuk Handler.
//
// Service melayani seluruh perkara modul ini. Wajib.
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul dapat
// dipindahkan tanpa menariknya serta.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul Master Pasal AI.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("masterpasalai/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Basic: base}, nil
}
