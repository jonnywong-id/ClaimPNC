package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterpenyebabkerugian/usecase"
	"claim-pnc/internal/portal"
)

// repoGagal menjawab setiap operasi dengan galat yang sama.
type repoGagal struct{ err error }

func (r repoGagal) List(context.Context) ([]masterpenyebabkerugian.CauseOfLoss, error) {
	return nil, r.err
}

func (r repoGagal) Get(context.Context, string) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, r.err
}

func (r repoGagal) Insert(context.Context, string) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, r.err
}

func (r repoGagal) Update(context.Context, string, string) (masterpenyebabkerugian.CauseOfLoss, error) {
	return masterpenyebabkerugian.CauseOfLoss{}, r.err
}

func serviceOver(t *testing.T, repo masterpenyebabkerugian.Repo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpenyebabkerugian.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)
	return service
}

// Galat bentrok ID dan tanpa situs diteruskan apa adanya; galat lain dibungkus.
func TestCreatePassesDomainErrorsThroughAndWrapsTheRest(t *testing.T) {
	ctx := context.Background()

	for _, domainErr := range []error{
		masterpenyebabkerugian.ErrIDTaken, masterpenyebabkerugian.ErrNoSite,
	} {
		_, err := serviceOver(t, repoGagal{err: domainErr}).Create(ctx, portalASM, "x")
		require.Same(t, domainErr, err)
	}

	_, err := serviceOver(t, repoGagal{err: errors.New("putus")}).Create(ctx, portalASM, "x")
	require.EqualError(t, err, "masterpenyebabkerugian/usecase: menambah penyebab kerugian: putus")
}

// Galat penyimpanan pada Get dan Update dibungkus dengan ID-nya.
func TestGetAndUpdateWrapStorageFailures(t *testing.T) {
	ctx := context.Background()
	service := serviceOver(t, repoGagal{err: errors.New("putus")})

	_, err := service.Get(ctx, portalASM, " 1001 ")
	require.EqualError(t, err,
		`masterpenyebabkerugian/usecase: membaca penyebab kerugian " 1001 ": putus`)

	_, err = service.Update(ctx, portalASM, " 1001 ", "x")
	require.EqualError(t, err,
		`masterpenyebabkerugian/usecase: mengubah penyebab kerugian "1001": putus`)
}

// Penyuntingan dengan deskripsi terlalu panjang ditolak sebelum penyimpanan disentuh.
func TestUpdateRejectsTooLongDescription(t *testing.T) {
	service, asm, _ := newService(t)

	_, err := service.Update(context.Background(), portalASM, "1001",
		strings.Repeat("x", masterpenyebabkerugian.MaxDescriptionLength+1))
	var validation *masterpenyebabkerugian.ValidationError
	require.ErrorAs(t, err, &validation)

	cause, err := asm.Get(context.Background(), "1001")
	require.NoError(t, err)
	require.Equal(t, "Contoh Golongan A", cause.Description)
}

// Portal yang tidak dapat dilayani ditolak pada penyuntingan.
func TestUpdateOnAnUnservedPortalIsRejected(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Update(context.Background(), "SMAS", "1001", "x")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Get(context.Background(), "SMAS", "1001")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(context.Background(), "SMAS", "x")
	require.ErrorIs(t, err, portal.ErrNotReady)
}
