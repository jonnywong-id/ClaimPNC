package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// CommitteeTransferCommand adalah tombol Transfer Komite pada satu baris Adjustment. Object,
// Coverage, dan Adjustment berbasis 1.
type CommitteeTransferCommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int
}

// CommitteeTransferResult adalah klaim sesudah transfer beserta kasus komitenya.
type CommitteeTransferResult struct {
	Claim     registrasi.Claim
	Committee registrasi.CommitteeCase
}

// Pesan pemeriksaan Transfer Komite — `ValidationTypePaymentAdj` step 2, dan
// `SetListComiteeClaimPerObjAdj` step 63 (`local.ErrKom`) bila tidak ada penyetuju.
const (
	msgTransferPaymentType = "Silahkan Pilih Tipe Pembayaran"
	msgTransferPropose     = "Nilai Proposed Adjustment Harus Diisi"
	msgTransferAdjusterFee = "Nilai Professional Fee, Survey Expenses, dan VAT Harus Diisi"
	msgTransferNoApprover  = "Error Case Komite tidak kebuat. Silakan transfer ulang"
	msgTransferDone        = "This adjustment has already been transferred to committee."
)

// TransferCommittee memindahkan satu baris adjustment ke komite (`ValidationTypePaymentAdj`
// → `SetListComiteeClaimPerObjAdj`).
//
// Penjaganya sama dengan tombol Tambah di grid yang sama. Baris dibekukan: begitu
// ditransfer ia tidak dapat ditransfer ulang (`IsKomiteTransfer := 1`), status akseptasinya
// 0, dan Status Klaim menjadi 1149 Claim Committee.
func (l *Service) TransferCommittee(ctx context.Context, p CommitteeTransferCommand, by Caller) (CommitteeTransferResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputSurveyor})
	if err != nil {
		return CommitteeTransferResult{}, err
	}
	if claim.ID != p.ClaimID {
		return CommitteeTransferResult{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !settlementStages[task.Stage] {
		return CommitteeTransferResult{}, registrasi.ErrStageMismatch
	}
	if !l.canWork(task, by) {
		return CommitteeTransferResult{}, registrasi.ErrNotTaskOwner
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return CommitteeTransferResult{}, err
	}
	if err := checkTransfer(*line); err != nil {
		return CommitteeTransferResult{}, err
	}

	value := registrasi.CommitteeValue(*line, claim.Policy)
	approvers, err := l.tiering.Approvers(ctx, registrasi.CommitteeLine(claim.Policy), value, by.Identity)
	if err != nil {
		return CommitteeTransferResult{}, fmt.Errorf("registrasi/usecase: menghitung penjenjangan komite: %w", err)
	}
	if len(approvers) == 0 {
		return CommitteeTransferResult{}, transferViolation(registrasi.ViolationCommitteeNoApprover, msgTransferNoApprover)
	}

	now := l.clock.Now().UTC()
	var committee registrasi.CommitteeCase
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		id, err := l.committees.NextCaseID(ctx)
		if err != nil {
			return err
		}
		committee = registrasi.NewCommitteeCase(id, claim.Number, approvers, *line, value, now)
		line.CommitteeCaseID = id
		line.CommitteeTransferredAt = now
		line.AcceptanceStatus = registrasi.DecisionPending
		claim.ClaimStatus = registrasi.StatusClaimCommittee
		claim.UpdatedBy, claim.UpdatedAt = by.Identity, now

		if err := l.committees.Save(ctx, committee); err != nil {
			return err
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "ADJUSTMENT_TRANSFER_KOMITE",
			Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Adjustment Transfer To Committee — objek %d jaminan %d adjustment %d, %s, %d jenjang",
				p.Object, p.Coverage, p.Adjustment, id, len(approvers)),
		})
	})
	if err != nil {
		return CommitteeTransferResult{}, err
	}
	return CommitteeTransferResult{Claim: claim, Committee: committee}, nil
}

