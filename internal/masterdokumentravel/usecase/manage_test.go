package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/repo/memory"
)

var errPortal = errors.New("portal tidak siap")

func newTestService(t *testing.T) (*Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewRepo(memory.SampleList()...)
	service, err := NewService(Options{RepoSelector: func(alias string) (masterdokumentravel.Repo, error) {
		if alias != "ASM" {
			return nil, errPortal
		}
		return repo, nil
	}})
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceRequiresSelector(t *testing.T) {
	_, err := NewService(Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")
}

func TestServiceListGetCreate(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	rows, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, rows, 6)

	doc, err := service.Get(ctx, "ASM", "100002")
	require.NoError(t, err)
	require.Equal(t, "Tiket Perjalanan", doc.Name)

	created, err := service.Create(ctx, "ASM", masterdokumentravel.Input{Name: "  Visa  "})
	require.NoError(t, err)
	require.Equal(t, masterdokumentravel.TravelDocument{ID: "100007", Name: "Visa"}, created)

	require.NoError(t, service.EnsurePortalReady("ASM"))
}

func TestServiceUpdate(t *testing.T) {
	service, repo := newTestService(t)
	ctx := context.Background()

	updated, err := service.Update(ctx, "ASM", "100001", masterdokumentravel.Input{Name: " Paspor Baru "})
	require.NoError(t, err)
	require.Equal(t, masterdokumentravel.TravelDocument{ID: "100001", Name: "Paspor Baru"}, updated)

	stored, err := repo.Get(ctx, "100001")
	require.NoError(t, err)
	require.Equal(t, "Paspor Baru", stored.Name)

	// Baris yang tidak ada ditolak sebelum menulis apa pun.
	_, err = service.Update(ctx, "ASM", "999999", masterdokumentravel.Input{Name: "x"})
	require.ErrorIs(t, err, masterdokumentravel.ErrNotFound)
}

func TestServiceUpdateReportsWriteFailure(t *testing.T) {
	boom := errors.New("tulis gagal")
	service, err := NewService(Options{RepoSelector: func(string) (masterdokumentravel.Repo, error) {
		return failingUpdateRepo{Repo: memory.NewRepo(memory.SampleList()...), err: boom}, nil
	}})
	require.NoError(t, err)

	_, err = service.Update(context.Background(), "ASM", "100001", masterdokumentravel.Input{Name: "x"})
	require.ErrorIs(t, err, boom)
}

// failingUpdateRepo membaca dari repo memory tetapi menggagalkan setiap Update.
type failingUpdateRepo struct {
	*memory.Repo
	err error
}

func (r failingUpdateRepo) Update(context.Context, masterdokumentravel.TravelDocument) error {
	return r.err
}

func TestServicePropagatesPortalFailure(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "XX")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "XX", "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "XX", masterdokumentravel.Input{})
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "XX", "1", masterdokumentravel.Input{})
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("XX")
	require.ErrorIs(t, err, errPortal)
	require.ErrorContains(t, err, `portal "XX" tidak dapat dilayani`)
}
