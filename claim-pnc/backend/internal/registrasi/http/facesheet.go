package registrasihttp

import (
	"net/http"
	"strconv"

	"claim-pnc/internal/registrasi/usecase"
)

// FaceSheetRequest adalah badan POST /api/registrasi/klaim/{klaimID}/cfs. Objek dan
// jaminan berbasis 1, sama dengan nomor baris di layar.
type FaceSheetRequest struct {
	TaskID   string `json:"tugas_id"`
	Object   int    `json:"objek"`
	Coverage int    `json:"jaminan"`
}

// FaceSheet menangani tombol "Download Claim Face Sheet".
//
// POST, bukan GET: permintaan ini mencatat revisi dan mengunci estimasi. Jawabannya
// berkas PDF; galat tetap dijawab JSON seperti endpoint lain.
func (h *Handler) FaceSheet(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body FaceSheetRequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.DownloadFaceSheet(r.Context(), usecase.FaceSheetCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage,
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