// checkTransfer memeriksa baris seperti `ValidationTypePaymentAdj` step 16–28: tipe
// pembayaran, Total Klaim untuk tipe berbasis propose, dan fee untuk fee adjuster.
func checkTransfer(line registrasi.SettlementLine) error {
	if line.Transferred() {
		return transferViolation(registrasi.ViolationCommitteeTransferred, msgTransferDone)
	}
	switch line.PaymentType {
	case "":
		return transferViolation(registrasi.ViolationCommitteeIncomplete, msgTransferPaymentType)
	case registrasi.PaymentAdjusterFee:
		if line.Gross == 0 {
			return transferViolation(registrasi.ViolationCommitteeIncomplete, msgTransferAdjusterFee)
		}
	case registrasi.PaymentReject, registrasi.PaymentSalvage:
	default:
		if line.Propose == 0 {
			return transferViolation(registrasi.ViolationCommitteeIncomplete, msgTransferPropose)
		}
	}
	return nil
}

func transferViolation(code registrasi.ViolationCode, message string) error {
	return &registrasi.ValidationError{Violation: []registrasi.Violation{{Code: code, Field: "adjustment", Message: message}}}
}

// settlementAt menunjuk satu baris adjustment klaim; indeks berbasis 1.
func settlementAt(claim *registrasi.Claim, object, coverage, adjustment int) (*registrasi.SettlementLine, error) {
	if object < 1 || object > len(claim.InsuredItem) {
		return nil, fmt.Errorf("%w: objek %d tidak ada", registrasi.ErrInvalidAction, object)
	}
	item := &claim.InsuredItem[object-1]
	if coverage < 1 || coverage > len(item.Coverage) {
		return nil, fmt.Errorf("%w: jaminan %d tidak ada", registrasi.ErrInvalidAction, coverage)
	}
	cov := &item.Coverage[coverage-1]
	if adjustment < 1 || adjustment > len(cov.Settlement) {
		return nil, fmt.Errorf("%w: adjustment %d tidak ada", registrasi.ErrInvalidAction, adjustment)
	}
	return &cov.Settlement[adjustment-1], nil
}

// settlementOfCommittee menunjuk baris adjustment yang CASEIDKOMITE-nya kasus itu.
func settlementOfCommittee(claim *registrasi.Claim, caseID string) (object, coverage, adjustment int, line *registrasi.SettlementLine) {
	for i := range claim.InsuredItem {
		for j := range claim.InsuredItem[i].Coverage {
			lines := claim.InsuredItem[i].Coverage[j].Settlement
			for k := range lines {
				if strings.EqualFold(strings.TrimSpace(lines[k].CommitteeCaseID), strings.TrimSpace(caseID)) {
					return i + 1, j + 1, k + 1, &lines[k]
				}
			}
		}
	}
	return 0, 0, 0, nil
}

// CommitteeItem adalah satu putusan komite yang menunggu pemanggil, beserta konteksnya.
type CommitteeItem struct {
	Member      registrasi.CommitteeMember
	Levels      int
	ClaimID     string
	PolicyNo    string
	InsuredName string
	ObjectName  string
	Coverage    string
	Adjustment  int
	Line        registrasi.SettlementLine
}

