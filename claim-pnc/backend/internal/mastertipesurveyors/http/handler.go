package mastertipesurveyorshttp

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/platform/httpjson"
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

	surveyorType, err := h.Service.Get(r.Context(), active.Alias, chi.URLParam(r, "kode"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
		SurveyorType: toDTO(surveyorType),
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

	saved, err := h.Service.Update(r.Context(), active.Alias, code, request.Description)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
		SurveyorType: toDTO(saved),
		Portal:       active.Alias,
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

func toDTO(t mastertipesurveyors.SurveyorType) SurveyorTypeDTO {
	return SurveyorTypeDTO{
		Code:        t.Code,
		Description: t.Description,
		LegacyCode:  t.LegacyCode,
	}
}
