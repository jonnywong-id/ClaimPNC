package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// acceptedWithSpreading membawa klaim Fire berspreading tertentu sampai diakseptasi.
func (l environment) acceptedWithSpreading(t *testing.T, spreading []usecase.SpreadingInput) registrasi.Task {
	t.Helper()
	ctx := context.Background()
	_, reg := l.upToInputRegister(t, firePolicy)
	input := validInput(reg.ID)
	input.InsuredItem[0].Coverage[0].Spreading = spreading
	result, err := l.service.SaveRegister(ctx, input, l.caller)
	require.NoError(t, err)
	task := *result.NextTask

	command := oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1)
	_, err = l.service.SaveEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	next, err := l.service.CompleteEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	task = *next.NextTask

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)
	transferred, err := l.service.TransferCommittee(ctx, transfer(task), l.caller)
	require.NoError(t, err)
	for _, c := range []usecase.Caller{committee1, committee2} {
		_, err = l.service.DecideCommittee(ctx, decide(transferred.Committee.ID, registrasi.DecisionApprove, ""), c)
		require.NoError(t, err)
	}
	_, err = l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, ReceiverID: "1", AccountNo: "9876543210", Email: "a@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.Policy.BusinessCode = "10140"
	require.NoError(t, l.store.Save(ctx, claim))
	_, err = l.service.AcceptSettlement(ctx, acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err)
	return task
}

// Spreading BPPDAN, treaty, dan fac out masing-masing diproses lewat jalurnya sendiri;
// spreading yang dihapus diabaikan.
func TestListDLARoutesSpreading(t *testing.T) {
	l := setup(t)
	l.dla.Cases["10008"] = "2" // treaty
	l.dla.Cases["10009"] = "3" // fac out
	l.dla.Treaties["10008"] = registrasi.TreatyArrangement{
		Reinsurers: []registrasi.TreatyReinsurer{{PctShare: "100", Name: "REAS TREATY", ID: "R1"}},
	}
	task := l.acceptedWithSpreading(t, []usecase.SpreadingInput{
		{TreatyKind: "10010", Name: "BPPDAN", Share: 200_000},
		{TreatyKind: "10008", Name: "Treaty", Share: 300_000},
		{TreatyKind: "10009", Name: "Fac", Share: 500_000},
		{TreatyKind: "10007", Name: "OR", Share: 100_000, Removed: true},
	})
	l.dla.Policies[firePolicy] = registrasi.DLAPolicy{StartYear: 2026}

	list, err := l.service.ListDLA(context.Background(), dlaCommand(task), l.caller)
	require.NoError(t, err)
	recipients := map[string]bool{}
	for _, d := range list.DLA {
		recipients[d.Recipient] = true
	}
	require.True(t, recipients["BPPDAN"], "DLA = %+v", list.DLA)
	// Treaty tanpa limit dan fac out tanpa Fac Offer polis tidak menghasilkan baris DLA.
	require.False(t, recipients["REAS TREATY"], "DLA = %+v", list.DLA)
	require.Len(t, list.DLA, 1)
	require.Equal(t, list.Issued, len(list.DLA))
}

// Pre DLA yang sudah ada disalin menjadi DLA bernomor sama, tanpa perhitungan baru.
func TestListDLACopiesPreDLA(t *testing.T) {
	l := setup(t)
	task, _ := l.acceptedForDLA(t)
	claim, err := l.store.Get(context.Background(), task.ClaimID)
	require.NoError(t, err)
	object := claim.InsuredItem[0]
	l.dla.Pre = []registrasi.DLA{{
		ClaimID: claim.ID, ObjectID: object.ID, CoverageSeq: 1, AdjustmentSeq: 1,
		Number: "PRE-1", Recipient: "PRA PENERIMA", RecipientCode: "55", Printed: true, Sent: true,
	}}
	list, err := l.service.ListDLA(context.Background(), dlaCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 1, list.Issued)
	require.Len(t, list.DLA, 1)
	d := list.DLA[0]
	require.Equal(t, "PRE-1", d.Number)
	require.False(t, d.Printed)
	require.False(t, d.Sent)
	require.NotEmpty(t, d.AcceptedNo)
}
