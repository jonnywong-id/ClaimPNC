package inboxlaporanklaimhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Form Input Receive Document memuat dua isian bernarasi panjang — kronologis kejadian
// dan rincian kerusakan, masing-masing 4.000 karakter. 64 KiB sudah jauh lebih dari
// cukup, dan batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan
// memori, bukan sesudah.
const maxRequestBody = 64 << 10

// CallerLookup membaca identitas pemanggil dari konteks permintaan.
//
// Ia disuntikkan dari cmd, bukan diimpor dari modul auth: modul tidak saling mengimpor
// lapisan transport-nya — itulah yang membuat modul dapat dipindahkan tanpa menariknya
// serta. Pola yang sama dipakai modul Master Rekening dan modul menu.
type CallerLookup func(ctx context.Context) (inboxlaporanklaim.Caller, bool)

// Handler melayani permintaan Inbox Laporan Klaim.
type Handler struct {
	service       *usecase.Service
	caller        CallerLookup
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerLookup
	Logger  *slog.Logger

	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Inbox Laporan Klaim.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("inboxlaporanklaim/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("inboxlaporanklaim/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("inboxlaporanklaim/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /inbox/laporan-klaim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	result, err := h.service.List(r.Context(), active.Alias, caller, readQuery(r))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	now := h.service.Now()
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Laporan:  toListDTO(result.Page.Report, now),
		Kategori: toCategoryDTO(result.Summary),
		Halaman: PaginationDTO{
			Halaman:      result.Page.Pagination.Page,
			Ukuran:       result.Page.Pagination.Size,
			Total:        result.Page.Total,
			TotalHalaman: result.Page.TotalPages(),
		},
		BatasCabang: result.BranchScope,
		Portal:      active.Alias,
	})
}

// Options menangani GET /inbox/laporan-klaim/pilihan.
//
// Lencana pada daftar tab di sini sengaja KOSONG: menghitungnya menuntut penyaring yang
// baru dipilih pengguna setelah layar tergambar. Angkanya datang bersama daftar.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	region, err := h.service.Regions(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	category := make([]CategoryDTO, 0)
	for _, info := range inboxlaporanklaim.ListCategories() {
		category = append(category, CategoryDTO{
			Kode:       string(info.Category),
			Judul:      info.Title,
			Komunikasi: info.Message,
		})
	}

	business := make([]OptionDTO, 0)
	for _, line := range inboxlaporanklaim.ListBusinessLines() {
		business = append(business, OptionDTO{Kode: string(line.Line), Nama: line.Name})
	}

	area := make([]OptionDTO, 0, len(region))
	for _, item := range region {
		area = append(area, OptionDTO{Kode: item.Code, Nama: item.Name})
	}

	h.writeResponse(w, r, http.StatusOK, OptionResponse{
		Kategori: category,
		Bisnis:   business,
		Kanwil:   area,
		Portal:   active.Alias,
	})
}

// Get menangani GET /inbox/laporan-klaim/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrNotFound)
		return
	}

	report, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, singleResponse(report, active.Alias, h.service.Now()))
}

// Save menangani PUT /inbox/laporan-klaim/{id} — tombol Simpan pada form Input Receive
// Document.
//
// PUT, bukan PATCH: form mengirim SELURUH isian setiap kali disimpan, sehingga
// permintaannya menggantikan dan idempoten. Menekan Simpan dua kali menghasilkan keadaan
// yang sama persis.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrNotFound)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Save(r.Context(), active.Alias, id, caller, toDetail(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, singleResponse(saved, active.Alias, h.service.Now()))
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest

	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan MENGOSONGKAN isian yang sebenarnya
	// terisi — pada permintaan yang menggantikan seluruh isi, itu kehilangan data.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}

	return request, true
}

// singleResponse menyusun jawaban yang memuat satu berkas beserta isian formnya.
func singleResponse(
	report inboxlaporanklaim.ClaimReport,
	portalAlias string,
	now time.Time,
) SingleResponse {
	detail := toDetailDTO(inboxlaporanklaim.DetailOf(report))
	return SingleResponse{
		Laporan: toDTO(report, now),
		Isian:   &detail,
		// Berkas milik Pega hanya dapat dibaca dari sini selama masa paralel
		// (`ADR-0004`, `P-1`). Layar memakai penanda ini untuk menggambar formnya dalam
		// modus baca saja — bukan menyimpulkannya sendiri dari kolom `asal`.
		DapatDisunting: report.Origin == inboxlaporanklaim.OriginNew,
		Portal:         portalAlias,
	}
}

// Create menangani POST /inbox/laporan-klaim — tombol "Buat Baru".
//
// Badan permintaan TIDAK dibaca, dan itu bukan kelalaian: tombolnya di sistem lama tidak
// meminta satu pun isian. Lihat usecase.Service.Create.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	report, err := h.service.Create(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta nomornya. Nomor
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya — dan layar
	// LANGSUNG membuka form isiannya, persis seperti alur Pega yang meneruskan ke
	// assignment "Receive Document" begitu berkasnya dibuat.
	h.writeResponse(w, r, http.StatusCreated, singleResponse(report, active.Alias, h.service.Now()))
}

// prepare mengambil portal aktif dan identitas pemanggil sekaligus.
//
// Nilai ketiga false berarti responsnya sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) prepare(
	w http.ResponseWriter,
	r *http.Request,
) (portal.Portal, inboxlaporanklaim.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxlaporanklaim.Caller{}, false
	}

	caller, known := h.caller(r.Context())
	if !known {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrCallerUnknown)
		return portal.Portal{}, inboxlaporanklaim.Caller{}, false
	}
	return active, caller, true
}

// readQuery membaca penyaring dari parameter kueri.
//
// Nilai yang tidak dapat dibaca sebagai angka DIABAIKAN, bukan ditolak: halaman dan
// ukuran halaman adalah kendali tampilan, dan menolak seluruh permintaan karena satu
// parameter cacat hanya memindahkan kerepotan ke layar. Penyaring yang MENENTUKAN ISI —
// kategori dan lini bisnis — tetap ditolak bila tidak dikenal, dan penolakannya terjadi
// di lapisan usecase.
//
// Kategori yang tidak disebut sama sekali jatuh ke tab pertama layar, sama seperti
// membuka layarnya di Pega.
func readQuery(r *http.Request) usecase.Query {
	value := r.URL.Query()

	category := strings.TrimSpace(value.Get("kategori"))
	if category == "" {
		category = string(inboxlaporanklaim.CategoryOutstanding)
	}

	return usecase.Query{
		Category:     inboxlaporanklaim.Category(category),
		RegionCode:   value.Get("kanwil"),
		BusinessLine: inboxlaporanklaim.BusinessLine(strings.TrimSpace(value.Get("bisnis"))),
		Keyword:      value.Get("cari"),
		Pagination: inboxlaporanklaim.Pagination{
			Page: number(value.Get("halaman")),
			Size: number(value.Get("ukuran")),
		},
	}
}

func number(text string) int {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0
	}
	return n
}
