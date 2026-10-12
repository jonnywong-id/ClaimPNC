package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
)

func TestListMasterXOLReturnsCopy(t *testing.T) {
	repo := NewRepo(WithMasters(inboxxol.MasterXOL{ID: "A"}, inboxxol.MasterXOL{ID: "B"}))
	got, err := repo.ListMasterXOL(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	// Mengubah hasil tidak boleh mengubah isi penyimpanan.
	got[0].ID = "Z"
	again, _ := repo.ListMasterXOL(context.Background())
	require.Equal(t, "A", again[0].ID)
}

func TestListPendingMasterApprovalFiltersAndSortsByYear(t *testing.T) {
	repo := NewRepo(WithMasters(
		inboxxol.MasterXOL{ID: "A", Year: "2025", CommitteeStatus: "0"},
		inboxxol.MasterXOL{ID: "B", Year: "2023", CommitteeStatus: " 0 "},
		inboxxol.MasterXOL{ID: "C", Year: "2020", CommitteeStatus: "1"},
	))
	got, err := repo.ListPendingMasterApproval(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"B", "A"}, []string{got[0].ID, got[1].ID})
}

func TestSummarizeClaims(t *testing.T) {
	repo := NewRepo(WithSummaries(" 2024 ", inboxxol.ClaimSummary{LossDate: "x"}))
	ctx := context.Background()

	got, err := repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: "2024"})
	require.NoError(t, err)
	require.Nil(t, got, "tanpa group business tidak ada baris")

	got, err = repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: " 2024", BusinessGroupIDs: []string{"1"}})
	require.NoError(t, err)
	require.Equal(t, []inboxxol.ClaimSummary{{LossDate: "x"}}, got)

	got, err = repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: "1999", BusinessGroupIDs: []string{"1"}})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestBreakdowns(t *testing.T) {
	repo := NewRepo(
		WithBreakdown("01/01/2024", "Banjir", inboxxol.BusinessBreakdown{BusinessGroup: "own"}),
		WithTreatyInward("01/01/2024", "banjir", inboxxol.BusinessBreakdown{BusinessGroup: "treaty"}),
	)
	ctx := context.Background()
	full := inboxxol.BreakdownFilter{LossDate: " 01/01/2024 ", CauseOfLoss: "BANJIR", BusinessGroupIDs: []string{"1"}}

	got, err := repo.BreakdownByBusiness(ctx, full)
	require.NoError(t, err)
	require.Equal(t, "own", got[0].BusinessGroup)

	got, err = repo.BreakdownByBusiness(ctx, inboxxol.BreakdownFilter{LossDate: "01/01/2024", CauseOfLoss: "BANJIR"})
	require.NoError(t, err)
	require.Nil(t, got)

	// Treaty inward tidak membutuhkan group business.
	got, err = repo.BreakdownTreatyInward(ctx, inboxxol.BreakdownFilter{LossDate: "01/01/2024", CauseOfLoss: "Banjir"})
	require.NoError(t, err)
	require.Equal(t, "treaty", got[0].BusinessGroup)

	got, err = repo.BreakdownTreatyInward(ctx, inboxxol.BreakdownFilter{LossDate: "01/01/2024"})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestSearchAdviceFiltersAndSorts(t *testing.T) {
	repo := NewRepo(WithAdvices(
		inboxxol.Advice{Number: "1", Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: "Banjir", Revision: "1", LayerID: "B"},
		inboxxol.Advice{Number: "2", Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: "banjir ", Revision: "1", LayerID: "A"},
		inboxxol.Advice{Number: "3", Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: "Banjir", Revision: "0", LayerID: "Z"},
		inboxxol.Advice{Number: "4", Type: inboxxol.AdviceDLA, Year: "2024", CauseOfLoss: "Banjir"},
		inboxxol.Advice{Number: "5", Type: inboxxol.AdvicePLA, Year: "2023", CauseOfLoss: "Banjir"},
		inboxxol.Advice{Number: "6", Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: "Api"},
	))
	got, err := repo.SearchAdvice(context.Background(),
		inboxxol.AdviceFilter{Year: " 2024 ", CauseOfLoss: "BANJIR", Type: inboxxol.AdvicePLA})
	require.NoError(t, err)
	numbers := []string{}
	for _, a := range got {
		numbers = append(numbers, a.Number)
	}
	require.Equal(t, []string{"3", "2", "1"}, numbers)
}

func TestListPendingAdviceApprovalGroupsAndKeepsLatestText(t *testing.T) {
	repo := NewRepo(WithAdvices(
		inboxxol.Advice{Type: inboxxol.AdvicePLA, Year: "2023", CauseOfLoss: "Api", ApprovalStatus: "0", IssuedOn: "01/01/2023"},
		inboxxol.Advice{Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: "Banjir", ApprovalStatus: "0", IssuedOn: "05/01/2024"},
		inboxxol.Advice{Type: inboxxol.AdvicePLA, Year: "2024", CauseOfLoss: " Banjir ", ApprovalStatus: " 0", IssuedOn: "09/01/2024"},
		inboxxol.Advice{Type: inboxxol.AdviceDLA, Year: "2024", CauseOfLoss: "Banjir", ApprovalStatus: "0", IssuedOn: "02/02/2024"},
		inboxxol.Advice{Type: inboxxol.AdviceDLA, Year: "2024", CauseOfLoss: "Banjir", ApprovalStatus: "1", IssuedOn: "99/99/9999"},
	))
	got, err := repo.ListPendingAdviceApproval(context.Background())
	require.NoError(t, err)
	require.Equal(t, []inboxxol.ApprovalItem{
		{Year: "2024", CauseOfLoss: "Banjir", Type: inboxxol.AdviceDLA, LastInsertedAt: "02/02/2024"},
		{Year: "2024", CauseOfLoss: "Banjir", Type: inboxxol.AdvicePLA, LastInsertedAt: "09/01/2024"},
		{Year: "2023", CauseOfLoss: "Api", Type: inboxxol.AdvicePLA, LastInsertedAt: "01/01/2023"},
	}, got)
}

func TestListCauseOfLoss(t *testing.T) {
	repo := NewRepo(WithCauseOfLoss(inboxxol.CauseOfLoss{ID: "1", Description: "Banjir"}))
	got, err := repo.ListCauseOfLoss(context.Background())
	require.NoError(t, err)
	require.Equal(t, []inboxxol.CauseOfLoss{{ID: "1", Description: "Banjir"}}, got)
}

func TestSampleRepoCoversEveryTab(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	masters, err := repo.ListMasterXOL(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleMasters(), masters)

	causes, err := repo.ListCauseOfLoss(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleCauseOfLoss(), causes)

	summaries, err := repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: "2024", BusinessGroupIDs: []string{"x"}})
	require.NoError(t, err)
	require.Equal(t, SampleSummaries2024(), summaries)
	summaries, err = repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: "2023", BusinessGroupIDs: []string{"x"}})
	require.NoError(t, err)
	require.Equal(t, SampleSummaries2023(), summaries)

	filter := inboxxol.BreakdownFilter{LossDate: "12/03/2024", CauseOfLoss: "BANJIR", BusinessGroupIDs: []string{"x"}}
	own, err := repo.BreakdownByBusiness(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, SampleBreakdownBanjir(), own)
	treaty, err := repo.BreakdownTreatyInward(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, SampleTreatyBanjir(), treaty)

	filter = inboxxol.BreakdownFilter{LossDate: "28/07/2024", CauseOfLoss: "KEBAKARAN", BusinessGroupIDs: []string{"x"}}
	own, _ = repo.BreakdownByBusiness(ctx, filter)
	require.Equal(t, SampleBreakdownKebakaran(), own)
	treaty, _ = repo.BreakdownTreatyInward(ctx, filter)
	require.Equal(t, SampleTreatyKebakaran(), treaty)

	pending, err := repo.ListPendingAdviceApproval(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, pending)
	require.Len(t, SampleAdvices(), len(repo.advices))
}
