package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

func queryFor(t *testing.T, code, login string, seeAll bool) inboxclaimtreatyprop.Query {
	t.Helper()
	q, err := inboxclaimtreatyprop.NewQuery(inboxclaimtreatyprop.QueryInput{Tab: code, SeeAll: seeAll},
		inboxclaimtreatyprop.Caller{Login: login})
	require.NoError(t, err)
	return q
}

func ids(page inboxclaimtreatyprop.Page) []string {
	var result []string
	for _, item := range page.Items {
		result = append(result, item.ClaimID)
	}
	return result
}

var wide = inboxclaimtreatyprop.Pagination{Page: 1, Size: 100}

func TestSampleStoreWorkListSortedNewestFirst(t *testing.T) {
	store := NewSampleStore()
	page, err := store.List(context.Background(), queryFor(t, inboxclaimtreatyprop.TabWorkList, "ADMINTREATY1", false), wide)
	require.NoError(t, err)
	// Hanya penugasan worklist berkelas treaty; PNC-9001 (kelas lain) dan antrean teknik
	// tidak ikut. Urutan: waktu penugasan menurun.
	require.Equal(t, []string{"CLMP-1002", "CLMP-1003", "CLMP-1001"}, ids(page))
	require.Equal(t, 3, page.Total)
}

func TestSampleStoreTechnicalTabReadsWorkbasketOnly(t *testing.T) {
	store := NewSampleStore()
	page, err := store.List(context.Background(), queryFor(t, inboxclaimtreatyprop.TabTechnical, "ADMINTREATY1", false), wide)
	require.NoError(t, err)
	require.Equal(t, []string{"CLMP-3001", "CLMP-2001", "CLMP-2002"}, ids(page))
}

func TestStoreScopesToCallerWhenTabRequiresIt(t *testing.T) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mine := Row{WorkClass: inboxclaimtreatyprop.WorkClass, AssignedAt: at,
		Item: inboxclaimtreatyprop.WorkItem{ClaimID: "B", AssignedOperator: "saya"}}
	other := Row{WorkClass: inboxclaimtreatyprop.WorkClass, AssignedAt: at,
		Item: inboxclaimtreatyprop.WorkItem{ClaimID: "A", AssignedOperator: "lain"}}
	store := NewStore(mine, other)

	// Tab buatan uji yang menyaring pemanggil — tidak ada tab bawaan yang menyaring.
	q := inboxclaimtreatyprop.Query{
		Tab:    inboxclaimtreatyprop.Tab{Code: inboxclaimtreatyprop.TabWorkList, ScopedToCaller: true},
		Caller: inboxclaimtreatyprop.Caller{Login: "SAYA"},
	}
	page, err := store.List(context.Background(), q, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"B"}, ids(page))

	// Lihat semua melepas penyaring; waktu sama → diurutkan Claim ID (tanpa Reference).
	q.SeeAll = true
	page, err = store.List(context.Background(), q, wide)
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"}, ids(page))
}

func TestStoreSlicesRequestedPage(t *testing.T) {
	store := NewSampleStore()
	page, err := store.List(context.Background(), queryFor(t, inboxclaimtreatyprop.TabWorkList, "X", false),
		inboxclaimtreatyprop.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"CLMP-1001"}, ids(page))
	require.Equal(t, 3, page.Total)
}

func TestRowKeyPrefersReference(t *testing.T) {
	require.Equal(t, "R", rowKey(inboxclaimtreatyprop.WorkItem{Reference: "R", ClaimID: "C"}))
	require.Equal(t, "C", rowKey(inboxclaimtreatyprop.WorkItem{ClaimID: "C"}))
}
