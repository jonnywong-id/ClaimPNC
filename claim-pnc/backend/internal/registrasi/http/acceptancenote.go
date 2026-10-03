package registrasihttp

import (
	"net/http"
	"strconv"

	"claim-pnc/internal/registrasi/usecase"
)

// AcceptanceNoteRequest adalah badan tombol PRINT Draft Persetujuan. Indeks berbasis 1.
type AcceptanceNoteRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`
}

// AcceptanceNote menangani POST …/akseptasi/draft — PDF Draft Persetujuan
// (`PrintPDFAcceptanceNote`). Galat dijawab JSON seperti endpoint lain.
func (h *Handler) AcceptanceNote(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body AcceptanceNoteRequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.PrintAcceptanceNote(r.Context(), usecase.AcceptanceNoteCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage, Adjustment: body.Adjustment,
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
