package masterpicteknikhttp

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/platform/httpjson"
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

	technician, err := h.Service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
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

	employee, err := h.Service.Lookup(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, EmployeeResponse{
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

// readRequest membaca badan permintaan simpan. Nilai kedua false berarti jawabannya sudah
// ditulis dan pemanggil harus berhenti.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	ok := httpjson.Decode(w, r, maxSaveBodyBytes, &request, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
	return request, ok
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
