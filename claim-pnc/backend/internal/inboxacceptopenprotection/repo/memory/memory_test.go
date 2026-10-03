package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxacceptopenprotection"
	"claim-pnc/internal/inboxacceptopenprotection/repo/memory"
)

var wib = time.FixedZone("WIB", 7*60*60)

func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func pending(number, typ string, at time.Time) inboxacceptopenprotection.Protection {
	return inboxacceptopenprotection.Protection{
		Number:         number,
		PolicyNumber:   "99.001.2026.00000001",
		ClaimNumber:    "PNCN.26.0001",
		ClaimReference: "PNCN.26.0001",
		Type:           typ,
		InputDate:      at,
	}
}

func TestSampleRepoNonPremiumQueue(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{
		Queue: inboxacceptopenprotection.QueueNonPremium,
	})
	require.NoError(t, err)

	// Contoh: 0002, 0004, 0005, OPC-216 lengkap dan belum diputuskan; 0006 tanpa klaim dan
	// 0007 sudah disetujui tidak tampil; 0003 milik antrean PREMI.
	require.Equal(t, 4, page.Total)
	nomor := []string{}
	for _, p := range page.Protections {
		nomor = append(nomor, p.Number)
	}
	// Diurutkan menurun menurut tanggal dibuat.
	require.Equal(t, []string{"OPCN.26.0005", "OPCN.26.0004", "OPCN.26.0002", "OPC-216"}, nomor)
}

func TestSampleRepoPremiumQueue(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{
		Queue: inboxacceptopenprotection.QueuePremium,
	})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	require.Equal(t, "OPCN.26.0003", page.Protections[0].Number)
}

func TestListSearchMatchesNumberPolicyAndClaim(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	ctx := context.Background()

	for _, kata := range []string{"opcn.26.0004", "00000005", "pncn.26.0009"} {
		page, err := repo.List(ctx, inboxacceptopenprotection.Filter{Search: kata})
		require.NoError(t, err)
		require.Equal(t, 1, page.Total, "kata %q", kata)
		require.Equal(t, "OPCN.26.0004", page.Protections[0].Number)
	}

	page, err := repo.List(ctx, inboxacceptopenprotection.Filter{Search: "tidak-ada"})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Protections)
}

func TestListPaginationAndTieBreak(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, wib)
	repo := memory.NewRepo()
	repo.Add(pending("A", "1", at), pending("C", "1", at), pending("B", "1", at))

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	// Tanggal sama: nomor menurun sebagai pemutus seri.
	require.Equal(t, "C", page.Protections[0].Number)
	require.Equal(t, "B", page.Protections[1].Number)

	page, err = repo.List(context.Background(), inboxacceptopenprotection.Filter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	require.Len(t, page.Protections, 1)
	require.Equal(t, "A", page.Protections[0].Number)

	// Offset melewati jumlah: halaman kosong, total tetap.
	page, err = repo.List(context.Background(), inboxacceptopenprotection.Filter{Offset: 10})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.NotNil(t, page.Protections)
	require.Empty(t, page.Protections)
}

func TestListSkipsRowsWithoutPolicy(t *testing.T) {
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, wib)
	repo := memory.NewRepo()
	p := pending("A", "1", at)
	p.PolicyNumber = " "
	repo.Add(p)

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{})
	require.NoError(t, err)
	require.Zero(t, page.Total)
}

func TestCancelledContextIsReturned(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	ctx := cancelled()

	_, err := repo.List(ctx, inboxacceptopenprotection.Filter{})
	require.ErrorIs(t, err, context.Canceled)

	_, err = repo.Get(ctx, "OPCN.26.0002")
	require.ErrorIs(t, err, context.Canceled)

	_, err = repo.Decide(ctx, "OPCN.26.0002", inboxacceptopenprotection.DecisionApprove, "X", time.Now())
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetIgnoresCaseAndStatus(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	p, err := repo.Get(context.Background(), "  opcn.26.0007 ")
	require.NoError(t, err)
	require.True(t, p.Approved())
	require.Equal(t, "KOLEKSICONTOH", p.AcceptedBy)

	_, err = repo.Get(context.Background(), "")
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrNotFound)

	_, err = repo.Get(context.Background(), "TIDAK-ADA")
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrNotFound)
}

