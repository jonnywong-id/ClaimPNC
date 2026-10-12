package registrasihttp

import (
	"net/http"
	"time"

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

	// ReceiverID adalah Penerima Klaim modal Transfer Claim ke Komite (lini Travel); boleh kosong.
	ReceiverID string `json:"penerima_klaim,omitempty"`
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
		ReceiverID: body.ReceiverID,
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

// CommitteeNoteDTO adalah isian modal "Transfer Claim ke Komite" satu jaminan — lihat
// registrasi.CommitteeNote. Inisial dan tanggal komite hanya dikirim server.
type CommitteeNoteDTO struct {
	Circumstances       string `json:"kronologi_kejadian"`
	ExtentOfLoss        string `json:"jumlah_kerugian"`
	LegalLiability      string `json:"polis_liability"`
	Remarks             string `json:"remarks"`
	RemarkInvestigation string `json:"remarks_investigasi"`
	Diagnose            string `json:"diagnosa"`
	DiagnoseCode        string `json:"kode_diagnosa"`
	DiagnoseDesc        string `json:"desc_diagnosa"`
	Receiver            string `json:"penerima_klaim"`
	InitialName         string `json:"inisial,omitempty"`
	CommitteeDate       string `json:"tanggal_komite,omitempty"`
}

// CommitteeNoteRequest adalah badan POST /api/registrasi/klaim/{id}/jaminan/isian-komite.
// Objek dan jaminan berbasis 1.
type CommitteeNoteRequest struct {
	TaskID   string `json:"tugas_id"`
	Object   int    `json:"objek"`
	Coverage int    `json:"jaminan"`
	CommitteeNoteDTO
}

func committeeNoteDTO(n registrasi.CommitteeNote) CommitteeNoteDTO {
	d := CommitteeNoteDTO{
		Circumstances: n.Circumstances, ExtentOfLoss: n.ExtentOfLoss, LegalLiability: n.LegalLiability,
		Remarks: n.Remarks, RemarkInvestigation: n.RemarkInvestigation, Diagnose: n.Diagnose,
		DiagnoseCode: n.DiagnoseCode, DiagnoseDesc: n.DiagnoseDesc, Receiver: n.Receiver,
		InitialName: n.InitialName,
	}
	if !n.CommitteeDate.IsZero() {
		d.CommitteeDate = n.CommitteeDate.Format(time.RFC3339)
	}
	return d
}

// SaveCommitteeNote menangani POST /api/registrasi/klaim/{klaimID}/jaminan/isian-komite.
func (h *Handler) SaveCommitteeNote(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body CommitteeNoteRequest
	if !h.readBody(w, r, &body) {
		return
	}
	claim, err := h.service.SaveCommitteeNote(r.Context(), usecase.CommitteeNoteCommand{
		ClaimID: claimID, TaskID: body.TaskID, Object: body.Object, Coverage: body.Coverage,
		Note: registrasi.CommitteeNote{
			Circumstances: body.Circumstances, ExtentOfLoss: body.ExtentOfLoss,
			LegalLiability: body.LegalLiability, Remarks: body.Remarks,
			RemarkInvestigation: body.RemarkInvestigation, Diagnose: body.Diagnose,
			DiagnoseCode: body.DiagnoseCode, DiagnoseDesc: body.DiagnoseDesc, Receiver: body.Receiver,
		},
	}, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}
