package masterpanelhttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/usecase"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/platform/masterhttp"
)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja. Ia TIDAK pernah tersimpan ke basis data — `POOLDATA.PANEL_HE` tidak
// punya kolom pencatat pelaku maupun waktu — dan hanya dipakai untuk log.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master panel.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[masterpanel.Panel, masterpanel.Input, masterpanel.ApprovalStatus, usecase.Actor, SaveRequest, struct{}]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master panel.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("masterpanel/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/panel.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nama panel
	//
	// Yang pertama adalah cerminan ketiga tab `Section/BrowsePanelHE-Section.xml`, yang
	// ketiganya memuat section berbeda dengan nilai penyaring "0", "1", dan "2".
	//
	// Yang kedua DITAMBAHKAN. Report definition lama membatasi hasilnya di 500 baris
	// (`pyMaxRecords=500`) tanpa satu pun cara mempersempitnya dari layar; pencarian di sini
	// yang menggantikan pemotongan itu.
	//
	// Get menangani GET /master/panel/{id}.
	//
	// Create menangani POST /master/panel.
	//
	// Save menangani PUT /master/panel/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
	// persis seperti `Activity/CNMUpdatePanelHE_act` yang menetapkan `APPROVAL := "0"` tanpa
	// syarat apa pun. Keputusan komite menempuh Decide.
	//
	// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
	// ditulis.
	h.Handler = masterhttp.New(masterhttp.Spec[masterpanel.Panel, masterpanel.Input, masterpanel.ApprovalStatus, usecase.Actor, SaveRequest, struct{}]{
		Module:           "masterpanel/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterpanel.ErrNotFound,
		DefaultStatus:    masterpanel.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) masterpanel.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []masterpanel.Panel, status masterpanel.ApprovalStatus, alias string) any {
			return ListResponse{
				Panel:  toListDTO(list),
				Status: string(status),
				Portal: alias,
			}
		},
		SingleBody: func(r *http.Request, row masterpanel.Panel, alias string) any {
			return SingleResponse{
				Panel:  toDTO(row),
				Portal: alias,
			}
		},
	})
	return h, nil
}
