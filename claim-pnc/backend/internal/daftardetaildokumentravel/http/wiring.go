package daftardetaildokumentravelhttp

import (
	"net/http"

	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Handler melayani permintaan Daftar daftardetaildokumentravel.Detail Dokumen Travel.
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

// NewHandler membentuk handler modul Daftar daftardetaildokumentravel.Detail Dokumen Travel.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewBasic("daftardetaildokumentravel/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Basic: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /api/master/daftar-detail-dokumen-travel.
	//
	// Menggantikan Report Definition `BrowseLstDocTravel_RD` yang mengisi grid layar
	// `ListDocumentTravel_Harness`.
	//
	// Get menangani GET /api/master/daftar-detail-dokumen-travel/{id}.
	//
	// Menggantikan `CNMSetDetailTravelDocument_act`, yang mengisi form dari baris terpilih.
	// Berbeda dari List, jawaban ini MEMBAWA daftar pembatasan plan dan jaminannya — dan
	// form memang membutuhkannya tepat saat baris dibuka, bukan saat daftarnya ditampilkan.
	//
	// Create menangani POST /api/master/daftar-detail-dokumen-travel.
	//
	// Menggantikan tombol Tambah lalu Simpan pada harness, yang memanggil
	// `CNMInsertDocumentTravel_act` — activity yang TIDAK ADA di export (`R-16`), sehingga
	// yang ditiru adalah apa yang disimpannya, bukan urutan langkahnya.
	//
	// Update menangani PUT /api/master/daftar-detail-dokumen-travel/{id}.
	//
	// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali, sehingga
	// permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama dua kali
	// menghasilkan keadaan akhir yang sama — termasuk untuk daftar pembatasan plan-nya, yang
	// memang diganti seluruhnya.
	h.Handler = crudhttp.New(crudhttp.Spec[SaveRequest]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         daftardetaildokumentravel.ErrNotFound,
		Read:             h.readRequest,
		List: crudhttp.Load(h.Service.List, func(list []daftardetaildokumentravel.Detail, alias string) any {
			content := toListDTO(list)
			return ListResponse{
				Detail: content,
				Total:  len(content),
				Portal: alias,
			}
		}),
		Get: crudhttp.LoadOne(h.Service.Get, func(row daftardetaildokumentravel.Detail, alias string) any {
			return SingleResponse{
				Detail: toDTO(row),
				Portal: alias,
			}
		}),
		Create: crudhttp.Save(func(r *http.Request, alias string, request SaveRequest) (daftardetaildokumentravel.Detail, error) {
			return h.Service.Create(r.Context(), alias, toInput(request))
		}, func(saved daftardetaildokumentravel.Detail, alias string) any {
			return SingleResponse{
				Detail: toDTO(saved),
				Portal: alias,
			}
		}),
		Update: crudhttp.SaveOne(func(r *http.Request, alias, id string, request SaveRequest) (daftardetaildokumentravel.Detail, error) {
			return h.Service.Update(r.Context(), alias, id, toInput(request))
		}, func(saved daftardetaildokumentravel.Detail, alias string) any {
			return SingleResponse{
				Detail: toDTO(saved),
				Portal: alias,
			}
		}),
	})
	return h, nil
}
