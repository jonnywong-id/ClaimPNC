package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

	// ReceiverID adalah Penerima Klaim modal "Transfer Claim ke Komite" (`.TempReceiver`,
	// lini Travel). Kosong berarti TEMPRECEIVER jaminan yang tersimpan.
	ReceiverID string
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

	// ValidationTypePayment step 9–12 dan setValidasiReceiverClaim_act (lini Travel).
	msgTransferReceiver        = "Receiver Claim harus di isi"
	msgTransferReceiverBank    = "Nama Bank Belum Di isi"
	msgTransferReceiverAccount = "No Rekening Belum Di isi"
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
		return CommitteeTransferResult{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, task.Stage)
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
	if err := checkTravelReceiver(claim, p); err != nil {
		return CommitteeTransferResult{}, err
	}

	value := registrasi.CommitteeValue(*line, claim.Policy)
	businessLine := registrasi.CommitteeLine(claim.Policy, value)
	route, err := l.tiering.Route(ctx, businessLine, value, by.Identity)
	if err != nil {
		return CommitteeTransferResult{}, fmt.Errorf("registrasi/usecase: menghitung penjenjangan komite: %w", err)
	}
	if len(route.Approvers) == 0 {
		return CommitteeTransferResult{}, transferViolation(registrasi.ViolationCommitteeNoApprover,
			fmt.Sprintf("%s (no active committee member in Master Komite for %s at IDR %s, excluding %s).",
				msgTransferNoApprover, businessLine, rupiahText(value), by.Identity))
	}

	now := l.clock.Now().UTC()
	var committee registrasi.CommitteeCase
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		id, err := l.committees.NextCaseID(ctx, now)
		if err != nil {
			return err
		}
		committee = registrasi.NewCommitteeCase(id, claim.Number, route.Approvers, *line, value, now)
		committee.ClaimID = claim.ID
		committee.ObjectID = claim.InsuredItem[p.Object-1].ID
		committee.CoverageSeq, committee.AdjustmentSeq = p.Coverage, p.Adjustment
		committee.Line, committee.Band = businessLine, route.Band
		committee.Applicant, committee.CreatedBy, committee.UpdatedBy = by.Identity, by.Identity, by.Identity
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
			Note: fmt.Sprintf("Adjustment Transfer To Committee — objek %d jaminan %d adjustment %d, %s, %s, %d jenjang",
				p.Object, p.Coverage, p.Adjustment, id, businessLine, len(route.Approvers)),
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

// checkTravelReceiver memeriksa Penerima Klaim lini Travel sebelum transfer —
// `ValidationTypePayment` step 6 dan 9–12 (`setValidasiReceiverClaim_act` menonaktifkan tombolnya
// lebih dulu di layar): penerima yang dipilih (`.TempReceiver`) harus ada dan data banknya
// lengkap, karena pembayarannya kelak dikirim ke kasir.
//
// Satu pemeriksaan Pega tidak dibawa: "Cabang Bank Belum Di isi" (`.BranchOfBank`) — kolomnya
// tidak ada di POOLDATA.T_CLAIM_RECEIVER.
func checkTravelReceiver(claim registrasi.Claim, p CommitteeTransferCommand) error {
	if claim.Policy.Line != registrasi.LineTravel && claim.Policy.BusinessType != "Travel" {
		return nil
	}
	id := strings.TrimSpace(p.ReceiverID)
	if id == "" {
		id = strings.TrimSpace(claim.InsuredItem[p.Object-1].Coverage[p.Coverage-1].Committee.Receiver)
	}
	reject := func(msg string) error {
		return &registrasi.ValidationError{Violation: []registrasi.Violation{{
			Code: registrasi.ViolationCommitteeIncomplete, Field: "penerima_klaim", Message: msg,
		}}}
	}
	if id == "" {
		return reject(msgTransferReceiver)
	}
	for _, r := range claim.Receiver {
		if strings.TrimSpace(r.ID) != id {
			continue
		}
		var v []registrasi.Violation
		if strings.TrimSpace(r.BankName) == "" {
			v = append(v, registrasi.Violation{Code: registrasi.ViolationCommitteeIncomplete, Field: "penerima_klaim", Message: msgTransferReceiverBank})
		}
		if strings.TrimSpace(r.AccountNo) == "" {
			v = append(v, registrasi.Violation{Code: registrasi.ViolationCommitteeIncomplete, Field: "penerima_klaim", Message: msgTransferReceiverAccount})
		}
		if len(v) > 0 {
			return &registrasi.ValidationError{Violation: v}
		}
		return nil
	}
	return reject(msgTransferReceiver)
}

