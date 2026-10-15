package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// AcceptanceTypeOptionDTO adalah satu pilihan Tipe Akseptasi Klaim.
type AcceptanceTypeOptionDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// AcceptanceDefaultsResponse adalah isian yang sudah terisi saat form AcceptationLOD dibuka
// (`AcceptationLOD_PreAct`). nilai_lod_sen hanya ada untuk Non-MBU.
type AcceptanceDefaultsResponse struct {
	Type          string                    `json:"tipe_akseptasi"`
	TypeOptions   []AcceptanceTypeOptionDTO `json:"pilihan_tipe_akseptasi"`
	CommitteeName string                    `json:"nama_komite_akseptasi"`
	LODValueCents *int64                    `json:"nilai_lod_sen,omitempty"`
	Warning       []string                  `json:"peringatan"`
}

// AcceptanceDefaults menangani POST …/akseptasi/awal. Badannya sama dengan tombol PRINT Draft
// Persetujuan: alamat baris adjustment.
func (h *Handler) AcceptanceDefaults(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body AcceptanceNoteRequest
	if !h.readBody(w, r, &body) {
		return
	}
	d, err := h.service.AcceptanceDefaults(r.Context(), usecase.AcceptanceDefaultsCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage, Adjustment: body.Adjustment,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := AcceptanceDefaultsResponse{Type: d.Type, CommitteeName: d.CommitteeName, Warning: []string{}}
	for _, o := range registrasi.AcceptanceTypeOptions() {
		out.TypeOptions = append(out.TypeOptions, AcceptanceTypeOptionDTO{ID: o.ID, Name: o.Name})
	}
	if d.HasLODValue {
		cents := int64(d.LODValue)
		out.LODValueCents = &cents
	}
	if d.LocationMissing {
		out.Warning = append(out.Warning, registrasi.MsgAcceptanceLocation)
	}
	h.writeResponse(w, r, http.StatusOK, out)
}
