package usecase

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// AcceptanceNoteCommand adalah permintaan tombol PRINT di samping Nomor Akseptasi. Object,
// Coverage, dan Adjustment berbasis 1.
type AcceptanceNoteCommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int
}

// AcceptanceNoteResult adalah PDF Draft Persetujuan yang diunduh.
type AcceptanceNoteResult struct {
	FileName string
	Content  []byte
}

// PrintAcceptanceNote membentuk Draft Persetujuan satu adjustment (`PrintPDFAcceptanceNote`).
//
// Ia hanya MEMBACA: tidak ada kolom klaim yang ditulis, hanya jejak audit
// DRAFT_AKSEPTASI_CETAK. Langkah activity yang tidak dibawa tercatat di acceptancenote.go.
func (l *Service) PrintAcceptanceNote(ctx context.Context, p AcceptanceNoteCommand, by Caller) (AcceptanceNoteResult, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	if err := registrasi.CanPrintAcceptanceNote(*line, claim.Policy); err != nil {
		return AcceptanceNoteResult{}, err
	}
	object := claim.InsuredItem[p.Object-1]
	coverage := object.Coverage[p.Coverage-1]

	names, _, err := l.currencyNames(ctx)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	policyDoc, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	cause := ""
	if coverage.CauseOfLoss != "" {
		if cause, err = l.faceSheet.CauseOfLossName(ctx, coverage.CauseOfLoss); err != nil {
			return AcceptanceNoteResult{}, err
		}
	}
	qs, err := l.acceptanceNoteQS(ctx, claim, coverage, policyDoc)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	receiver, err := l.acceptanceNoteReceiver(ctx, claim, *line)
	if err != nil {
		return AcceptanceNoteResult{}, err
	}
	entity := plaEntity(claim.Portal)
	personalAccident := claim.Policy.Line == registrasi.LinePersonalAccident
	var paSigners []registrasi.AcceptanceNoteSigner
	if personalAccident {
		// SetSignaturePA: dua tanda tangan tetap per entitas dari POOLDATA.M_SIGNATURE1.
		for i, id := range registrasi.AcceptanceNotePASignatureIDs(entity) {
			signer := registrasi.AcceptanceNoteSigner{Title: registrasi.AcceptanceNotePATitles[i]}
			if id != "" {
				if signer.Name, signer.Signature, err = l.pla.PASignature(ctx, id); err != nil {
					return AcceptanceNoteResult{}, err
				}
			}
			paSigners = append(paSigners, signer)
		}
	}
	signerName := strings.TrimSpace(line.Acceptance.Form.CommitteeName)
	signerID := registrasi.AcceptanceNoteSignerID(signerName)
	var signature []byte
	if signerID != "" {
		if _, signature, err = l.pla.Signature(ctx, signerID); err != nil {
			return AcceptanceNoteResult{}, err
		}
	}

	spread, parts := registrasi.AcceptanceNoteSpreading(coverage.Spreading, line.Value, qs)
	note := registrasi.AcceptanceNote{
		Entity:           entity,
		PersonalAccident: personalAccident,
		PASigners:        paSigners,
		AcceptedNo:       line.AcceptedNo,
		PaymentLabel:     registrasi.AcceptanceNotePaymentLabel(line.PaymentType),

		PolicyNumber:    claim.Policy.Number,
		ClaimNumber:     claim.Number,
		InsuredName:     firstText(claim.Policy.QQName, claim.Policy.InsuredName),
		PolicyCurrency:  firstText(names[claim.Policy.Currency], claim.Policy.Currency),
		SumInsured:      coverage.TSI,
		LossLocation:    claim.Location,
		PolicyCondition: coverage.Name,
		ObjectName:      object.Name,
		PeriodStart:     claim.Policy.CoverageStart,
		PeriodEnd:       claim.Policy.CoverageEnd,
		DateOfLoss:      claim.DateOfLoss,
		NatureOfLoss:    cause,

		StatusBusiness: policyDoc.StatusBusiness,
		Installments:   l.acceptanceNoteInstallments(ctx, claim),

		Currency:    firstText(names[line.Currency], line.Currency),
		PaymentType: line.PaymentType,
		Gross:       line.Gross,
		Own:         line.Value,
		Spread:      spread,
		QS:          parts,

		Receiver: receiver,
		Remark:   line.Acceptance.Form.Remark,

		SignedAt:   line.Acceptance.AcceptedAt,
		SignerName: signerName,
		SignerID:   signerID,
		Signature:  signature,
	}
	content, err := l.acceptanceNoteRenderer.Render(note)
	if err != nil {
		return AcceptanceNoteResult{}, fmt.Errorf("registrasi/usecase: membentuk Draft Persetujuan: %w", err)
	}
	if err := l.audit.Record(ctx, registrasi.AuditTrail{
		ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "DRAFT_AKSEPTASI_CETAK", Actor: by.Identity,
		At: l.clock.Now().UTC(),
		Note: fmt.Sprintf("Draft Persetujuan %s — objek %d jaminan %d adjustment %d",
			line.AcceptedNo, p.Object, p.Coverage, p.Adjustment),
	}); err != nil {
		return AcceptanceNoteResult{}, err
	}
	return AcceptanceNoteResult{FileName: registrasi.AcceptanceNoteFileName(line.AcceptedNo), Content: content}, nil
}

