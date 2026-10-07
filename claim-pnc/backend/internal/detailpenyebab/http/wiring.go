package detailpenyebabhttp

import (
	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/detailpenyebab/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan ia dipakai HANYA untuk mengisi log — tabelnya tidak punya kolom pencatat
// pelaku sama sekali. Berbeda dari Master Login, identitas pemanggil di modul ini TIDAK
// menjadi data yang tersimpan; lihat usecase.Actor.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan Detail Penyebab Kerugian.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*crudhttp.Handler[struct{}]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul Detail Penyebab Kerugian.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("detailpenyebab/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// Get menangani GET /master/detail-penyebab/{id}.
	//
	// Jalurnya memakai D_COL_ID — kunci barisnya, dan satu-satunya yang membedakan dua baris.
	h.Handler = crudhttp.New(crudhttp.Spec[struct{}]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         detailpenyebab.ErrNotFound,
		Get: crudhttp.LoadOne(h.Service.Get, func(found detailpenyebab.CauseOfLossDetail, alias string) any {
			return SingleResponse{
				Detail: toDTO(found),
				Portal: alias,
			}
		}),
	})
	return h, nil
}
