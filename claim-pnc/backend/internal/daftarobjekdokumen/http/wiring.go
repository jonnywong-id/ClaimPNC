package daftarobjekdokumenhttp

import (
	"net/http"

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Daftar Objek Dokumen.
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

// NewHandler membentuk handler modul Daftar Objek Dokumen.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("daftarobjekdokumen/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/objek-dokumen.
	//
	// Get menangani GET /master/objek-dokumen/{id}.
	//
	// Rute tersendiri, bukan sekadar mencari di hasil List, karena hanya di sini pemetaan
	// bisnisnya ikut dimuat — dan layar membutuhkannya tepat saat baris dibuka untuk disunting,
	// bukan saat daftarnya ditampilkan.
	//
	// Create menangani POST /master/objek-dokumen.
	//
	// Update menangani PUT /master/objek-dokumen/{id}.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         daftarobjekdokumen.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []daftarobjekdokumen.DocumentObject, alias string) any {
			content := toListDTO(list)
			return ListResponse{
				DocumentObject: content,
				Total:          len(content),
				Portal:         alias,
			}
		}),
		Get: crudhttp.LoadOne(h.Service.Get, func(row daftarobjekdokumen.DocumentObject, alias string) any {
			return SingleResponse{
				DocumentObject: toDTO(row),
				Portal:         alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (daftarobjekdokumen.DocumentObject, error) {
			return h.Service.Create(r.Context(), alias, requestToInput(request))
		}, func(saved daftarobjekdokumen.DocumentObject, alias string) any {
			return SingleResponse{
				DocumentObject: toDTO(saved),
				Portal:         alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (daftarobjekdokumen.DocumentObject, error) {
			return h.Service.Update(r.Context(), alias, id, requestToInput(request))
		}, func(saved daftarobjekdokumen.DocumentObject, alias string) any {
			return SingleResponse{
				DocumentObject: toDTO(saved),
				Portal:         alias,
			}
		}),
	})
	return h, nil
}
