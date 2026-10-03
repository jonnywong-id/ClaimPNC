package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterautoclaim/repo/memory"
)

func newRepo() *memory.Repo {
	return memory.NewRepo(memory.Options{
		Rows: []masterautoclaim.AutoClaim{
			{Initial: "ZZ", Status: masterautoclaim.StatusPending, Committee: "K1"},
			{Initial: "AA", Status: masterautoclaim.StatusPending, Committee: " k1 "},
			{Initial: "BB", Status: masterautoclaim.StatusApproved, Committee: "K2"},
		},
		Sources:   []masterautoclaim.BusinessSource{{ID: "BRI", Name: "Bank Rakyat"}, {ID: "MDR", Name: "B.Mandiri"}},
		Clients:   []masterautoclaim.Client{{ID: "C2", Name: "Zeta"}, {ID: "C1", Name: "Alfa"}},
		Banks:     []masterautoclaim.Bank{{Code: "009", Name: "BNI"}, {Code: "002", Name: "BRI"}},
		Committee: "KOMITE1",
	})
}

func TestListFiltersStatusAndCommittee(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	pending, err := repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusPending})
	require.NoError(t, err)
	require.Equal(t, []string{"AA", "ZZ"}, []string{pending[0].Initial, pending[1].Initial})

	mine, err := repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusPending, CommitteeOnly: true, CommitteeID: "K1"})
	require.NoError(t, err)
	require.Len(t, mine, 2)
	none, _ := repo.List(ctx, masterautoclaim.Filter{Status: masterautoclaim.StatusApproved, CommitteeOnly: true, CommitteeID: "K1"})
	require.Empty(t, none)
}

func TestGetInsertUpdate(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	got, err := repo.Get(ctx, " BB ")
	require.NoError(t, err)
	require.Equal(t, "K2", got.Committee)
	_, err = repo.Get(ctx, "XX")
	require.ErrorIs(t, err, masterautoclaim.ErrNotFound)

	require.ErrorIs(t, repo.Insert(ctx, masterautoclaim.AutoClaim{Initial: " AA "}), masterautoclaim.ErrInitialTaken)
	require.NoError(t, repo.Insert(ctx, masterautoclaim.AutoClaim{Initial: "CC", ReceiverName: "Penerima"}))

	require.NoError(t, repo.Update(ctx, masterautoclaim.AutoClaim{Initial: "CC", ReceiverName: "DIABAIKAN",
		BankName: "BRI", AccountNumber: "1", MaxPercent: "5", ReporterPIC: "P", ReporterEmail: "e",
		ClaimAllowed: "1", ReceiverAddress: "A", Status: masterautoclaim.StatusRejected, SubmittedBy: "S",
		Committee: "K", ClientID: "C", ClientName: "N"}))
	updated, _ := repo.Get(ctx, "CC")
	// Nama penerima tidak pernah diubah, sama seperti kueri update di SQL.
	require.Equal(t, "Penerima", updated.ReceiverName)
	require.Equal(t, masterautoclaim.StatusRejected, updated.Status)
	require.Equal(t, "N", updated.ClientName)

	require.ErrorIs(t, repo.Update(ctx, masterautoclaim.AutoClaim{Initial: "XX"}), masterautoclaim.ErrNotFound)
}

func TestLookups(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	sources, err := repo.SearchBusinessSources(ctx, " b.mandiri ")
	require.NoError(t, err)
	require.Equal(t, []masterautoclaim.BusinessSource{{ID: "MDR", Name: "B.Mandiri"}}, sources)
	sources, _ = repo.SearchBusinessSources(ctx, "bri")
	require.Equal(t, "BRI", sources[0].ID)
	sources, _ = repo.SearchBusinessSources(ctx, "  ")
	require.Empty(t, sources, "kata kunci kosong tidak mengembalikan apa pun")

	clients, err := repo.SearchClients(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, []string{"Alfa", "Zeta"}, []string{clients[0].Name, clients[1].Name})
	clients, _ = repo.SearchClients(ctx, "c2")
	require.Equal(t, "C2", clients[0].ID)

	banks, err := repo.ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"BNI", "BRI"}, []string{banks[0].Name, banks[1].Name})

	bank, err := repo.FindBankByName(ctx, " bri ")
	require.NoError(t, err)
	require.Equal(t, "002", bank.Code)
	_, err = repo.FindBankByName(ctx, "")
	require.ErrorIs(t, err, masterautoclaim.ErrBankNotFound)
	_, err = repo.FindBankByName(ctx, "BCA")
	require.ErrorIs(t, err, masterautoclaim.ErrBankNotFound)

	committee, err := repo.Committee(ctx)
	require.NoError(t, err)
	require.Equal(t, "KOMITE1", committee)
}

func TestLookupStopsAtLimit(t *testing.T) {
	sources := make([]masterautoclaim.BusinessSource, 0, masterautoclaim.MaxLookupRows+3)
	clients := make([]masterautoclaim.Client, 0, masterautoclaim.MaxLookupRows+3)
	for i := 0; i < masterautoclaim.MaxLookupRows+3; i++ {
		sources = append(sources, masterautoclaim.BusinessSource{ID: "S", Name: "SAMA"})
		clients = append(clients, masterautoclaim.Client{ID: "C", Name: "SAMA"})
	}
	repo := memory.NewRepo(memory.Options{Sources: sources, Clients: clients})
	got, err := repo.SearchBusinessSources(context.Background(), "sama")
	require.NoError(t, err)
	require.Len(t, got, masterautoclaim.MaxLookupRows)
	gotClients, err := repo.SearchClients(context.Background(), "sama")
	require.NoError(t, err)
	require.Len(t, gotClients, masterautoclaim.MaxLookupRows)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := newRepo()
	failure := errors.New("rusak")
	repo.SetError(failure)
	ctx := context.Background()

	_, err := repo.List(ctx, masterautoclaim.Filter{})
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "AA")
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, repo.Insert(ctx, masterautoclaim.AutoClaim{Initial: "N"}), failure)
	require.ErrorIs(t, repo.Update(ctx, masterautoclaim.AutoClaim{Initial: "AA"}), failure)
	_, err = repo.SearchBusinessSources(ctx, "a")
	require.ErrorIs(t, err, failure)
	_, err = repo.SearchClients(ctx, "a")
	require.ErrorIs(t, err, failure)
	_, err = repo.ListBanks(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.FindBankByName(ctx, "BRI")
	require.ErrorIs(t, err, failure)
	_, err = repo.Committee(ctx)
	require.ErrorIs(t, err, failure)
}

func TestSampleRepo(t *testing.T) {
	repo := memory.NewSampleRepo()
	banks, err := repo.ListBanks(context.Background())
	require.NoError(t, err)
	require.Len(t, banks, len(memory.SampleBanks()))
	committee, _ := repo.Committee(context.Background())
	require.Equal(t, memory.SampleCommittee, committee)
	require.NotEmpty(t, memory.SampleList())
	require.NotEmpty(t, memory.SampleClients())
	require.NotEmpty(t, memory.SampleBusinessSources())
}
