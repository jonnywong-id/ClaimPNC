package masterbengkelhttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/usecase"
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/platform/masterhttp"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja. Ia TIDAK pernah tersimpan ke basis data — `POOLDATA.BENGKEL_HE` tidak
// punya kolom pencatat pelaku maupun waktu — dan hanya dipakai untuk log.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master bengkel.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[masterbengkel.Workshop, masterbengkel.Input, masterbengkel.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master bengkel.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("masterbengkel/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/bengkel.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nama bengkel, kota, atau cabang
	//
	// Yang pertama adalah cerminan ketiga tab `Section/BrowseMasterHE-Section.xml`:
	//
	//	APPROVE           status=1
	//	WAITING APPROVAL  status=0
	//	REJECT            status=2
	//
// Yang kedua DITAMBAHKAN. Report definition lama membatasi hasilnya di 500 baris
// (`pyMaxRecords=500`) tanpa satu pun cara mempersempitnya dari layar; pencarian di sini
// yang menggantikan pemotongan itu.
	//
	// Get menangani GET /master/bengkel/{id}.
	//
	// Create menangani POST /master/bengkel.
	//
	// Save menangani PUT /master/bengkel/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
	// persis seperti `Activity/UpdateBengkelHE_act` step 7 yang menetapkan `APPROVAL := "0"`
	// tanpa syarat apa pun. Keputusan komite menempuh Decide.
	//
	// Decide menangani POST /master/bengkel/keputusan.
	//
	// Ia padanan `Activity/SetApprovalAllMaster`: daftar baris bercentang beserta satu
	// status, ditetapkan sekaligus.
	//
	// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
	//
	// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
	// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai
	// peristiwa, bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan
	// membuat keputusan yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
	//
	// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya
	// sudah ditulis.
	h.Handler = masterhttp.New(masterhttp.Spec[masterbengkel.Workshop, masterbengkel.Input, masterbengkel.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]{
		Module:           "masterbengkel/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterbengkel.ErrNotFound,
		DefaultStatus:    masterbengkel.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) masterbengkel.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []masterbengkel.Workshop, status masterbengkel.ApprovalStatus, alias string) any {
			return ListResponse{
				Bengkel: toListDTO(list),
				Status:  string(status),
				Portal:  alias,
			}
		},
		SingleBody: func(r *http.Request, row masterbengkel.Workshop, alias string) any {
			return SingleResponse{
				Bengkel: toDTO(row),
				Portal:  alias,
			}
		},
		Decider:         o.Service,
		DecisionIDs:     func(request DecisionRequest) []string { return request.ID },
		DecisionStatus:  func(request DecisionRequest) string { return request.Status },
		MaxDecisionRows: maxDecisionRows,
		TooManyRows: masterbengkel.OneViolation("id_bengkel",
			"Terlalu banyak bengkel dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."),
		DecisionBody: func(changed int, status masterbengkel.ApprovalStatus, alias string) any {
			return DecisionResponse{
				Changed:     changed,
				Status:      string(status),
				StatusLabel: status.Label(),
				Portal:      alias,
			}
		},
	})
	return h, nil
}
