package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/detailpenyebab"
)

func seeded() *Repo {
	repo := NewRepo()
	repo.seed(
		[]detailpenyebab.CauseOfLossDetail{
			{ID: "990002", MasterID: "M1", Description: "kebakaran pabrik", LossCode: "K1",
				Business: []detailpenyebab.Business{{ID: " 01 ", Name: "Fire"}}},
			{ID: "990001", MasterID: "M2", Description: "Banjir", LossCode: "B1"},
			{ID: "LAIN", Description: "Angin"},
		},
		[]detailpenyebab.MasterOption{{ID: "M1", Label: "Kebakaran"}, {ID: "M2", Label: "Alam"}},
		[]detailpenyebab.Business{{ID: "01", Name: "Fire"}, {ID: "02", Name: "Marine"}},
	)
	return repo
}

func cancelled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestListFiltersAndSortsByDescription(t *testing.T) {
	repo := seeded()
	ctx := context.Background()

	all, err := repo.List(ctx, detailpenyebab.Filter{})
	require.NoError(t, err)
	require.Equal(t, []string{"Angin", "Banjir", "kebakaran pabrik"},
		[]string{all[0].Description, all[1].Description, all[2].Description})
	require.Nil(t, all[2].Business, "daftar tidak membawa lini bisnis")
	require.Equal(t, "Kebakaran", all[2].MasterLabel)

	byKeyword, _ := repo.List(ctx, detailpenyebab.Filter{Keyword: " b1 "})
	require.Len(t, byKeyword, 1)
	require.Equal(t, "990001", byKeyword[0].ID)

	byID, _ := repo.List(ctx, detailpenyebab.Filter{Keyword: "9900"})
	require.Len(t, byID, 2)

	byMaster, _ := repo.List(ctx, detailpenyebab.Filter{MasterID: "M2"})
	require.Len(t, byMaster, 1)

	byBusiness, _ := repo.List(ctx, detailpenyebab.Filter{BusinessID: "01"})
	require.Len(t, byBusiness, 1)
	require.Equal(t, "990002", byBusiness[0].ID)

	_, err = repo.List(cancelled(), detailpenyebab.Filter{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGet(t *testing.T) {
	repo := seeded()
	got, err := repo.Get(context.Background(), " 990002 ")
	require.NoError(t, err)
	require.Equal(t, "Kebakaran", got.MasterLabel)
	require.Len(t, got.Business, 1)

	_, err = repo.Get(context.Background(), "tidak-ada")
	require.ErrorIs(t, err, detailpenyebab.ErrNotFound)
	_, err = repo.Get(cancelled(), "990002")
	require.ErrorIs(t, err, context.Canceled)
}

func TestInsertContinuesSeededSequence(t *testing.T) {
	repo := seeded()
	got, err := repo.Insert(context.Background(), detailpenyebab.Input{MasterID: "M2", Description: "Gempa"})
	require.NoError(t, err)
	require.Equal(t, "990003", got.ID)
	require.Equal(t, "Alam", got.MasterLabel)

	_, err = repo.Insert(cancelled(), detailpenyebab.Input{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestInsertRejectsTakenID(t *testing.T) {
	repo := seeded()
	// Nomor urut tertinggal dari isi — keadaan yang menuntut penolakan, bukan penimpaan.
	repo.sequence = 1
	_, err := repo.Insert(context.Background(), detailpenyebab.Input{Description: "x"})
	require.ErrorIs(t, err, detailpenyebab.ErrIDTaken)
}

func TestUpdate(t *testing.T) {
	repo := seeded()
	got, err := repo.Update(context.Background(), " 990001 ", detailpenyebab.Input{MasterID: "M1", Description: "Baru",
		Business: []detailpenyebab.Business{{ID: "02"}}})
	require.NoError(t, err)
	require.Equal(t, "Kebakaran", got.MasterLabel)
	stored, _ := repo.Get(context.Background(), "990001")
	require.Equal(t, "Baru", stored.Description)
	require.Equal(t, []detailpenyebab.Business{{ID: "02"}}, stored.Business)

	_, err = repo.Update(context.Background(), "tidak-ada", detailpenyebab.Input{})
	require.ErrorIs(t, err, detailpenyebab.ErrNotFound)
	_, err = repo.Update(cancelled(), "990001", detailpenyebab.Input{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSearchMasterAndBusiness(t *testing.T) {
	repo := seeded()
	ctx := context.Background()

	masters, err := repo.SearchMaster(ctx, " ala ")
	require.NoError(t, err)
	require.Equal(t, []detailpenyebab.MasterOption{{ID: "M2", Label: "Alam"}}, masters)
	masters, _ = repo.SearchMaster(ctx, "m1")
	require.Equal(t, "M1", masters[0].ID)
	masters, _ = repo.SearchMaster(ctx, "")
	require.Len(t, masters, 2)
	_, err = repo.SearchMaster(cancelled(), "")
	require.ErrorIs(t, err, context.Canceled)

	business, err := repo.SearchBusiness(ctx, "mar")
	require.NoError(t, err)
	require.Equal(t, []detailpenyebab.Business{{ID: "02", Name: "Marine"}}, business)
	business, _ = repo.SearchBusiness(ctx, "01")
	require.Equal(t, "01", business[0].ID)
	business, _ = repo.SearchBusiness(ctx, "")
	require.Len(t, business, 2)
	_, err = repo.SearchBusiness(cancelled(), "")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSearchStopsAtLookupLimit(t *testing.T) {
	repo := NewRepo()
	master := make([]detailpenyebab.MasterOption, 0, detailpenyebab.MaxLookupRows+5)
	business := make([]detailpenyebab.Business, 0, detailpenyebab.MaxLookupRows+5)
	for i := 0; i < detailpenyebab.MaxLookupRows+5; i++ {
		master = append(master, detailpenyebab.MasterOption{ID: "M", Label: "X"})
		business = append(business, detailpenyebab.Business{ID: "B", Name: "X"})
	}
	repo.seed(nil, master, business)
	got, err := repo.SearchMaster(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, got, detailpenyebab.MaxLookupRows)
	lines, err := repo.SearchBusiness(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, lines, detailpenyebab.MaxLookupRows)
}

func TestSampleRepoIsUsable(t *testing.T) {
	repo := NewSampleRepo()
	list, err := repo.List(context.Background(), detailpenyebab.Filter{})
	require.NoError(t, err)
	require.Len(t, list, len(sampleRows()))
	masters, _ := repo.SearchMaster(context.Background(), "")
	require.Equal(t, sampleMaster(), masters)
	business, _ := repo.SearchBusiness(context.Background(), "")
	require.Equal(t, sampleBusiness(), business)
}
