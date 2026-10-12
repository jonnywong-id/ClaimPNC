package registrasihttp

import (
	"net/http"
	"strconv"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// DLARequest adalah badan dua rute DLA. Objek, jaminan, dan adjustment berbasis 1. Nomor
// memilih satu DLA untuk dicetak (kosong = seluruhnya); Remarks berisi REMARKS per nomor
// DLA yang belum pernah dicetak.
type DLARequest struct {
	TaskID      string            `json:"tugas_id"`
	Object      int               `json:"objek"`
	Coverage    int               `json:"jaminan"`
	Adjustment  int               `json:"adjustment"`
	Number      string            `json:"nomor"`
	Remarks     map[string]string `json:"catatan"`
	AsPerPolicy bool              `json:"sesuai_polis"`
}

// DLARowDTO adalah satu baris grid layar PrintDLA.
type DLARowDTO struct {
	Number    string `json:"nomor"`
	Recipient string `json:"penerima"`
	Type      string `json:"tipe"`
	Note      string `json:"catatan"`
	Email     string `json:"email"`
	Date      string `json:"tanggal"`
	Value     string `json:"nilai"`
	Currency  string `json:"mata_uang"`
	Printed   bool   `json:"sudah_cetak"`
	Sent      bool   `json:"sudah_kirim"`
}

// DLAListResponse adalah isi layar PrintDLA.
type DLAListResponse struct {
	Issued   int         `json:"baru_terbit"`
	ExGratia bool        `json:"ex_gratia"`
	Warning  []string    `json:"peringatan"`
	DLA      []DLARowDTO `json:"dla"`
}

func (b DLARequest) command(claimID string) usecase.DLACommand {
	return usecase.DLACommand{
		ClaimID: claimID, TaskID: b.TaskID, Object: b.Object, Coverage: b.Coverage, Adjustment: b.Adjustment,
		Number: b.Number, Remarks: b.Remarks, AsPerPolicy: b.AsPerPolicy,
	}
}

func dlaListDTO(l usecase.DLAList) DLAListResponse {
	out := DLAListResponse{Issued: l.Issued, ExGratia: l.ExGratia, Warning: l.Warning, DLA: make([]DLARowDTO, 0, len(l.DLA))}
	if out.Warning == nil {
		out.Warning = []string{}
	}
	for _, d := range l.DLA {
		out.DLA = append(out.DLA, dlaRowDTO(d))
	}
	return out
}

func dlaRowDTO(d registrasi.DLA) DLARowDTO {
	return DLARowDTO{
		Number: d.Number, Recipient: d.Recipient, Type: d.Type, Note: d.Note, Email: d.Info.Email,
		Date: formatDate(d.Date), Value: d.Value, Currency: d.Currency, Printed: d.Printed, Sent: d.Sent,
	}
}

// ListDLA menangani POST …/dla/daftar — membuka layar PrintDLA (menerbitkan bila belum).
func (h *Handler) ListDLA(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body DLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	list, err := h.service.ListDLA(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, dlaListDTO(list))
}

// DLA menangani POST …/dla — tombol PRINT (satu nomor) dan Print All DLA. Jawabannya PDF,
// atau ZIP bila lebih dari satu DLA. Galat dijawab JSON seperti endpoint lain.
func (h *Handler) DLA(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body DLARequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.PrintDLA(r.Context(), body.command(claimID), caller)
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
