package masterloginhttp

import (
	"net/http"

	"claim-pnc/internal/masterlogin/usecase"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpkit"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan ia dipakai DUA hal — perbedaan yang menentukan:
//
//  1. **Menurunkan LOGINLEADER** pada penambahan. Itu DATA yang tersimpan, bukan
//     pencatatan. Lihat usecase.Service.Create.
//  2. **Mengisi log.** Tabelnya tidak punya kolom pencatat pelaku sama sekali.
//
// Karena yang pertama, identitas pemanggil di modul ini BUKAN sekadar pelengkap: tanpa
// nilainya, setiap baris baru lahir tanpa tautan tim.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master login surveyor.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*crudhttp.Handler[struct{}]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master login.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("masterlogin/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Handler baku dijalankan mesin bersama crudhttp; catatan perilakunya:
	// List menangani GET /master/login.
	//
	// # Penyaringnya
	//
	//	?cari=...  mempersempit pada Nama, Login, dan Email
	//
	// TANPA penyaring status: tabelnya tidak punya kolom APPROVAL, dan layar lamanya tidak
	// bertab. Lihat banner masterlogin.SurveyorLogin.
	//
	// Cakupan daftarnya SELURUH baris tabel; rule yang mengisi grid Pega tidak ada di export
	// (`R-16`), dan alasannya beserta bacaan lain yang mungkin ada pada masterlogin.Filter.
	h.Handler = crudhttp.New(crudhttp.Spec[struct{}]{
		WriteResponse:    h.WriteResponse,
		WriteModuleError: h.writeModuleError,
		List: func(r *http.Request, alias string) (any, error) {
			list, err := h.Service.List(r.Context(), alias, r.URL.Query().Get("cari"))
			if err != nil {
				return nil, err
			}
			return ListResponse{
				Login:  toListDTO(list),
				Portal: alias,
			}, nil
		},
	})
	return h, nil
}
