package masterpicteknikhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Form ini hanya mengirim beberapa isian pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const maxSaveBodyBytes = 8 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Ia dinyatakan sebagai antarmuka sempit di sisi PEMAKAI, bukan diimpor dari usecase,
// supaya handler dapat diuji tanpa membentuk seluruh layanan beserta penyimpanannya.
type Service interface {
	List(ctx context.Context, portalAlias string) ([]masterpicteknik.Technician, error)
	Get(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Technician, error)
	Lookup(ctx context.Context, portalAlias, operatorID string) (masterpicteknik.Employee, error)
	Create(ctx context.Context, portalAlias string, t masterpicteknik.Technician) (masterpicteknik.Technician, error)
	Update(ctx context.Context, portalAlias, operatorID string, t masterpicteknik.Technician) (masterpicteknik.Technician, error)
}

// Handler melayani permintaan Master PIC Teknik.
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

// NewHandler membentuk handler modul Master PIC Teknik.
//
// Ia menolak bahan yang tidak lengkap saat perakitan di cmd, bukan saat permintaan pertama
// datang: rakitan yang setengah jadi harus gagal saat start.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterpicteknik/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterpicteknik/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/master/pic-teknik.
//
// Menggantikan Report Definition `BrowseVMstUserTeknis_RD` yang mengisi grid layar
// `UserTeknisInbox`, termasuk penyaring `STS_AKTIF = '1'` yang dipatok di dalamnya.
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

	content := make([]TechnicianDTO, 0, len(list))
	for _, t := range list {
		content = append(content, toDTO(t))
	}
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Technician: content,
		Total:      len(content),
		Portal:     active.Alias,
	})
}

// Get menangani GET /api/master/pic-teknik/{id}.
//
// Menggantikan `SetMstUserTeknisValue_act`, yang menjalankan `GetMasterPICTeknis` lalu
// menyalin hasilnya ke halaman `TempDcol` untuk diisi ke form.
//
// Petugas NONAKTIF tetap dijawab di sini meski tidak muncul di daftar — sama seperti
// `GetMasterPICTeknis` yang tidak menyaring status aktif sama sekali.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	technician, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Technician: toDTO(technician),
		Portal:     active.Alias,
	})
}

// Lookup menangani GET /api/master/pic-teknik/direktori/{id}.
//
// Menggantikan `SetMstUserTeknisMstUser_act`, yang berjalan di layar begitu ID operator
// diisi dan mengisi `TempDcol.MCL_NAME`. Ia TIDAK menyimpan apa pun — hanya membaca
// direktori.
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	employee, err := h.service.Lookup(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, EmployeeResponse{
		Employee: EmployeeDTO{
			OperatorID:     employee.OperatorID,
			Name:           employee.Name,
			Email:          employee.Email,
			Supervisor:     employee.SupervisorID,
			SupervisorName: employee.SupervisorName,
		},
		Portal: active.Alias,
	})
}

// Create menangani POST /api/master/pic-teknik.
//
// Menggantikan `CNMInsertMstUserTeknis_act`, termasuk penolakan bila ID operatornya tidak
// ditemukan di direktori pegawai.
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

	saved, err := h.service.Create(r.Context(), active.Alias, fromRequest(request.OperatorID, request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	// 201, bukan 200: sumber daya baru terbentuk.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Technician: toDTO(saved),
		Portal:     active.Alias,
	})
}

// Update menangani PUT /api/master/pic-teknik/{id}.
//
// ID diambil dari jalur URL, tidak pernah dari badan permintaan.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpicteknik.ErrNotFound)
		return
	}

	request, read := h.readRequest(w, r)
	if !read {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, fromRequest(id, request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Technician: toDTO(saved),
		Portal:     active.Alias,
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
	// tanda bahwa ada yang salah. Ia juga yang menolak klien yang mencoba mengirim `nama`,
	// `grup_panel`, atau `beban_kerja` — tiga nilai yang tidak boleh datang dari luar.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dikembalikan: memantulkan masukan mentah ke
		// peramban adalah kebiasaan yang tidak layak dimulai di satu tempat pun.
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

// fromRequest menyusun bentuk domain dari isian form.
//
// Name, PanelGroup, dan Workload sengaja dibiarkan kosong: ketiganya tidak pernah datang
// dari klien. Name diisi usecase dari direktori; PanelGroup dan Workload dipertahankan
// dari baris yang sudah ada.
func fromRequest(operatorID string, request SaveRequest) masterpicteknik.Technician {
	return masterpicteknik.Technician{
		OperatorID:    operatorID,
		Email:         request.Email,
		BusinessLine:  request.BusinessLine,
		Group:         request.Group,
		Supervisor:    request.Supervisor,
		Quota:         request.Quota,
		ExternalQuota: request.ExternalQuota,
		Active:        request.Active,
	}
}

func toDTO(t masterpicteknik.Technician) TechnicianDTO {
	return TechnicianDTO{
		OperatorID:    t.OperatorID,
		Name:          t.Name,
		Email:         t.Email,
		BusinessLine:  t.BusinessLine,
		Group:         t.Group,
		Supervisor:    t.Supervisor,
		Quota:         t.Quota,
		ExternalQuota: t.ExternalQuota,
		Workload:      t.Workload,
		PanelGroup:    t.PanelGroup,
		Active:        t.Active,
	}
}
