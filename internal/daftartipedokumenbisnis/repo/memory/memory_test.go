package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/daftartipedokumenbisnis/repo/memory"
)

var errBoom = errors.New("boom")

func TestNewRepoTrimsRowsAndDropsEmptyCoverages(t *testing.T) {
	repo := memory.NewRepo(daftartipedokumenbisnis.DocumentRule{
		ID: " 10005 ", BusinessID: " 001 ", BusinessName: " Fire ", DetailDocument: " Laporan ",
		Coverages: []daftartipedokumenbisnis.Coverage{{ID: " 10009 "}, {ID: "  "}},
	})

	got, err := repo.Get(context.Background(), "10005")
	require.NoError(t, err)
	require.Equal(t, "001", got.BusinessID)
	require.Equal(t, "Fire", got.BusinessName)
	require.Equal(t, "Laporan", got.DetailDocument)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10009"}}, got.Coverages)
}

func TestListBusinessesSortedByNameAndSkipsEmptyBusiness(t *testing.T) {
	rows := append(memory.SampleList(), daftartipedokumenbisnis.DocumentRule{ID: "x1"})
	repo := memory.NewRepo(rows...)

	got, err := repo.ListBusinesses(context.Background())
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Business{
		{ID: "001", Name: "Fire"},
		{ID: "004", Name: "Marine Cargo"},
	}, got)
}

func TestListByBusinessSortedByID(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	got, err := repo.ListByBusiness(context.Background(), " 001 ")
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, []string{"10001", "10002", "10003"}, []string{got[0].ID, got[1].ID, got[2].ID})

	empty, err := repo.ListByBusiness(context.Background(), "005")
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestGetMissingIsNotFound(t *testing.T) {
	_, err := memory.NewRepo().Get(context.Background(), "nope")
	require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
}

func TestInsertBatchIssuesIDsAfterExistingSequence(t *testing.T) {
	// Baris contoh berakhir di 10004, sehingga ID berikutnya 10005 dan 10006.
	repo := memory.NewRepo(memory.SampleList()...)

	saved, err := repo.InsertBatch(context.Background(), daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"005", "003"},
		Rules: []daftartipedokumenbisnis.Input{{
			DocumentTypeID: "20001", ObjectDocID: "30001", DetailTypeDocID: "40001",
			DetailDocument: "Laporan", Mandatory: true, MinDocument: 2,
		}},
	}, daftartipedokumenbisnis.Editor{})
	require.NoError(t, err)
	require.Len(t, saved, 2)
	require.Equal(t, "10005", saved[0].ID)
	require.Equal(t, "005", saved[0].BusinessID)
	require.Equal(t, "10006", saved[1].ID)
	require.Equal(t, "003", saved[1].BusinessID)
	require.True(t, saved[0].Mandatory)
	require.Equal(t, 2, saved[0].MinDocument)

	stored, err := repo.Get(context.Background(), "10006")
	require.NoError(t, err)
	require.Equal(t, "Laporan", stored.DetailDocument)
}

func TestInsertBatchIgnoresForeignShapedIDsForSequence(t *testing.T) {
	// ID tak berawalan situs, atau yang ekornya bukan angka, tidak memajukan urutan;
	// urutan berikutnya tetap diturunkan dari ID berbentuk sah dan baris lama tidak tertimpa.
	repo := memory.NewRepo(
		daftartipedokumenbisnis.DocumentRule{ID: "x9", BusinessID: "001"},
		daftartipedokumenbisnis.DocumentRule{ID: "1abc", BusinessID: "001"},
		daftartipedokumenbisnis.DocumentRule{ID: "10001", BusinessID: "001", DetailDocument: "lama"},
	)
	saved, err := repo.InsertBatch(context.Background(), daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"002"},
		Rules:       []daftartipedokumenbisnis.Input{{DetailDocument: "baru"}},
	}, daftartipedokumenbisnis.Editor{})
	require.NoError(t, err)
	require.Equal(t, "10002", saved[0].ID)

	old, err := repo.Get(context.Background(), "10001")
	require.NoError(t, err)
	require.Equal(t, "lama", old.DetailDocument)
}

func TestUpdateKeepsBusinessAndCoverages(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	got, err := repo.Update(context.Background(), " 10001 ", daftartipedokumenbisnis.Input{
		DocumentTypeID: "20004", DetailTypeDocID: "40005", DetailDocument: "Kuitansi", MinDocument: 3,
	}, daftartipedokumenbisnis.Editor{})
	require.NoError(t, err)
	require.Equal(t, "001", got.BusinessID)
	require.Equal(t, "Fire", got.BusinessName)
	require.Equal(t, "Kuitansi", got.DetailDocument)
	require.False(t, got.Mandatory)
	require.Empty(t, got.ObjectDocID)
	// Jaminan disimpan terpisah dan tetap ada setelah baris diubah.
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10009"}}, got.Coverages)

	_, err = repo.Update(context.Background(), "nope", daftartipedokumenbisnis.Input{}, daftartipedokumenbisnis.Editor{})
	require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
}

func TestAddCoverageAppendsSortedAndIgnoresDuplicate(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)

	got, err := repo.AddCoverage(context.Background(), "10001", " 10001 ")
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10001"}, {ID: "10009"}}, got.Coverages)

	again, err := repo.AddCoverage(context.Background(), "10001", "10009")
	require.NoError(t, err)
	require.Len(t, again.Coverages, 2)

	_, err = repo.AddCoverage(context.Background(), "nope", "1")
	require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(errBoom)
	ctx := context.Background()

	_, err := repo.ListBusinesses(ctx)
	require.ErrorIs(t, err, errBoom)
	_, err = repo.ListByBusiness(ctx, "001")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Get(ctx, "10001")
	require.ErrorIs(t, err, errBoom)
	_, err = repo.InsertBatch(ctx, daftartipedokumenbisnis.BatchInput{}, daftartipedokumenbisnis.Editor{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.Update(ctx, "10001", daftartipedokumenbisnis.Input{}, daftartipedokumenbisnis.Editor{})
	require.ErrorIs(t, err, errBoom)
	_, err = repo.AddCoverage(ctx, "10001", "1")
	require.ErrorIs(t, err, errBoom)
}

func TestBusinessAndReferenceReposReturnCopies(t *testing.T) {
	businesses := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	list, err := businesses.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, memory.SampleBusinessList(), list)
	// Mengubah salinan tidak mengubah isi repo.
	list[0].Name = "diubah"
	again, err := businesses.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Fire", again[0].Name)

	for _, sample := range [][]daftartipedokumenbisnis.Reference{
		memory.SampleDocumentTypeList(), memory.SampleObjectDocList(), memory.SampleDetailTypeDocList(),
	} {
		refs := memory.NewReferenceRepo(sample...)
		got, err := refs.List(context.Background())
		require.NoError(t, err)
		require.Equal(t, sample, got)
	}
	require.Len(t, memory.SampleDocumentTypeList(), 6)
}