// rupiahText menulis nilai sen sebagai rupiah bulat berpemisah titik: 2.500.000.
func rupiahText(v registrasi.Money) string {
	digits := fmt.Sprintf("%d", int64(v)/100)
	var out []byte
	for i, d := range []byte(digits) {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, d)
	}
	return string(out)
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
		// Kasus komite survey (TransferType 1) diputus lewat link email Pega KomiteAcceptSurvey
		// yang belum dibangun — keputusan Work Owner 2026-10-11.
		if strings.TrimSpace(c.TransferType) == registrasi.SurveyCommitteeTransfer {
			return registrasi.ErrSurveyCommitteeAwaiting
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
		c.UpdatedAt, c.UpdatedBy = now, by.Identity
		if outcome := c.Outcome(); outcome != "" {
			c.DecidedAt = now
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
		if err := l.committeeProgress(ctx, claim.Number, c.Outcome(), by, now); err != nil {
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

// committeeProgress menulis progres putusan akhir komite (`KomitePost_Adjustment`).
//
//   - Setuju di jenjang terakhir — langkah 19–21: buka posisi AKSEPTASI "On Progress"
//     ("Auto Create AKSEPTASI", 014/60), yang kelak ditutup Submit akseptasi; langkah 22–23:
//     tutup posisi KOMITE ("Auto Finish KOMITE", 006/24, Done).
//   - Tolak — langkah 61–62: tutup posisi KOMITE ("Auto Reject KOMITE", 006/24, Done).
//
// Posisi KOMITE dibuka Pega saat Transfer Komite (`SetChildKomitePerAdjustment_act` langkah 25),
// yang belum dibangun di aplikasi ini; bila tidak ada posisi KOMITE terbuka, penutupannya
// dilewati — POSISIID wajib terisi.
func (l *Service) committeeProgress(ctx context.Context, claimNumber, outcome string, by Caller, now time.Time) error {
	if outcome == "" {
		return nil // jenjang berikutnya masih menunggu
	}
	if outcome == registrasi.DecisionApprove {
		if _, err := l.acceptance.StartProgress(ctx, registrasi.ProgressStart{
			ClaimNumber: claimNumber, CaseID: claimNumber, Position: registrasi.PositionAcceptance,
			Note: registrasi.CommitteeAcceptNote, Progress1: registrasi.AcceptanceProgress1,
			Progress2: registrasi.AcceptanceProgress2, User: by.Identity, At: now,
		}); err != nil {
			return err
		}
	}
	committee, err := l.acceptance.OpenPosition(ctx, claimNumber, registrasi.PositionCommittee)
	if err != nil || committee == "" {
		return err
	}
	note := registrasi.CommitteeFinishNote
	if outcome == registrasi.DecisionReject {
		note = registrasi.CommitteeRejectNote
	}
	return l.acceptance.AddProgress(ctx, registrasi.ProgressUpdate{
		ClaimNumber: claimNumber, PositionID: committee, Note: note,
		Progress1: registrasi.CommitteeProgress1, Progress2: registrasi.CommitteeProgress2,
		Position: registrasi.AcceptanceProgressDone, User: by.Identity, At: now,
	})
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
