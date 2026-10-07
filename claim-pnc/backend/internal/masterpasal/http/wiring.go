package masterpasalhttp

import (
	"claim-pnc/internal/masterpasal/usecase"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Master Pasal Kerugian.
//
// # Tidak ada Caller di sini, dan itu bukan kelalaian
//
// Modul master lain menerima identitas pemanggil untuk mengisi kolom pencatat siapa.
// POOLDATA.V_M_DATA_PASAL tidak punya kolom semacam itu — hanya IDDATA, IDPASAL, dan
// JSONPASAL — sehingga tidak ada tempat untuk menuliskannya.
//
// Akibatnya dicatat sebagai keterbatasan, bukan ditambal dengan kolom yang dikarang:
// perubahan dan penghapusan di layar ini TIDAK MENINGGALKAN JEJAK di basis data.
// Menambah kolom menempuh `D-63`.
type Handler struct {
	httpkit.Basic[*usecase.Service]
}

// Options adalah bahan pembentuk Handler.
//
// Service melayani seluruh perkara modul ini. Wajib.
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
// dapat dipindahkan tanpa menariknya serta.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul Master Pasal Kerugian.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("masterpasal/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Basic: base}, nil
}
