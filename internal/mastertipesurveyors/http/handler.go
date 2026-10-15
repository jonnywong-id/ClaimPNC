package mastertipesurveyorshttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim satu isian pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const maxSaveBodyBytes = 4 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string) ([]mastertipesurveyors.SurveyorType, error)
	Get(ctx context.Context, portalAlias, code string) (mastertipesurveyors.SurveyorType, error)
	Create(ctx context.Context, portalAlias, description string) (mastertipesurveyors.SurveyorType, error)
	Update(ctx context.Context, portalAlias, code, description string) (mastertipesurveyors.SurveyorType, error)
}

// Handler melayani permintaan Master Tipe Surveyors.
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

	// WriteResponse dan WriteError dipasok dari luar supaya seluruh modul menuliskan
	// respons dan galat dengan cara yang sama. WriteError yang disuntikkan cmd sudah
	// dibungkus portalhttp.WithPortalError, sehingga galat portal terpetakan seragam.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Tipe Surveyors.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan
// pertama datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("mastertipesurveyors/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("mastertipesurveyors/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/master/tipe-surveyor.
//
// Menggantikan Report Definition `BrowseVMSurveyors_RD` yang mengisi grid layar
// `SurveyorsInbox`.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.List(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]SurveyorTypeDTO, 0, len(list))
	for _, t := range list {
		content = append(content, toDTO(t))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		SurveyorType: content,
		Total:        len(content),
		Portal:       active.Alias,
	})
}

// Get menangani GET /api/master/tipe-surveyor/{kode}.
//
// Menggantikan `SetSurveryorsValue_act(msurveyid)`, yang menjalankan
// `SelectVMSurveyors_RD` lalu menyalin hasilnya ke halaman `TempSurveyors` untuk diisi ke
// form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	surveyorType, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "kode"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		SurveyorType: toDTO(surveyorType),
		Portal:       active.Alias,
	})
}

// Create menangani POST /api/master/tipe-surveyor.
//
// Menggantikan tombol Tambah pada harness, yang mengirim sentinel `"UnknownID"` supaya
// procedure membentuk kodenya. Di sini kodenya tidak pernah ikut di badan permintaan sama
// sekali — sentinel itu tidak dibawa.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, request.Description)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk dan kodenya baru diketahui di sini.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		SurveyorType: toDTO(saved),
		Portal:       active.Alias,
	})
}

// Update menangani PUT /api/master/tipe-surveyor/{kode}.
//
// Menggantikan tombol Simpan pada baris yang sedang diubah. Kode diambil dari jalur URL,
// tidak pernah dari badan permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	code := chi.URLParam(r, "kode")
	if code == "" {
		h.writeModuleError(w, r, mastertipesurveyors.ErrNotFound)
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, code, request.Description)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		SurveyorType: toDTO(saved),
		Portal:       active.Alias,
	})
}

// readRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya sudah
// ditulis dan pemanggil harus berhenti.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest

	reader := http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah. Ia juga yang menolak klien yang mencoba mengirim `kode`.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan. Di modul ini ia tidak memuat
		// rahasia, tetapi memantulkan masukan mentah ke peramban adalah kebiasaan yang
		// tidak layak dimulai di satu tempat pun.
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

func toDTO(t mastertipesurveyors.SurveyorType) SurveyorTypeDTO {
	return SurveyorTypeDTO{
		Code:        t.Code,
		Description: t.Description,
		LegacyCode:  t.LegacyCode,
	}
}
