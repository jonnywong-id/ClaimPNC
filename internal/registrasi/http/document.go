package registrasihttp

import (
	"io"
	"net/http"
	"strings"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// maxDocumentUpload adalah batas satu unggahan — sama dengan modul dokumen penunjang
// (`dokumenpenunjang.BatasUkuranBerkas`, 20 MiB), ditambah ruang untuk batas multipart.
const maxDocumentUpload = 20<<20 + 1<<20

// canDelete menilai tombol Delete per lampiran untuk pemanggil.
func (h *Handler) canDelete(caller usecase.Caller) func(registrasi.Attachment) bool {
	return func(a registrasi.Attachment) bool { return h.service.CanDeleteAttachment(a, caller) }
}

// DeleteDocument menangani POST /api/registrasi/klaim/{klaimID}/dokumen/{lampiranID}/hapus —
// tombol Delete. Jawabannya checklist yang sudah diperbarui, seperti SetCountAttach_act
// yang Pega jalankan sesudahnya.
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request, claimID, attachmentID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.failure(w, r, portal.ErrNotStated)
		return
	}
	if err := h.service.DeleteDocument(r.Context(), usecase.DeleteDocumentCommand{
		ClaimID: claimID, Portal: active.Alias, AttachmentID: attachmentID,
	}, caller); err != nil {
		h.failure(w, r, err)
		return
	}
	view, err := h.service.Documents(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, documentsResponse(view, h.canDelete(caller)))
}

// DocumentLink menangani GET /api/registrasi/klaim/{klaimID}/dokumen/{lampiranID}/tautan —
// tombol "Lihat dokumen" membuka berkasnya lewat alamat ini.
func (h *Handler) DocumentLink(w http.ResponseWriter, r *http.Request, claimID, attachmentID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.failure(w, r, portal.ErrNotStated)
		return
	}
	link, err := h.service.DocumentLink(r.Context(), usecase.DocumentLinkCommand{
		ClaimID: claimID, Portal: active.Alias, AttachmentID: attachmentID,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, DocumentLinkResponse{URL: link.URL, ValidUntil: formatMoment(link.ExpiresAt)})
}

// UploadDocument menangani POST /api/registrasi/klaim/{klaimID}/dokumen — tombol
// "Unggah Dokumen" pada satu baris checklist.
//
// Badannya multipart: `berkas` (isi), `jenis_dokumen` (DOC_TYPE_DT_ID baris), dan
// `catatan` (opsional). Jawabannya checklist yang sudah diperbarui, sehingga "Total Sudah
// Diunggah" langsung bertambah.
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.failure(w, r, portal.ErrNotStated)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentUpload)
	file, header, err := r.FormFile("berkas")
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			h.writeResponse(w, r, http.StatusRequestEntityTooLarge, ErrorResponse{
				Code: CodeMalformedRequest, Message: "The file is larger than 20 MB.",
			})
			return
		}
		h.failure(w, r, registrasi.ErrDocumentFileEmpty)
		return
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(file)
	if err != nil {
		h.failure(w, r, err)
		return
	}

	_, err = h.service.UploadDocument(r.Context(), usecase.UploadDocumentCommand{
		ClaimID:        claimID,
		Portal:         active.Alias,
		DocumentTypeID: strings.TrimSpace(r.FormValue("jenis_dokumen")),
		FileName:       header.Filename,
		Content:        content,
		Note:           r.FormValue("catatan"),
	}, caller)
	if err != nil {
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
