package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/menu/repo/memory"
)

func TestListSortsBySequenceAndChecksApp(t *testing.T) {
	repo := memory.NewRepo([]menu.Item{{ID: 1, Sequence: 3}, {ID: 2, Sequence: 1}}, nil, nil)
	ctx := context.Background()
	items, err := repo.List(ctx, " "+menu.AppName+" ")
	require.NoError(t, err)
	require.Equal(t, []int{2, 1}, []int{items[0].ID, items[1].ID})

	_, err = repo.List(ctx, "LAIN")
	require.ErrorIs(t, err, menu.ErrAppNotFound)
}

func TestGroupsAndGrantsAreCaseInsensitive(t *testing.T) {
	repo := memory.NewRepo(nil,
		map[string][]string{" admin ": {"IT", "KOMITE"}},
		map[string][]int{"it": {53, 2}, "KOMITE": {2, 7}})
	ctx := context.Background()

	groups, err := repo.GroupsOf(ctx, "ADMIN")
	require.NoError(t, err)
	require.Equal(t, []string{"IT", "KOMITE"}, groups)
	groups, _ = repo.GroupsOf(ctx, "tidak-ada")
	require.Empty(t, groups)

	ids, err := repo.AuthorizedIDs(ctx, menu.AppName, []string{" IT ", "komite", "lain"})
	require.NoError(t, err)
	require.Equal(t, []int{2, 7, 53}, ids, "MENU_ID ganda hanya muncul sekali, terurut")

	_, err = repo.AuthorizedIDs(ctx, "LAIN", []string{"IT"})
	require.ErrorIs(t, err, menu.ErrAppNotFound)
}

func TestSetErrorFailsEveryRead(t *testing.T) {
	repo := memory.NewSampleRepo()
	failure := errors.New("rusak")
	repo.SetError(failure)
	ctx := context.Background()
	_, err := repo.List(ctx, menu.AppName)
	require.ErrorIs(t, err, failure)
	_, err = repo.GroupsOf(ctx, "x")
	require.ErrorIs(t, err, failure)
	_, err = repo.AuthorizedIDs(ctx, menu.AppName, []string{"IT"})
	require.ErrorIs(t, err, failure)
}

func TestDevRepoGrantsDevelopmentLogins(t *testing.T) {
	repo := memory.NewDevRepo()
	for _, login := range []string{"adminpnc", "PICTEKNIKS", "brokercontoh", "profilbolong"} {
		groups, err := repo.GroupsOf(context.Background(), login)
		require.NoError(t, err)
		require.Equal(t, []string{"IT"}, groups, login)
	}
	items, err := repo.List(context.Background(), menu.AppName)
	require.NoError(t, err)
	require.Len(t, items, len(memory.SampleItems()))
	require.NotEmpty(t, memory.SampleGrants())
}
