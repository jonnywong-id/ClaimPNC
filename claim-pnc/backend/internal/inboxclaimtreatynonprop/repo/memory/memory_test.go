package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
)

// fixedNow mematok jam supaya umur pekerjaan tidak berubah setiap hari.
var fixedNow = time.Date(2026, time.September, 28, 3, 0, 0, 0, time.UTC)

func mustQuery(
	t *testing.T, tab string, seeAll, tba bool, login string,
) inboxclaimtreatynonprop.Query {
	t.Helper()
	q, err := inboxclaimtreatynonprop.NewQuery(
		inboxclaimtreatynonprop.QueryInput{Tab: tab, SeeAll: seeAll, TBAOnly: tba},
		inboxclaimtreatynonprop.Caller{Login: login},
	)
	require.NoError(t, err)
	return q
}

func ids(page inboxclaimtreatynonprop.Page) []string {
	out := []string{}
	for _, item := range page.Items {
		out = append(out, item.ClaimID)
	}
	return out
}

func list(
	t *testing.T, store *memory.Store, q inboxclaimtreatynonprop.Query,
) inboxclaimtreatynonprop.Page {
	t.Helper()
	page, err := store.List(context.Background(), q,
		inboxclaimtreatynonprop.Pagination{Page: 1, Size: inboxclaimtreatynonprop.MaxPageSize})
	require.NoError(t, err)
	return page
}

// Tab Admin hanya membaca worklist pemanggil, terbaru lebih dulu.
func TestListAdminOwnRowsNewestFirst(t *testing.T) {
	store := memory.NewStoreAt(func() time.Time { return fixedNow }, memory.SampleRows()...)

	page := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false, "adminnonprop1"))
	require.Equal(t, []string{"CLMNP-1004", "CLMNP-1002", "CLMNP-1001"}, ids(page),
		"pencocokan login tidak membedakan huruf besar-kecil")
	require.Equal(t, 3, page.Total)
}

// Baris dihias dengan teks waktu pembuatan dan umur dalam hari kalender WIB.
func TestListDecoratesCreatedAtAndAging(t *testing.T) {
	store := memory.NewStoreAt(func() time.Time { return fixedNow }, memory.SampleRows()...)

	page := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabTechnical, false, false, "X"))
	require.Equal(t, []string{"CLMNP-2001", "CLMNP-2002"}, ids(page),
		"tab Teknik menaik; baris antrean lain tersaring")
	require.Equal(t, "20260917T030000.000 GMT", page.Items[0].CreatedAt)
	require.Equal(t, 11, page.Items[0].AgingDays)
}

// TBA saja menyaring polis kosong milik pemanggil; See All melepas kepemilikan saja.
func TestListFilters(t *testing.T) {
	store := memory.NewSampleStore()

	tba := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, false, true, "ADMINNONPROP1"))
	require.Equal(t, []string{"CLMNP-1004"}, ids(tba))

	all := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false, "ORANGLAIN"))
	require.Equal(t, []string{"CLMNP-1004", "CLMNP-1002", "CLMNP-1003", "CLMNP-1001"}, ids(all))

	none := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false, "ORANGLAIN"))
	require.Empty(t, none.Items)
	require.Equal(t, 0, none.Total)
}

// Awalan harus di depan; waktu sama diurutkan menurut nomor klaim supaya deterministik.
func TestListPrefixAnchoredAndTieBreak(t *testing.T) {
	at := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
	store := memory.NewStore(
		memory.Row{WorkCreatedAt: at, Item: inboxclaimtreatynonprop.WorkItem{
			ClaimID: "clmnp-0002", AssignedOperator: "A"}},
		memory.Row{WorkCreatedAt: at, Item: inboxclaimtreatynonprop.WorkItem{
			ClaimID: "CLMNP-0001", AssignedOperator: "A"}},
		memory.Row{WorkCreatedAt: at, Item: inboxclaimtreatynonprop.WorkItem{
			ClaimID: "X-CLMNP-0003", AssignedOperator: "A"}},
	)

	page := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false, "A"))
	require.Equal(t, []string{"CLMNP-0001", "clmnp-0002"}, ids(page))
}

// Paginasi memotong setelah penyaringan, total tetap jumlah seluruh yang cocok.
func TestListPaginates(t *testing.T) {
	store := memory.NewSampleStore()
	q := mustQuery(t, inboxclaimtreatynonprop.TabAdmin, true, false, "A")

	page, err := store.List(context.Background(), q, inboxclaimtreatynonprop.Pagination{Page: 2, Size: 3})
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.Equal(t, []string{"CLMNP-1001"}, ids(page))
}

// Baris tanpa waktu objek kerja tidak menjadi tanggal tahun satu, umurnya nol.
func TestListZeroCreationTime(t *testing.T) {
	store := memory.NewStore(memory.Row{Item: inboxclaimtreatynonprop.WorkItem{
		ClaimID: "CLMNP-9", AssignedOperator: "A"}})

	page := list(t, store, mustQuery(t, inboxclaimtreatynonprop.TabAdmin, false, false, "A"))
	require.Len(t, page.Items, 1)
	require.Equal(t, "", page.Items[0].CreatedAt)
	require.Equal(t, 0, page.Items[0].AgingDays)
}
