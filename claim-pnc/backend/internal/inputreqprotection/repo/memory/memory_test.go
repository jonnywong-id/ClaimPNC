package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
	"claim-pnc/internal/inputreqprotection/repo/memory"
)

var wib = time.FixedZone("WIB", 7*60*60)

func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestSampleRepoListsAllPendingRowsNewestFirst(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	page, err := repo.List(context.Background(), inputreqprotection.Filter{})
	require.NoError(t, err)
	require.Equal(t, 6, page.Total)
	require.Len(t, page.Protections, 6)

	// Diurutkan menurun menurut waktu pembuatan.
	require.Equal(t, "OPCN.26.0005", page.Protections[0].Number)
	require.Equal(t, "OPC-201", page.Protections[5].Number)
}

func TestListTieBreaksOnNumberDescending(t *testing.T) {
	// Dua baris dengan waktu pembuatan identik diurutkan menurut nomor, menurun.
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	repo := memory.NewRepo()
	repo.Add(
		inputreqprotection.Protection{Number: "OPCN.26.0001", CreatedAt: at},
		inputreqprotection.Protection{Number: "OPCN.26.0003", CreatedAt: at},
		inputreqprotection.Protection{Number: "OPCN.26.0002", CreatedAt: at},
	)

	page, err := repo.List(context.Background(), inputreqprotection.Filter{})
	require.NoError(t, err)
	require.Equal(t, []string{"OPCN.26.0003", "OPCN.26.0002", "OPCN.26.0001"},
		[]string{page.Protections[0].Number, page.Protections[1].Number, page.Protections[2].Number})
}

func TestListSearchesNumberPolicyAndClaimCaseInsensitive(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	byNumber, err := repo.List(context.Background(), inputreqprotection.Filter{Search: "opc-201"})
	require.NoError(t, err)
	require.Equal(t, 1, byNumber.Total)
	require.Equal(t, "OPC-201", byNumber.Protections[0].Number)

	byPolicy, err := repo.List(context.Background(), inputreqprotection.Filter{Search: "00000004"})
	require.NoError(t, err)
	require.Equal(t, 1, byPolicy.Total)
	require.Equal(t, "OPCN.26.0003", byPolicy.Protections[0].Number)

	byClaim, err := repo.List(context.Background(), inputreqprotection.Filter{Search: "pncn.26.0010"})
	require.NoError(t, err)
	require.Equal(t, 1, byClaim.Total)
	require.Equal(t, "OPCN.26.0005", byClaim.Protections[0].Number)

	none, err := repo.List(context.Background(), inputreqprotection.Filter{Search: "tidak-ada"})
	require.NoError(t, err)
	require.Zero(t, none.Total)
}

func TestListPaginatesAndOffsetBeyondTotalIsEmpty(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	page, err := repo.List(context.Background(), inputreqprotection.Filter{Limit: 2, Offset: 4})
	require.NoError(t, err)
	require.Equal(t, 6, page.Total)
	require.Len(t, page.Protections, 2)
	require.Equal(t, "OPCN.26.0001", page.Protections[0].Number)

	beyond, err := repo.List(context.Background(), inputreqprotection.Filter{Offset: 6})
	require.NoError(t, err)
	require.Equal(t, 6, beyond.Total)
	require.NotNil(t, beyond.Protections)
	require.Empty(t, beyond.Protections)
}

func TestListSkipsAcceptedRows(t *testing.T) {
	repo := memory.NewRepo()
	repo.Add(
		inputreqprotection.Protection{Number: "A", AcceptStatus: inputreqprotection.AcceptApproved},
		inputreqprotection.Protection{Number: "B"},
	)
	page, err := repo.List(context.Background(), inputreqprotection.Filter{})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "B", page.Protections[0].Number)
}

