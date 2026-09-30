package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Penyetuju tetap mode memori: KOMITE01 lalu KOMITE02.
var (
	committee1 = usecase.Caller{Identity: "KOMITE01"}
	committee2 = usecase.Caller{Identity: "KOMITE02"}
)

// transferredClaim membawa klaim ke Choose Surveyor, menambah satu adjustment Final, lalu
// mentransfernya ke komite.
func (l environment) transferredClaim(t *testing.T) (registrasi.Task, usecase.CommitteeTransferResult) {
	t.Helper()
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	_, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)
	result, err := l.service.TransferCommittee(ctx, transfer(task), l.caller)
	require.NoError(t, err)
	return task, result
}

func transfer(task registrasi.Task) usecase.CommitteeTransferCommand {
	return usecase.CommitteeTransferCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1}
}

// Transfer Komite membentuk kasus berjenjang, membekukan baris (CASEIDKOMITE, status 0), dan
// Status Klaim menjadi 1149 Claim Committee.
func TestTransferCommitteeCreatesCaseAndFreezesLine(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, result := l.transferredClaim(t)

	c := result.Committee
	require.Equal(t, "KMTN.26.1", c.ID)
	require.Equal(t, task.ClaimID, c.ClaimID)
	require.Equal(t, 1, c.CoverageSeq)
	require.Equal(t, 1, c.AdjustmentSeq)
	require.NotEmpty(t, c.ObjectID)
	require.Equal(t, registrasi.CommitteeLineNonMBUAB, c.Line, "Rp 9 jt Non-MBU masuk grup A/B")
	require.Equal(t, testOperator, c.Applicant)
	require.Equal(t, registrasi.CommitteeCaseOpen, c.Status())
	require.Equal(t, 1, c.Level())
	require.Len(t, c.Members, 2)
	require.Equal(t, "KOMITE01", c.Members[0].Operator)
	require.Equal(t, 1, c.Members[0].Level)
	require.Equal(t, 2, c.Members[1].Level)
	for _, m := range c.Members {
		require.Equal(t, registrasi.DecisionPending, m.Decision)
		require.Equal(t, registrasi.CommitteeCaseOpen, m.CaseStatus)
		require.Equal(t, registrasi.CommitteeTransferType, m.TransferType)
		require.Equal(t, registrasi.Rupiah(9_000_000), m.Value, "nilai pembanding = nilai ASM × kurs")
	}

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	line := stored.InsuredItem[0].Coverage[0].Settlement[0]
	require.Equal(t, "KMTN.26.1", line.CommitteeCaseID)
	require.Equal(t, registrasi.DecisionPending, line.AcceptanceStatus)
	require.False(t, line.CommitteeTransferredAt.IsZero())
	require.Equal(t, registrasi.StatusClaimCommittee, stored.ClaimStatus)

	_, err = l.service.TransferCommittee(ctx, transfer(task), l.caller)
	violation(t, err, registrasi.ViolationCommitteeTransferred)
}

// Jenjang berikutnya baru menunggu setelah jenjang sebelumnya setuju; setuju di jenjang
// terakhir mengakseptasi adjustment dan mengosongkan Status Klaim.
func TestCommitteeApprovalWalksTheLevels(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, result := l.transferredClaim(t)
	id := result.Committee.ID

	_, err := l.service.DecideCommittee(ctx, decide(id, registrasi.DecisionApprove, ""), committee2)
	require.ErrorIs(t, err, registrasi.ErrNotCommitteeTurn, "jenjang 2 belum boleh memutuskan")

	pending, err := l.service.PendingCommittees(ctx, committee1)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, 2, pending[0].Levels)
	require.Equal(t, 1, pending[0].Adjustment)
	none, err := l.service.PendingCommittees(ctx, committee2)
	require.NoError(t, err)
	require.Empty(t, none)

	c, err := l.service.DecideCommittee(ctx, decide(id, registrasi.DecisionApprove, "ok jenjang 1"), committee1)
	require.NoError(t, err)
	require.Empty(t, c.Outcome(), "masih ada jenjang 2")
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.DecisionPending, stored.InsuredItem[0].Coverage[0].Settlement[0].AcceptanceStatus)

	pending, err = l.service.PendingCommittees(ctx, committee2)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	c, err = l.service.DecideCommittee(ctx, decide(id, registrasi.DecisionApprove, "disetujui"), committee2)
	require.NoError(t, err)
	require.Equal(t, registrasi.DecisionApprove, c.Outcome())
	for _, m := range c.Members {
		require.Equal(t, registrasi.CommitteeCaseClosed, m.CaseStatus)
	}

	stored, err = l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	line := stored.InsuredItem[0].Coverage[0].Settlement[0]
	require.Equal(t, registrasi.DecisionApprove, line.AcceptanceStatus)
	require.Equal(t, "disetujui", line.Notes)
	require.False(t, line.CommitteeDecidedAt.IsZero())
	require.Empty(t, stored.ClaimStatus)

	_, err = l.service.DecideCommittee(ctx, decide(id, registrasi.DecisionApprove, ""), committee2)
	require.ErrorIs(t, err, registrasi.ErrNotCommitteeTurn, "kasus yang selesai tidak menerima putusan")
}

