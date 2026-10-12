package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/repo/memory"
)

func at(year int, month time.Month, date int) *time.Time {
	value := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &value
}

var today = time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC)

func numbersOf(page inboxprogressclaim.ClaimPage) []string {
	result := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		result = append(result, item.ClaimNumber)
	}
	return result
}

func TestListClaimsOrdersByProcessDateThenNumberWithUndatedLast(t *testing.T) {
	store := memory.NewStore([]memory.ClaimRecord{
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "D"}},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "C"}},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "B", ProcessDate: at(2026, 8, 5)}},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "A", ProcessDate: at(2026, 8, 5)}},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "E", ProcessDate: at(2026, 8, 1)}},
	}, nil)

	page, err := store.ListClaims(context.Background(), inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding}, inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, []string{"E", "A", "B", "C", "D"}, numbersOf(page))
	require.Equal(t, 5, page.Total)
	require.Equal(t, inboxprogressclaim.Pagination{Page: 1, Size: inboxprogressclaim.DefaultPageSize}, page.Pagination)
}

func TestListClaimsKeywordMatchesClaimPolicyAndPIC(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	cases := map[string]string{
		"pncn.26.0101":      "PNCN.26.0101",
		"16.001.2026.00102": "PNCN.26.0102",
		"buditeknik":        "PNCN.26.0102",
	}
	for keyword, want := range cases {
		page, err := store.ListClaims(ctx, inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding, Keyword: "  " + keyword + " "}, inboxprogressclaim.Pagination{})
		require.NoError(t, err)
		require.Contains(t, numbersOf(page), want, keyword)
	}

	page, err := store.ListClaims(ctx, inboxprogressclaim.ClaimQuery{Keyword: "TIDAK-ADA"}, inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
}

func TestListClaimsNextFollowUpKeepsOnlyDueRecords(t *testing.T) {
	store := memory.NewStore([]memory.ClaimRecord{
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "LATE"}, LastFollowUpByPIC: at(2026, 9, 20)},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "TODAY"}, LastFollowUpByPIC: &today},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "FUTURE"}, LastFollowUpByPIC: at(2026, 9, 22)},
		{Item: inboxprogressclaim.ClaimRow{ClaimNumber: "NEVER"}},
	}, nil)

	page, err := store.ListClaims(context.Background(), inboxprogressclaim.ClaimQuery{
		View: inboxprogressclaim.ViewNextFollowUp, Today: today.Add(15 * time.Hour),
	}, inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"LATE", "TODAY"}, numbersOf(page))
}

func TestListClaimsPaginates(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()
	q := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding}

	all, err := store.ListClaims(ctx, q, inboxprogressclaim.Pagination{Size: 100})
	require.NoError(t, err)
	require.Greater(t, all.Total, 2)

	second, err := store.ListClaims(ctx, q, inboxprogressclaim.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, all.Total, second.Total)
	require.Equal(t, numbersOf(all)[2:min(4, all.Total)], numbersOf(second))

	beyond, err := store.ListClaims(ctx, q, inboxprogressclaim.Pagination{Page: 99, Size: 2})
	require.NoError(t, err)
	require.Empty(t, beyond.Items)
	require.Equal(t, all.Total, beyond.Total)
}

func TestListPICSummaryScopesBusinessCallerAndRange(t *testing.T) {
	store := memory.NewSampleStore()
	ctx := context.Background()

	rows, err := store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessNonMBU,
		Caller:   inboxprogressclaim.Caller{Login: "adminklaim"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 12, rows[0].ClaimCount)

	// Rentang tanggal yang tidak memuat 3 Agustus menyaring habis.
	rows, err = store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessNonMBU,
		Caller:   inboxprogressclaim.Caller{Login: memory.SampleOwner},
		From:     at(2026, 8, 4),
	})
	require.NoError(t, err)
	require.Empty(t, rows)
	require.NotNil(t, rows)

	rows, err = store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessNonMBU,
		Caller:   inboxprogressclaim.Caller{Login: memory.SampleOwner},
		To:       at(2026, 8, 2),
	})
	require.NoError(t, err)
	require.Empty(t, rows)

	rows, err = store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessNonMBU,
		Caller:   inboxprogressclaim.Caller{Login: memory.SampleOwner},
		From:     at(2026, 8, 3), To: at(2026, 8, 3),
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestListPICSummaryRecordWithoutDateIsExcludedOnlyWhenRangeGiven(t *testing.T) {
	store := memory.NewStore(nil, []memory.PICRecord{
		{Item: inboxprogressclaim.PICSummary{PIC: "B"}, Business: inboxprogressclaim.BusinessPA},
		{Item: inboxprogressclaim.PICSummary{PIC: "b"}, Business: inboxprogressclaim.BusinessPA},
	})
	ctx := context.Background()

	rows, err := store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{Business: inboxprogressclaim.BusinessPA, Caller: inboxprogressclaim.Caller{Login: "B"}})
	require.NoError(t, err)
	// Diurutkan menurut nama PIC; pencocokan pemanggil tidak peka huruf.
	require.Equal(t, []string{"B", "b"}, []string{rows[0].PIC, rows[1].PIC})

	rows, err = store.ListPICSummary(ctx, inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessPA, Caller: inboxprogressclaim.Caller{Login: "B"}, From: at(2026, 1, 1),
	})
	require.NoError(t, err)
	require.Empty(t, rows)
}
