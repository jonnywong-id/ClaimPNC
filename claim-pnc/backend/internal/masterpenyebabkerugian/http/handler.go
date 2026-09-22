package masterpenyebabkerugianhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim satu field pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const maxSaveBodyBytes = 4 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string) ([]masterpenyebabkerugian.CauseOfLoss, error)
	Get(ctx context.Context, portalAlias, id string) (masterpenyebabkerugian.CauseOfLoss, error)
	Create(ctx context.Context, portalAlias, description string) (masterpenyebabkerugian.CauseOfLoss, error)
	Update(ctx context.Context, portalAlias, id, description string) (masterpenyebabkerugian.CauseOfLoss, error)
}

// Handler melayani permintaan Master Penyebab Kerugian.
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

// NewHandler membentuk handler modul Master Penyebab Kerugian.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/master/penyebab-kerugian.
//
// Menggantikan `Report Definition/BrowseVMCauseOfLoss_RD` yang mengisi grid pada harness
// `CauseOfLossInbox`.
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

	content := make([]CauseOfLossDTO, 0, len(list))
	for _, c := range list {
		content = append(content, toDTO(c))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		PenyebabKerugian: content,
		Total:            len(content),
		Portal:           active.Alias,
	})
}

// Get menangani GET /api/master/penyebab-kerugian/{id}.
//
// Menggantikan pengisian halaman `TempCauseOfLoss` dari baris terpilih sebelum form Ubah
// dibuka.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	cause, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		PenyebabKerugian: toDTO(cause),
		Portal:           active.Alias,
	})
}

// Create menangani POST /api/master/penyebab-kerugian.
//
// Menggantikan tombol **Tambah** pada harness, yang menjalankan `CNMInsertCauseOfLoss_act`
// dengan `TempCauseOfLoss.M_COL_ID = "UnknownID"` sebagai penanda baris baru. Di sini
// penandanya tidak dibawa sama sekali — metode HTTP yang membedakannya.
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

	cause, err := h.service.Create(r.Context(), active.Alias, request.Deskripsi)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan nomornya baru diketahui di sini.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		PenyebabKerugian: toDTO(cause),
		Portal:           active.Alias,
	})
}

// Update menangani PUT /api/master/penyebab-kerugian/{id}.
//
// Menggantikan jalur yang sama dengan `M_COL_ID` terisi. ID diambil dari jalur URL, tidak
// pernah dari badan permintaan.
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

	cause, err := h.service.Update(r.Context(), active.Alias, chi.URLParam(r, "id"), request.Deskripsi)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		PenyebabKerugian: toDTO(cause),
		Portal:           active.Alias,
	})
}

// readRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya sudah
// ditulis dan pemanggil harus berhenti.
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

func toDTO(c masterpenyebabkerugian.CauseOfLoss) CauseOfLossDTO {
	return CauseOfLossDTO{
		ID:        c.ID,
		Deskripsi: c.Description,
		IDLama:    c.LegacyID,
	}
}
