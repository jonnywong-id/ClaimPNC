package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/repo/memory"
)

var ctx = context.Background()

func ids(rows []mastersurveyors.Surveyor) []string {
	list := make([]string, 0, len(rows))
	for _, row := range rows {
		list = append(list, row.ID)
	}
	return list
}

func TestListMenyaringDanMengurutkanNama(t *testing.T) {
	repo := memory.NewRepo()

	all, total, err := repo.List(ctx, mastersurveyors.Filter{})
	require.NoError(t, err)
	require.Equal(t, len(all), total)
	for i := 1; i < len(all); i++ {
		require.LessOrEqual(t, all[i-1].Name, all[i].Name, "diurutkan menurut nama")
	}

	pending, _, err := repo.List(ctx, mastersurveyors.Filter{Status: mastersurveyors.StatusPending})
	require.NoError(t, err)
	for _, row := range pending {
		require.Equal(t, mastersurveyors.StatusPending, row.Status)
	}

	byName, total, err := repo.List(ctx, mastersurveyors.Filter{Name: " contoh dua "})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, []string{"1000002"}, ids(byName))

	byLogin, _, err := repo.List(ctx, mastersurveyors.Filter{AppLogin: "surveyorsatu"})
	require.NoError(t, err)
	require.Equal(t, []string{"1000001"}, ids(byLogin))

	byType, _, err := repo.List(ctx, mastersurveyors.Filter{TypeCode: " 1003 "})
	require.NoError(t, err)
	require.Equal(t, []string{"1000003"}, ids(byType))
}

// Antrean komite hanya memuat baris menunggu milik komite yang sedang masuk.
func TestListAntreanKomiteSaya(t *testing.T) {
	repo := memory.NewRepo()

	mine, _, err := repo.List(ctx, mastersurveyors.Filter{
		MyCommitteeOnly: true, CommitteeIdentity: "komitecontoh",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"1000002"}, ids(mine),
		"baris berstatus selain menunggu dan komite lain tidak ikut")

	// Identitas kosong tidak mempersempit apa pun.
	everyone, total, err := repo.List(ctx, mastersurveyors.Filter{MyCommitteeOnly: true})
	require.NoError(t, err)
	require.Equal(t, len(everyone), total)
	require.Greater(t, total, 1)
}

func TestListPaginasi(t *testing.T) {
	repo := memory.NewRepo()

	_, total, err := repo.List(ctx, mastersurveyors.Filter{})
	require.NoError(t, err)

	page, pageTotal, err := repo.List(ctx, mastersurveyors.Filter{Limit: 1, Offset: -3})
	require.NoError(t, err)
	require.Equal(t, total, pageTotal)
	require.Len(t, page, 1)

	beyond, beyondTotal, err := repo.List(ctx, mastersurveyors.Filter{Offset: total})
	require.NoError(t, err)
	require.Nil(t, beyond)
	require.Equal(t, total, beyondTotal)
}

func TestGetDanPencarian(t *testing.T) {
	repo := memory.NewRepo()

	got, err := repo.Get(ctx, " 1000001 ")
	require.NoError(t, err)
	require.Equal(t, "SURVEYORSATU", got.AppLogin)

	_, err = repo.Get(ctx, "x")
	require.ErrorIs(t, err, mastersurveyors.ErrNotFound)

	byName, err := repo.FindByNameKey(ctx, "surveyor contoh   SATU")
	require.NoError(t, err)
	require.Equal(t, []string{"1000001"}, ids(byName))

	none, err := repo.FindByNameKey(ctx, "   ")
	require.NoError(t, err)
	require.Nil(t, none)

	byLogin, err := repo.FindByAppLogin(ctx, " surveyorsatu ")
	require.NoError(t, err)
	require.Equal(t, []string{"1000001"}, ids(byLogin))

	none, err = repo.FindByAppLogin(ctx, "")
	require.NoError(t, err)
	require.Nil(t, none)
}

// Kode baru melompati kode yang sudah dipakai contoh dan memakai enam digit.
func TestInsertMenerbitkanKode(t *testing.T) {
	repo := memory.NewRepo()

	saved, err := repo.Insert(ctx, mastersurveyors.Surveyor{Name: " Baru ", AppLogin: "BARU"})
	require.NoError(t, err)
	require.Len(t, saved.ID, 7)
	require.Equal(t, "1", saved.ID[:1])
	require.Equal(t, "Baru", saved.Name)

	again, err := repo.Get(ctx, saved.ID)
	require.NoError(t, err)
	require.Equal(t, saved, again)

	_, err = repo.Insert(ctx, mastersurveyors.Surveyor{Name: "baru"})
	require.ErrorIs(t, err, mastersurveyors.ErrNameTaken)

	_, err = repo.Insert(ctx, mastersurveyors.Surveyor{Name: "Lain", AppLogin: "baru"})
	require.ErrorIs(t, err, mastersurveyors.ErrLoginTaken)
}

func TestUpdate(t *testing.T) {
	repo := memory.NewRepo()

	row, err := repo.Get(ctx, "1000002")
	require.NoError(t, err)

	// Mengubah baris tanpa mengubah namanya sendiri tidak ditolak oleh namanya.
	row.Address = "Alamat Baru"
	require.NoError(t, repo.Update(ctx, row))
	again, err := repo.Get(ctx, "1000002")
	require.NoError(t, err)
	require.Equal(t, "Alamat Baru", again.Address)

	clash := row
	clash.Name = "SURVEYOR CONTOH SATU"
	require.ErrorIs(t, repo.Update(ctx, clash), mastersurveyors.ErrNameTaken)

	clash = row
	clash.AppLogin = "surveyorsatu"
	require.ErrorIs(t, repo.Update(ctx, clash), mastersurveyors.ErrLoginTaken)

	require.ErrorIs(t, repo.Update(ctx, mastersurveyors.Surveyor{ID: "999"}),
		mastersurveyors.ErrNotFound)
}
