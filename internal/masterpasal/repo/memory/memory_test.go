package memory_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasal/repo/memory"
)

func TestListSortsByNumberAndDropsBusiness(t *testing.T) {
	repo := memory.NewRepo([]masterpasal.Clause{
		{ID: "9", Number: "B"},
		{ID: "2", Number: "A"},
		{ID: "1", Number: "A", Business: []masterpasal.Business{{ID: "2001"}}},
	}, nil)

	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Clause{
		{ID: "1", Number: "A"}, {ID: "2", Number: "A"}, {ID: "9", Number: "B"},
	}, rows)
}

func TestGetRefreshesBusinessNamesFromTheMaster(t *testing.T) {
	repo := memory.NewRepo([]masterpasal.Clause{{
		ID: "1", Number: "PSL",
		Business: []masterpasal.Business{
			{ID: " 2001 ", Name: "nama lama"},
			{ID: "", Name: "tanpa kode"},
			{ID: "9999", Name: "tidak di master"},
			{ID: "2007", Name: "master tanpa nama"},
		},
	}, {ID: "2", Number: "KOSONG"}}, append(memory.SampleBusiness(), masterpasal.Business{ID: "2007"}))

	clause, err := repo.Get(context.Background(), " 1 ")
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Business{
		{ID: " 2001 ", Name: "Personal Accident"},
		{ID: "", Name: "tanpa kode"},
		{ID: "9999", Name: "tidak di master"},
		{ID: "2007", Name: "master tanpa nama"},
	}, clause.Business)

	clause, err = repo.Get(context.Background(), "2")
	require.NoError(t, err)
	require.Nil(t, clause.Business)

	_, err = repo.Get(context.Background(), "404")
	require.ErrorIs(t, err, masterpasal.ErrNotFound)
}

func TestInsertUpdateDelete(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	created, err := repo.Insert(ctx, masterpasal.Input{
		Number: "PSL-004", Text: "teks", Description: "ket",
		Category: masterpasal.CategoryException,
		Business: []masterpasal.Business{{ID: "2001", Name: "Personal Accident"}},
	})
	require.NoError(t, err)
	require.Equal(t, "4", created.ID, "urutan berikutnya setelah 3")
	require.Equal(t, "Pengecualian", created.CategoryLabel)

	updated, err := repo.Update(ctx, "4", masterpasal.Input{Number: "PSL-004B"})
	require.NoError(t, err)
	require.Equal(t, "4", updated.ID)
	require.Equal(t, "Notifikasi", updated.CategoryLabel)

	_, err = repo.Update(ctx, "404", masterpasal.Input{Number: "x"})
	require.ErrorIs(t, err, masterpasal.ErrNotFound)

	require.NoError(t, repo.Delete(ctx, "4"))
	require.ErrorIs(t, repo.Delete(ctx, "4"), masterpasal.ErrNotFound)

	rows, _ := repo.List(ctx)
	require.Len(t, rows, len(memory.SampleList()))
}

func TestSearchBusinessMatchesNameOrCodeAndCapsTheRows(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	rows, err := repo.SearchBusiness(ctx, "  ")
	require.NoError(t, err)
	require.Nil(t, rows)

	rows, _ = repo.SearchBusiness(ctx, "ar")
	require.Equal(t, []masterpasal.Business{{ID: "2003", Name: "Marine Cargo"}}, rows)

	rows, _ = repo.SearchBusiness(ctx, "2002")
	require.Equal(t, []masterpasal.Business{{ID: "2002", Name: "Travel"}}, rows)

	many := []masterpasal.Business{{ID: "", Name: "LINI TANPA KODE"}}
	for i := 0; i < masterpasal.MaxLookupRows+5; i++ {
		many = append(many, masterpasal.Business{ID: fmt.Sprint(i), Name: fmt.Sprintf("LINI %03d", i)})
	}
	rows, _ = memory.NewRepo(nil, many).SearchBusiness(ctx, "lini")
	require.Len(t, rows, masterpasal.MaxLookupRows)
	require.Equal(t, "LINI 000", rows[0].Name, "urut menurut nama; baris tanpa kode dilewati")
}
