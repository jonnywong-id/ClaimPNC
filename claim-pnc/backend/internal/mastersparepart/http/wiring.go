package masterspareparthttp

import (
	"context"
	"net/http"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/usecase"
	"claim-pnc/internal/platform/httpkit"
	"claim-pnc/internal/platform/masterhttp"
)

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja, dan berbeda dari Master Panel ia BENAR-BENAR TERSIMPAN:
// `POOLDATA.SPAREPART_HE` punya kolom `USER_UPDATE`.
type Caller = httpkit.Caller

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader = httpkit.CallerReader

// Handler melayani permintaan master sparepart.
type Handler struct {
	httpkit.Master[*usecase.Service, CallerReader]
	*masterhttp.Handler[mastersparepart.Sparepart, mastersparepart.Input, mastersparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]
}

// Options adalah bahan pembentuk Handler.
//
// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
type Options = httpkit.MasterOptions[*usecase.Service, CallerReader]

// NewHandler membentuk handler modul master sparepart.
func NewHandler(o Options) (*Handler, error) {
	base, err := httpkit.NewMaster("mastersparepart/http", o)
	if err != nil {
		return nil, err
	}
	h := &Handler{Master: base}
	// Kelima handler dijalankan mesin bersama masterhttp; catatan perilakunya:
	// List menangani GET /master/sparepart.
	//
	// # Penyaringnya
	//
	//	?status=0|1|2  wajib dikenal; bila tidak disebut dianggap "1"
	//	?cari=...      mempersempit pada nama, nomor, DAN kode sparepart
	//
	// Yang pertama adalah cerminan ketiga tab `Section/BrowseMasterSparepartHE-Section.xml`,
	// yang ketiganya memuat section berbeda dengan nilai penyaring "1", "2", dan "0".
	//
	// Yang kedua DITAMBAHKAN. Report definition lama membatasi hasilnya di 500 baris
	// (`pyMaxRecords=500`) tanpa satu pun cara mempersempitnya dari layar; pencarian di sini
	// yang menggantikan pemotongan itu.
	//
	// # Kenapa daftar acuan ikut dibaca di jalur daftar
	//
	// Grid menampilkan NAMA kategori dan tipe, sedangkan tabelnya hanya menyimpan ID-nya. Tanpa
	// ini layar harus memanggil `/pilihan` sendiri lalu mencocokkan di peramban — dua perjalanan
	// untuk menggambar satu tabel, dan satu kesempatan lagi bagi keduanya menjadi tidak sinkron.
	//
	// Kegagalan membaca daftar acuan TIDAK menggagalkan daftarnya: bila tabel acuannya tidak
	// dapat dibaca, kolom namanya dikirim kosong dan barisnya tetap tampil dengan kodenya. Daftar
	// sparepart yang hilang seluruhnya karena tabel LAIN bermasalah adalah kegagalan yang jauh
	// lebih besar daripada yang dicegahnya.
	//
	// Get menangani GET /master/sparepart/{id}.
	//
	// Create menangani POST /master/sparepart.
	//
	// Save menangani PUT /master/sparepart/{id}.
	//
	// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
	// persis seperti `Activity/UpdateSparepartHE_act` yang menetapkan `APPROVAL := "0"` tanpa
	// syarat apa pun. Keputusan komite menempuh Decide.
	//
	// Decide menangani POST /master/sparepart/keputusan.
	//
	// Ia padanan `Activity/SetApprovalAllMaster` beserta layarnya
	// `Section/ApprovalMasterSparepartHE-Section.xml`: daftar baris bercentang dan satu status,
	// ditetapkan sekaligus.
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
	h.Handler = masterhttp.New(masterhttp.Spec[mastersparepart.Sparepart, mastersparepart.Input, mastersparepart.ApprovalStatus, usecase.Actor, SaveRequest, DecisionRequest]{
		Module:           "mastersparepart/http",
		Service:          o.Service,
		Caller:           func(ctx context.Context) (string, bool) { c, ok := o.Caller(ctx); return c.Login, ok },
		Actor:            func(login string) usecase.Actor { return usecase.Actor{Login: login} },
		Logger:           o.Logger,
		WriteResponse:    o.WriteResponse,
		WriteError:       o.WriteError,
		WriteModuleError: h.writeModuleError,
		NotFound:         mastersparepart.ErrNotFound,
		DefaultStatus:    mastersparepart.StatusApproved,
		MaxBody:          maxRequestBody,
		Malformed: ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		},
		ToInput: func(request SaveRequest) mastersparepart.Input { return request.toInput() },
		ListBody: func(r *http.Request, list []mastersparepart.Sparepart, status mastersparepart.ApprovalStatus, alias string) any {
			return ListResponse{
				Sparepart: toListDTO(list, h.nameIndexOf(r, alias)),
				Status:    string(status),
				Portal:    alias,
			}
		},
		SingleBody: func(r *http.Request, row mastersparepart.Sparepart, alias string) any {
			return SingleResponse{
				Sparepart: toDTO(row, h.nameIndexOf(r, alias)),
				Portal:    alias,
			}
		},
		Decider:         o.Service,
		DecisionIDs:     func(request DecisionRequest) []string { return request.ID },
		DecisionStatus:  func(request DecisionRequest) string { return request.Status },
		MaxDecisionRows: maxDecisionRows,
		TooManyRows: mastersparepart.OneViolation("id_sparepart",
			"Terlalu banyak sparepart dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."),
		DecisionBody: func(changed int, status mastersparepart.ApprovalStatus, alias string) any {
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