// Tolak di jenjang mana pun menghentikan seluruh komite (`KomitePost_Adjustment` step 31).
func TestCommitteeRejectionStopsAllLevels(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, result := l.transferredClaim(t)

	c, err := l.service.DecideCommittee(ctx, decide(result.Committee.ID, registrasi.DecisionReject, "tidak dijamin"), committee1)
	require.NoError(t, err)
	require.Equal(t, registrasi.DecisionReject, c.Outcome())
	require.Equal(t, registrasi.DecisionReject, c.Members[1].Decision)

	pending, err := l.service.PendingCommittees(ctx, committee2)
	require.NoError(t, err)
	require.Empty(t, pending)

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.DecisionReject, stored.InsuredItem[0].Coverage[0].Settlement[0].AcceptanceStatus)
}

func TestCommitteeDecisionRejectsUnknownDecision(t *testing.T) {
	l := setup(t)
	_, result := l.transferredClaim(t)
	_, err := l.service.DecideCommittee(context.Background(), decide(result.Committee.ID, "9", ""), committee1)
	require.True(t, errors.Is(err, registrasi.ErrInvalidAction))
}

func decide(id, decision, note string) usecase.CommitteeDecisionCommand {
	return usecase.CommitteeDecisionCommand{CaseID: id, Decision: decision, Note: note}
}

// SetEmailKomite step 4 dan 15–19: lini komite dari jenis bisnis polis dan nilainya.
func TestCommitteeLineFollowsBusinessType(t *testing.T) {
	big := registrasi.Rupiah(75_000_000)
	cases := map[string]registrasi.Policy{
		"BONDING": {BusinessType: "BondingKBG"},
		"TRAVEL":  {BusinessType: "Travel"},
		"PA":      {BusinessType: "PA"},
		"NONMBU":  {BusinessType: "Fire"},
	}
	for want, p := range cases {
		require.Equal(t, want, registrasi.CommitteeLine(p, big), p.BusinessType)
	}
	require.Equal(t, "BONDING", registrasi.CommitteeLine(registrasi.Policy{BusinessType: "Fire", BusinessCode: "10168"}, big))
}

// Non-MBU sampai Rp 50.000.000 diputus komite grup A/B (NONMBUAB); satu rupiah di atasnya
// kembali ke tangga NONMBU. PA dan Travel tidak terpengaruh.
func TestCommitteeLineSendsSmallNonMBUToGroupAB(t *testing.T) {
	cargo := registrasi.Policy{BusinessType: "MarineCargo"}
	require.Equal(t, registrasi.CommitteeLineNonMBUAB, registrasi.CommitteeLine(cargo, registrasi.Rupiah(2_500_000)))
	require.Equal(t, registrasi.CommitteeLineNonMBUAB, registrasi.CommitteeLine(cargo, registrasi.Rupiah(50_000_000)))
	require.Equal(t, registrasi.CommitteeLineNonMBU, registrasi.CommitteeLine(cargo, registrasi.Rupiah(50_000_001)))
	require.Equal(t, "PA", registrasi.CommitteeLine(registrasi.Policy{BusinessType: "PA"}, registrasi.Rupiah(1)))
}

// SetListComiteeClaimPerObjAdj step 28–36: nilai pembanding ambang komite.
func TestCommitteeValueFollowsPaymentTypeAndCoinsurance(t *testing.T) {
	usd := registrasi.ExchangeRate(15_000 * 10_000) // 1 USD = Rp 15.000
	line := registrasi.SettlementLine{
		PaymentType: registrasi.PaymentFinal, Rate: usd,
		Propose: 1_000_00, Gross: 800_00, Value: 400_00, // USD 1.000 · 800 · 400
	}
	member := registrasi.Policy{Coinsurance: registrasi.Coinsurance{Role: "MEMBER"}}
	leader := registrasi.Policy{Coinsurance: registrasi.Coinsurance{Role: "LEADER"}}

	require.Equal(t, registrasi.Rupiah(6_000_000), registrasi.CommitteeValue(line, member), "bagian ASM × kurs")
	require.Equal(t, registrasi.Rupiah(12_000_000), registrasi.CommitteeValue(line, leader), "leader memakai gross")

	fee := registrasi.SettlementLine{PaymentType: registrasi.PaymentAdjusterFee, Rate: registrasi.ExchangeRate(10_000), Gross: registrasi.Rupiah(3_000_000), Value: registrasi.Rupiah(1)}
	require.Equal(t, registrasi.Rupiah(3_000_000), registrasi.CommitteeValue(fee, member), "fee adjuster memakai gross fee")

	reject := registrasi.SettlementLine{PaymentType: registrasi.PaymentReject, Rate: registrasi.ExchangeRate(10_000)}
	require.Equal(t, registrasi.Rupiah(500_000_001), registrasi.CommitteeValue(reject, member), "Tolak Klaim melewati ambang tertinggi")

	cents := registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, Rate: registrasi.ExchangeRate(10_000), Propose: 150, Value: 150}
	require.Equal(t, registrasi.Money(200), registrasi.CommitteeValue(cents, member), "dibulatkan ke atas ke rupiah penuh")
}

// Decide mengabaikan kapitalisasi operator dan mengisi waktu putusan.
func TestCommitteeCaseDecideMatchesOperatorCaseInsensitively(t *testing.T) {
	now := time.Date(2026, 9, 28, 3, 0, 0, 0, time.UTC)
	c := registrasi.NewCommitteeCase("KMTN.26.9", "PNCN.26.0001",
		[]registrasi.CommitteeApprover{{OperatorID: "KOMITE01"}}, registrasi.SettlementLine{PaymentType: "1"}, 100, now)
	require.NoError(t, c.Decide("komite01", registrasi.DecisionApprove, "", now))
	require.Equal(t, registrasi.DecisionApprove, c.Outcome())
	require.Equal(t, now, c.Members[0].DecidedAt)
}
