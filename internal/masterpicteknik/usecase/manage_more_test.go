package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterpicteknik/directory"
	"claim-pnc/internal/masterpicteknik/repo/memory"
	"claim-pnc/internal/masterpicteknik/usecase"
)

var (
	errStore    = errors.New("penyimpanan rusak")
	errNotReady = errors.New("portal belum siap")
)

func serviceWith(t *testing.T, repo *memory.Repo, dir *directory.Fake) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpicteknik.Repo, error) {
			if alias != "ASM" {
				return nil, errNotReady
			}
			return repo, nil
		},
		Directory: dir,
	})
	require.NoError(t, err)
	return service
}

func TestEnsurePortalReadyNamesThePortal(t *testing.T) {
	service := serviceWith(t, memory.NewRepo(), directory.NewFake())

	require.NoError(t, service.EnsurePortalReady("ASM"))
	err := service.EnsurePortalReady("SMAS")
	require.ErrorIs(t, err, errNotReady)
	require.ErrorContains(t, err, `portal "SMAS" tidak dapat dilayani`)
}

func TestReadsWrapStorageFailures(t *testing.T) {
	repo := memory.NewRepo(memory.SampleList()...)
	repo.SetError(errStore)
	service := serviceWith(t, repo, directory.NewFake(directory.SampleEmployees()...))
	ctx := context.Background()

	_, err := service.List(ctx, "ASM")
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "membaca daftar PIC teknik")

	_, err = service.Get(ctx, "ASM", "PICTEKNIK01")
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, `membaca PIC "PICTEKNIK01"`)

	_, err = service.List(ctx, "SMAS")
	require.ErrorIs(t, err, errNotReady)
	_, err = service.Get(ctx, "SMAS", "PICTEKNIK01")
	require.ErrorIs(t, err, errNotReady)
	_, err = service.Lookup(ctx, "SMAS", "PICTEKNIK01")
	require.ErrorIs(t, err, errNotReady)
}

func TestAnUnexpectedDirectoryFailureIsWrapped(t *testing.T) {
	dir := directory.NewFake()
	dir.SetError(errors.New("tak terduga"))
	service := serviceWith(t, memory.NewRepo(), dir)

	_, err := service.Lookup(context.Background(), "ASM", "PICTEKNIK01")
	require.ErrorContains(t, err, `mencari pegawai "PICTEKNIK01"`)

	dir.SetError(masterpicteknik.ErrDirectoryUnreachable)
	_, err = service.Lookup(context.Background(), "ASM", "PICTEKNIK01")
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
}
