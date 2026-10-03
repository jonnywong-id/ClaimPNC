package usecase_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/usecase"
)

// manyPICRepo mengembalikan rekap PIC sebanyak count baris.
type manyPICRepo struct{ count int }

func (manyPICRepo) ListClaims(context.Context, inboxprogressclaim.ClaimQuery, inboxprogressclaim.Pagination) (inboxprogressclaim.ClaimPage, error) {
	return inboxprogressclaim.ClaimPage{}, nil
}

func (r manyPICRepo) ListPICSummary(context.Context, inboxprogressclaim.PICQuery) ([]inboxprogressclaim.PICSummary, error) {
	rows := make([]inboxprogressclaim.PICSummary, r.count)
	for i := range rows {
		rows[i].PIC = fmt.Sprintf("PIC%d", i)
	}
	return rows, nil
}

func perPICInput() inboxprogressclaim.QueryInput {
	return inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewPerPIC, Business: "NONMBU"}
}

func newServiceWith(t *testing.T, repo inboxprogressclaim.Repo, logger *slog.Logger) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxprogressclaim.Repo, error) { return repo, nil },
		Clock:        fixedClock{at: today},
		Logger:       logger,
	})
	require.NoError(t, err)
	return service
}

func TestLargePerPICRecapIsLoggedAsWarning(t *testing.T) {
	logs := &bytes.Buffer{}
	service := newServiceWith(t, manyPICRepo{count: usecase.LargeRecapWarning + 1}, slog.New(slog.NewJSONHandler(logs, nil)))

	listed, err := service.List(context.Background(), "asm", inboxprogressclaim.Caller{Login: "ADMIN"}, perPICInput(), inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Len(t, listed.PICRows, usecase.LargeRecapWarning+1)
	require.Contains(t, logs.String(), "rekap Progress Klaim per PIC menarik sangat banyak baris")
	require.Contains(t, logs.String(), `"jumlah_baris":501`)
}

func TestRecapAtThresholdIsNotLogged(t *testing.T) {
	logs := &bytes.Buffer{}
	service := newServiceWith(t, manyPICRepo{count: usecase.LargeRecapWarning}, slog.New(slog.NewJSONHandler(logs, nil)))

	_, err := service.List(context.Background(), "asm", inboxprogressclaim.Caller{Login: "ADMIN"}, perPICInput(), inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Empty(t, logs.String())

	// Tanpa logger sama sekali, rekap besar tetap dikembalikan.
	service = newServiceWith(t, manyPICRepo{count: usecase.LargeRecapWarning + 5}, nil)
	listed, err := service.List(context.Background(), "asm", inboxprogressclaim.Caller{Login: "ADMIN"}, perPICInput(), inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Len(t, listed.PICRows, usecase.LargeRecapWarning+5)
}

func TestPerPICStorageFailureIsWrapped(t *testing.T) {
	service := newServiceWith(t, failingRepo{}, nil)
	_, err := service.List(context.Background(), "asm", inboxprogressclaim.Caller{Login: "ADMIN"}, perPICInput(), inboxprogressclaim.Pagination{})
	require.ErrorContains(t, err, "mengambil isi bagian per-pic")
}

func TestInvalidQueryIsReturnedBeforeTouchingStorage(t *testing.T) {
	service := newServiceWith(t, failingRepo{}, nil)
	_, err := service.List(context.Background(), "asm", inboxprogressclaim.Caller{}, inboxprogressclaim.QueryInput{}, inboxprogressclaim.Pagination{})
	require.ErrorIs(t, err, inboxprogressclaim.ErrCallerUnknown)
}
