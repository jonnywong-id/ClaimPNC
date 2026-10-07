package mastergroupingspareparthttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/mastergroupingsparepart/usecase"
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/platform/masterhttp"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan berbeda dari Master Sparepart ia TIDAK TERSIMPAN: kedua tabel modul ini
// tidak punya kolom pencatat pelaku. Ia dibawa hanya untuk dicatat di log — satu-satunya
// tempat yang tersedia sampai `S-5` Jejak Audit dibangun.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master grouping sparepart.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[mastergroupingsparepart.Grouping, mastergroupingsparepart.Input, mastergroupingsparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master grouping sparepart.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("mastergroupingsparepart/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/grouping-sparepart.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nomor sparepart, nama sparepart, nama panel, dan no rangka
	//
	// Yang pertama adalah cerminan ketiga tab `Section/PNCMasterGroupingSparepartHE-Section.xml`.
	//
	// Yang kedua DITAMBAHKAN sebagai penyaring sisi server. Layar lama punya kotak pencarian, dan
	// ia menyaring di klipboard — atas daftar yang sudah terlanjur dimuat seluruhnya.
	//
	// Get menangani GET /master/grouping-sparepart/{id}.
	//
	// Create menangani POST /master/grouping-sparepart.
	//
	// Save menangani PUT /master/grouping-sparepart/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan.
	// Keputusan komite menempuh Decide; lihat usecase.Service.Save untuk selisihnya terhadap
	// sistem lama, yang menerima statusnya sebagai parameter.
	//
	// Decide menangani POST /master/grouping-sparepart/keputusan.
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
	h.Handler = masterhttp.New(masterhttp.Spec[mastergroupingsparepart.Grouping, mastergroupingsparepart.Input, mastergroupingsparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]{
		Module:           "mastergroupingsparepart/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         mastergroupingsparepart.ErrNotFound,
		DefaultStatus:    mastergroupingsparepart.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) mastergroupingsparepart.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []mastergroupingsparepart.Grouping, status mastergroupingsparepart.ApprovalStatus, alias string) any {
			return ListResponse{
				Grouping: toListDTO(list),
				Status:   string(status),
				Portal:   alias,
			}
		},
		SingleBody: func(r *http.Request, row mastergroupingsparepart.Grouping, alias string) any {
			return SingleResponse{
				Grouping: toDTO(row),
				Portal:   alias,
			}
		},
		Decider:         o.Service,
		DecisionIDs:     func(request DecisionRequest) []string { return request.ID },
		DecisionStatus:  func(request DecisionRequest) string { return request.Status },
		MaxDecisionRows: maxDecisionRows,
		TooManyRows: mastergroupingsparepart.OneViolation("id_grouping",
			"Terlalu banyak grouping dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."),
		DecisionBody: func(changed int, status mastergroupingsparepart.ApprovalStatus, alias string) any {
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
