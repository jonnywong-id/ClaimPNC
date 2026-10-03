package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memory"
	"claim-pnc/internal/masterstatus/usecase"
)

var errStore = errors.New("penyimpanan rusak")

// flakyRepo membungkus repo memori dan menggagalkan operasi yang ditandai.
type flakyRepo struct {
	*memory.Repo
	failList, failGet, failInsert, failUpdate error
}

func (f *flakyRepo) List(ctx context.Context) ([]masterstatus.ClaimStatus, error) {
	if f.failList != nil {
		return nil, f.failList
	}
	return f.Repo.List(ctx)
}

func (f *flakyRepo) Get(ctx context.Context, code string) (masterstatus.ClaimStatus, error) {
	if f.failGet != nil {
		return masterstatus.ClaimStatus{}, f.failGet
	}
	return f.Repo.Get(ctx, code)
}

func (f *flakyRepo) Insert(ctx context.Context, label string) (masterstatus.ClaimStatus, error) {
	if f.failInsert != nil {
		return masterstatus.ClaimStatus{}, f.failInsert
	}
	return f.Repo.Insert(ctx, label)
}

func (f *flakyRepo) Update(ctx context.Context, code, label string) (masterstatus.ClaimStatus, error) {
	if f.failUpdate != nil {
		return masterstatus.ClaimStatus{}, f.failUpdate
	}
	return f.Repo.Update(ctx, code, label)
}

func flakyService(t *testing.T, repo *flakyRepo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterstatus.Repo, error) {
			if alias != portalASM {
				return nil, errPortalNotReady
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func newFlaky() *flakyRepo {
	return &flakyRepo{Repo: memory.NewRepo(memory.SampleList()...)}
}

func TestEnsurePortalReady(t *testing.T) {
	service := flakyService(t, newFlaky())
	require.NoError(t, service.EnsurePortalReady(portalASM))
	err := service.EnsurePortalReady("SMAS")
	require.ErrorIs(t, err, errPortalNotReady)
	require.ErrorContains(t, err, `"SMAS"`)
}

func TestEveryOperationForwardsPortalFailure(t *testing.T) {
	service := flakyService(t, newFlaky())
	ctx := context.Background()
	_, err := service.List(ctx, "SMAS")
	require.ErrorIs(t, err, errPortalNotReady)
	_, err = service.Get(ctx, "SMAS", "1134")
	require.ErrorIs(t, err, errPortalNotReady)
	_, err = service.Create(ctx, "SMAS", "Baru")
	require.ErrorIs(t, err, errPortalNotReady)
	_, err = service.Update(ctx, "SMAS", "1134", "Baru")
	require.ErrorIs(t, err, errPortalNotReady)
}

func TestStoreFailuresAreWrapped(t *testing.T) {
	ctx := context.Background()

	repo := newFlaky()
	repo.failList = errStore
	service := flakyService(t, repo)
	_, err := service.List(ctx, portalASM)
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "membaca daftar status")
	_, err = service.Create(ctx, portalASM, "Baru")
	require.ErrorContains(t, err, "memeriksa keunikan label")

	repo = newFlaky()
	repo.failGet = errStore
	service = flakyService(t, repo)
	_, err = service.Get(ctx, portalASM, "1134")
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, `membaca status "1134"`)
	_, err = service.Update(ctx, portalASM, "1134", "Baru")
	require.ErrorIs(t, err, errStore)

	repo = newFlaky()
	repo.failInsert = errStore
	service = flakyService(t, repo)
	_, err = service.Create(ctx, portalASM, "Baru Sekali")
	require.ErrorContains(t, err, "menambah status")

	repo = newFlaky()
	repo.failInsert = masterstatus.ErrCodeTaken
	service = flakyService(t, repo)
	_, err = service.Create(ctx, portalASM, "Baru Sekali")
	require.ErrorIs(t, err, masterstatus.ErrCodeTaken)

	repo = newFlaky()
	repo.failUpdate = errStore
	service = flakyService(t, repo)
	_, err = service.Update(ctx, portalASM, "1134", "Baru Sekali")
	require.ErrorContains(t, err, `mengubah status "1134"`)

	repo = newFlaky()
	repo.failUpdate = masterstatus.ErrNotFound
	service = flakyService(t, repo)
	_, err = service.Update(ctx, portalASM, "1134", "Baru Sekali")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
}

func TestUpdateValidatesBeforeReading(t *testing.T) {
	service := flakyService(t, newFlaky())
	_, err := service.Update(context.Background(), portalASM, "1134", "  ")
	var validation *masterstatus.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestUpdateRejectsLabelOfAnotherStatus(t *testing.T) {
	service := flakyService(t, newFlaky())
	_, err := service.Update(context.Background(), portalASM, "1134", " paid ")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)
}
