package masterdominanfactorhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim satu field pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan
// badan permintaan yang dikarang.
const maxSaveBodyBytes = 4 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string) ([]masterdominanfactor.DominantFactor, error)
	Get(ctx context.Context, portalAlias, id string) (masterdominanfactor.DominantFactor, error)
	Create(ctx context.Context, portalAlias, name string) (masterdominanfactor.DominantFactor, error)
	Update(ctx context.Context, portalAlias, id, name string) (masterdominanfactor.DominantFactor, error)
}

// Handler melayani permintaan Master Dominan Factor.
type Handler struct {
	service       Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service Service
	Logger  *slog.Logger

	// WriteResponse dan FallbackErrorWriter dipasok dari luar supaya seluruh modul
	// menuliskan respons dan galat portal dengan cara yang sama.
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Master Dominan Factor.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/master/dominan-factor.
//
// Menggantikan `RDB List/GetDataDominanFactor-SQL.xml` yang mengisi grid harness
// `DetailDominanFactor`.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.List(r.Context(), active.Alias)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	content := make([]DominantFactorDTO, 0, len(list))
	for _, f := range list {
		content = append(content, toDTO(f))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		DominantFactor: content,
		Total:          len(content),
		Portal:         active.Alias,
	})
}

// Get menangani GET /api/master/dominan-factor/{id}.
//
// Menggantikan `Activity/SetDominanFactor-Act.xml`, yang mengisi halaman `TempFactor`
// dari baris terpilih sebelum form Ubah dibuka.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	factor, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		DominantFactor: toDTO(factor),
		Portal:         active.Alias,
	})
}

// Create menangani POST /api/master/dominan-factor.
//
// Menggantikan tombol **Tambah** pada harness, yang menjalankan `DominanFactor_DT`
// untuk mengisi `TempFactor.Status = "Insert"` lalu memanggil `InsertDominanFactor`.
// Di sini penandanya tidak dibawa sama sekali — metode HTTP yang membedakannya.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	factor, err := h.service.Create(r.Context(), active.Alias, request.Nama)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan nomornya baru diketahui di sini.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		DominantFactor: toDTO(factor),
		Portal:         active.Alias,
	})
}

// Update menangani PUT /api/master/dominan-factor/{id}.
//
// Menggantikan tombol **Ubah**, yang menjalankan `DominanFactorUpdate_DT` untuk mengisi
// `TempFactor.Status = "Update"`. ID diambil dari jalur URL, tidak pernah dari badan
// permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	factor, err := h.service.Update(r.Context(), active.Alias, chi.URLParam(r, "id"), request.Nama)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		DominantFactor: toDTO(factor),
		Portal:         active.Alias,
	})
}

// readRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya
// sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan. Di modul ini ia tidak memuat
		// rahasia, tetapi memantulkan masukan mentah ke peramban adalah kebiasaan yang
		// tidak layak dimulai di satu tempat pun.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    ErrCodeBadRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}
	return request, true
}

func toDTO(f masterdominanfactor.DominantFactor) DominantFactorDTO {
	return DominantFactorDTO{ID: f.ID, Nama: f.Name}
}
