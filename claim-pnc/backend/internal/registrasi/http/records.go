package registrasihttp

import (
	"net/http"
	"time"

	"claim-pnc/internal/platform/clock"
)

// formatMoment mengirim waktu sebagai RFC 3339 dalam WIB; kosong bila tidak ada.
func formatMoment(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(clock.ZoneWIB).Format(time.RFC3339)
}

// SurveyDTO adalah satu baris tab Survey.
type SurveyDTO struct {
	CaseID         string `json:"kasus_id"`
	Type           string `json:"tipe"`
	SurveyorName   string `json:"nama_surveyor"`
	Date           string `json:"tanggal_survey"`
	SurveyLocation string `json:"lokasi_survey"`
	ObjectName     string `json:"nama_objek"`
	ObjectLocation string `json:"lokasi_objek"`
	Index          string `json:"urutan"`
	Status         string `json:"status"`
	Note           string `json:"keterangan"`
	InputDate      string `json:"tanggal_input"`
}

// SurveysResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/survey.
type SurveysResponse struct {
	Survey []SurveyDTO `json:"survey"`
}

// DocumentRowDTO adalah satu baris checklist dokumen.
type DocumentRowDTO struct {
	ID         string `json:"id"`
	CategoryID string `json:"jenis_id"`
	Name       string `json:"nama"`
	Required   bool   `json:"wajib"`
	MinDoc     string `json:"minimal"`
	Uploaded   int    `json:"terunggah"`
}

// DocumentCategoryDTO adalah satu kategori checklist dokumen.
type DocumentCategoryDTO struct {
	Code  string           `json:"kode"`
	Label string           `json:"nama"`
	Row   []DocumentRowDTO `json:"dokumen"`
}

// AttachmentDTO adalah satu berkas yang sudah diunggah.
type AttachmentDTO struct {
	ID          string `json:"id"`
	Name        string `json:"nama"`
	MimeType    string `json:"jenis_berkas"`
	Note        string `json:"catatan"`
	Category    string `json:"kategori"`
	SubCategory string `json:"sub_kategori"`
	Stored      bool   `json:"tersimpan"`
	UploadedBy  string `json:"diunggah_oleh"`
	UploadedAt  string `json:"diunggah_pada"`
}

// DocumentsResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/dokumen.
type DocumentsResponse struct {
	Category   []DocumentCategoryDTO `json:"kategori"`
	Attachment []AttachmentDTO       `json:"berkas"`
}

// ProgressDTO adalah satu catatan progres.
type ProgressDTO struct {
	Seq          int    `json:"urutan"`
	InputAt      string `json:"tanggal_input"`
	Status1      string `json:"status_1"`
	Status1Name  string `json:"status_1_nama"`
	Status2      string `json:"status_2"`
	Status2Name  string `json:"status_2_nama"`
	Note         string `json:"keterangan"`
	NextFollowUp string `json:"tindak_lanjut"`
	InputBy      string `json:"diinput_oleh"`
}

// CommunicationDTO adalah satu percakapan.
type CommunicationDTO struct {
	CaseID      string `json:"kasus_id"`
	ID          string `json:"id"`
	SentAt      string `json:"tanggal"`
	SenderName  string `json:"pengirim"`
	Message     string `json:"pesan"`
	Reply       string `json:"balasan"`
	ReplierName string `json:"penjawab"`
	RepliedAt   string `json:"tanggal_balasan"`
}

// ProgressResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/progres.
type ProgressResponse struct {
	Progress      []ProgressDTO      `json:"progres"`
	Communication []CommunicationDTO `json:"komunikasi"`
}

// Surveys menangani GET /api/registrasi/klaim/{klaimID}/survey.
func (h *Handler) Surveys(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	view, err := h.service.Surveys(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := SurveysResponse{Survey: make([]SurveyDTO, 0, len(view.Survey))}
	for _, s := range view.Survey {
		body.Survey = append(body.Survey, SurveyDTO{
			CaseID: s.CaseID, Type: s.Type, SurveyorName: s.SurveyorName, Date: formatDate(s.Date),
			SurveyLocation: s.SurveyLocation, ObjectName: s.ObjectName, ObjectLocation: s.ObjectLocation,
			Index: s.Index, Status: s.Status, Note: s.Note, InputDate: formatDate(s.InputDate),
		})
	}
	h.writeResponse(w, r, http.StatusOK, body)
}

// Documents menangani GET /api/registrasi/klaim/{klaimID}/dokumen.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	view, err := h.service.Documents(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := DocumentsResponse{
		Category:   make([]DocumentCategoryDTO, 0, len(view.Category)),
		Attachment: make([]AttachmentDTO, 0, len(view.Attachment)),
	}
	for _, c := range view.Category {
		dto := DocumentCategoryDTO{Code: c.Code, Label: c.Label, Row: make([]DocumentRowDTO, 0, len(c.Row))}
		for _, row := range c.Row {
			dto.Row = append(dto.Row, DocumentRowDTO{
				ID: row.Type.ID, CategoryID: row.Type.CategoryID, Name: row.Type.Name,
				Required: row.Required, MinDoc: row.Type.MinDoc, Uploaded: row.Uploaded,
			})
		}
		body.Category = append(body.Category, dto)
	}
	for _, a := range view.Attachment {
		body.Attachment = append(body.Attachment, AttachmentDTO{
			ID: a.ID, Name: a.Name, MimeType: a.MimeType, Note: a.Note, Category: a.Category,
			SubCategory: a.SubCategory, Stored: a.ImageID != "", UploadedBy: a.UploadedBy,
			UploadedAt: formatMoment(a.UploadedAt),
		})
	}
	h.writeResponse(w, r, http.StatusOK, body)
}

// Progress menangani GET /api/registrasi/klaim/{klaimID}/progres.
func (h *Handler) Progress(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	view, err := h.service.ProgressRecords(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := ProgressResponse{
		Progress:      make([]ProgressDTO, 0, len(view.Progress)),
		Communication: make([]CommunicationDTO, 0, len(view.Communication)),
	}
	for _, p := range view.Progress {
		body.Progress = append(body.Progress, ProgressDTO{
			Seq: p.Seq, InputAt: formatMoment(p.InputAt), Status1: p.Status1, Status1Name: p.Status1Name,
			Status2: p.Status2, Status2Name: p.Status2Name, Note: p.Note,
			NextFollowUp: formatMoment(p.NextFollowUp), InputBy: p.InputBy,
		})
	}
	for _, c := range view.Communication {
		body.Communication = append(body.Communication, CommunicationDTO{
			CaseID: c.CaseID, ID: c.ID, SentAt: formatMoment(c.SentAt), SenderName: c.SenderName,
			Message: c.Message, Reply: c.Reply, ReplierName: c.ReplierName, RepliedAt: formatMoment(c.RepliedAt),
		})
	}
	h.writeResponse(w, r, http.StatusOK, body)
}
