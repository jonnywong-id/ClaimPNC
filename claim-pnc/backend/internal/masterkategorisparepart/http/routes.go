package masterkategorispareparthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/masterkategorisparepart/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat SATU isian pendek pada jalur simpan, dan satu daftar kunci
// pada jalur keputusan. 32 KiB sudah jauh lebih dari cukup — angkanya disamakan dengan
// modul master lain alih-alih diperkecil, supaya tidak ada satu modul yang diam-diam
// menolak permintaan yang diterima modul tetangganya.
const maxRequestBody = 32 << 10

// maxDecisionRows membatasi banyaknya baris pada satu keputusan borongan.
//
// Sistem lama tidak membatasinya. Batas di sini bukan aturan bisnis melainkan penjaga
// sumber daya: setiap baris menjadi satu pernyataan UPDATE di dalam satu transaksi, dan
// transaksi yang menahan ratusan kunci baris menghalangi Pega yang sedang melayani produksi
// pada tabel yang sama (D-21).
//
// Dua ratus, sama dengan modul master lain. Pada tabel penggolongan yang isinya berorde
// puluhan sampai ratusan baris, batas ini praktis tidak akan tersentuh.
const maxDecisionRows = 200

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan berbeda dari Master Sparepart ia **TIDAK TERSIMPAN**:
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak punya kolom pencatat pelaku sama sekali. Ia
// tetap diminta karena dipakai LOG — lihat usecase.Actor.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master kategori sparepart.
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

// NewHandler membentuk handler modul master kategori sparepart.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterkategorisparepart/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterkategorisparepart/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"masterkategorisparepart/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

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
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	status := masterkategorisparepart.StatusApproved
	if raw := r.URL.Query().Get("status"); raw != "" {
		status = masterkategorisparepart.ApprovalStatus(raw)
	}

	list, err := h.service.List(r.Context(), active.Alias, status, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Category: toListDTO(list),
		Status:   string(status),
		Portal:   active.Alias,
	})
}

// Get menangani GET /master/kategori-sparepart/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterkategorisparepart.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Category: toDTO(found),
		Portal:   active.Alias,
	})
}

// Create menangani POST /master/kategori-sparepart.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras, bukan diam-diam mencatat log tanpa pelaku — dan pada modul
		// ini log adalah SATU-SATUNYA tempat pelaku terekam, karena tabelnya tidak punya
		// kolom untuk itu.
		h.writeError(w, r, errors.New(
			"masterkategorisparepart/http: identitas pemanggil tidak ada di konteks"))
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

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID dan APPROVAL
	// yang keduanya diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Category: toDTO(saved),
		Portal:   active.Alias,
	})
}

// Save menangani PUT /master/kategori-sparepart/{id}.
//
// Ia TIDAK menerima status: menyimpan selalu mengembalikan baris ke antrean persetujuan,
// persis seperti `Activity/UpdateKategoriSparepart_act2` yang menetapkan APPROVAL := "0"
// tanpa syarat apa pun. Keputusan persetujuan menempuh Decide.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"masterkategorisparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterkategorisparepart.ErrNotFound)
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
		Category: toDTO(saved),
		Portal:   active.Alias,
	})
}

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
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"masterkategorisparepart/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request DecisionRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	if len(request.ID) > maxDecisionRows {
		h.writeModuleError(w, r, masterkategorisparepart.OneViolation("id_kategori_sparepart",
			"Terlalu banyak kategori dipilih sekaligus. Putuskan paling banyak 200 dalam satu kali."))
		return
	}

	status := masterkategorisparepart.ApprovalStatus(request.Status)
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

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Pada modul berisi SATU isian, akibatnya paling terang —
	// seluruh badan permintaan menjadi tidak berarti apa-apa.
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

// Mount mendaftarkan rute modul master kategori sparepart.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` ada di basis data SETIAP entitas dan isinya berbeda.
// Tidak ada satu pun rute di modul ini yang isinya milik aplikasi, sehingga tidak ada yang
// dikecualikan — berbeda dari Master Panel, yang daftar pilihannya konstanta yang ditanam
// di activity Pega.
//
// # Urutan pendaftaran
//
// Rute bersegmen statis didaftarkan LEBIH DULU daripada rute ber-parameter `{id}` pada
// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya
// tidak menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui
// hal itu untuk yakin bahwa `/keputusan` tidak pernah terbaca sebagai sebuah ID kategori.
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
// **Tidak ada DELETE.** Sistem lama tidak punya satu pun terhadap tabel ini — kesembilan
// rule yang menyentuhnya tidak memuat satu pun pernyataan hapus — dan `D-66` melarang
// penghapusan fisik data bernilai bisnis. Kategori yang tidak lagi dipakai DITOLAK, bukan
// dibuang; dengan begitu sparepart lama yang sudah menunjuknya tetap dapat menampilkan
// namanya.
//
// **Tidak ada endpoint `/pilihan`.** Modul ini tidak punya tabel acuan: ia SENDIRI yang
// menjadi acuan bagi Master Sparepart dan Master Tipe Sparepart. Keduanya membacanya lewat
// endpoint mereka masing-masing, bukan lewat modul ini.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Post("/master/kategori-sparepart/keputusan", h.Decide)

		perPortal.Get("/master/kategori-sparepart", h.List)
		perPortal.Post("/master/kategori-sparepart", h.Create)
		perPortal.Get("/master/kategori-sparepart/{id}", h.Get)
		perPortal.Put("/master/kategori-sparepart/{id}", h.Save)
	})
}