func TestEveryMethodHonoursCancelledContext(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	ctx := cancelled()

	_, err := repo.List(ctx, inputreqprotection.Filter{})
	require.ErrorIs(t, err, context.Canceled)
	_, err = repo.Get(ctx, "OPC-201")
	require.ErrorIs(t, err, context.Canceled)
	_, err = repo.HasDuplicate(ctx, inputreqprotection.DuplicateKey{}, "")
	require.ErrorIs(t, err, context.Canceled)
	_, err = repo.Create(ctx, inputreqprotection.Draft{}, inputreqprotection.Claim{}, "U", time.Now())
	require.ErrorIs(t, err, context.Canceled)
	_, err = repo.Update(ctx, "OPC-201", inputreqprotection.Draft{}, inputreqprotection.Claim{}, "U", time.Now())
	require.ErrorIs(t, err, context.Canceled)

	_, err = memory.NewTypeRepoWithSamples().ListTypes(ctx)
	require.ErrorIs(t, err, context.Canceled)
	_, err = memory.NewClaimRepoWithSamples().FindClaim(ctx, "PNCN.26.0007")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetFindsByNormalizedNumber(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	p, err := repo.Get(context.Background(), "  opcn.26.0004 ")
	require.NoError(t, err)
	require.Equal(t, inputreqprotection.TypeChangeLossDate, p.Type)
	require.NotNil(t, p.ChangeDetail.LossDateBefore)

	_, err = repo.Get(context.Background(), "")
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)
	_, err = repo.Get(context.Background(), "OPC-999")
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)
}

func TestHasDuplicateComparesPolicyTypeAndWIBDay(t *testing.T) {
	// 2026-09-22 23.30 UTC adalah 2026-09-23 06.30 WIB.
	repo := memory.NewRepo()
	repo.Add(inputreqprotection.Protection{
		Number:       "OPCN.26.0001",
		PolicyNumber: "pol-1",
		Type:         "1",
		InputDate:    time.Date(2026, time.September, 22, 23, 30, 0, 0, time.UTC),
	})
	day := time.Date(2026, time.September, 23, 0, 0, 0, 0, wib)
	ctx := context.Background()

	dup, err := repo.HasDuplicate(ctx, inputreqprotection.DuplicateKey{PolicyNumber: " POL-1 ", Type: "1", Day: day}, "")
	require.NoError(t, err)
	require.True(t, dup)

	// Mengecualikan dirinya sendiri.
	dup, err = repo.HasDuplicate(ctx, inputreqprotection.DuplicateKey{PolicyNumber: "POL-1", Type: "1", Day: day}, "opcn.26.0001")
	require.NoError(t, err)
	require.False(t, dup)

	// Polis berbeda, tipe berbeda, hari berbeda — bukan ganda.
	for _, key := range []inputreqprotection.DuplicateKey{
		{PolicyNumber: "POL-2", Type: "1", Day: day},
		{PolicyNumber: "POL-1", Type: "2", Day: day},
		{PolicyNumber: "POL-1", Type: "1", Day: day.AddDate(0, 0, 1)},
	} {
		dup, err = repo.HasDuplicate(ctx, key, "")
		require.NoError(t, err)
		require.False(t, dup, "%+v", key)
	}
}

func TestCreateIssuesGlobalSequenceAcrossYearsAndDerivesDetail(t *testing.T) {
	repo := memory.NewRepo()
	dol := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	after := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	claim := inputreqprotection.Claim{
		Number: "PNCN.26.0007", PolicyNumber: "POL-1", LossDate: &dol,
		CauseOfLoss: "Kebakaran", ObjectName: "OBJ", BranchName: "CAB",
	}
	draft := inputreqprotection.Draft{
		ClaimNumber: "PNCN.26.0007", Type: "7", Note: "catatan",
		Change: inputreqprotection.ChangeRequest{LossDateAfter: &after, CauseOfLossAfter: "COL-2"},
	}

	at := time.Date(2026, time.December, 31, 10, 0, 0, 0, wib)
	first, err := repo.Create(context.Background(), draft, claim, "ADMIN", at)
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0001", first.Number)
	require.Equal(t, "POL-1", first.PolicyNumber)
	require.Equal(t, "PNCN.26.0007", first.ClaimNumber)
	require.Equal(t, "PNCN.26.0007", first.ClaimReference)
	require.Equal(t, "ADMIN", first.CreatedBy)
	require.Equal(t, at, first.CreatedAt)
	require.Equal(t, inputreqprotection.AcceptPending, first.AcceptStatus)
	require.Equal(t, inputreqprotection.ChangeDetail{
		LossDateBefore: &dol, LossDateAfter: &after,
		CauseOfLossID: "Kebakaran", CauseOfLossMasterID: "COL-2",
		ObjectName: "OBJ", BranchName: "CAB",
	}, first.ChangeDetail)

	// Pencacah global: menembus pergantian tahun tanpa reset.
	second, err := repo.Create(context.Background(), draft, claim, "ADMIN", at.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, "OPCN.27.0002", second.Number)

	got, err := repo.Get(context.Background(), "OPCN.26.0001")
	require.NoError(t, err)
	require.Equal(t, first, got)
}

