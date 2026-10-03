package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumen"
	"claim-pnc/internal/daftartipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"
)

// failingUpdateRepo lolos saat Get tetapi gagal saat Update — meniru baris yang hilang
// atau koneksi yang putus di antara pemeriksaan dan penyimpanan.
type failingUpdateRepo struct {
	*recordingRepo
	err error
}

func (r *failingUpdateRepo) Update(context.Context, daftartipedokumen.DocumentType, daftartipedokumen.Editor) error {
	return r.err
}

// mustService merakit Service dengan selektor repo pilihan uji dan jam tetap.
func mustService(t *testing.T, selector daftartipedokumen.RepoSelector) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{RepoSelector: selector, Clock: clock.FixedAt(savedAt)})
	require.NoError(t, err)
	return service
}

func TestGetAndUpdateRejectUnavailablePortal(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Get(context.Background(), "ASI", "10001")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Update(context.Background(), "ASI", "10001", daftartipedokumen.Input{}, "SOMEUSER")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

func TestUpdateTrimsIDAndInputBeforeSaving(t *testing.T) {
	service, repo := newService(t)

	saved, err := service.Update(context.Background(), "ASM", " 10003 ",
		daftartipedokumen.Input{Type: "  Dokumen Komite Baru  ", ProcessStatus: " Komite "}, " SOMEUSER ")
	require.NoError(t, err)
	require.Equal(t, daftartipedokumen.DocumentType{ID: "10003", Type: "Dokumen Komite Baru", ProcessStatus: "Komite"}, saved)
	require.Equal(t, "SOMEUSER", repo.lastEditor.Identity)
	require.Equal(t, savedAt, repo.lastEditor.At)
}

func TestUpdatePropagatesRepositoryUpdateFailure(t *testing.T) {
	_, base := newService(t)
	boom := errors.New("koneksi terputus")
	repo := &failingUpdateRepo{recordingRepo: base, err: boom}

	service := mustService(t, func(string) (daftartipedokumen.Repo, error) { return repo, nil })
	_, err := service.Update(context.Background(), "ASM", "10001", daftartipedokumen.Input{Type: "X"}, "SOMEUSER")
	require.ErrorIs(t, err, boom)
}

func TestEnsurePortalReady(t *testing.T) {
	service, _ := newService(t)

	require.NoError(t, service.EnsurePortalReady("ASM"))

	err := service.EnsurePortalReady("ASI")
	require.ErrorIs(t, err, portal.ErrNotReady)
	require.Contains(t, err.Error(), `portal "ASI" tidak dapat dilayani`)
}
