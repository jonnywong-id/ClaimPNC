package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/repo/memory"
)

func ids(list []masterpanel.Panel) []string {
	out := make([]string, 0, len(list))
	for _, p := range list {
		out = append(out, p.ID)
	}
	return out
}

// Daftar disaring menurut status dan kata kunci nama, terurut ID menurun.
func TestListFiltersByStatusAndNameKeyword(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	approved, err := repo.List(ctx, masterpanel.Filter{Status: masterpanel.StatusApproved})
	require.NoError(t, err)
	require.Contains(t, ids(approved), "01000001")
	for _, p := range approved {
		require.Equal(t, masterpanel.StatusApproved, p.Status)
	}

	pending, err := repo.List(ctx, masterpanel.Filter{
		Status: masterpanel.StatusPending, Keyword: " kaca "})
	require.NoError(t, err)
	require.Equal(t, []string{"01000002"}, ids(pending))

	none, err := repo.List(ctx, masterpanel.Filter{
		Status: masterpanel.StatusPending, Keyword: "tidak-ada"})
	require.NoError(t, err)
	require.Empty(t, none)

	sorted := memory.NewRepo(memory.Options{Rows: []masterpanel.Panel{
		{ID: "A1", Status: masterpanel.StatusPending},
		{ID: "B2", Status: masterpanel.StatusPending},
	}})
	list, err := sorted.List(ctx, masterpanel.Filter{Status: masterpanel.StatusPending})
	require.NoError(t, err)
	require.Equal(t, []string{"B2", "A1"}, ids(list))
}

// Get mengembalikan salinan, sehingga mengubah lokasinya tidak mengubah repo.
func TestGetReturnsAnIndependentCopy(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	panel, err := repo.Get(ctx, " 01000001 ")
	require.NoError(t, err)
	require.Len(t, panel.Location, 2)
	panel.Location[0].Name = "DIUBAH"

	again, err := repo.Get(ctx, "01000001")
	require.NoError(t, err)
	require.Equal(t, "KIRI", again.Location[0].Name)

	_, err = repo.Get(ctx, "99")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
}

// FindByName mengabaikan besar-kecil huruf dan tidak membawa lokasi.
func TestFindByNameIgnoresCaseAndDropsLocations(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	panel, err := repo.FindByName(ctx, "  pintu DEPAN ")
	require.NoError(t, err)
	require.Equal(t, "01000001", panel.ID)
	require.Nil(t, panel.Location)

	_, err = repo.FindByName(ctx, " ")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
	_, err = repo.FindByName(ctx, "tidak ada")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
}

// Penambahan menolak nama ganda; penyuntingan tidak pernah mengganti ID.
func TestInsertAndUpdate(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	require.ErrorIs(t, repo.Insert(ctx, masterpanel.Panel{ID: "X", Name: " PINTU depan"}),
		masterpanel.ErrNameTaken)

	fresh := masterpanel.Panel{ID: "01000009", Name: "Spion", Status: masterpanel.StatusPending}
	require.NoError(t, repo.Insert(ctx, fresh))
	stored, err := repo.Get(ctx, "01000009")
	require.NoError(t, err)
	require.Equal(t, fresh, stored)

	require.NoError(t, repo.Update(ctx, masterpanel.Panel{
		ID: " 01000009 ", Name: "Spion Kanan",
		Location: []masterpanel.PanelLocation{{Name: "KANAN", Side: masterpanel.SideRight}},
	}))
	updated, err := repo.Get(ctx, "01000009")
	require.NoError(t, err)
	require.Equal(t, "01000009", updated.ID)
	require.Equal(t, "Spion Kanan", updated.Name)
	require.Len(t, updated.Location, 1)

	require.ErrorIs(t, repo.Update(ctx, masterpanel.Panel{ID: "99"}), masterpanel.ErrNotFound)
}

// SetStatus hanya menghitung baris yang benar-benar berubah, dan menulis alasannya.
func TestSetStatusCountsOnlyChangedRows(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()

	changed, err := repo.SetStatus(ctx, []string{" 01000002 ", "01000001", "99"},
		masterpanel.StatusApproved, "")
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	changed, err = repo.SetStatus(ctx, []string{"01000002"}, masterpanel.StatusRejected,
		"foto kurang")
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	panel, err := repo.Get(ctx, "01000002")
	require.NoError(t, err)
	require.Equal(t, masterpanel.StatusRejected, panel.Status)
	require.Equal(t, "foto kurang", panel.RejectReason)
}

// NextID melanjutkan nomor urut dengan bentuk kode situs + enam digit.
func TestNextIDContinuesTheSequence(t *testing.T) {
	repo := memory.NewSampleRepo()

	id, err := repo.NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, masterpanel.ComposeID(memory.SamplePanelSite,
		memory.SamplePanelSequence+1, 6), id)

	// Situs kosong jatuh ke situs contoh.
	empty := memory.NewRepo(memory.Options{})
	first, err := empty.NextID(context.Background())
	require.NoError(t, err)
	require.Equal(t, masterpanel.ComposeID(memory.SamplePanelSite, 1, 6), first)
}

// Galat yang dipasang dijawab setiap operasi.
func TestSetErrorFailsEveryOperation(t *testing.T) {
	failure := errors.New("oracle mati")
	repo := memory.NewSampleRepo()
	repo.SetError(failure)
	ctx := context.Background()

	_, err := repo.List(ctx, masterpanel.Filter{})
	require.ErrorIs(t, err, failure)
	_, err = repo.Get(ctx, "01000001")
	require.ErrorIs(t, err, failure)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, repo.Insert(ctx, masterpanel.Panel{}), failure)
	require.ErrorIs(t, repo.Update(ctx, masterpanel.Panel{}), failure)
	_, err = repo.SetStatus(ctx, []string{"x"}, masterpanel.StatusApproved, "")
	require.ErrorIs(t, err, failure)
	_, err = repo.NextID(ctx)
	require.ErrorIs(t, err, failure)
}
