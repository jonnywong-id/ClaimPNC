package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
	"claim-pnc/internal/inboxadmin/usecase"
)

// failingRepo selalu gagal membaca antrean.
type failingRepo struct{ err error }

func (r failingRepo) List(context.Context, inboxadmin.Query) ([]inboxadmin.WorkItem, error) {
	return nil, r.err
}

func TestRepoErrorIsWrappedWithTheTabCode(t *testing.T) {
	boom := errors.New("ORA-03113")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return failingRepo{err: boom}, nil },
		Clock:        fixedClock{at: now},
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "utama", caller,
		inboxadmin.QueryInput{Tab: inboxadmin.TabAll}, inboxadmin.Pagination{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "mengambil isi tab "+inboxadmin.TabAll)
}

func TestVeryLargeResultIsWarnedButNotCut(t *testing.T) {
	rows := make([]memory.Row, 0, inboxadmin.LargeResultWarning+1)
	for i := 0; i <= inboxadmin.LargeResultWarning; i++ {
		rows = append(rows, memory.Row{
			Tab:  inboxadmin.TabRCLPUCL,
			Item: inboxadmin.WorkItem{CaseID: fmt.Sprintf("PNC-%05d", i)},
		})
	}
	store := memory.NewStore(rows...)

	logs := &bytes.Buffer{}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return store, nil },
		Clock:        fixedClock{at: now},
		Logger:       slog.New(slog.NewTextHandler(logs, nil)),
	})
	require.NoError(t, err)

	listed, err := service.List(context.Background(), "utama", caller,
		inboxadmin.QueryInput{Tab: inboxadmin.TabRCLPUCL}, inboxadmin.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)

	// Total tetap seluruh baris; peringatan hanya ditulis ke log.
	require.Equal(t, inboxadmin.LargeResultWarning+1, listed.Page.Total)
	require.Len(t, listed.Page.Items, 10)
	require.Contains(t, logs.String(), "level=WARN")
	require.Contains(t, logs.String(), fmt.Sprintf("jumlah_baris=%d", inboxadmin.LargeResultWarning+1))
}
