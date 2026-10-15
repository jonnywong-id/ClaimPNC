package detailpenyebabhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/detailpenyebab/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat lima isian pendek ditambah daftar lini bisnis yang panjangnya
// tidak dibatasi apa pun. 32 KiB sudah jauh lebih dari cukup — angkanya disamakan dengan
// modul master lain alih-alih diperbesar, supaya tidak ada satu modul yang diam-diam
// menerima permintaan yang ditolak modul tetangganya.
const maxRequestBody = 32 << 10

// maxKeywordLength membatasi panjang kata kunci pencarian.
//
// Kata kunci yang sangat panjang tidak pernah berasal dari layar — isiannya jauh lebih
// pendek — dan menolaknya lebih awal menghindarkan basis data dari pola LIKE raksasa yang
// pasti tidak cocok dengan apa pun.
const maxKeywordLength = 100

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan ia dipakai HANYA untuk mengisi log — tabelnya tidak punya kolom pencatat
// pelaku sama sekali. Berbeda dari Master Login, identitas pemanggil di modul ini TIDAK
// menjadi data yang tersimpan; lihat usecase.Actor.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan Detail Penyebab Kerugian.
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

// NewHandler membentuk handler modul Detail Penyebab Kerugian.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("detailpenyebab/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("detailpenyebab/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"detailpenyebab/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/detail-penyebab.
//
// # Ketiga penyaingnya
//
//	?cari=...       mempersempit pada Deskripsi Kerugian, Kode Kehilangan, dan ID
//	?id_master=...  mempersempit pada satu induk
//	?bisnis=...     mempersempit pada satu lini bisnis
//
// Kedua penyaring terakhir adalah padanan panel "Cari Data" di layar lama; yang pertama
// lebih luas daripada panel itu, dan alasannya ada pada detailpenyebab.Filter.
//
// Cakupan daftarnya SELURUH baris, TANPA menyaring yang tidak aktif. Rule yang mengisi grid
// Pega tidak ada di export (`R-16`), dan alasan memilih bacaan ini ada pada banner paket
// detailpenyebab.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	query := r.URL.Query()
	if !h.keywordWithinLimit(w, r, query.Get("cari")) {
		return
	}

	list, err := h.service.List(r.Context(), active.Alias, detailpenyebab.Filter{
		Keyword:    query.Get("cari"),
		MasterID:   query.Get("id_master"),
		BusinessID: query.Get("bisnis"),
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Detail: toListDTO(list),
		Portal: active.Alias,
	})
}

// Get menangani GET /master/detail-penyebab/{id}.
//
// Jalurnya memakai D_COL_ID — kunci barisnya, dan satu-satunya yang membedakan dua baris.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, detailpenyebab.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Detail: toDTO(found),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/detail-penyebab.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras alih-alih menghasilkan log tanpa pelaku.
		h.writeError(w, r, errors.New(
			"detailpenyebab/http: identitas pemanggil tidak ada di konteks"))
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

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID dan sebutan
	// induknya, yang keduanya diturunkan server sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Detail: toDTO(saved),
		Portal: active.Alias,
	})
}

// Save menangani PUT /master/detail-penyebab/{id}.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"detailpenyebab/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, detailpenyebab.ErrNotFound)
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
		Detail: toDTO(saved),
		Portal: active.Alias,
	})
}

// SearchMaster menangani GET /master/detail-penyebab/pilihan/master.
func (h *Handler) SearchMaster(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	keyword := r.URL.Query().Get("cari")
	if !h.keywordWithinLimit(w, r, keyword) {
		return
	}

	list, err := h.service.SearchMaster(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, LookupResponse{
		Master: toMasterDTO(list),
		Bisnis: []BusinessDTO{},
		Portal: active.Alias,
	})
}

// SearchBusiness menangani GET /master/detail-penyebab/pilihan/bisnis.
func (h *Handler) SearchBusiness(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	keyword := r.URL.Query().Get("cari")
	if !h.keywordWithinLimit(w, r, keyword) {
		return
	}

	list, err := h.service.SearchBusiness(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, LookupResponse{
		Master: []MasterOptionDTO{},
		Bisnis: toBusinessDTO(list),
		Portal: active.Alias,
	})
}

// Options menangani GET /master/detail-penyebab/pilihan.
//
// Ia TIDAK dipasangi pemeriksaan portal, dan itu satu-satunya rute di modul ini yang
// demikian: isinya milik aplikasi, bukan milik entitas. Kedua pilihan Status Aktif sama di
// keempat portal — nilainya konstanta domain, bukan baris basis data.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	list := detailpenyebab.ActiveOptions()
	options := make([]ActiveOptionDTO, 0, len(list))
	for _, one := range list {
		options = append(options, ActiveOptionDTO{Kode: one.Code, Label: one.Label})
	}
	h.writeResponse(w, r, http.StatusOK, OptionsResponse{StatusAktif: options})
}

// keywordWithinLimit menolak kata kunci yang terlalu panjang. False berarti responsnya
// sudah ditulis.
func (h *Handler) keywordWithinLimit(
	w http.ResponseWriter,
	r *http.Request,
	keyword string,
) bool {
	if len([]rune(keyword)) <= maxKeywordLength {
		return true
	}
	h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Kata pencarian terlalu panjang.",
	})
	return false
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam. Pada modul ini itu lebih
	// dari sekadar menangkap salah ketik: `id`, `nama_master`, dan `label_status_aktif`
	// DIKIRIM pada setiap jawaban tetapi tidak dapat dikirim balik, dan klien yang
	// mengembalikan seluruh objek apa adanya harus mengetahuinya saat pertama dicoba —
	// bukan menemukan bahwa ketiganya diam-diam tidak berpengaruh.
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

// Mount mendaftarkan rute modul Detail Penyebab Kerugian.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal, KECUALI satu
//
// `POOLDATA.D_CAUSE_OF_LOSS` ada di basis data SETIAP entitas dan isinya berbeda — ia
// menentukan sebab kerugian apa saja yang dapat dipilih pada badan hukum itu. Satu-satunya
// yang dikecualikan adalah `/pilihan`, yang isinya konstanta domain; lihat Handler.Options.
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
// **Tidak ada DELETE.** Layar lamanya tidak punya tombolnya, `pyDeleteSQL` pada rule
// simpannya kosong, dan tidak ada satu pun activity penghapus di export. `D-66` pun
// melarang penghapusan fisik data bernilai bisnis. Baris yang tidak lagi dipakai
// dinyatakan lewat **Status Aktif** — lihat doc comment detailpenyebab.Repo.
//
// # Urutan pendaftaran MENENTUKAN di sini
//
// `/pilihan` dan `/pilihan/...` didaftarkan SEBELUM `/{id}`. chi mencocokkan segmen statis
// lebih dulu sehingga urutannya sebenarnya aman, tetapi menuliskannya berurutan membuat
// pembaca tidak perlu mengetahui hal itu — dan melindungi dari perubahan router kelak yang
// tidak lagi demikian.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	// Pilihan yang isinya milik aplikasi — tanpa pemeriksaan portal.
	r.Get("/master/detail-penyebab/pilihan", h.Options)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/detail-penyebab/pilihan/master", h.SearchMaster)
		perPortal.Get("/master/detail-penyebab/pilihan/bisnis", h.SearchBusiness)

		perPortal.Get("/master/detail-penyebab", h.List)
		perPortal.Post("/master/detail-penyebab", h.Create)
		perPortal.Get("/master/detail-penyebab/{id}", h.Get)
		perPortal.Put("/master/detail-penyebab/{id}", h.Save)
	})
}
