package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
)

func TestSampleRepoListsMastersWithoutChildren(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()
	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, len(SampleMaster()))
	for i, m := range list {
		require.Nil(t, m.Business)
		require.Nil(t, m.Layer)
		if i > 0 {
			require.Less(t, list[i-1].ID, m.ID)
		}
	}
	years, err := repo.ListYear(ctx)
	require.NoError(t, err)
	require.Equal(t, SampleYear(), years)
}

func TestGetReturnsIndependentCopy(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()
	got, err := repo.Get(ctx, " 10001 ")
	require.NoError(t, err)
	require.NotEmpty(t, got.Layer)
	got.Layer[0].Reinsurer[0].Name = "DIUBAH"
	again, _ := repo.Get(ctx, "10001")
	require.NotEqual(t, "DIUBAH", again.Layer[0].Reinsurer[0].Name)

	_, err = repo.Get(ctx, "99999")
	require.ErrorIs(t, err, masterxol.ErrNotFound)
}

func TestSaveNewAndUpdateKeepsCommitteeColumns(t *testing.T) {
	repo := NewRepo()
	ctx := context.Background()

	saved, err := repo.Save(ctx, masterxol.Master{Name: " A ", ExchangeRate: 2,
		Layer: []masterxol.Layer{{Name: "L1", Limit: 5}, {ID: "777", Name: "L0"}}})
	require.NoError(t, err)
	require.Equal(t, "10001", saved.ID)
	require.Equal(t, "A", saved.Name)
	require.Equal(t, "10001", saved.Layer[0].ID)
	require.Equal(t, "777", saved.Layer[1].ID)
	require.Equal(t, masterxol.Amount(10), saved.Layer[0].ConvertedLimit)

	second, err := repo.Save(ctx, masterxol.Master{Name: "B", Layer: []masterxol.Layer{{Name: "x"}}})
	require.NoError(t, err)
	require.Equal(t, "10002", second.ID)
	// Nomor layer melanjutkan nomor terbesar dari seluruh induk.
	require.Equal(t, "10002", second.Layer[0].ID)

	require.NoError(t, repo.SubmitToCommittee(ctx, "10001", " PIC1 ", " catatan "))
	updated, err := repo.Save(ctx, masterxol.Master{ID: "10001", Name: "A2", PIC: "lain",
		CommitteeStatus: masterxol.CommitteeApproved})
	require.NoError(t, err)
	require.Equal(t, "PIC1", updated.PIC)
	require.Equal(t, masterxol.CommitteePending, updated.CommitteeStatus)

	_, err = repo.Save(ctx, masterxol.Master{ID: "55555"})
	require.ErrorIs(t, err, masterxol.ErrNotFound)
}

func TestSaveSkipsNonNumericIDs(t *testing.T) {
	// Nomor non-angka diabaikan NextID; nomor berikutnya melanjutkan yang terbesar.
	repo := NewRepo()
	repo.master["10001"] = masterxol.Master{ID: "10001"}
	repo.master["X"] = masterxol.Master{ID: "X"}
	saved, err := repo.Save(context.Background(), masterxol.Master{Name: "n"})
	require.NoError(t, err)
	require.Equal(t, "10002", saved.ID)
}

func TestDeleteOperations(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()
	master, err := repo.Get(ctx, "10001")
	require.NoError(t, err)
	layerID := master.Layer[0].ID
	reasID := master.Layer[0].Reinsurer[0].ID
	businessID := master.Business[0].ID

	require.NoError(t, repo.DeleteReinsurer(ctx, " "+layerID+" ", reasID))
	require.ErrorIs(t, repo.DeleteReinsurer(ctx, "tidak-ada", "x"), masterxol.ErrLayerNotFound)

	require.NoError(t, repo.DeleteBusiness(ctx, "10001", businessID))
	require.ErrorIs(t, repo.DeleteBusiness(ctx, "99999", "x"), masterxol.ErrNotFound)

	after, _ := repo.Get(ctx, "10001")
	require.Len(t, after.Business, len(master.Business)-1)
	require.Len(t, after.Layer[0].Reinsurer, len(master.Layer[0].Reinsurer)-1)

	require.NoError(t, repo.DeleteLayer(ctx, "10001", layerID))
	require.ErrorIs(t, repo.DeleteLayer(ctx, "10001", layerID), masterxol.ErrLayerNotFound)
	require.ErrorIs(t, repo.DeleteLayer(ctx, "99999", layerID), masterxol.ErrNotFound)

	require.NoError(t, repo.DeleteMaster(ctx, " 10001 "))
	require.ErrorIs(t, repo.DeleteMaster(ctx, "10001"), masterxol.ErrNotFound)
	require.ErrorIs(t, repo.SubmitToCommittee(ctx, "10001", "p", "r"), masterxol.ErrNotFound)
}

func TestListBusinessGroupReproducesLegacyMatching(t *testing.T) {
	repo := NewSampleRepo()
	ctx := context.Background()
	names := func(list []masterxol.Business) []string {
		result := []string{}
		for _, b := range list {
			result = append(result, b.Name)
		}
		return result
	}

	accident, err := repo.ListBusinessGroup(ctx, masterxol.TypeAccident)
	require.NoError(t, err)
	// Kejanggalan yang ditiru: AEROSPACE memuat "PA", sedangkan PA dan GA tidak muncul.
	require.Equal(t, []string{"AVIATION HULL", masterxol.TreatyInwardName}, names(accident))

	property, err := repo.ListBusinessGroup(ctx, masterxol.TypeProperty)
	require.NoError(t, err)
	require.Contains(t, names(property), "HEAVY EQUIPMENT")
	require.NotContains(t, names(property), "MARINE CARGO")

	all, err := repo.ListBusinessGroup(ctx, masterxol.TypeUnknown)
	require.NoError(t, err)
	require.Len(t, all, len(sampleBusinessGroup())+1)
}

func TestNumericSortFallsBackToText(t *testing.T) {
	list := []masterxol.Master{{ID: "B"}, {ID: "10010"}, {ID: "1009"}, {ID: "A"}}
	sortByID(list)
	require.Equal(t, "1009", list[0].ID)
	require.Equal(t, "10010", list[1].ID)
}
