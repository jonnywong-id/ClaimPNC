package masterdokumentravelhttp

import (
	"net/http"

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Master Dokumen Travel.
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

// NewHandler membentuk handler modul Master Dokumen Travel.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("masterdokumentravel/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/dokumen-travel.
	//
	// Menggantikan Report Definition `BrowseMstDocTravel_RD` yang mengisi grid layar
	// `BrowseMasterDocumentTravel_Harness`.
	//
	// Create menangani POST /api/master/dokumen-travel.
	//
	// Menggantikan tombol Tambah lalu Simpan pada harness, yang mengirim sentinel
	// `"UnknownID"` supaya procedure memilih cabang INSERT. Di sini ID tidak pernah ikut di
	// badan permintaan sama sekali — sentinel itu tidak dibawa.
	//
	// Update menangani PUT /api/master/dokumen-travel/{id}.
	//
	// PUT, bukan PATCH: seluruh isi yang boleh diubah — satu field, judul — dikirim setiap
	// kali, sehingga permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama
	// dua kali menghasilkan keadaan akhir yang sama.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterdokumentravel.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []masterdokumentravel.TravelDocument, alias string) any {
			content := toListDTO(list)
			return ListResponse{
				TravelDocument: content,
				Total:          len(content),
				Portal:         alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (masterdokumentravel.TravelDocument, error) {
			return h.Service.Create(r.Context(), alias, masterdokumentravel.Input{Name: request.Name})
		}, func(saved masterdokumentravel.TravelDocument, alias string) any {
			return SingleResponse{
				TravelDocument: toDTO(saved),
				Portal:         alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (masterdokumentravel.TravelDocument, error) {
			return h.Service.Update(r.Context(), alias, id, masterdokumentravel.Input{Name: request.Name})
		}, func(saved masterdokumentravel.TravelDocument, alias string) any {
			return SingleResponse{
				TravelDocument: toDTO(saved),
				Portal:         alias,
			}
		}),
	})
	return h, nil
}
