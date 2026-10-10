package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/laporanhasilai/usecase"
)

// brokenRepo gagal pada operasi yang ditandai.
type brokenRepo struct {
	listErr error
}

func (r brokenRepo) List(context.Context, laporanhasilai.Filter, laporanhasilai.Pagination) (laporanhasilai.Page, error) {
	return laporanhasilai.Page{}, r.listErr
}

func serviceOver(t *testing.T, repo laporanhasilai.Repo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (laporanhasilai.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)
	return service
}

func TestSearchStopsOnListFailure(t *testing.T) {
	boom := errors.New("daftar gagal")
	_, err := serviceOver(t, brokenRepo{listErr: boom}).Search(
		context.Background(), "asm", wholeSeptember(), laporanhasilai.Pagination{})
	require.ErrorIs(t, err, boom)
}

func TestExportRefusesUnknownPortal(t *testing.T) {
	_, err := serviceWithSample(t).ListForExport(
		context.Background(), "entitas-lain", wholeSeptember(), laporanhasilai.Pagination{})
	require.ErrorContains(t, err, "portal tidak dikenal")
}

func TestExportPassesListFailureThrough(t *testing.T) {
	boom := errors.New("daftar gagal")
	_, err := serviceOver(t, brokenRepo{listErr: boom}).ListForExport(
		context.Background(), "asm", wholeSeptember(), laporanhasilai.Pagination{})
	require.ErrorIs(t, err, boom)
}
