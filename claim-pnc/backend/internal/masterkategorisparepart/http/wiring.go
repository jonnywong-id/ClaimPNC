package masterkategorispareparthttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterkategorisparepart/usecase"
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
// Satu field, dan berbeda dari Master Sparepart ia **TIDAK TERSIMPAN**:
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya kolom pencatat pelaku sama sekali. Ia
// tetap diminta karena dipakai LOG — lihat usecase.Actor.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master kategori sparepart.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[masterkategorisparepart.PartCategory, masterkategorisparepart.Input, masterkategorisparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master kategori sparepart.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("masterkategorisparepart/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/kategori-sparepart.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nama kategori
	//
	// Yang pertama adalah cerminan ketiga tab `Section/MasterKategoriSparepartHE-Section.xml`.
	// Bawaannya "1" — tab Approve — karena itulah tab pertama pada layar lama, dan karena baris
	// berstatus itulah yang benar-benar dipakai modul hilir.
	//
	// Yang kedua DITAMBAHKAN; lihat masterkategorisparepart.Filter.
	//
	// Get menangani GET /master/kategori-sparepart/{id}.
	//
	// Create menangani POST /master/kategori-sparepart.
	//
	// Save menangani PUT /master/kategori-sparepart/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
	// persis seperti `Activity/UpdateKategoriSparepart_act2` yang menetapkan APPROVAL := "0"
	// tanpa syarat apa pun. Keputusan persetujuan menempuh Decide.
	//
	// Decide menangani POST /master/kategori-sparepart/keputusan.
	//
	// Ia padanan `Section/ApprovalMasterKategoriSparepartHE-Section.xml`: daftar baris
	// bercentang dan satu status, ditetapkan sekaligus.
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
	h.Handler = masterhttp.New(masterhttp.Spec[masterkategorisparepart.PartCategory, masterkategorisparepart.Input, masterkategorisparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]{
		Module:           "masterkategorisparepart/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         masterkategorisparepart.ErrNotFound,
		DefaultStatus:    masterkategorisparepart.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) masterkategorisparepart.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []masterkategorisparepart.PartCategory, status masterkategorisparepart.ApprovalStatus, alias string) any {
			return ListResponse{
				Category: toListDTO(list),
				Status:   string(status),
				Portal:   alias,
			}
		},
		SingleBody: func(r *http.Request, row masterkategorisparepart.PartCategory, alias string) any {
			return SingleResponse{
				Category: toDTO(row),
				Portal:   alias,
			}
		},
		Decider:         o.Service,
		DecisionIDs:     func(request DecisionRequest) []string { return request.ID },
		DecisionStatus:  func(request DecisionRequest) string { return request.Status },
		MaxDecisionRows: maxDecisionRows,
		TooManyRows: masterkategorisparepart.OneViolation("id_kategori_sparepart",
			"Terlalu banyak kategori dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."),
		DecisionBody: func(changed int, status masterkategorisparepart.ApprovalStatus, alias string) any {
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
