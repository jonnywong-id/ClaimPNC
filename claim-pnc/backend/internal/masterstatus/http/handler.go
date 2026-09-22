package masterstatushttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterstatus"
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
	List(ctx context.Context, portalAlias string) ([]masterstatus.ClaimStatus, error)
	Get(ctx context.Context, portalAlias, code string) (masterstatus.ClaimStatus, error)
	Create(ctx context.Context, portalAlias, label string) (masterstatus.ClaimStatus, error)
	Update(ctx context.Context, portalAlias, code, label string) (masterstatus.ClaimStatus, error)
}

// Handler melayani permintaan Master Status Klaim.
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
	// menuliskan respons dan galat sesi dengan cara yang sama.
	WriteResponse       JSONWriter
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Master Status Klaim.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    WriteError(o.Logger, o.WriteResponse, o.FallbackErrorWriter),
	}
}

// List menangani GET /api/master/status-klaim.
//
// Menggantikan Report Definition BrowseVStsClaim_RD yang mengisi grid layar
// StatusClaimInbox.
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

	content := make([]ClaimStatusDTO, 0, len(list))
	for _, s := range list {
		content = append(content, toDTO(s))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		ClaimStatus: content,
		Total:       len(content),
		Portal:      active.Alias,
	})
}

// Ambil menangani GET /api/master/status-klaim/{kode}.
//
// Menggantikan SetStsClaimValue_act(lscid), yang menjalankan SelectVStsClaim_RD lalu
// menyalin hasilnya ke halaman TempStsClaim untuk diisi ke form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	status, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "kode"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		ClaimStatus: toDTO(status),
		Portal:      active.Alias,
	})
}

// Tambah menangani POST /api/master/status-klaim.
//
// Menggantikan tombol Tambah pada harness, yang mengirim sentinel "UnknownID" supaya
// procedure membentuk kodenya. Di sini kodenya tidak pernah ikut di badan permintaan
// sama sekali — sentinel itu tidak dibawa.
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

	status, err := h.service.Create(r.Context(), active.Alias, request.Label)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan kodenya baru diketahui di sini.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		ClaimStatus: toDTO(status),
		Portal:      active.Alias,
	})
}

// Ubah menangani PUT /api/master/status-klaim/{kode}.
//
// Menggantikan tombol Simpan pada baris yang sedang diubah. Kode diambil dari jalur URL,
// tidak pernah dari badan permintaan.
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

	status, err := h.service.Update(r.Context(), active.Alias, chi.URLParam(r, "kode"), request.Label)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		ClaimStatus: toDTO(status),
		Portal:      active.Alias,
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

func toDTO(s masterstatus.ClaimStatus) ClaimStatusDTO {
	return ClaimStatusDTO{Code: s.Code, Label: s.Label, LegacyCode: s.LegacyCode}
}
