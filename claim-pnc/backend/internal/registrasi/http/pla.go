package registrasihttp

import (
	"net/http"
	"strconv"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// PLARequest adalah badan tiga rute PLA. Objek dan jaminan berbasis 1. Nomor memilih satu
// PLA untuk dicetak (kosong = seluruhnya); Notes berisi REMARKS per nomor PLA.
type PLARequest struct {
	TaskID   string            `json:"tugas_id"`
	Object   int               `json:"objek"`
	Coverage int               `json:"jaminan"`
	Number   string            `json:"nomor"`
	Notes    map[string]string `json:"catatan"`
	// Emails berisi isian Email per nomor PLA; hanya nomor yang dikirim yang diubah.
	Emails map[string]string `json:"email,omitempty"`
}

// PLARowDTO adalah satu baris grid layar PrintPLA_dtl.
type PLARowDTO struct {
	Number    string `json:"nomor"`
	Recipient string `json:"penerima"`
	Type      string `json:"tipe"`
	Note      string `json:"catatan"`
	Email     string `json:"email"`
	Date      string `json:"tanggal"`
	// Sent adalah T_PLALIST.ISKIRIM = '1' — PLA sudah dikirim lewat email.
	Sent bool `json:"terkirim"`
}

// PLASendDTO adalah hasil SEND ALL PLA untuk satu PLA.
type PLASendDTO struct {
	Number  string `json:"nomor"`
	Sent    bool   `json:"terkirim"`
	Skipped bool   `json:"dilewati"`
	Error   string `json:"galat,omitempty"`
}

// PLASendResponse adalah jawaban SEND ALL PLA beserta daftar PLA terbaru.
type PLASendResponse struct {
	Result []PLASendDTO    `json:"hasil"`
	List   PLAListResponse `json:"daftar"`
}

// PLAListResponse adalah isi layar PrintPLA_dtl.
type PLAListResponse struct {
	Revision int         `json:"revisi_cfs"`
	Issued   int         `json:"baru_terbit"`
	PLA      []PLARowDTO `json:"pla"`
}

func (b PLARequest) command(claimID string) usecase.PLACommand {
	return usecase.PLACommand{ClaimID: claimID, TaskID: b.TaskID, Object: b.Object, Coverage: b.Coverage, Number: b.Number}
}

func plaListDTO(l usecase.PLAList) PLAListResponse {
	out := PLAListResponse{Revision: l.Revision, Issued: l.Issued, PLA: make([]PLARowDTO, 0, len(l.PLA))}
	for _, p := range l.PLA {
		out.PLA = append(out.PLA, plaRowDTO(p))
	}
	return out
}

func plaRowDTO(p registrasi.PLA) PLARowDTO {
	return PLARowDTO{Number: p.Number, Recipient: p.Recipient, Type: p.Type, Note: p.Note, Email: p.Info.Email, Date: formatDate(p.Date), Sent: p.Sent}
}

// ListPLA menangani POST …/pla/daftar — membuka layar PrintPLA_dtl (menerbitkan bila belum).
func (h *Handler) ListPLA(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body PLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	list, err := h.service.ListPLA(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, plaListDTO(list))
}

// SavePLANotes menangani POST …/pla/catatan — menyimpan isian REMARKS dan Email.
func (h *Handler) SavePLANotes(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body PLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	list, err := h.service.SavePLADetails(r.Context(), body.command(claimID), body.Notes, body.Emails, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, plaListDTO(list))
}

// SendAllPLA menangani POST …/pla/kirim — tombol SEND ALL PLA. Jawabannya hasil per PLA dan
// daftar PLA terbaru (status terkirim sudah berubah).
func (h *Handler) SendAllPLA(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body PLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.SendAllPLA(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	list, err := h.service.ListPLA(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := PLASendResponse{Result: make([]PLASendDTO, 0, len(result)), List: plaListDTO(list)}
	for _, o := range result {
		out.Result = append(out.Result, PLASendDTO{Number: o.Number, Sent: o.Sent, Skipped: o.Skipped, Error: o.Error})
	}
	h.writeResponse(w, r, http.StatusOK, out)
}

// PLA menangani POST …/pla — tombol Print PLA (satu nomor) dan Print All PLA. Jawabannya
// PDF, atau ZIP bila lebih dari satu PLA. Galat dijawab JSON seperti endpoint lain.
func (h *Handler) PLA(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body PLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.PrintPLA(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+result.FileName+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(result.Content)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Content)
}
