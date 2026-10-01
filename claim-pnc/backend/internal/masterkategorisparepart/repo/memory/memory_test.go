package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterkategorisparepart"
)

func ids(list []masterkategorisparepart.PartCategory) []string {
	result := make([]string, 0, len(list))
	for _, c := range list {
		result = append(result, c.ID)
	}
	return result
}

func TestListFiltersByStatusAndKeyword(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	approved, err := repo.List(ctx, masterkategorisparepart.Filter{Status: masterkategorisparepart.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, []string{"1", "2", "3"}, ids(approved))

	found, err := repo.List(ctx, masterkategorisparepart.Filter{
		Status: masterkategorisparepart.StatusApproved, Keyword: " draul ",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"2"}, ids(found))
}

func TestListSortsNumericIDsBeforeText(t *testing.T) {
	repo := NewRepo(Options{Rows: []masterkategorisparepart.PartCategory{
		{ID: "B", Status: "1"},
		{ID: "10", Status: "1"},
		{ID: "A", Status: "1"},
		{ID: "2", Status: "1"},
	}})

	rows, err := repo.List(context.Background(), masterkategorisparepart.Filter{Status: "1"})
	require.NoError(t, err)
	require.Equal(t, []string{"2", "10", "A", "B"}, ids(rows))
}

func TestGetAndFindByName(t *testing.T) {
	repo := NewRepo(Options{Rows: append(SampleList(),
		masterkategorisparepart.PartCategory{ID: "0", Name: " engine ", Status: "1"})})
	ctx := context.Background()

	got, err := repo.Get(ctx, " 4 ")
	require.NoError(t, err)
	require.Equal(t, "ELECTRICAL", got.Name)

	_, err = repo.Get(ctx, "99")
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)

	// Nama dibandingkan tanpa memperhatikan huruf dan spasi; ID terkecil yang menang.
	got, err = repo.FindByName(ctx, "Engine")
	require.NoError(t, err)
	require.Equal(t, "0", got.ID)

	_, err = repo.FindByName(ctx, "TIDAK ADA")
	require.ErrorIs(t, err, masterkategorisparepart.ErrNotFound)
}

func TestInsertIssuesNextIDAndRejectsTakenName(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	next, err := repo.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, "7", next)

	saved, err := repo.Insert(ctx, masterkategorisparepart.PartCategory{Name: "BRAKE", Status: "0"})
	require.NoError(t, err)
	require.Equal(t, masterkategorisparepart.PartCategory{ID: "7", Name: "BRAKE", Status: "0"}, saved)

	_, err = repo.Insert(ctx, masterkategorisparepart.PartCategory{Name: " brake "})
	require.ErrorIs(t, err, masterkategorisparepart.ErrNameTaken)
}

func TestUpdateAndSetStatus(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()

	require.NoError(t, repo.Update(ctx, masterkategorisparepart.PartCategory{ID: "4", Name: "ELEC", Status: "1"}))
	got, err := repo.Get(ctx, "4")
	require.NoError(t, err)
	require.Equal(t, masterkategorisparepart.PartCategory{ID: "4", Name: "ELEC", Status: "1"}, got)

	require.ErrorIs(t, repo.Update(ctx, masterkategorisparepart.PartCategory{ID: "99"}),
		masterkategorisparepart.ErrNotFound)

	// "1" sudah disetujui, "99" tidak ada; hanya "5" yang berubah.
	changed, err := repo.SetStatus(ctx, []string{"1", "5", "99"}, masterkategorisparepart.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	boom := errors.New("boom")
	repo := NewSampleRepo()
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx, masterkategorisparepart.Filter{})
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "1")
	require.ErrorIs(t, err, boom)
	_, err = repo.FindByName(ctx, "ENGINE")
	require.ErrorIs(t, err, boom)
	_, err = repo.Insert(ctx, masterkategorisparepart.PartCategory{})
	require.ErrorIs(t, err, boom)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, repo.Update(ctx, masterkategorisparepart.PartCategory{ID: "1"}), boom)
	_, err = repo.SetStatus(ctx, []string{"1"}, "0")
	require.ErrorIs(t, err, boom)
}