// PendingCommittees mengembalikan putusan komite klaim PNCN yang menunggu pemanggil.
func (l *Service) PendingCommittees(ctx context.Context, by Caller) ([]CommitteeItem, error) {
	members, err := l.committees.Pending(ctx, by.Identity)
	if err != nil {
		return nil, err
	}
	items := make([]CommitteeItem, 0, len(members))
	for _, m := range members {
		item, err := l.committeeItem(ctx, m)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (l *Service) committeeItem(ctx context.Context, m registrasi.CommitteeMember) (CommitteeItem, error) {
	item := CommitteeItem{Member: m}
	if c, err := l.committees.Get(ctx, m.CaseID); err == nil {
		item.Levels = len(c.Members)
	} else if !errors.Is(err, registrasi.ErrCommitteeNotFound) {
		return CommitteeItem{}, err
	}
	claim, err := l.claim.GetByNumber(ctx, m.ClaimNumber)
	if errors.Is(err, registrasi.ErrClaimNotFound) {
		return item, nil
	}
	if err != nil {
		return CommitteeItem{}, err
	}
	item.ClaimID, item.PolicyNo, item.InsuredName = claim.ID, claim.Policy.Number, claim.Policy.InsuredName
	if o, c, a, line := settlementOfCommittee(&claim, m.CaseID); line != nil {
		item.ObjectName = claim.InsuredItem[o-1].Name
		item.Coverage = claim.InsuredItem[o-1].Coverage[c-1].Name
		item.Adjustment = a
		item.Line = *line
	}
	return item, nil
}

// Committee membaca satu kasus komite — status per jenjang pada grid Adjustment.
func (l *Service) Committee(ctx context.Context, caseID string) (registrasi.CommitteeCase, error) {
	return l.committees.Get(ctx, caseID)
}

// CommitteeDecisionCommand adalah putusan satu anggota komite.
type CommitteeDecisionCommand struct {
	CaseID   string
	Decision string // registrasi.DecisionApprove atau DecisionReject
	Note     string
}

// DecideCommittee mencatat putusan anggota yang sedang ditunggu (`KomitePost_Adjustment`).
//
// Setuju di jenjang terakhir mengakseptasi adjustment (STATUSAKSEPTASI 1); tolak di jenjang
// mana pun menolaknya (STATUSAKSEPTASI 2). Keduanya mengosongkan Status Klaim dan menyalin
// catatan komite ke Notes adjustment (step 18, 28, 32). Nomor akseptasi belum terbit di sini.
func (l *Service) DecideCommittee(ctx context.Context, p CommitteeDecisionCommand, by Caller) (registrasi.CommitteeCase, error) {
	now := l.clock.Now().UTC()
	var result registrasi.CommitteeCase
	err := l.unit.Run(ctx, func(ctx context.Context) error {
		c, err := l.committees.Get(ctx, p.CaseID)
		if err != nil {
			return err
		}
		current, _ := c.Current()
		if err := c.Decide(by.Identity, p.Decision, p.Note, now); err != nil {
			return err
		}
		claim, err := l.claim.GetByNumber(ctx, c.ClaimNumber)
		if err != nil {
			return err
		}
		_, _, _, line := settlementOfCommittee(&claim, c.ID)
		if line == nil {
			return fmt.Errorf("%w: adjustment kasus komite %s tidak ditemukan pada klaim %s", registrasi.ErrInvalidAction, c.ID, claim.Number)
		}
		if outcome := c.Outcome(); outcome != "" {
			line.AcceptanceStatus = outcome
			line.CommitteeDecidedAt = now
			if note := strings.TrimSpace(p.Note); note != "" {
				line.Notes = note
			}
			claim.ClaimStatus = ""
		}
		claim.UpdatedBy, claim.UpdatedAt = by.Identity, now

		if err := l.committees.Save(ctx, c); err != nil {
			return err
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		event := "KOMITE_SETUJU"
		if p.Decision == registrasi.DecisionReject {
			event = "KOMITE_TOLAK"
		}
		result = c
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: event, Actor: by.Identity, At: now,
			Note: fmt.Sprintf("%s jenjang %d dari %d%s", c.ID, current.Level, len(c.Members), outcomeNote(c.Outcome())),
		})
	})
	if err != nil {
		return registrasi.CommitteeCase{}, err
	}
	return result, nil
}

func outcomeNote(outcome string) string {
	switch outcome {
	case registrasi.DecisionApprove:
		return " — Accept Adjustment by Committee"
	case registrasi.DecisionReject:
		return " — Reject Adjustment by Committee"
	}
	return ""
}