func TestUpdateKeepsOriginAndRejectsLockedOrAccepted(t *testing.T) {
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	repo := memory.NewRepo()
	repo.Add(
		inputreqprotection.Protection{Number: "E", PolicyNumber: "OLD", Type: "1", CreatedBy: "PEMBUAT", CreatedAt: at},
		inputreqprotection.Protection{Number: "L", ClaimNumber: "PNCN.26.0001"},
		inputreqprotection.Protection{Number: "A", AcceptStatus: inputreqprotection.AcceptRejected},
	)
	claim := inputreqprotection.Claim{PolicyNumber: "NEW", ObjectName: "OBJ"}
	draft := inputreqprotection.Draft{ClaimNumber: "PNCN.26.0009", Type: "8", Note: "baru",
		Change: inputreqprotection.ChangeRequest{CauseOfLossAfter: "COL-9"}}
	ctx := context.Background()

	saved, err := repo.Update(ctx, "e", draft, claim, "PENYUNTING", at.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, "E", saved.Number)
	require.Equal(t, "NEW", saved.PolicyNumber)
	require.Equal(t, "PNCN.26.0009", saved.ClaimNumber)
	require.Equal(t, "PNCN.26.0009", saved.ClaimReference)
	require.Equal(t, "8", saved.Type)
	require.Equal(t, "baru", saved.Note)
	require.Equal(t, "COL-9", saved.ChangeDetail.CauseOfLossMasterID)
	require.Equal(t, "OBJ", saved.ChangeDetail.ObjectName)
	require.Equal(t, "PEMBUAT", saved.CreatedBy)
	require.Equal(t, at, saved.CreatedAt)

	_, err = repo.Update(ctx, "L", draft, claim, "U", at)
	require.ErrorIs(t, err, inputreqprotection.ErrLocked)
	_, err = repo.Update(ctx, "A", draft, claim, "U", at)
	require.ErrorIs(t, err, inputreqprotection.ErrAccepted)
	_, err = repo.Update(ctx, "X", draft, claim, "U", at)
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)
}

func TestTypeRepoReturnsCopyAndSupportsAdd(t *testing.T) {
	repo := memory.NewTypeRepo()
	types, err := repo.ListTypes(context.Background())
	require.NoError(t, err)
	require.Empty(t, types)

	repo.Add(inputreqprotection.ProtectionType{ID: "10", Name: "Baru"})
	types, err = repo.ListTypes(context.Background())
	require.NoError(t, err)
	require.Equal(t, []inputreqprotection.ProtectionType{{ID: "10", Name: "Baru"}}, types)

	// Mengubah hasil tidak boleh mengubah isi repo.
	types[0].Name = "diubah"
	again, err := repo.ListTypes(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Baru", again[0].Name)
}

func TestClaimRepoFindsByNormalizedNumber(t *testing.T) {
	repo := memory.NewClaimRepoWithSamples()

	c, err := repo.FindClaim(context.Background(), " pnc-1865 ")
	require.NoError(t, err)
	require.Equal(t, "ASM-FW-GCNMFW-WORK PNC-1865", c.PegaID)

	noDOL, err := repo.FindClaim(context.Background(), "PNCN.26.0008")
	require.NoError(t, err)
	require.Nil(t, noDOL.LossDate)

	_, err = repo.FindClaim(context.Background(), "PNCN.26.9999")
	require.ErrorIs(t, err, inputreqprotection.ErrClaimNotFound)

	empty := memory.NewClaimRepo()
	empty.Add(inputreqprotection.Claim{Number: " abc "})
	got, err := empty.FindClaim(context.Background(), "ABC")
	require.NoError(t, err)
	require.Equal(t, " abc ", got.Number)
}
