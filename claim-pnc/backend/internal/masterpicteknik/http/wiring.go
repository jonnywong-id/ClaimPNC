package masterpicteknikhttp

import (
	"net/http"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Master PIC Teknik.
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

// NewHandler membentuk handler modul Master PIC Teknik.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan pertama
// datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("masterpicteknik/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/pic-teknik.
	//
	// Menggantikan Report Definition `BrowseVMstUserTeknis_RD` yang mengisi grid layar
	// `UserTeknisInbox`, termasuk penyaring `STS_AKTIF = '1'` yang dipatok di dalamnya.
	//
	// Create menangani POST /api/master/pic-teknik.
	//
	// Menggantikan `CNMInsertMstUserTeknis_act`, termasuk penolakan bila ID operatornya tidak
	// ditemukan di direktori pegawai.
	//
	// Update menangani PUT /api/master/pic-teknik/{id}.
	//
	// ID diambil dari jalur URL, tidak pernah dari badan permintaan.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterpicteknik.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []masterpicteknik.Technician, alias string) any {
			content := make([]TechnicianDTO, 0, len(list))
			for _, t := range list {
				content = append(content, toDTO(t))
			}
			return ListResponse{
				Technician: content,
				Total:      len(content),
				Portal:     alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (masterpicteknik.Technician, error) {
			return h.Service.Create(r.Context(), alias, fromRequest(request.OperatorID, request))
		}, func(saved masterpicteknik.Technician, alias string) any {
			return SingleResponse{
				Technician: toDTO(saved),
				Portal:     alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (masterpicteknik.Technician, error) {
			return h.Service.Update(r.Context(), alias, id, fromRequest(id, request))
		}, func(saved masterpicteknik.Technician, alias string) any {
			return SingleResponse{
				Technician: toDTO(saved),
				Portal:     alias,
			}
		}),
	})
	return h, nil
}
