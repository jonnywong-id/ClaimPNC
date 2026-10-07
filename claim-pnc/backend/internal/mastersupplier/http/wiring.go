package mastersupplierhttp

import (
	"net/http"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Berbeda dari Master Bengkel, identitas ini BENAR-BENAR TERSIMPAN: ia menjadi kunci
// `USERKLAIMID` di dalam dokumen supplier dan kolom `USER_REQ` pada baris permintaan
// persetujuan. Keduanya ditulis sistem lama juga.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master supplier.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*crudhttp.Handler[struct{}]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master supplier.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("mastersupplier/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/supplier.
	//
	// # Penyaringnya
	//
	//	?cari=...  mempersempit pada nama, kota, atau contact person
	//
	// Ia DITAMBAHKAN. Layar lama tidak punya penyaring apa pun — `Section/InboxMasterSupplier`
	// hanya punya tombol New mastersupplier.Supplier, Edit, dan Refresh — dan gridnya membaca page list yang
	// tidak ada satu pun rule di export yang mengisinya (`R-16`). Tanpa kueri lamanya, tidak
	// ada yang dapat ditiru; yang dapat dilakukan adalah membuat daftar yang dapat
	// dipersempit.
	//
	// Get menangani GET /master/supplier/{id}.
	//
	// Ia padanan `Activity/GetDataSupplier_pre`, termasuk penurunan JENIS_STATUS dari
	// SUPPLIER_HE yang dikerjakan adapter saat membaca.
	h.Handler = crudhttp.New(crudhttp.Spec[struct{}]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		NotFound:         mastersupplier.ErrNotFound,
		List: func(r *http.Request, alias string) (any, error) {
			list, err := h.Service.List(r.Context(), alias, r.URL.Query().Get("cari"))
			if err != nil {
				return nil, err
			}
			return ListResponse{
				Supplier: toListDTO(list),
				Portal:   alias,
			}, nil
		},
		Get: crudhttp.LoadOne(h.Service.Get, func(found mastersupplier.Supplier, alias string) any {
			return SingleResponse{
				Supplier: toDTO(found),
				Portal:   alias,
			}
		}),
	})
	return h, nil
}
