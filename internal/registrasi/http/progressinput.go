package registrasihttp

import (
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Modal Input Progress Claim — lihat registrasi/progressinput.go.

// ProgressPositionDTO adalah satu posisi progres.
type ProgressPositionDTO struct {
	ID      string `json:"id"`
	CaseID  string `json:"kasus_id"`
	Name    string `json:"posisi"`
	Status  string `json:"status"`
	Date    string `json:"tanggal,omitempty"`
	Terbuka bool   `json:"terbuka"`
}

// ProgressOptionDTO adalah satu pilihan master progres.
type ProgressOptionDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// ProgressInputFormResponse adalah isi awal modal.
type ProgressInputFormResponse struct {
	Positions   []ProgressPositionDTO `json:"posisi"`
	Status1     []ProgressOptionDTO   `json:"status_progress_1"`
	Statuses    []string              `json:"status"`
	DocTypes    []string              `json:"tipe_dokumen"`
	EmailBroker string                `json:"email_progress"`
	// EmailStatuses: Status Progress 1 yang mewajibkan Email Progress Klaim.
	EmailStatuses []string `json:"status_wajib_email"`
	DocStatus     string   `json:"status_tipe_dokumen"`
}

// ProgressInputBody adalah badan Submit.
type ProgressInputBody struct {
	PositionID   string   `json:"posisi_id"`
	Progress1    string   `json:"status_progress_1"`
	NeedIDs      []string `json:"status_perlu"`
	Status       string   `json:"status"`
	Remarks      string   `json:"remarks"`
	NextFollowUp string   `json:"tgl_next_follow_up"`
	DocType      string   `json:"tipe_dokumen"`
	EmailBroker  string   `json:"email_progress"`
}

// ProgressInputResponse adalah jawaban Submit.
type ProgressInputResponse struct {
	Warnings []string `json:"peringatan"`
}

func progressOptionsDTO(list []registrasi.ProgressOption) []ProgressOptionDTO {
	out := make([]ProgressOptionDTO, 0, len(list))
	for _, o := range list {
		out = append(out, ProgressOptionDTO{ID: o.ID, Name: o.Name})
	}
	return out
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ProgressInputForm menangani GET …/progres/form.
func (h *Handler) ProgressInputForm(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	f, err := h.service.ProgressInputForm(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := ProgressInputFormResponse{Positions: []ProgressPositionDTO{}, Status1: progressOptionsDTO(f.Status1),
		Statuses: nonNilStrings(f.Statuses), DocTypes: nonNilStrings(f.DocTypes), EmailBroker: f.EmailBroker,
		EmailStatuses: registrasi.ProgressEmailStatuses(), DocStatus: registrasi.ProgressDocumentStatus}
	for _, p := range f.Positions {
		d := ProgressPositionDTO{ID: p.ID, CaseID: p.CaseID, Name: p.Name, Status: p.Status, Terbuka: p.Open()}
		if !p.Date.IsZero() {
			d.Date = formatMoment(p.Date)
		}
		out.Positions = append(out.Positions, d)
	}
	h.writeResponse(w, r, http.StatusOK, out)
}

// ProgressNeedOptions menangani GET /registrasi/progres/status-perlu?status1=&tipe_dokumen=.
func (h *Handler) ProgressNeedOptions(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	q := r.URL.Query()
	list, err := h.service.ProgressNeedOptions(r.Context(), q.Get("status1"), q.Get("tipe_dokumen"))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, progressOptionsDTO(list))
}

// InputProgress menangani POST …/progres — tombol Submit modal Input Progress Claim.
func (h *Handler) InputProgress(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body ProgressInputBody
	if !h.readBody(w, r, &body) {
		return
	}
	var follow time.Time
	if s := strings.TrimSpace(body.NextFollowUp); s != "" {
		t, err := parseDate(s, "Tgl Next Follow Up")
		if err != nil {
			h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: err.Error()})
			return
		}
		follow = t
	}
	res, err := h.service.InputProgress(r.Context(), usecase.ProgressInputCommand{
		ClaimID: claimID, PositionID: body.PositionID, Progress1: body.Progress1, NeedIDs: body.NeedIDs,
		Status: body.Status, Remarks: body.Remarks, NextFollowUp: follow, DocType: body.DocType, EmailBroker: body.EmailBroker,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ProgressInputResponse{Warnings: nonNilStrings(res.Warnings)})
}
