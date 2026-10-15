package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/repo/memory"
)

func logins(list []masterlogin.SurveyorLogin) []string {
	out := []string{}
	for _, one := range list {
		out = append(out, one.Login)
	}
	return out
}

func TestListSortsByNameThenLoginAndFilters(t *testing.T) {
	repo := memory.NewRepo(memory.Options{Rows: []masterlogin.SurveyorLogin{
		{Name: "B", Login: "b2"},
		{Name: "A", Login: "a1", Email: "x@contoh"},
		{Name: "B", Login: "b1"},
	}})

	all, err := repo.List(context.Background(), masterlogin.Filter{})
	require.NoError(t, err)
	require.Equal(t, []string{"a1", "b1", "b2"}, logins(all))

	byEmail, err := repo.List(context.Background(), masterlogin.Filter{Keyword: " X@CONTOH "})
	require.NoError(t, err)
	require.Equal(t, []string{"a1"}, logins(byEmail))

	byLogin, err := repo.List(context.Background(), masterlogin.Filter{Keyword: "B2"})
	require.NoError(t, err)
	require.Equal(t, []string{"b2"}, logins(byLogin))

	none, err := repo.List(context.Background(), masterlogin.Filter{Keyword: "zzz"})
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestGetAndFindLeaderOf(t *testing.T) {
	repo := memory.NewSampleRepo()

	found, err := repo.Get(context.Background(), " RinaAyuLestari ")
	require.NoError(t, err)
	require.Equal(t, "BudiHartono", found.LeaderLogin)

	_, err = repo.Get(context.Background(), "  ")
	require.ErrorIs(t, err, masterlogin.ErrNotFound)

	leader, err := repo.FindLeaderOf(context.Background(), "RinaAyuLestari")
	require.NoError(t, err)
	require.Equal(t, "BudiHartono", leader)

	_, err = repo.FindLeaderOf(context.Background(), "tidakada")
	require.ErrorIs(t, err, masterlogin.ErrNotFound)
}

func TestInsertRejectsTakenLoginCaseInsensitive(t *testing.T) {
	repo := memory.NewSampleRepo()

	_, err := repo.Insert(context.Background(), masterlogin.SurveyorLogin{Login: " budihartono "})
	require.ErrorIs(t, err, masterlogin.ErrLoginTaken)

	saved, err := repo.Insert(context.Background(), masterlogin.SurveyorLogin{Name: "N", Login: "Baru"})
	require.NoError(t, err)
	require.Equal(t, "Baru", saved.Login)

	got, err := repo.Get(context.Background(), "Baru")
	require.NoError(t, err)
	require.Equal(t, "N", got.Name)
}

func TestUpdateReplacesExistingRow(t *testing.T) {
	repo := memory.NewSampleRepo()

	require.NoError(t, repo.Update(context.Background(),
		masterlogin.SurveyorLogin{Login: "DewiKartika", Name: "DEWI BARU"}))
	got, err := repo.Get(context.Background(), "DewiKartika")
	require.NoError(t, err)
	require.Equal(t, "DEWI BARU", got.Name)

	require.ErrorIs(t, repo.Update(context.Background(),
		masterlogin.SurveyorLogin{Login: "tidakada"}), masterlogin.ErrNotFound)
}

func TestSetErrorFailsEveryOperation(t *testing.T) {
	repo := memory.NewSampleRepo()
	boom := errors.New("basis data mati")
	repo.SetError(boom)

	_, err := repo.List(context.Background(), masterlogin.Filter{})
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(context.Background(), "DewiKartika")
	require.ErrorIs(t, err, boom)
	_, err = repo.FindLeaderOf(context.Background(), "DewiKartika")
	require.ErrorIs(t, err, boom)
	_, err = repo.Insert(context.Background(), masterlogin.SurveyorLogin{Login: "x"})
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, repo.Update(context.Background(), masterlogin.SurveyorLogin{Login: "x"}), boom)
}
