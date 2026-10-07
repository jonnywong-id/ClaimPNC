package mastertipesurveyorshttp

import (
	"net/http"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Master Tipe Surveyors.
type Handler struct {
	httpkit.Basic[Service]
	*crudhttp.Handler[SaveRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
type Options = httpkit.BasicOptions[Service]

// NewHandler membentuk handler modul Master Tipe Surveyors.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan
// pertama datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("mastertipesurveyors/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/tipe-surveyor.
	//
	// Menggantikan Report Definition `BrowseVMSurveyors_RD` yang mengisi grid layar
	// `SurveyorsInbox`.
	//
	// Create menangani POST /api/master/tipe-surveyor.
	//
	// Menggantikan tombol Tambah pada harness, yang mengirim sentinel `"UnknownID"` supaya
	// procedure membentuk kodenya. Di sini kodenya tidak pernah ikut di badan permintaan sama
	// sekali — sentinel itu tidak dibawa.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []mastertipesurveyors.SurveyorType, alias string) any {
			content := make([]SurveyorTypeDTO, 0, len(list))
			for _, t := range list {
				content = append(content, toDTO(t))
			}
			return ListResponse{
				SurveyorType: content,
				Total:        len(content),
				Portal:       alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (mastertipesurveyors.SurveyorType, error) {
			return h.Service.Create(r.Context(), alias, request.Description)
		}, func(saved mastertipesurveyors.SurveyorType, alias string) any {
			return SingleResponse{
				SurveyorType: toDTO(saved),
				Portal:       alias,
			}
		}),
	})
	return h, nil
}
