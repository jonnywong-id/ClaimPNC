package mastersurveyorshttp

import (
	"context"

	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah identitas orang yang mengirim permintaan.
//
// Ia sengaja tipe milik modul ini, bukan tipe milik modul auth: modul tidak saling
// mengimpor, dan yang dibutuhkan di sini hanyalah dua field. Cara mengisinya diberikan
// saat perakitan di cmd/claimpnc lewat Options.Caller, sehingga modul ini tidak pernah
// tahu bagaimana sesi bekerja.
type Caller struct {
	// Identity adalah nilai yang dibandingkan dengan kolom KOMITE. Keduanya berisi
	// **Operator ID** — lihat catatan di paket committee.
	Identity string
	Name     string
}

// Handler melayani permintaan Master Surveyors.
type Handler struct {
	httpkit.Master[Service, func(context.Context) (Caller, bool)]
}

// Options adalah bahan pembentuk Handler.
//
// Caller membaca identitas pemanggil dari context. Diisi saat perakitan di cmd.
// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
type Options = httpkit.MasterOptions[Service, func(context.Context) (Caller, bool)]

// NewHandler membentuk handler modul Master Surveyors.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan
// pertama datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("mastersurveyors/http", o)
	if err != nil {
		return nil, err
	}
	return &Handler{Master: base}, nil
}
