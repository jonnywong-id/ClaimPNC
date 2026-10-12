package registrasihttp

import "net/http"

// DiagnosisDTO adalah satu kode diagnosa.
type DiagnosisDTO struct {
	Code        string `json:"kode"`
	Description string `json:"deskripsi"`
}

// DiagnosisResponse adalah hasil pencarian kode diagnosa.
type DiagnosisResponse struct {
	Option []DiagnosisDTO `json:"pilihan"`
}

// SearchDiagnosis menangani GET /api/registrasi/diagnosa?cari=… — tombol Cari "Cari Kode / Desc
// Diagnose" modal "Transfer Claim ke Komite" (PA).
func (h *Handler) SearchDiagnosis(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	found, err := h.service.SearchDiagnosis(r.Context(), r.URL.Query().Get("cari"))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := make([]DiagnosisDTO, 0, len(found))
	for _, d := range found {
		body = append(body, DiagnosisDTO{Code: d.Code, Description: d.Description})
	}
	h.writeResponse(w, r, http.StatusOK, DiagnosisResponse{Option: body})
}
