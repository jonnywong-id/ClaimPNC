package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// AccountDTO adalah satu rekening Master Rekening untuk isian No Rekening penerima klaim.
// Tanggal kosong bila kolomnya belum diisi di master.
type AccountDTO struct {
	Number              string `json:"nomor_rekening"`
	Name                string `json:"nama"`
	BankName            string `json:"nama_bank"`
	Branch              string `json:"nama_cabang_bank"`
	Address             string `json:"alamat"`
	BankID              string `json:"id_bank"`
	Email               string `json:"email"`
	Telephone           string `json:"telepon"`
	CashierApprovedAt   string `json:"tanggal_approve_kasir"`
	CommitteeApprovedAt string `json:"tanggal_approve_komite"`
}

func accountDTO(a registrasi.BankAccount) AccountDTO {
	return AccountDTO{
		Number: a.Number, Name: a.Name, BankName: a.BankName, Branch: a.Branch, Address: a.Address,
		BankID: a.BankID, Email: a.Email, Telephone: a.Telephone,
		CashierApprovedAt: formatDate(a.CashierApprovedAt), CommitteeApprovedAt: formatDate(a.CommitteeApprovedAt),
	}
}

// FindAccount menangani GET …/rekening/{nomor} — isian No Rekening InputReceiver.
func (h *Handler) FindAccount(w http.ResponseWriter, r *http.Request, number string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	account, err := h.service.FindAccount(r.Context(), number)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, accountDTO(account))
}

// ReceiverRequest adalah badan tombol Simpan InputReceiver. ID kosong berarti penerima baru
// (tombol Tambah).
type ReceiverRequest struct {
	TaskID     string `json:"tugas_id"`
	ReceiverID string `json:"id"`
	AccountNo  string `json:"nomor_rekening"`
	Email      string `json:"email"`
	Telephone  string `json:"telepon"`
}

// SaveReceiver menangani POST …/klaim/{klaimID}/penerima.
func (h *Handler) SaveReceiver(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body ReceiverRequest
	if !h.readBody(w, r, &body) {
		return
	}
	claim, err := h.service.SaveReceiver(r.Context(), usecase.ReceiverCommand{
		ClaimID: claimID, TaskID: body.TaskID, ReceiverID: body.ReceiverID,
		AccountNo: body.AccountNo, Email: body.Email, Telephone: body.Telephone,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}
