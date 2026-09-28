package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// CommitteeMemberDTO adalah satu anggota kasus komite (baris T_CLAIM_KOMITE_LIST).
type CommitteeMemberDTO struct {
	Level     int    `json:"jenjang"`
	Operator  string `json:"komite"`
	Decision  string `json:"keputusan"`
	Note      string `json:"catatan"`
	DecidedAt string `json:"tanggal_putusan,omitempty"`
}

// CommitteeDTO adalah satu kasus komite. Status: "berjalan", "disetujui", atau "ditolak".
type CommitteeDTO struct {
	ID          string               `json:"id"`
	ClaimNumber string               `json:"nomor_klaim"`
	Status      string               `json:"status"`
	Current     string               `json:"menunggu,omitempty"`
	Members     []CommitteeMemberDTO `json:"anggota"`
}

func committeeDTO(c registrasi.CommitteeCase) CommitteeDTO {
	out := CommitteeDTO{ID: c.ID, ClaimNumber: c.ClaimNumber, Members: make([]CommitteeMemberDTO, 0, len(c.Members))}
	switch c.Outcome() {
	case registrasi.DecisionApprove:
		out.Status = "disetujui"
	case registrasi.DecisionReject:
		out.Status = "ditolak"
	default:
		out.Status = "berjalan"
	}
	if m, ok := c.Current(); ok {
		out.Current = m.Operator
	}
	for _, m := range c.Members {
		out.Members = append(out.Members, CommitteeMemberDTO{
			Level: m.Level, Operator: m.Operator, Decision: m.Decision, Note: m.Note, DecidedAt: formatDate(m.DecidedAt),
		})
	}
	return out
}

// CommitteeTransferRequest adalah badan tombol Transfer Komite. Indeks berbasis 1.
type CommitteeTransferRequest struct {
	TaskID     string `json:"tugas_id"`
	Object     int    `json:"objek"`
	Coverage   int    `json:"jaminan"`
	Adjustment int    `json:"adjustment"`
}

// CommitteeTransferResponse adalah klaim sesudah transfer beserta kasus komitenya.
type CommitteeTransferResponse struct {
	Claim     ClaimDTO     `json:"klaim"`
	Committee CommitteeDTO `json:"komite"`
}

// TransferCommittee menangani POST …/klaim/{klaimID}/adjustment/komite.
func (h *Handler) TransferCommittee(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CommitteeTransferRequest
	if !h.readBody(w, r, &body) {
		return
	}
	result, err := h.service.TransferCommittee(r.Context(), usecase.CommitteeTransferCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage, Adjustment: body.Adjustment,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, CommitteeTransferResponse{
		Claim: claimDTO(result.Claim), Committee: committeeDTO(result.Committee),
	})
}

// CommitteeItemDTO adalah satu putusan komite yang menunggu pemanggil.
type CommitteeItemDTO struct {
	CommitteeID     string `json:"komite_id"`
	Level           int    `json:"jenjang"`
	Levels          int    `json:"jumlah_jenjang"`
	ClaimID         string `json:"klaim_id"`
	ClaimNumber     string `json:"nomor_klaim"`
	PolicyNo        string `json:"nomor_polis"`
	InsuredName     string `json:"nama_tertanggung"`
	ObjectName      string `json:"nama_objek"`
	Coverage        string `json:"nama_coverage"`
	Adjustment      int    `json:"adjustment"`
	PaymentTypeName string `json:"nama_tipe_pembayaran"`
	Currency        string `json:"mata_uang"`
	ValueCents      int64  `json:"nilai_asm_sen"`
	CommitteeCents  int64  `json:"nilai_komite_sen"`
	TransferredAt   string `json:"tanggal_transfer"`
}

// CommitteeListResponse adalah daftar putusan komite yang menunggu pemanggil.
type CommitteeListResponse struct {
	Items []CommitteeItemDTO `json:"komite"`
}

// PendingCommittees menangani GET …/komite.
func (h *Handler) PendingCommittees(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	items, err := h.service.PendingCommittees(r.Context(), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := CommitteeListResponse{Items: make([]CommitteeItemDTO, 0, len(items))}
	for _, i := range items {
		out.Items = append(out.Items, CommitteeItemDTO{
			CommitteeID: i.Member.CaseID, Level: i.Member.Level, Levels: i.Levels,
			ClaimID: i.ClaimID, ClaimNumber: i.Member.ClaimNumber, PolicyNo: i.PolicyNo,
			InsuredName: i.InsuredName, ObjectName: i.ObjectName, Coverage: i.Coverage,
			Adjustment: i.Adjustment, PaymentTypeName: registrasi.PaymentTypeName(i.Member.PaymentType),
			Currency: i.Line.Currency, ValueCents: int64(i.Line.Value), CommitteeCents: int64(i.Member.Value),
			TransferredAt: formatDate(i.Member.CreatedAt),
		})
	}
	h.writeResponse(w, r, http.StatusOK, out)
}

// Committee menangani GET …/komite/{komiteID}.
func (h *Handler) Committee(w http.ResponseWriter, r *http.Request, caseID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	c, err := h.service.Committee(r.Context(), caseID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, committeeDTO(c))
}

// CommitteeDecisionRequest adalah badan tombol Setuju/Tolak: keputusan "1" setuju, "2" tolak.
type CommitteeDecisionRequest struct {
	Decision string `json:"keputusan"`
	Note     string `json:"catatan"`
}

// DecideCommittee menangani POST …/komite/{komiteID}/putusan.
func (h *Handler) DecideCommittee(w http.ResponseWriter, r *http.Request, caseID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CommitteeDecisionRequest
	if !h.readBody(w, r, &body) {
		return
	}
	c, err := h.service.DecideCommittee(r.Context(), usecase.CommitteeDecisionCommand{
		CaseID: caseID, Decision: body.Decision, Note: body.Note,
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, committeeDTO(c))
}
