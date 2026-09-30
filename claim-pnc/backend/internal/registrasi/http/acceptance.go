package registrasihttp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// AcceptanceRequest adalah isian form Persetujuan / Akseptasi (`AcceptationLOD_Sect`).
// Tanggal berformat YYYY-MM-DD; kosong berarti tidak diisi. Indeks berbasis 1. Tanggal Cetak
// LOD tidak dikirim: ia baca saja di form dan diambil dari baris (diisi saat Print LOD).
type AcceptanceRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`

	LODStatus          string `json:"persetujuan_tertanggung"`
	Type               string `json:"tipe_akseptasi"`
	ReceiveDate        string `json:"tanggal_terima_lod"`
	PayableDate        string `json:"tanggal_boleh_bayar"`
	AnalystReceiveDate string `json:"tanggal_terima_lod_analis"`
	LODValueCents      *int64 `json:"nilai_lod_sen"`
	ReceiverID         string `json:"penerima"`
	CommitteeName      string `json:"nama_komite_akseptasi"`
	Remark             string `json:"catatan_penerima"`
	MinutesNote        string `json:"berita_acara"`
}

// maxAcceptanceUpload adalah batas seluruh badan Simpan akseptasi beserta berkasnya.
const maxAcceptanceUpload = 5 * maxDocumentUpload

// AcceptSettlement menangani POST …/klaim/{klaimID}/akseptasi.
//
// Badannya multipart: `isian` (JSON AcceptanceRequest), lalu pasangan berulang `berkas` dan
// `jenis_dokumen` (DOC_TYPE_DT_ID) berurutan sama — "Unggah Dokumen Persetujuan LOD".
func (h *Handler) AcceptSettlement(w http.ResponseWriter, r *http.Request, claimID string) {
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
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code: CodeMalformedRequest, Message: "The acceptance form could not be read.",
		})
		return
	}
	var body AcceptanceRequest
	if err := json.Unmarshal([]byte(r.FormValue("isian")), &body); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code: CodeMalformedRequest, Message: "The acceptance form could not be read.",
		})
		return
	}
	form, err := acceptanceForm(body)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: err.Error()})
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

	claim, err := h.service.AcceptSettlement(r.Context(), usecase.AcceptanceCommand{
		ClaimID: claimID, TaskID: body.TaskID, Portal: active.Alias,
		Object: body.Object, Coverage: body.Coverage, Adjustment: body.Adjustment,
		Form: form, Files: files,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}

func acceptanceForm(b AcceptanceRequest) (registrasi.AcceptanceForm, error) {
	f := registrasi.AcceptanceForm{
		LODStatus: strings.TrimSpace(b.LODStatus), Type: strings.TrimSpace(b.Type),
		ReceiverID: strings.TrimSpace(b.ReceiverID), CommitteeName: strings.TrimSpace(b.CommitteeName),
		Remark: b.Remark, MinutesNote: b.MinutesNote,
	}
	if b.LODValueCents != nil {
		f.LODValue, f.HasLODValue = registrasi.Money(*b.LODValueCents), true
	}
	var err error
	if f.ReceiveDate, err = optionalDate(b.ReceiveDate, "tanggal_terima_lod"); err != nil {
		return f, err
	}
	if f.PayableDate, err = optionalDate(b.PayableDate, "tanggal_boleh_bayar"); err != nil {
		return f, err
	}
	if f.AnalystReceiveDate, err = optionalDate(b.AnalystReceiveDate, "tanggal_terima_lod_analis"); err != nil {
		return f, err
	}
	return f, nil
}

// optionalDate membaca tanggal YYYY-MM-DD yang boleh kosong; wajib-tidaknya diperiksa aturan
// domain, bukan di sini.
func optionalDate(s, name string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, nil
	}
	return parseDate(strings.TrimSpace(s), name)
}
