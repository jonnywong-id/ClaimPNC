package daftartipedokumenbisnishttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/daftartipedokumenbisnis/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan.
//
// Ia penjaga sumber daya, BUKAN validasi: tanpa batas, satu permintaan bertubuh besar
// dapat menghabiskan memori proses. Batasnya dilonggarkan dari modul lain karena
// penambahan di layar ini mengirim perkalian bisnis kali dokumen dalam satu badan —
// puluhan bisnis dikali puluhan baris masih harus muat.
const maxRequestBody = 256 << 10

// Caller adalah identitas pemanggil, sejauh yang dibutuhkan modul ini.
//
// Tipe milik modul, bukan impor dari modul auth: modul tidak saling mengimpor lapisan
// transport-nya. Yang menjembatani keduanya adalah cmd.
type Caller struct {
	Identity string

	// Position adalah pyPosition pemanggil.
	//
	// Dipakai HANYA untuk memutuskan apakah tombol "Pilih semua" ditampilkan, meniru
	// `pyVisible` layar lama. Ia bukan kewenangan — lihat Service.MayBulkSelect.
	Position string
}

// Handler melayani rute modul Daftar Tipe Dokumen Bisnis.
type Handler struct {
	service       *usecase.Service
	caller        func(context.Context) (Caller, bool)
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah ketergantungan Handler.
type Options struct {
	Service       *usecase.Service
	Logger        *slog.Logger
	Caller        func(context.Context) (Caller, bool)
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk Handler dan menolak ketergantungan yang belum diisi.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("daftartipedokumenbisnis/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("daftartipedokumenbisnis/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// ListBusinesses menangani GET /master/tipe-dokumen-bisnis — grid tingkat pertama.
func (h *Handler) ListBusinesses(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.ListBusinesses(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := toBusinessListDTO(list)
	h.writeResponse(w, r, http.StatusOK, BusinessListResponse{
		Business: content,
		Total:    len(content),
		Portal:   active.Alias,
	})
}

// ListByBusiness menangani GET /master/tipe-dokumen-bisnis/bisnis/{businessID} — grid
// tingkat kedua.
func (h *Handler) ListByBusiness(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	businessID := chi.URLParam(r, "businessID")
	list, err := h.service.ListByBusiness(r.Context(), active.Alias, businessID)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// Daftar KOSONG bukan galat. Lini bisnis yang belum punya aturan memang belum punya
	// baris, dan layar Tambah justru menuju ke sana — menjawab 404 akan menutup jalan
	// menuju satu-satunya layar yang dapat memperbaikinya.
	content := toRuleListDTO(list)
	h.writeResponse(w, r, http.StatusOK, RuleListResponse{
		Rules:  content,
		Total:  len(content),
		Portal: active.Alias,
	})
}

// Get menangani GET /master/tipe-dokumen-bisnis/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	rule, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{Rule: toDTO(rule), Portal: active.Alias})
}

// Create menangani POST /master/tipe-dokumen-bisnis.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, toBatchInput(request), h.identity(r))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := toRuleListDTO(saved)
	h.writeResponse(w, r, http.StatusCreated, CreateResponse{
		Rules:  content,
		Total:  len(content),
		Portal: active.Alias,
	})
}

// Update menangani PUT /master/tipe-dokumen-bisnis/{id}.
//
// PUT, bukan PATCH: badan permintaan memuat SELURUH isian baris, dan mengirimnya dua kali
// menghasilkan keadaan yang sama.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, daftartipedokumenbisnis.ErrNotFound)
		return
	}

	var request UpdateRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, toInput(request.RuleRequest), h.identity(r))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{Rule: toDTO(saved), Portal: active.Alias})
}

// AddCoverage menangani POST /master/tipe-dokumen-bisnis/{id}/jenis-klaim.
//
// Sub-sumber daya, bukan field pada badan Update, karena penambahannya memang operasi
// tersendiri di sistem lama — `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc` melayaninya lewat
// cabang yang berbeda, dan tidak ada jalur yang mengganti seluruh daftarnya sekaligus.
func (h *Handler) AddCoverage(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	var request CoverageRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.AddCoverage(r.Context(), active.Alias, chi.URLParam(r, "id"), request.CoverageID)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{Rule: toDTO(saved), Portal: active.Alias})
}

