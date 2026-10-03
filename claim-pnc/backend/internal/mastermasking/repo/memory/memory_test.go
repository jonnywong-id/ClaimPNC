package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/mastermasking/repo/memory"
)

var base = time.Date(2026, time.September, 20, 3, 0, 0, 0, time.UTC)

func row(id, branchID, branchName, login string, active bool, at time.Time) mastermasking.Masking {
	return mastermasking.Masking{
		ID: id, BranchID: branchID, BranchName: branchName, Login: login,
		Module: "KLAIM", Active: active, InputAt: at,
	}
}

func newRepo() *memory.Repo {
	return memory.NewRepo(
		row("1", "01", "JAKARTA", "ANDI", true, base),
		row("2", "02", "BANDUNG", "budi", false, base.Add(time.Hour)),
		row("3", "01", "JAKARTA", "CITRA", true, base.Add(time.Hour)),
		row("X", "", "", "TANPA_CABANG", true, base.Add(-time.Hour)),
	)
}

func ids(list []mastermasking.Masking) []string {
	out := []string{}
	for _, m := range list {
		out = append(out, m.ID)
	}
	return out
}

// Terbaru lebih dulu; waktu sama dipecah menurut ID menurun, ID bukan angka dianggap nol.
func TestListOrderAndFilters(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	all, err := repo.List(ctx, mastermasking.Filter{})
	require.NoError(t, err)
	require.Equal(t, []string{"3", "2", "1", "X"}, ids(all))

	byLogin, err := repo.List(ctx, mastermasking.Filter{By: mastermasking.SearchByLogin, Keyword: "BUD"})
	require.NoError(t, err)
	require.Equal(t, []string{"2"}, ids(byLogin), "pencarian login tidak peka huruf besar-kecil")

	byBranch, err := repo.List(ctx, mastermasking.Filter{By: mastermasking.SearchByBranch, Keyword: "jakar"})
	require.NoError(t, err)
	require.Equal(t, []string{"3", "1"}, ids(byBranch))

	inactive, err := repo.List(ctx, mastermasking.Filter{By: mastermasking.SearchByStatus, Keyword: "tidak aktif"})
	require.NoError(t, err)
	require.Equal(t, []string{"2"}, ids(inactive))

	invalid, err := repo.List(ctx, mastermasking.Filter{By: mastermasking.SearchByStatus, Keyword: "entah"})
	require.NoError(t, err)
	require.Empty(t, invalid)
}

// ID berikutnya melanjutkan ID angka terbesar; repo kosong mulai dari 1.
func TestInsertAssignsNextIDAndBranchName(t *testing.T) {
	repo := newRepo()
	saved, err := repo.Insert(context.Background(), mastermasking.Masking{
		BranchID: " 02 ", Login: " DEWI ", Module: "KLAIM",
	})
	require.NoError(t, err)
	require.Equal(t, "4", saved.ID)
	require.Equal(t, "BANDUNG", saved.BranchName, "nama cabang dibaca ulang dari daftar cabang")
	require.Equal(t, "DEWI", saved.Login)

	got, err := repo.Get(context.Background(), " 4 ")
	require.NoError(t, err)
	require.Equal(t, saved, got)

	empty := memory.NewRepo()
	first, err := empty.Insert(context.Background(), mastermasking.Masking{BranchID: "99"})
	require.NoError(t, err)
	require.Equal(t, "1", first.ID)
	require.Equal(t, "", first.BranchName, "cabang tak dikenal tidak punya nama")
}

func TestGetAndFindByPair(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	_, err := repo.Get(ctx, "999")
	require.ErrorIs(t, err, mastermasking.ErrNotFound)

	found, err := repo.FindByPair(ctx, " 02", "BUDI ")
	require.NoError(t, err)
	require.Equal(t, "2", found.ID)

	_, err = repo.FindByPair(ctx, "02", "ANDI")
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
}

func TestUpdateAndSetActive(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	updated, err := repo.Update(ctx, mastermasking.Masking{ID: "1", BranchID: "02", Login: "ANDI", Active: false})
	require.NoError(t, err)
	require.Equal(t, "BANDUNG", updated.BranchName)
	require.False(t, updated.Active)

	_, err = repo.Update(ctx, mastermasking.Masking{ID: "999"})
	require.ErrorIs(t, err, mastermasking.ErrNotFound)

	at := base.Add(48 * time.Hour)
	changed, err := repo.SetActive(ctx, " 1 ", true, " ADMIN ", at)
	require.NoError(t, err)
	require.True(t, changed.Active)
	require.Equal(t, "ADMIN", changed.InputBy)
	require.Equal(t, at, changed.InputAt)

	_, err = repo.SetActive(ctx, "999", true, "ADMIN", at)
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
}

// Cabang tambahan tidak menggandakan cabang yang sudah ada; daftar diurutkan menurut nama.
func TestBranches(t *testing.T) {
	repo := newRepo().WithBranches(
		mastermasking.Branch{ID: " 03 ", Name: " AMBON "},
		mastermasking.Branch{ID: "01", Name: "GANDA"},
		mastermasking.Branch{ID: "  ", Name: "KOSONG"},
	)
	ctx := context.Background()

	all, err := repo.ListBranches(ctx, "", 10)
	require.NoError(t, err)
	require.Equal(t, []mastermasking.Branch{
		{ID: "03", Name: "AMBON"}, {ID: "02", Name: "BANDUNG"}, {ID: "01", Name: "JAKARTA"},
	}, all)

	limited, err := repo.ListBranches(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, limited, 1, "batas nol dibetulkan menjadi satu")

	byName, err := repo.ListBranches(ctx, "band", 10)
	require.NoError(t, err)
	require.Equal(t, []mastermasking.Branch{{ID: "02", Name: "BANDUNG"}}, byName)

	byID, err := repo.ListBranches(ctx, "03", 10)
	require.NoError(t, err)
	require.Equal(t, []mastermasking.Branch{{ID: "03", Name: "AMBON"}}, byID)

	exists, err := repo.BranchExists(ctx, " 03 ")
	require.NoError(t, err)
	require.True(t, exists)

	exists, err = repo.BranchExists(ctx, "77")
	require.NoError(t, err)
	require.False(t, exists)
}

// Data contoh membentuk repo yang konsisten: setiap cabang barisnya dikenal.
func TestSampleIsConsistent(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...).WithBranches(memory.SampleBranches()...)
	ctx := context.Background()

	list, err := repo.List(ctx, mastermasking.Filter{})
	require.NoError(t, err)
	require.Len(t, list, len(memory.SampleList()))

	for _, m := range list {
		exists, err := repo.BranchExists(ctx, m.BranchID)
		require.NoError(t, err)
		require.Truef(t, exists, "cabang %q baris %q tidak dikenal", m.BranchID, m.ID)
	}
}
