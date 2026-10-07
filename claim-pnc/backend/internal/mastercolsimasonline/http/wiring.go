package mastercolsimasonlinehttp

import (
	"net/http"

	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/mastercolsimasonline/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan master COL Simas Online.
type Handler struct {
	httpkit.Basic[*usecase.Service]
	*crudhttp.Handler[SaveRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
// dapat dipindahkan tanpa menariknya serta.
type Options = httpkit.BasicOptions[*usecase.Service]

// NewHandler membentuk handler modul master COL Simas Online.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("mastercolsimasonline/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/col-simas-online.
	//
	// Get menangani GET /master/col-simas-online/{id}.
	//
	// Rute tersendiri, bukan sekadar mencari di hasil List, karena hanya di sini pemetaan
	// bisnisnya ikut dimuat — dan layar membutuhkannya tepat saat baris dibuka untuk
	// disunting, bukan saat daftarnya ditampilkan.
	//
	// Create menangani POST /master/col-simas-online.
	//
	// Update menangani PUT /master/col-simas-online/{id}.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         mastercolsimasonline.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []mastercolsimasonline.CauseOfLoss, alias string) any {
			return ListResponse{
				CauseOfLoss: toListDTO(list),
				Portal:      alias,
			}
		}),
		Get: crudhttp.LoadOne(h.Service.Get, func(row mastercolsimasonline.CauseOfLoss, alias string) any {
			return SingleResponse{
				CauseOfLoss: toDTO(row),
				Portal:      alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (mastercolsimasonline.CauseOfLoss, error) {
			return h.Service.Create(r.Context(), alias, requestToInput(request))
		}, func(saved mastercolsimasonline.CauseOfLoss, alias string) any {
			return SingleResponse{
				CauseOfLoss: toDTO(saved),
				Portal:      alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (mastercolsimasonline.CauseOfLoss, error) {
			return h.Service.Update(r.Context(), alias, id, requestToInput(request))
		}, func(saved mastercolsimasonline.CauseOfLoss, alias string) any {
			return SingleResponse{
				CauseOfLoss: toDTO(saved),
				Portal:      alias,
			}
		}),
	})
	return h, nil
}
