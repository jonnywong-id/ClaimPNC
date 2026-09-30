package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi/usecase"
)

// CashierRequest adalah badan dua rute Transfer Kasir. Indeks berbasis 1.
type CashierRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`
}

// CashierPreviewResponse adalah isi dialog konfirmasi Transfer Kasir.
type CashierPreviewResponse struct {
	AcceptedNo string `json:"nomor_akseptasi"`
	Receiver   string `json:"penerima"`
	AccountNo  string `json:"nomor_rekening"`
	BankName   string `json:"nama_bank"`
	Email      string `json:"email"`
	NettCents  int64  `json:"nilai_nett_sen"`
	Currency   string `json:"mata_uang"`
	Problem    string `json:"masalah"`
}

func (b CashierRequest) command(claimID string) usecase.CashierCommand {
	return usecase.CashierCommand{ClaimID: claimID, TaskID: b.TaskID, Object: b.Object, Coverage: b.Coverage, Adjustment: b.Adjustment}
}

// PreviewCashier menangani POST …/kasir/pratinjau — isi dialog konfirmasi.
func (h *Handler) PreviewCashier(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CashierRequest
	if !h.readBody(w, r, &body) {
		return
	}
	p, err := h.service.PreviewCashierTransfer(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, CashierPreviewResponse{
		AcceptedNo: p.AcceptedNo, Receiver: p.Receiver.Name, AccountNo: p.Receiver.AccountNo, BankName: p.Receiver.BankName,
		Email: p.Receiver.Email, NettCents: int64(p.Nett), Currency: p.Currency, Problem: p.Problem,
	})
}

// TransferCashier menangani POST …/kasir — Submit dialog Transfer Kasir.
func (h *Handler) TransferCashier(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CashierRequest
	if !h.readBody(w, r, &body) {
		return
	}
	claim, err := h.service.TransferCashier(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}
