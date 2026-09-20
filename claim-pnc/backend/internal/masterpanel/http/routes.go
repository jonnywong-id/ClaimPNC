package masterpanelhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat sepuluh isian pendek ditambah daftar lokasi yang dibatasi
// masterpanel.MaxLocationRows baris — dan setiap barisnya hanya dua nilai pendek. 64 KiB
// sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh raksasa ditolak
// sebelum memakan memori.
const maxRequestBody = 64 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya: `SetApprovalAllMaster` menelusuri seluruh baris
// bercentang berapa pun jumlahnya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya — setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ribuan kunci baris menghalangi Pega yang sedang melayani produksi
// pada tabel yang sama (D-21).
const maxDecisionRows = 200

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja. Ia TIDAK pernah tersimpan ke basis data — `POOLDATA.PANEL_HE` tidak
// punya kolom pencatat pelaku maupun waktu — dan hanya dipakai untuk log.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master panel.
type Handler struct {
	service       *usecase.Service
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerReader
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul master panel.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterpanel/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterpanel/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterpanel/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

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
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	status := masterpanel.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = masterpanel.ApprovalStatus(raw)
	}

	list, err := h.service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Panel:  toListDTO(list),
		Status: string(status),
		Portal: active.Alias,
	})
}

// Get menangani GET /master/panel/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpanel.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Panel:  toDTO(found),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/panel.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam menulis master tanpa identitas pemanggil
		// di log.
		h.writeError(w, r, errors.New("masterpanel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID_PANEL dan
	// APPROVAL yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Panel:  toDTO(saved),
		Portal: active.Alias,
	})
}

// Save menangani PUT /master/panel/{id}.
//
// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
// persis seperti `Activity/CNMUpdatePanelHE_act` yang menetapkan `APPROVAL := "0"` tanpa
// syarat apa pun. Keputusan komite menempuh Decide.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterpanel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpanel.ErrNotFound)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Save(r.Context(), active.Alias, id, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Panel:  toDTO(saved),
		Portal: active.Alias,
	})
}

// Decide menangani POST /master/panel/keputusan.
//
// Ia padanan `Activity/SetApprovalAllMaster` beserta layarnya
// `Section/ApprovalMasterPanelHE-Section.xml`: daftar baris bercentang, satu status, dan
// satu catatan, ditetapkan sekaligus.
//
// # Kenapa POST ke sub-sumber daya, bukan PATCH pada tiap baris
//
// Karena yang terjadi adalah SATU peristiwa bisnis — sebuah keputusan atas sekumpulan
// pengajuan — dan `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai
// peristiwa, bukan sebagai pembaruan field. Memecahnya menjadi sederet PATCH juga akan
// membuat keputusan yang di Pega utuh menjadi sesuatu yang dapat gagal separuh jalan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New("masterpanel/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, masterpanel.OneViolation("id_panel",
			"Terlalu banyak panel dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := masterpanel.ApprovalStatus(request.Status)
	changed, err := h.service.Decide(r.Context(), active.Alias, request.ID, status, request.Reason,
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, DecisionResponse{
		Changed:     changed,
		Status:      string(status),
		StatusLabel: status.Label(),
		Portal:      active.Alias,
	})
}

// Options menangani GET /master/panel/pilihan.
//
// Isinya KONSTANTA — kelima nama lokasi dan ketiga sandi sisi ditanam di
// `Activity/SetLokasiSisiPanel-Act.xml`, bukan dibaca dari tabel acuan mana pun.
//
// Ia tetap disajikan lewat endpoint supaya layar tidak memuat salinan ketiganya:
// satu-satunya daftar pilihan modul ini yang artinya benar-benar diketahui tidak boleh
// hidup di dua tempat. Karena isinya tidak menyentuh basis data, ia TIDAK menuntut portal
// — lihat Mount.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(w, r, http.StatusOK, optionsDTO())
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Pada modul ini akibatnya lebih jauh daripada biasanya —
	// sepuluh isian induknya WAJIB, sehingga satu nama yang salah ketik akan ditolak
	// sebagai "wajib diisi" dan pengguna mencari kesalahan di tempat yang keliru.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	return true
}

// Mount mendaftarkan rute modul master panel.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Yang dipasangi pemeriksaan portal, dan yang tidak
//
// SELURUH rute data dipasangi PortalAktif: POOLDATA.PANEL_HE dan LOKASI_PANEL_HE ada di
// basis data setiap entitas, dan menyajikan daftar satu entitas kepada entitas lain adalah
// kebocoran yang justru dicegah R-20.
//
// `/pilihan` TIDAK, dan itu bukan kelalaian: isinya konstanta yang ditanam di dalam
// activity Pega, bukan bacaan basis data mana pun. Memasangi pemeriksaan portal padanya
// akan membuat form tidak dapat menggambar dropdown-nya sebelum portal dipilih —
// penolakan yang tidak melindungi apa pun. Perlakuannya sama dengan daftar Kategori pada
// Master Pasal Kerugian.
//
// # Urutan pendaftaran
//
// Kedua rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}`
// pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu
// mengetahui hal itu untuk yakin bahwa `/keputusan` tidak pernah terbaca sebagai sebuah ID
// panel.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan
//
// Tidak ada DELETE terhadap panel. Sistem lama tidak punya satu pun terhadap tabel induk,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis — panel yang tidak lagi
// dipakai ditolak atau ditandai lewat STS_AKTIF, bukan dibuang.
//
// Baris LOKASI memang dibuang saat penyimpanan, dan itu pertentangan dengan `D-66` yang
// dinyatakan terbuka di banner masterpanel.sql — bukan disembunyikan di balik ketiadaan
// endpoint DELETE.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Get("/master/panel/pilihan", h.Options)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Post("/master/panel/keputusan", h.Decide)

		perPortal.Get("/master/panel", h.List)
		perPortal.Post("/master/panel", h.Create)
		perPortal.Get("/master/panel/{id}", h.Get)
		perPortal.Put("/master/panel/{id}", h.Save)
	})
}