func TestDecideErrors(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	ctx := context.Background()
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, wib)

	_, err := repo.Decide(ctx, "OPCN.26.0002", inboxacceptopenprotection.Decision("x"), "A", at)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrUnknownDecision)

	_, err = repo.Decide(ctx, "TIDAK-ADA", inboxacceptopenprotection.DecisionApprove, "A", at)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrNotFound)

	_, err = repo.Decide(ctx, "OPCN.26.0007", inboxacceptopenprotection.DecisionApprove, "A", at)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrAlreadyDecided)
}

func TestDecideApprovesAndAppliesLossDate(t *testing.T) {
	repo := memory.NewRepoWithSamples()
	at := time.Date(2026, 9, 25, 9, 0, 0, 0, wib)

	saved, err := repo.Decide(context.Background(), "OPCN.26.0004", inboxacceptopenprotection.DecisionApprove, "PETUGAS", at)
	require.NoError(t, err)
	require.Equal(t, inboxacceptopenprotection.AcceptApproved, saved.AcceptStatus)
	require.Equal(t, "PETUGAS", saved.AcceptedBy)
	require.True(t, saved.AcceptedAt.Equal(at))

	tanggal, ada := repo.LossDateOf("pncn.26.0009")
	require.True(t, ada)
	require.Equal(t, time.Date(2026, 8, 5, 0, 0, 0, 0, wib), tanggal)

	// Keputusan tersimpan.
	again, err := repo.Get(context.Background(), "OPCN.26.0004")
	require.NoError(t, err)
	require.False(t, again.Pending())
}

func TestDecideRejectDoesNotTouchClaim(t *testing.T) {
	repo := memory.NewRepoWithSamples()

	saved, err := repo.Decide(context.Background(), "OPCN.26.0004", inboxacceptopenprotection.DecisionReject, "P", time.Now())
	require.NoError(t, err)
	require.Equal(t, inboxacceptopenprotection.AcceptRejected, saved.AcceptStatus)

	tanggal, ada := repo.LossDateOf("PNCN.26.0009")
	require.True(t, ada)
	require.Equal(t, time.Date(2026, 8, 3, 0, 0, 0, 0, wib), tanggal)
}

func TestDecideLossDateWithoutClaimRowIsNotSynced(t *testing.T) {
	after := time.Date(2026, 8, 9, 0, 0, 0, 0, wib)
	p := pending("A", inboxacceptopenprotection.TypeChangeLossDate, after)
	p.Change.LossDateAfter = &after

	// Repo tanpa klaim sama sekali.
	repo := memory.NewRepo()
	repo.Add(p)
	_, err := repo.Decide(context.Background(), "A", inboxacceptopenprotection.DecisionApprove, "P", after)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)

	// Keputusan TIDAK tersimpan.
	got, err := repo.Get(context.Background(), "A")
	require.NoError(t, err)
	require.True(t, got.Pending())

	// Repo berklaim lain.
	repo.AddClaim("KLAIM-LAIN", after)
	_, err = repo.Decide(context.Background(), "A", inboxacceptopenprotection.DecisionApprove, "P", after)
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)

	_, ada := repo.LossDateOf("PNCN.26.0001")
	require.False(t, ada)
}

func TestGroupRepo(t *testing.T) {
	repo := memory.NewSampleGroupRepo()

	groups, err := repo.GroupsOf(context.Background(), " keduacontoh ")
	require.NoError(t, err)
	require.Equal(t, []string{"CaseManager", "pnccollection"}, groups)

	// Salinan: mengubah hasil tidak mengubah isi repo.
	groups[0] = "Diubah"
	again, err := repo.GroupsOf(context.Background(), "KEDUACONTOH")
	require.NoError(t, err)
	require.Equal(t, "CaseManager", again[0])

	none, err := repo.GroupsOf(context.Background(), "TIDAKTERDAFTAR")
	require.NoError(t, err)
	require.Nil(t, none)

	require.Len(t, memory.SampleGroups(), 4)
}
