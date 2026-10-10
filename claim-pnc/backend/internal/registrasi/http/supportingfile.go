package registrasihttp

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// UploadSupportingFiles menangani POST …/klaim/{klaimID}/adjustment/file-penunjang — tombol
// "Unggah File Penunjang" pada satu baris adjustment (local action `UploadDokumen_Adj`).
//
// Badannya multipart: `tugas_id`, `objek`, `jaminan`, `adjustment` (berbasis 1), lalu pasangan
// berulang `berkas`, `jenis_dokumen` (DOC_TYPE_DT_ID), dan `catatan_berkas` berurutan sama.
// Jawabannya checklist dokumen klaim yang sudah diperbarui.
func (h *Handler) UploadSupportingFiles(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.failure(w, r, portal.ErrNotStated)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAcceptanceUpload)
	if err := r.ParseMultipartForm(maxDocumentUpload); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeResponse(w, r, http.StatusRequestEntityTooLarge, ErrorResponse{
				Code: CodeMalformedRequest, Message: "The files are larger than the upload limit.",
			})
			return
		}
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code: CodeMalformedRequest, Message: "The supporting files could not be read.",
		})
		return
	}

	index := func(name string) (int, bool) {
		n, err := strconv.Atoi(strings.TrimSpace(r.FormValue(name)))
		return n, err == nil && n > 0
	}
	object, okObject := index("objek")
	coverage, okCoverage := index("jaminan")
	adjustment, okAdjustment := index("adjustment")
	if !okObject || !okCoverage || !okAdjustment {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code: CodeMalformedRequest, Message: "objek, jaminan, and adjustment must be positive numbers.",
		})
		return
	}

	var files []usecase.AcceptanceFile
	kinds := r.MultipartForm.Value["jenis_dokumen"]
	notes := r.MultipartForm.Value["catatan_berkas"]
	for i, header := range r.MultipartForm.File["berkas"] {
		f, err := header.Open()
		if err != nil {
			h.failure(w, r, err)
			return
		}
		content, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			h.failure(w, r, err)
			return
		}
		file := usecase.AcceptanceFile{FileName: header.Filename, Content: content}
		if i < len(kinds) {
			file.DocumentTypeID = strings.TrimSpace(kinds[i])
		}
		if i < len(notes) {
			file.Note = notes[i]
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		h.failure(w, r, registrasi.ErrDocumentFileEmpty)
		return
	}

	if _, err := h.service.UploadSupportingFiles(r.Context(), usecase.SupportingFilesCommand{
		ClaimID: claimID, TaskID: strings.TrimSpace(r.FormValue("tugas_id")), Portal: active.Alias,
		Object: object, Coverage: coverage, Adjustment: adjustment, Files: files,
	}, caller); err != nil {
		h.failure(w, r, err)
		return
	}

	view, err := h.service.Documents(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusCreated, documentsResponse(view, h.canDelete(caller)))
}
