package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/repo/memory"
)

var errStore = errors.New("penyimpanan rusak")

func TestRepoListSortsAndReturnsACopy(t *testing.T) {
	repo := memory.NewRepo(
		masterstatusprogres.ProgressStatus{ID: "03", Name: "C"},
		masterstatusprogres.ProgressStatus{ID: "01", Name: "A"},
	)
	rows, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"01", "03"}, []string{rows[0].ID, rows[1].ID})

	rows[0].Name = "DIUBAH"
	again, _ := repo.List(context.Background())
	require.Equal(t, "A", again[0].Name)
}

func TestRepoGetInsertUpdate(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	ctx := context.Background()

	got, err := repo.Get(ctx, " 02 ")
	require.NoError(t, err)
	require.Equal(t, "DOKUMEN BELUM LENGKAP", got.Name)
	_, err = repo.Get(ctx, "99")
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)

	fresh, err := repo.InsertNew(ctx, masterstatusprogres.Input{Name: "BARU", PositionCode: "SURVEY"})
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.ProgressStatus{
		ID: masterstatusprogres.FormatID(7), Name: "BARU", PositionCode: "SURVEY",
	}, fresh)

	require.NoError(t, repo.Update(ctx, masterstatusprogres.ProgressStatus{
		ID: "01", Name: "UBAH", PositionCode: "KOMITE",
	}))
	got, _ = repo.Get(ctx, "01")
	require.Equal(t, "UBAH", got.Name)
	require.Equal(t, "KOMITE", got.PositionCode)

	err = repo.Update(ctx, masterstatusprogres.ProgressStatus{ID: "99"})
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)
}

func TestRepoInsertSkipsATakenID(t *testing.T) {
	// "02" sudah dipakai walau angka tertingginya 1 — kandidat berikutnya dilewati.
	repo := memory.NewRepo(
		masterstatusprogres.ProgressStatus{ID: "1"},
		masterstatusprogres.ProgressStatus{ID: "02"},
		masterstatusprogres.ProgressStatus{ID: "X"},
	)
	fresh, err := repo.InsertNew(context.Background(), masterstatusprogres.Input{Name: "N"})
	require.NoError(t, err)
	require.Equal(t, "03", fresh.ID)
}

func TestRepoSetErrorFailsEveryOperation(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(errStore)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errStore)
	_, err = repo.Get(ctx, "01")
	require.ErrorIs(t, err, errStore)
	_, err = repo.InsertNew(ctx, masterstatusprogres.Input{})
	require.ErrorIs(t, err, errStore)
	require.ErrorIs(t, repo.Update(ctx, masterstatusprogres.ProgressStatus{}), errStore)
}

func TestRepo2FollowsItsParent(t *testing.T) {
	parent := memory.NewRepo(memory.SampleList()...)
	repo := memory.NewRepo2(parent, memory.SampleList2()...)
	ctx := context.Background()

	rows, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, len(memory.SampleList2()))
	require.Equal(t, "1", rows[0].ID)

	got, err := repo.Get(ctx, " 3 ")
	require.NoError(t, err)
	require.Equal(t, "02", got.ParentID)
	_, err = repo.Get(ctx, "99")
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)

	fresh, err := repo.InsertNew(ctx, masterstatusprogres.Input2{Name: "ANAK", ParentID: "04"})
	require.NoError(t, err)
	require.Equal(t, masterstatusprogres.FormatID2(7), fresh.ID)
	require.Equal(t, "SURVEI SELESAI", fresh.ParentName)

	_, err = repo.InsertNew(ctx, masterstatusprogres.Input2{Name: "ANAK", ParentID: "99"})
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)

	updated, err := repo.Update(ctx, " 1 ", masterstatusprogres.Input2{Name: "UBAH", ParentID: "05"})
	require.NoError(t, err)
	require.Equal(t, "MENUNGGU PERSETUJUAN KOMITE", updated.ParentName)

	_, err = repo.Update(ctx, "99", masterstatusprogres.Input2{Name: "x", ParentID: "05"})
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)
	_, err = repo.Update(ctx, "1", masterstatusprogres.Input2{Name: "x", ParentID: "99"})
	require.ErrorIs(t, err, masterstatusprogres.ErrParentNotFound)
}

func TestRepo2InsertSkipsATakenID(t *testing.T) {
	parent := memory.NewRepo(memory.SampleList()...)
	repo := memory.NewRepo2(parent,
		masterstatusprogres.ProgressStatus2{ID: "1"},
		masterstatusprogres.ProgressStatus2{ID: masterstatusprogres.FormatID2(2)},
		masterstatusprogres.ProgressStatus2{ID: "Z"},
	)
	fresh, err := repo.InsertNew(context.Background(), masterstatusprogres.Input2{ParentID: "01"})
	require.NoError(t, err)
	require.NotEqual(t, masterstatusprogres.FormatID2(2), fresh.ID)
	require.NotEqual(t, "1", fresh.ID)
}

func TestRepo2SetErrorFailsEveryOperation(t *testing.T) {
	parent := memory.NewRepo(memory.SampleList()...)
	repo := memory.NewRepo2(parent, memory.SampleList2()...)
	repo.SetError(errStore)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errStore)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, errStore)
	_, err = repo.InsertNew(ctx, masterstatusprogres.Input2{ParentID: "01"})
	require.ErrorIs(t, err, errStore)
	_, err = repo.Update(ctx, "1", masterstatusprogres.Input2{ParentID: "01"})
	require.ErrorIs(t, err, errStore)
}