// acceptanceNoteQS membaca rincian QS per kode treaty QS pada spreading jaminan (langkah
// 66–71): grup treaty dari BusinessCode polis dan tahun mulai polis, lalu searchQSReins2.
func (l *Service) acceptanceNoteQS(ctx context.Context, claim registrasi.Claim, coverage registrasi.Coverage,
	policy registrasi.DLAPolicy) (map[string][]registrasi.TreatyQSPart, error) {
	out := map[string][]registrasi.TreatyQSPart{}
	year := policy.StartYear
	if year == 0 && !claim.Policy.CoverageStart.IsZero() {
		year = clock.DateWIB(claim.Policy.CoverageStart).Year()
	}
	business := firstText(policy.BusinessCode, claim.Policy.BusinessCode)
	for _, s := range coverage.Spreading {
		if s.Removed || !registrasi.TreatyUsesQS(s.TreatyKind) {
			continue
		}
		if _, done := out[s.TreatyKind]; done {
			continue
		}
		arr, err := l.dla.Treaty(ctx, business, year, s.TreatyKind)
		if err != nil {
			return nil, err
		}
		out[s.TreatyKind] = arr.QSParts
	}
	return out, nil
}

// acceptanceNoteReceiver adalah TempReceiver (langkah 37): penerima yang dipilih pada akseptasi,
// bank dan cabangnya dari rekening masternya.
func (l *Service) acceptanceNoteReceiver(ctx context.Context, claim registrasi.Claim, line registrasi.SettlementLine) (registrasi.AcceptanceNoteReceiver, error) {
	var out registrasi.AcceptanceNoteReceiver
	for _, r := range claim.Receiver {
		if strings.TrimSpace(r.ID) == strings.TrimSpace(line.Acceptance.Form.ReceiverID) {
			out = registrasi.AcceptanceNoteReceiver{Name: r.Name, Bank: r.BankName, AccountNo: r.AccountNo}
		}
	}
	if out.Name == "" {
		out.Name = line.Acceptance.ReceiverName
	}
	if out.AccountNo == "" {
		return out, nil
	}
	account, err := l.accounts.FindAccount(ctx, out.AccountNo)
	if err != nil {
		if errors.Is(err, registrasi.ErrAccountNotFound) {
			return out, nil
		}
		return out, err
	}
	out.Branch = strings.TrimSpace(account.Branch)
	if out.Bank == "" {
		out.Bank = strings.TrimSpace(account.BankName)
	}
	return out, nil
}

// acceptanceNoteInstallments adalah baris Premium Paid On: cicilan layanan premi yang
// PaymentAmount-nya > 0. Layanan yang tidak dapat dihubungi TIDAK menggagalkan cetak — Pega
// mencetak dari clipboard, dan cicilan kosong berarti bagian itu kosong.
func (l *Service) acceptanceNoteInstallments(ctx context.Context, claim registrasi.Claim) []registrasi.AcceptanceNoteInstallment {
	caseID, err := l.acceptance.PolicyCaseID(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return nil
	}
	statement, err := l.premium.Statement(ctx, portalOf("", claim.Portal), registrasi.PremiumQuery{
		PolicyNumber: claim.Policy.Number, ProdKe: claim.Policy.ProdKe,
		RequestedAt: claim.Policy.CoverageEnd, CaseID: registrasi.PremiumCaseID(caseID),
	})
	if err != nil {
		return nil
	}
	var out []registrasi.AcceptanceNoteInstallment
	for _, i := range statement.Installments {
		amount, ok := new(big.Rat).SetString(strings.ReplaceAll(strings.TrimSpace(i.PaymentAmount), ",", ""))
		if !ok || amount.Sign() <= 0 {
			continue
		}
		out = append(out, registrasi.AcceptanceNoteInstallment{
			Number: i.Number, PaidAt: i.PaidAt, Paid: i.PaymentDate, Amount: amount,
		})
	}
	return out
}