// BusinessChoices menangani GET /master/bisnis-pilihan.
func (h *Handler) BusinessChoices(w http.ResponseWriter, r *http.Request) {
	h.serveBusinessChoices(w, r)
}

// DocumentTypeChoices menangani GET /master/tipe-dokumen-pilihan.
func (h *Handler) DocumentTypeChoices(w http.ResponseWriter, r *http.Request) {
	h.serveReferences(w, r, h.service.ListDocumentTypes)
}

// DetailTypeDocChoices menangani GET /master/detail-dokumen-pilihan.
func (h *Handler) DetailTypeDocChoices(w http.ResponseWriter, r *http.Request) {
	h.serveReferences(w, r, h.service.ListDetailTypeDocs)
}

// ObjectDocChoices menangani GET /master/objek-dokumen-pilihan.
func (h *Handler) ObjectDocChoices(w http.ResponseWriter, r *http.Request) {
	h.serveReferences(w, r, h.service.ListObjectDocs)
}

// serveReferences melayani ketiga rute daftar pilihan yang bentuknya sama.
//
// Satu fungsi untuk ketiganya, bukan tiga salinan: ketiganya benar-benar berbeda hanya
// pada satu pemanggilan, dan tiga salinan berarti satu perbaikan harus diingat tiga kali.
func (h *Handler) serveReferences(
	w http.ResponseWriter,
	r *http.Request,
	list func(context.Context, string) ([]daftartipedokumenbisnis.Reference, error),
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	items, err := list(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := toReferenceListDTO(items)
	h.writeResponse(w, r, http.StatusOK, ReferenceListResponse{
		Items:  content,
		Total:  len(content),
		Portal: active.Alias,
	})
}

// serveBusinessChoices melayani daftar pilihan bisnis.
//
// Terpisah dari serveReferences karena bentuk jawabannya berbeda: bisnis memakai
// BusinessDTO dengan field `nama_bisnis`, sama dengan yang dipakai grid tingkat pertama,
// supaya layar tidak perlu dua bentuk untuk hal yang sama.
func (h *Handler) serveBusinessChoices(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.ListBusinessChoices(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := toBusinessListDTO(list)
	h.writeResponse(w, r, http.StatusOK, BusinessListResponse{
		Business:      content,
		Total:         len(content),
		Portal:        active.Alias,
		MayBulkSelect: h.service.MayBulkSelect(h.position(r)),
	})
}

// position mengembalikan pyPosition pemanggil, atau teks kosong bila tidak ada.
//
// Kosong berarti tombol "Pilih semua" tidak ditampilkan — sama seperti operator mana pun
// yang pyPosition-nya bukan NONMBU. Itu perilaku yang benar: tanpa identitas, tidak ada
// dasar untuk menampilkannya.
func (h *Handler) position(r *http.Request) string {
	if h.caller == nil {
		return ""
	}
	caller, exists := h.caller(r.Context())
	if !exists {
		return ""
	}
	return caller.Position
}

// identity mengembalikan identitas pemanggil, atau teks kosong bila tidak ada.
//
// Kosong TIDAK menghentikan penyimpanan — ia hanya mengisi kolom jejak. Lihat komentar
// Editor di lapisan domain.
func (h *Handler) identity(r *http.Request) string {
	if h.caller == nil {
		return ""
	}
	caller, exists := h.caller(r.Context())
	if !exists {
		return ""
	}
	return caller.Identity
}

// readRequest membaca badan permintaan JSON ke dalam target.
//
// Tiga penjaga, masing-masing menutup satu kelas kesalahan:
//   - MaxBytesReader membatasi besar badan
//   - DisallowUnknownFields menolak field yang tidak dikenal, sehingga salah ketik nama
//     field terbaca sebagai galat alih-alih diam-diam terabaikan — dan `id` maupun
//     `user_edit` tidak dapat diselundupkan
//   - dekode kedua memastikan tidak ada dokumen JSON kedua menempel di belakang
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}
	return true
}
