package masterspareparthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua puluh isian pendek dan tidak punya daftar anak sama
// sekali. 32 KiB sudah jauh lebih dari cukup, dan batasnya ada supaya permintaan bertubuh
// raksasa ditolak sebelum memakan memori.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya: `SetApprovalAllMaster` menelusuri seluruh baris bercentang
// berapa pun jumlahnya. Batas di sini bukan aturan bisnis melainkan penjaga sumber daya —
// setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan transaksi yang
// menahan ribuan kunci baris menghalangi Pega yang sedang melayani produksi pada tabel yang
// sama (D-21).
const maxDecisionRows = 200

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field saja, dan berbeda dari Master Panel ia BENAR-BENAR TERSIMPAN:
// `POOLDATA.SPAREPART_HE` punya kolom `USER_UPDATE`.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master sparepart.
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

// NewHandler membentuk handler modul master sparepart.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("mastersparepart/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("mastersparepart/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("mastersparepart/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

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
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	status := mastersparepart.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = mastersparepart.ApprovalStatus(raw)
	}

	list, err := h.service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Sparepart: toListDTO(list, h.nameIndexOf(r, active.Alias)),
		Status:    string(status),
		Portal:    active.Alias,
	})
}

// Get menangani GET /master/sparepart/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastersparepart.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Sparepart: toDTO(found, h.nameIndexOf(r, active.Alias)),
		Portal:    active.Alias,
	})
}

// Create menangani POST /master/sparepart.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam menulis USER_UPDATE yang kosong — dan pada
		// modul ini akibatnya nyata, karena kolom itu benar-benar tersimpan.
		h.writeError(w, r, errors.New(
			"mastersparepart/http: identitas pemanggil tidak ada di konteks"))
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

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID, APPROVAL,
	// USER_UPDATE, dan TGL_UPDATE_HARGA yang keempatnya diterbitkan server, sehingga layar
	// tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Sparepart: toDTO(saved, h.nameIndexOf(r, active.Alias)),
		Portal:    active.Alias,
	})
}

// Save menangani PUT /master/sparepart/{id}.
//
// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
// persis seperti `Activity/UpdateSparepartHE_act` yang menetapkan `APPROVAL := "0"` tanpa
// syarat apa pun. Keputusan komite menempuh Decide.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"mastersparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, mastersparepart.ErrNotFound)
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
		Sparepart: toDTO(saved, h.nameIndexOf(r, active.Alias)),
		Portal:    active.Alias,
	})
}

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
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"mastersparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, mastersparepart.OneViolation("id_sparepart",
			"Terlalu banyak sparepart dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := mastersparepart.ApprovalStatus(request.Status)
	changed, err := h.service.Decide(r.Context(), active.Alias, request.ID, status,
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

// Options menangani GET /master/sparepart/pilihan.
//
// Berbeda dari Master Panel yang daftar pilihannya konstanta di dalam kode, isi di sini
// DIBACA DARI BASIS DATA entitas yang sedang dibuka. Karena itu rutenya ikut dipasangi
// pemeriksaan portal — menyajikan daftar kategori satu entitas kepada entitas lain adalah
// kebocoran yang justru dicegah `R-20`.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	set, err := h.service.Options(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, toOptionsDTO(set.Category, set.Type, active.Alias))
}

// nameIndexOf menyusun pemetaan kode acuan ke namanya untuk satu permintaan.
//
// Kegagalannya DITELAN dengan sengaja, dan dicatat sebagai peringatan: nama kategori dan
// tipe adalah hiasan pada daftar yang isinya tetap terbaca tanpa keduanya. Lihat catatan
// pada List.
func (h *Handler) nameIndexOf(r *http.Request, portalAlias string) nameIndex {
	set, err := h.service.Options(r.Context(), portalAlias)
	if err != nil {
		logging.From(r.Context(), h.logger).Warn(
			"nama kategori dan tipe sparepart tidak dapat dibaca; daftar tetap disajikan dengan kodenya",
			slog.String("portal", portalAlias),
			slog.String("galat", err.Error()))
		return newNameIndex(nil, nil)
	}
	return newNameIndex(set.Category, set.Type)
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Pada modul ini akibatnya nyata — satu nama yang salah ketik
	// pada `harga_jual` akan ditolak sebagai "wajib diisi" dan pengguna mencari kesalahan di
	// tempat yang keliru.
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

// Mount mendaftarkan rute modul master sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Termasuk `/pilihan`, dan itu berbeda dari Master Panel. Alasannya: daftar pilihan modul
// ini DIBACA DARI BASIS DATA entitas — `GCNM_M_SPAREPART_CATEGORY` dan
// `GCNM_M_SPAREPART_TYPE` ada di basis data setiap entitas, dan isinya berbeda. Pada Master
// Panel daftarnya konstanta yang ditanam di dalam activity Pega, sehingga di sana pemeriksaan
// portal memang tidak melindungi apa pun.
//
// # Urutan pendaftaran
//
// Kedua rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui
// hal itu untuk yakin bahwa `/keputusan` tidak pernah terbaca sebagai sebuah ID sparepart.
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
// Tidak ada DELETE terhadap sparepart. Sistem lama tidak punya satu pun terhadap tabel ini,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis — sparepart yang tidak lagi
// dipakai ditolak atau ditandai lewat STS_AKTIF, bukan dibuang.
//
// Tidak ada pula endpoint unggah. Kedua tombolnya di layar lama — "Upload Document" dan
// "Upload Data Master Sparepart" — memanggil local action yang TIDAK ADA di export (`R-16`),
// sehingga susunan kolom CSV-nya, validasinya, dan apakah baris hasil unggah masuk antrean
// persetujuan seluruhnya tidak diketahui. Tombolnya tetap digambar dalam keadaan mati; lihat
// SparepartPage.tsx.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/sparepart/pilihan", h.Options)
		perPortal.Post("/master/sparepart/keputusan", h.Decide)

		perPortal.Get("/master/sparepart", h.List)
		perPortal.Post("/master/sparepart", h.Create)
		perPortal.Get("/master/sparepart/{id}", h.Get)
		perPortal.Put("/master/sparepart/{id}", h.Save)
	})
}
