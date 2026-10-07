package mastertipespareparthttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesparepart/usecase"
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/platform/masterhttp"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan seperti Master Kategori Sparepart ia **TIDAK TERSIMPAN**:
// `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak punya kolom pencatat pelaku sama sekali. Ia tetap
// diminta karena dipakai LOG — lihat usecase.Actor.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master tipe sparepart.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[mastertipesparepart.PartType, mastertipesparepart.Input, mastertipesparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master tipe sparepart.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("mastertipesparepart/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/tipe-sparepart.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nama tipe DAN nama kategorinya
	//
	// Yang pertama adalah cerminan ketiga tab `Section/MasterTipeSparepartHE-Section.xml`.
	// Bawaannya "1" — tab Approve — karena itulah tab pertama pada layar lama, dan karena baris
	// berstatus itulah yang benar-benar dipakai modul hilir.
	//
	// Yang kedua DITAMBAHKAN; lihat mastertipesparepart.Filter.
	//
	// Get menangani GET /master/tipe-sparepart/{id}.
	//
	// Create menangani POST /master/tipe-sparepart.
	//
	// Save menangani PUT /master/tipe-sparepart/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
	// persis seperti `Activity/UpdateTypeSparepart_act2` yang menetapkan APPROVAL := "0" tanpa
	// syarat apa pun. Keputusan persetujuan menempuh Decide.
	//
	// Decide menangani POST /master/tipe-sparepart/keputusan.
	//
	// Ia padanan `Section/ApprovalMasterTipeSparepartHE-Section.xml`: daftar baris bercentang
	// dan satu status, ditetapkan sekaligus.
	//
	// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
	//
	// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
	// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai
	// peristiwa, bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan
	// membuat keputusan yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
	//
	// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
	// ditulis.
	h.Handler = masterhttp.New(masterhttp.Spec[mastertipesparepart.PartType, mastertipesparepart.Input, mastertipesparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]{
		Module:           "mastertipesparepart/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         mastertipesparepart.ErrNotFound,
		DefaultStatus:    mastertipesparepart.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) mastertipesparepart.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []mastertipesparepart.PartType, status mastertipesparepart.ApprovalStatus, alias string) any {
			return ListResponse{
				Type:   toListDTO(list),
				Status: string(status),
				Portal: alias,
			}
		},
		SingleBody: func(r *http.Request, row mastertipesparepart.PartType, alias string) any {
			return SingleResponse{
				Type:   toDTO(row),
				Portal: alias,
			}
		},
		Decider:         o.Service,
		DecisionIDs:     func(request DecisionRequest) []string { return request.ID },
		DecisionStatus:  func(request DecisionRequest) string { return request.Status },
		MaxDecisionRows: maxDecisionRows,
		TooManyRows: mastertipesparepart.OneViolation("id_tipe_sparepart",
			"Terlalu banyak tipe dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."),
		DecisionBody: func(changed int, status mastertipesparepart.ApprovalStatus, alias string) any {
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
