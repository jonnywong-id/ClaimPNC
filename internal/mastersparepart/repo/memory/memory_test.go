package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/repo/memory"
)

func newRepo() *memory.Repo {
	return memory.NewRepo(memory.Options{
		Rows: []mastersparepart.Sparepart{
			{ID: "SP2", Name: "Filter Oli", Number: "1R-01", Code: "FLT-1", Status: mastersparepart.StatusApproved},
			{ID: "SP1", Name: "Piston", Number: "2R-02", Code: "PST-1", Status: mastersparepart.StatusApproved},
			{ID: "SP3", Name: "Seal", Number: "3R-03", Code: "SEL-1", Status: mastersparepart.StatusPending},
		},
		Category: []mastersparepart.Category{{ID: "K1", Name: "ENGINE"}},
		Type:     []mastersparepart.PartType{{ID: "T1", Name: "FILTER", CategoryID: "K1"}},
		Site:     " ZZ ",
		Sequence: 7,
	})
}

func TestListFiltersStatusAndKeyword(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	approved, err := repo.List(ctx, mastersparepart.Filter{Status: mastersparepart.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, []string{"SP1", "SP2"}, []string{approved[0].ID, approved[1].ID})

	for _, keyword := range []string{" filter ", "1r-01", "flt"} {
		got, err := repo.List(ctx, mastersparepart.Filter{Status: mastersparepart.StatusApproved, Keyword: keyword})
		require.NoError(t, err)
		require.Len(t, got, 1, keyword)
		require.Equal(t, "SP2", got[0].ID)
	}
	none, _ := repo.List(ctx, mastersparepart.Filter{Status: mastersparepart.StatusRejected})
	require.Empty(t, none)
}

func TestGetAndFind(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	got, err := repo.Get(ctx, " SP3 ")
	require.NoError(t, err)
	require.Equal(t, "Seal", got.Name)
	_, err = repo.Get(ctx, "SP9")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)

	byName, err := repo.FindByName(ctx, " piston ")
	require.NoError(t, err)
	require.Equal(t, "SP1", byName.ID)
	byNumber, err := repo.FindByNumber(ctx, "3r-03")
	require.NoError(t, err)
	require.Equal(t, "SP3", byNumber.ID)
	byCode, err := repo.FindByCode(ctx, "flt-1")
	require.NoError(t, err)
	require.Equal(t, "SP2", byCode.ID)

	_, err = repo.FindByName(ctx, "  ")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)
	_, err = repo.FindByCode(ctx, "TIDAK")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)
}

func TestListsOfCategoriesAndTypesAreCopies(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	categories, err := repo.ListCategories(ctx)
	require.NoError(t, err)
	categories[0].Name = "DIUBAH"
	again, _ := repo.ListCategories(ctx)
	require.Equal(t, "ENGINE", again[0].Name)

	types, err := repo.ListTypes(ctx)
	require.NoError(t, err)
	require.Equal(t, []mastersparepart.PartType{{ID: "T1", Name: "FILTER", CategoryID: "K1"}}, types)
}

func TestInsertRejectsTakenKeys(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	require.ErrorIs(t, repo.Insert(ctx, mastersparepart.Sparepart{Number: " 1r-01 "}), mastersparepart.ErrNumberTaken)
	require.ErrorIs(t, repo.Insert(ctx, mastersparepart.Sparepart{Name: "SEAL"}), mastersparepart.ErrNameTaken)
	require.ErrorIs(t, repo.Insert(ctx, mastersparepart.Sparepart{Code: "pst-1"}), mastersparepart.ErrCodeTaken)

	require.NoError(t, repo.Insert(ctx, mastersparepart.Sparepart{ID: "SP4", Name: "Hose", Number: "4R", Code: "HSE",
		Status: mastersparepart.StatusPending}))
	got, err := repo.Get(ctx, "SP4")
	require.NoError(t, err)
	require.Equal(t, "Hose", got.Name)
}

func TestUpdateKeepsID(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	require.NoError(t, repo.Update(ctx, mastersparepart.Sparepart{ID: " SP1 ", Name: "Piston Baru"}))
	got, _ := repo.Get(ctx, "SP1")
	require.Equal(t, "Piston Baru", got.Name)
	require.Equal(t, "SP1", got.ID)
	require.ErrorIs(t, repo.Update(ctx, mastersparepart.Sparepart{ID: "SP9"}), mastersparepart.ErrNotFound)
}

func TestSetStatusCountsOnlyRealChanges(t *testing.T) {
	repo := newRepo()
	changed, err := repo.SetStatus(context.Background(), []string{" SP3 ", "SP1", "SP9"}, mastersparepart.StatusApproved)
	require.NoError(t, err)
	require.Equal(t, 1, changed, "SP1 sudah disetujui; SP9 tidak ada")
	got, _ := repo.Get(context.Background(), "SP3")
	require.Equal(t, mastersparepart.StatusApproved, got.Status)
}

func TestNextIDUsesSiteAndPaddedSequence(t *testing.T) {
	repo := newRepo()
	id, err := repo.NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, "ZZ0000000008", id)

	fallback := memory.NewRepo(memory.Options{})
	id, err = fallback.NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, memory.SampleSite+"0000000001", id, "situs kosong jatuh ke situs contoh")
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := newRepo()
	failure := errors.New("rusak")
	repo.SetError(failure)
	ctx := context.Background()
	_, err := repo.List(ctx, mastersparepart.Filter{})
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "SP1")
	require.ErrorIs(t, err, failure)
	_, err = repo.FindByName(ctx, "Piston")
	require.ErrorIs(t, err, failure)
	_, err = repo.ListCategories(ctx)
	require.ErrorIs(t, err, failure)
	_, err = repo.ListTypes(ctx)
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, repo.Insert(ctx, mastersparepart.Sparepart{}), failure)
	require.ErrorIs(t, repo.Update(ctx, mastersparepart.Sparepart{ID: "SP1"}), failure)
	_, err = repo.SetStatus(ctx, []string{"SP1"}, mastersparepart.StatusRejected)
	require.ErrorIs(t, err, failure)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, failure)
}

func TestSampleRepo(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()
	categories, err := repo.ListCategories(ctx)
	require.NoError(t, err)
	require.Equal(t, memory.SampleCategories(), categories)
	types, _ := repo.ListTypes(ctx)
	require.Equal(t, memory.SampleTypes(), types)
	approved, _ := repo.List(ctx, mastersparepart.Filter{Status: mastersparepart.StatusApproved})
	require.NotEmpty(t, approved)
	id, _ := repo.NextID(ctx)
	require.Equal(t, mastersparepart.ComposeID(memory.SampleSite, memory.SampleSequence+1, 10), id)
	require.NotEmpty(t, memory.SampleList())
}
