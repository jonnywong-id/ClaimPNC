package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// CashierRequest adalah badan dua rute Transfer Kasir. Indeks berbasis 1.
type CashierRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`
	// TransferType adalah "Tipe Transfer Kasir": 1 Pembayaran Biasa, 2 Join Placement, 3 Fronting.
	TransferType string `json:"tipe_transfer"`
	// UnpaidFacOut adalah No DLA FAC OUT yang dicentang "Pilih Fac-out Tidak Dibayar".
	UnpaidFacOut []string `json:"fac_out_tidak_dibayar"`
}

// CashierFacOutDTO adalah satu baris tabel Fac-out dialog Transfer Pembayaran.
type CashierFacOutDTO struct {
	Number    string `json:"nomor_dla"`
	Reinsurer string `json:"nama_facout"`
	Value     string `json:"nilai_bayar"`
	Currency  string `json:"mata_uang"`
}

// CashierTypeDTO adalah satu pilihan "Tipe Transfer Kasir".
type CashierTypeDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
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
	// Confirmation adalah kalimat konfirmasi Pega (Pre_AlertTransferkasir).
	Confirmation string             `json:"konfirmasi"`
	Types        []CashierTypeDTO   `json:"tipe_transfer"`
	FacOut       []CashierFacOutDTO `json:"fac_out"`
}

func (b CashierRequest) command(claimID string) usecase.CashierCommand {
	return usecase.CashierCommand{ClaimID: claimID, TaskID: b.TaskID, Object: b.Object, Coverage: b.Coverage, Adjustment: b.Adjustment,
		TransferType: b.TransferType, UnpaidFacOut: b.UnpaidFacOut}
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
	types := make([]CashierTypeDTO, 0, len(registrasi.CashierTransferTypes))
	for _, t := range registrasi.CashierTransferTypes {
		types = append(types, CashierTypeDTO{ID: t.ID, Name: t.Name})
	}
	facOut := make([]CashierFacOutDTO, 0, len(p.FacOut))
	for _, f := range p.FacOut {
		facOut = append(facOut, CashierFacOutDTO{Number: f.Number, Reinsurer: f.Reinsurer, Value: f.Value, Currency: f.Currency})
	}
	h.writeResponse(w, r, http.StatusOK, CashierPreviewResponse{
		Types: types, FacOut: facOut,
		AcceptedNo: p.AcceptedNo, Receiver: p.Receiver.Name, AccountNo: p.Receiver.AccountNo, BankName: p.Receiver.BankName,
		Email: p.Receiver.Email, NettCents: int64(p.Nett), Currency: p.Currency, Problem: p.Problem,
		Confirmation: p.Confirmation,
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
