package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
	"claim-pnc/internal/masterpenolakan/usecase"
)

// newServiceKomite merakit layanan komite dengan satu portal "asm" yang dikenal.
func newServiceKomite(t *testing.T, repo *memory.RepoKomite) *usecase.ServiceKomite {
	t.Helper()
	service, err := usecase.NewServiceKomite(usecase.OptionsKomite{
		RepoSelector: func(alias string) (masterpenolakan.RepoKomite, error) {
			if alias != "asm" {
				return nil, errPortal
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceKomiteRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewServiceKomite(usecase.OptionsKomite{})
	require.ErrorContains(t, err, "RepoSelector komite wajib diisi")
}

func TestServiceKomiteListGetCreateUpdate(t *testing.T) {
	service := newServiceKomite(t, memory.NewRepoKomite(memory.SampleListKomite()...))
	ctx := context.Background()

	list, err := service.List(ctx, "asm")
	require.NoError(t, err)
	require.Len(t, list, 3)

	row, err := service.Get(ctx, "asm", "112")
	require.NoError(t, err)
	require.Equal(t, "DOKUMEN PENDUKUNG TIDAK MEYAKINKAN", row.Note)

	created, err := service.Create(ctx, "asm", masterpenolakan.InputKomite{Note: "  CATATAN BARU "})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "114", Note: "CATATAN BARU"}, created)

	updated, err := service.Update(ctx, "asm", "111", masterpenolakan.InputKomite{Note: " UBAH "})
	require.NoError(t, err)
	require.Equal(t, masterpenolakan.CommitteeRejection{ID: "111", Note: "UBAH"}, updated)

	_, err = service.Update(ctx, "asm", "999", masterpenolakan.InputKomite{Note: "X"})
	require.ErrorIs(t, err, masterpenolakan.ErrKomiteNotFound)
}

func TestServiceKomiteRejectsEmptyNote(t *testing.T) {
	service := newServiceKomite(t, memory.NewRepoKomite())
	ctx := context.Background()

	var validation *masterpenolakan.ValidationError
	_, err := service.Create(ctx, "asm", masterpenolakan.InputKomite{Note: "  "})
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "catatan", validation.Violation[0].Field)

	_, err = service.Update(ctx, "asm", "111", masterpenolakan.InputKomite{})
	require.ErrorAs(t, err, &validation)
}

func TestServiceKomiteUnknownPortal(t *testing.T) {
	service := newServiceKomite(t, memory.NewRepoKomite())
	ctx := context.Background()

	_, err := service.List(ctx, "x")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "x", "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "x", masterpenolakan.InputKomite{Note: "A"})
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "x", "1", masterpenolakan.InputKomite{Note: "A"})
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("x")
	require.ErrorIs(t, err, errPortal)
	require.NoError(t, service.EnsurePortalReady("asm"))
}
