package registrasihttp

import (
	"net/http"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// SettlementDTO adalah satu baris grid Adjustment (AdjustmentList). Nilai uang dalam sen,
// persen per 10.000 (100% = 1000000), kurs per 10.000.
type SettlementDTO struct {
	PaymentType      string `json:"tipe_pembayaran"`
	PaymentTypeName  string `json:"nama_tipe_pembayaran"`
	Currency         string `json:"mata_uang"`
	RateE4           int64  `json:"kurs_e4"`
	ProposeCents     int64  `json:"nilai_propose_sen"`
	SubmittedCents   int64  `json:"nilai_pengajuan_sen"`
	LOC              int64  `json:"loc"`
	SalvageACents    int64  `json:"nilai_salvage_sen"`
	SalvageBCents    int64  `json:"nilai_salvage_b_sen"`
	InterimCents     int64  `json:"nilai_interim_sen"`
	EstimationCents  int64  `json:"nilai_estimasi_sen"`
	RiskType         string `json:"tipe_resiko"`
	RiskPercent      int64  `json:"persen_resiko"`
	RiskValueCents   int64  `json:"nilai_resiko_sen"`
	GrossCents       int64  `json:"nilai_gross_sen"`
	ShareASM         int64  `json:"share_asm"`
	ValueCents       int64  `json:"nilai_asm_sen"`
	AcceptedCents    int64  `json:"nilai_akseptasi_sen"`
	Chronology       string `json:"kronologi"`
	Notes            string `json:"catatan"`
	AcceptanceStatus string `json:"status_akseptasi"`
	AcceptedNo       string `json:"nomor_akseptasi"`

	// Komite: nomor kasus (CASEIDKOMITE), tanggal transfer, dan tanggal putusan akhir.
	CommitteeID            string `json:"komite_id,omitempty"`
	CommitteeTransferredAt string `json:"tanggal_transfer_komite,omitempty"`
	CommitteeDecidedAt     string `json:"tanggal_putusan_komite,omitempty"`
}

func settlementDTO(lines []registrasi.SettlementLine) []SettlementDTO {
	if len(lines) == 0 {
		return nil
	}
	out := make([]SettlementDTO, 0, len(lines))
	for _, s := range lines {
		out = append(out, settlementLineDTO(s))
	}
	return out
}

func settlementLineDTO(s registrasi.SettlementLine) SettlementDTO {
	return SettlementDTO{
		PaymentType: s.PaymentType, PaymentTypeName: registrasi.PaymentTypeName(s.PaymentType),
		Currency: s.Currency, RateE4: int64(s.Rate), ProposeCents: int64(s.Propose), SubmittedCents: int64(s.Submitted),
		SalvageBCents: int64(s.SalvageB), InterimCents: int64(s.Interim), EstimationCents: int64(s.Estimation),
		LOC: int64(s.LOC), SalvageACents: int64(s.SalvageA), RiskType: s.RiskType,
		RiskPercent: int64(s.RiskPercent), RiskValueCents: int64(s.RiskValue),
		GrossCents: int64(s.Gross), ShareASM: int64(s.ShareASM), ValueCents: int64(s.Value),
		AcceptedCents: int64(s.Accepted), Chronology: s.Chronology, Notes: s.Notes,
		AcceptanceStatus: s.AcceptanceStatus, AcceptedNo: s.AcceptedNo,
		CommitteeID: s.CommitteeCaseID, CommitteeTransferredAt: formatDate(s.CommitteeTransferredAt),
		CommitteeDecidedAt: formatDate(s.CommitteeDecidedAt),
	}
}

// SettlementRequest adalah badan tombol Tambah. Objek dan jaminan berbasis 1.
type SettlementRequest struct {
	TaskID          string `json:"tugas_id"`
	Object          int    `json:"objek"`
	Coverage        int    `json:"jaminan"`
	PaymentType     string `json:"tipe_pembayaran"`
	Currency        string `json:"mata_uang"`
	ProposeCents    int64  `json:"nilai_propose_sen"`
	SubmittedCents  int64  `json:"nilai_pengajuan_sen"`
	LOC             int64  `json:"loc"`
	SalvageACents   int64  `json:"nilai_salvage_sen"`
	SalvageBCents   int64  `json:"nilai_salvage_b_sen"`
	RiskType        string `json:"tipe_resiko"`
	RiskPercent     int64  `json:"persen_resiko"`
	RiskValueCents  int64  `json:"nilai_resiko_sen"`
	ProfessionalFee int64  `json:"professional_fee_sen"`
	SurveyExpenses  int64  `json:"survey_expenses_sen"`
	VAT             int64  `json:"vat"`
	VATType         string `json:"tipe_vat"`
	Chronology      string `json:"kronologi"`
	Notes           string `json:"catatan"`
}

func (b SettlementRequest) command(claimID string) usecase.SettlementCommand {
	return usecase.SettlementCommand{
		ClaimID: claimID, TaskID: b.TaskID, Object: b.Object, Coverage: b.Coverage,
		Input: registrasi.SettlementInput{
			PaymentType: b.PaymentType, Currency: b.Currency,
			Propose: registrasi.Money(b.ProposeCents), Submitted: registrasi.Money(b.SubmittedCents),
			LOC: registrasi.Percent(b.LOC), SalvageB: registrasi.Money(b.SalvageBCents),
			RiskValue: registrasi.Money(b.RiskValueCents),
			SalvageA:  registrasi.Money(b.SalvageACents), RiskType: b.RiskType,
			RiskPercent: registrasi.Percent(b.RiskPercent),
			Fee: registrasi.AdjusterFee{
				ProfessionalFee: registrasi.Money(b.ProfessionalFee),
				SurveyExpenses:  registrasi.Money(b.SurveyExpenses),
				VAT:             registrasi.Percent(b.VAT),
				VATType:         b.VATType,
			},
			Chronology: b.Chronology, Notes: b.Notes,
		},
	}
}

// AddSettlement menangani POST …/klaim/{klaimID}/adjustment — tombol Tambah grid Adjustment.
func (h *Handler) AddSettlement(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SettlementRequest
	if !h.readBody(w, r, &body) {
		return
	}
	claim, err := h.service.AddSettlement(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}

// SettlementPreviewResponse adalah hasil hitungan baris yang belum disimpan, beserta
// spreading jaminan (tabel Tipe Treaty / Pembagian Persentase pada InputAdjustment).
type SettlementPreviewResponse struct {
	Line      SettlementDTO  `json:"adjustment"`
	Spreading []SpreadingDTO `json:"spreading"`
}

// PreviewSettlement menangani POST …/klaim/{klaimID}/adjustment/hitung — menghitung ulang
// nilai tampilan setiap isian berubah, tanpa menyimpan.
func (h *Handler) PreviewSettlement(w http.ResponseWriter, r *http.Request, claimID string) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body SettlementRequest
	if !h.readBody(w, r, &body) {
		return
	}
	preview, err := h.service.PreviewSettlement(r.Context(), body.command(claimID), caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	out := SettlementPreviewResponse{Line: settlementLineDTO(preview.Line), Spreading: make([]SpreadingDTO, 0, len(preview.Spreading))}
	for _, s := range preview.Spreading {
		if s.Removed {
			continue
		}
		out.Spreading = append(out.Spreading, SpreadingDTO{TreatyKind: s.TreatyKind, Name: s.Name, Share: Percent(s.Share)})
	}
	h.writeResponse(w, r, http.StatusOK, out)
}
