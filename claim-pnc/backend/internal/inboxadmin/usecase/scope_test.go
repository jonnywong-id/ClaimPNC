package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
	"claim-pnc/internal/inboxadmin/usecase"
)

type scopeClock struct{}

func (scopeClock) Now() time.Time { return time.Date(2026, time.October, 7, 8, 0, 0, 0, time.UTC) }

func scopedService(t *testing.T, store *memory.Store) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return store, nil },
		Clock:        scopeClock{},
	})
	require.NoError(t, err)
	return service
}

func TestBranchStaffOnlySeeTheirBranch(t *testing.T) {
	store := memory.NewStore(
		memory.Row{Tab: inboxadmin.TabAll, BranchCode: "100351", Item: inboxadmin.WorkItem{CaseID: "PNC-1"}},
		memory.Row{Tab: inboxadmin.TabAll, BranchCode: "100090", Item: inboxadmin.WorkItem{CaseID: "PNC-2"}},
	)
	service := scopedService(t, store)
	caller := inboxadmin.Caller{Login: "PETUGAS"}
	list := func() []string {
		listed, err := service.List(context.Background(), "ASM", caller,
			inboxadmin.QueryInput{Tab: inboxadmin.TabAll}, inboxadmin.Pagination{})
		require.NoError(t, err)
		var ids []string
		for _, row := range listed.Page.Items {
			ids = append(ids, row.CaseID)
		}
		return ids
	}

	// Belum terdaftar di cabang mana pun: tidak dibatasi, seperti cabang kosong di Pega.
	require.ElementsMatch(t, []string{"PNC-1", "PNC-2"}, list())

	store.SetBranch("PETUGAS", "100351")
	require.Equal(t, []string{"PNC-1"}, list())

	// Manajer melewati batas cabang.
	store.SetGroups("PETUGAS", "CaseManager")
	require.ElementsMatch(t, []string{"PNC-1", "PNC-2"}, list())

	info, err := service.Viewer(context.Background(), "ASM", caller)
	require.NoError(t, err)
	require.True(t, info.Viewer.Manager)
	require.Equal(t, memory.SampleRegions, info.Regions)
}
