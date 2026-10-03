package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
)

var errStore = errors.New("penyimpanan rusak")

func ids(cases []komite.CommitteeCase) []string {
	out := []string{}
	for _, c := range cases {
		out = append(out, c.CaseID)
	}
	return out
}

func TestThresholdRepoReturnsACopyAndCanFail(t *testing.T) {
	repo := memory.NewSampleRepo()
	rows, err := repo.ListThresholds(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, len(memory.SampleThresholds()))

	rows[0].BusinessLine = "DIUBAH"
	again, _ := repo.ListThresholds(context.Background())
	require.NotEqual(t, "DIUBAH", again[0].BusinessLine)

	repo.SetError(errStore)
	_, err = repo.ListThresholds(context.Background())
	require.ErrorIs(t, err, errStore)
}

func filter(kind komite.InboxKind) komite.InboxFilter {
	return komite.InboxFilter{Operator: memory.SampleOperator, Kind: kind, Limit: 25}
}

func TestInboxKindsSplitTheSampleCases(t *testing.T) {
	store := memory.NewSampleInboxStore()
	ctx := context.Background()

	page, err := store.ListCases(ctx, filter(komite.InboxOutstanding))
	require.NoError(t, err)
	require.Equal(t, []string{"K-2601", "K-2603", "K-2602"}, ids(page.Cases), "terlama di atas")
	require.Equal(t, 3, page.Total)

	page, _ = store.ListCases(ctx, filter(komite.InboxAccepted))
	require.Equal(t, []string{"K-2604"}, ids(page.Cases))

	page, _ = store.ListCases(ctx, filter(komite.InboxRejected))
	require.Equal(t, []string{"K-2605"}, ids(page.Cases))

	summary, err := store.Summarize(ctx, filter(komite.InboxOutstanding))
	require.NoError(t, err)
	require.Equal(t, komite.InboxSummary{Outstanding: 3, Accepted: 1, Rejected: 1}, summary)

	// Tanpa operator tidak ada yang terlihat.
	page, _ = store.ListCases(ctx, komite.InboxFilter{Kind: komite.InboxOutstanding, Limit: 25})
	require.Empty(t, page.Cases)
}

func TestInboxPaginatesSearchesAndFiltersByDate(t *testing.T) {
	store := memory.NewSampleInboxStore()
	ctx := context.Background()

	f := filter(komite.InboxOutstanding)
	f.Offset, f.Limit = 1, 1
	page, _ := store.ListCases(ctx, f)
	require.Equal(t, []string{"K-2603"}, ids(page.Cases))
	require.Equal(t, 3, page.Total)

	f.Offset, f.Limit = 2, 10
	page, _ = store.ListCases(ctx, f)
	require.Equal(t, []string{"K-2602"}, ids(page.Cases))

	f.Offset = 5
	page, _ = store.ListCases(ctx, f)
	require.Empty(t, page.Cases)
	require.Equal(t, 3, page.Total)

	f = filter(komite.InboxOutstanding)
	f.Search = "0102"
	page, _ = store.ListCases(ctx, f)
	require.Equal(t, []string{"K-2602"}, ids(page.Cases))

	f = filter(komite.InboxOutstanding)
	f.DateFrom = time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	page, _ = store.ListCases(ctx, f)
	require.Equal(t, []string{"K-2603", "K-2602"}, ids(page.Cases))
}

func TestDecisionsMoveACaseOutOfOutstanding(t *testing.T) {
	store := memory.NewSampleInboxStore()
	ctx := context.Background()

	require.NoError(t, store.Record(ctx, komite.Decision{
		ID: "D1", CaseID: "K-2601", Tier: 1, Kind: komite.DecisionApprove,
		ActorLogin: memory.SampleOperator,
	}))
	require.NoError(t, store.Record(ctx, komite.Decision{
		ID: "D2", CaseID: "K-2602", Tier: 1, Kind: komite.DecisionReject,
		ActorLogin: memory.SampleOperator,
	}))

	page, _ := store.ListCases(ctx, filter(komite.InboxOutstanding))
	require.Equal(t, []string{"K-2603"}, ids(page.Cases))
	page, _ = store.ListCases(ctx, filter(komite.InboxAccepted))
	require.ElementsMatch(t, []string{"K-2601", "K-2604"}, ids(page.Cases))
	page, _ = store.ListCases(ctx, filter(komite.InboxRejected))
	require.ElementsMatch(t, []string{"K-2602", "K-2605"}, ids(page.Cases))

	found, err := store.FindCase(ctx, "K-2601", memory.SampleOperator)
	require.NoError(t, err)
	require.Len(t, found.Progress.Decisions, 1)

	_, err = store.FindCase(ctx, "K-9999", memory.SampleOperator)
	require.ErrorIs(t, err, komite.ErrCaseNotFound)

	byCase, err := store.ListForCases(ctx, []string{"K-2601", "K-2603"})
	require.NoError(t, err)
	require.Len(t, byCase["K-2601"], 1)
	require.Empty(t, byCase["K-2603"])
	require.NotContains(t, byCase, "K-2602")
}

func TestInboxStoreFailuresAndHelpers(t *testing.T) {
	store := memory.NewInboxStore(komite.CommitteeCase{CaseID: " K-1 "})
	ctx := context.Background()

	require.True(t, store.Available(ctx))
	transfer, err := store.FindTransfer(ctx, "K-1")
	require.NoError(t, err)
	require.Equal(t, komite.TransferDetail{}, transfer)

	_, err = store.FindCase(ctx, "K-1", "")
	require.NoError(t, err, "kunci dibersihkan saat disimpan")

	store.SetError(errStore)
	_, err = store.ListCases(ctx, filter(komite.InboxOutstanding))
	require.ErrorIs(t, err, errStore)
	_, err = store.Summarize(ctx, filter(komite.InboxOutstanding))
	require.ErrorIs(t, err, errStore)
	_, err = store.FindCase(ctx, "K-1", "")
	require.ErrorIs(t, err, errStore)
	_, err = store.ListForCases(ctx, []string{"K-1"})
	require.ErrorIs(t, err, errStore)
	require.ErrorIs(t, store.Record(ctx, komite.Decision{}), errStore)
	_, err = store.FindTransfer(ctx, "K-1")
	require.ErrorIs(t, err, errStore)
}

func TestIDGeneratorProducesDistinctHexIDs(t *testing.T) {
	first, second := memory.IDGenerator{}.New(), memory.IDGenerator{}.New()
	require.Regexp(t, `^[0-9a-f]{32}$`, first)
	require.NotEqual(t, first, second)
}
