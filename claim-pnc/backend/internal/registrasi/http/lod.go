package registrasihttp

import (
	"net/http"
	"strconv"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// LODRequest adalah badan dialog Print LOD. Indeks berbasis 1; tipe_pdf adalah
// `.IDAdjustClaim` jenis LOD yang dipilih.
type LODRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`
	Type       string `json:"tipe_pdf"`
}

// LODTypeDTO adalah satu pilihan Tipe PDF.
type LODTypeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"nama"`
	Ready bool   `json:"tersedia"`
}

// LODTypesResponse adalah isi awal dialog Print LOD: pilihan Tipe PDF dan isian section
// PrintLODdanEmail (Email LOD, Nama Tertanggung).
type LODTypesResponse struct {
	Types       []LODTypeDTO `json:"tipe"`
	EmailLOD    string       `json:"email_lod"`
	InsuredName string       `json:"nama_tertanggung"`
}

// LODTypes menangani POST …/klaim/{klaimID}/lod/tipe — pilihan Tipe PDF dialog Print LOD.
func (h *Handler) LODTypes(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body LODRequest
	if !h.readBody(w, r, &body) {
		return
	}
	dialog, err := h.service.LODDialog(r.Context(), claimID, body.TaskID, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := LODTypesResponse{Types: make([]LODTypeDTO, 0, len(dialog.Types)), EmailLOD: dialog.EmailLOD, InsuredName: dialog.InsuredName}
	for _, t := range dialog.Types {
		out.Types = append(out.Types, LODTypeDTO{ID: t.ID, Name: t.Name, Ready: registrasi.LODTemplateReady(t.ID)})
	}
	h.writeResponse(w, r, http.StatusOK, out)
}

// SetLODType menangani POST …/klaim/{klaimID}/lod/pilih — dropdown Tipe LOD kolom Adjustment.
func (h *Handler) SetLODType(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body LODRequest
	if !h.readBody(w, r, &body) {
		return
	}
	claim, err := h.service.SetLODType(r.Context(), usecase.LODCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage,
		Adjustment: body.Adjustment, Type: body.Type,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}

// PrintLOD menangani POST …/klaim/{klaimID}/lod — mengunduh PDF Letter of Discharge.
func (h *Handler) PrintLOD(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body LODRequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.PrintLOD(r.Context(), usecase.LODCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage,
		Adjustment: body.Adjustment, Type: body.Type,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+result.FileName+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Content)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Content)
}
