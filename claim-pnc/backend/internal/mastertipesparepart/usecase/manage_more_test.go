package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesparepart"
	"claim-pnc/internal/mastertipesparepart/repo/memory"
	"claim-pnc/internal/mastertipesparepart/usecase"
)

var errStorage = errors.New("koneksi putus")

// stubStore membungkus repo memori dan dapat menggagalkan satu langkah tertentu, untuk
// keadaan yang tidak dapat dibentuk repo memori sendiri.
type stubStore struct {
	*memory.Repo
	insertID    *string
	insertErr   error
	findErr     error
	updateErr   error
	categoryErr error
}

func (s *stubStore) Insert(ctx context.Context, t mastertipesparepart.PartType) (mastertipesparepart.PartType, error) {
	if s.insertErr != nil {
		return mastertipesparepart.PartType{}, s.insertErr
	}
	saved, err := s.Repo.Insert(ctx, t)
	if s.insertID != nil {
		saved.ID = *s.insertID
	}
	return saved, err
}

func (s *stubStore) FindByName(ctx context.Context, name string) (mastertipesparepart.PartType, error) {
	if s.findErr != nil {
		return mastertipesparepart.PartType{}, s.findErr
	}
	return s.Repo.FindByName(ctx, name)
}

func (s *stubStore) Update(ctx context.Context, t mastertipesparepart.PartType) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Repo.Update(ctx, t)
}

func (s *stubStore) ListCategories(ctx context.Context) ([]mastertipesparepart.Category, error) {
	if s.categoryErr != nil {
		return nil, s.categoryErr
	}
	return s.Repo.ListCategories(ctx)
}

func newStubService(t *testing.T, store *stubStore) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastertipesparepart.Store, error) { return store, nil },
	})
	require.NoError(t, err)
	return service
}

func TestGetReturnsRowWithCategoryName(t *testing.T) {
	service, _ := newSampleService(t)
	got, err := service.Get(context.Background(), portalAlias, " 3 ")
	require.NoError(t, err)
	require.Equal(t, "HYDRAULIC PUMP", got.Name)
	require.Equal(t, "HYDRAULIC", got.CategoryName)
}

// Kegagalan membaca kategori menggagalkan pilihan, penambahan, dan penyimpanan.
func TestCategoryFailureIsPropagated(t *testing.T) {
	store := &stubStore{Repo: memory.NewSampleRepo(), categoryErr: errStorage}
	service := newStubService(t, store)
	ctx := context.Background()

	_, err := service.Choices(ctx, portalAlias)
	require.ErrorIs(t, err, errStorage)
	_, err = service.Create(ctx, portalAlias, newInput("SWING MOTOR"), actor(), nil)
	require.ErrorIs(t, err, errStorage)
	_, err = service.Save(ctx, portalAlias, "1", newInput("FUEL FILTER"), actor(), nil)
	require.ErrorIs(t, err, errStorage)
}

// Pilihan ditandai terpotong bila jumlahnya mencapai MaxLookupRows.
func TestChoicesMarksTruncation(t *testing.T) {
	many := make([]mastertipesparepart.Category, mastertipesparepart.MaxLookupRows)
	for i := range many {
		many[i] = mastertipesparepart.Category{ID: string(rune('a' + i%26)), Name: "K"}
	}
	service := newService(t, memory.NewRepo(memory.Options{Category: many}))
	set, err := service.Choices(context.Background(), portalAlias)
	require.NoError(t, err)
	require.True(t, set.Truncated)

	small, err := newService(t, memory.NewSampleRepo()).Choices(context.Background(), portalAlias)
	require.NoError(t, err)
	require.False(t, small.Truncated)
}

// Galat penyisipan diteruskan, dan ID kosong ditolak terang-terangan.
func TestCreateInsertFailures(t *testing.T) {
	ctx := context.Background()

	blank := "  "
	_, err := newStubService(t, &stubStore{Repo: memory.NewSampleRepo(), insertID: &blank}).
		Create(ctx, portalAlias, newInput("SWING MOTOR"), actor(), nil)
	require.ErrorContains(t, err, "ID tipe yang diterbitkan kosong")

	insertFails := newStubService(t, &stubStore{Repo: memory.NewSampleRepo(), insertErr: errStorage})
	_, err = insertFails.Create(ctx, portalAlias, newInput("SWING MOTOR"), actor(), nil)
	require.ErrorIs(t, err, errStorage)
}

// Isian tidak sah pada penyimpanan ditolak; galat pencarian nama dan penyimpanan diteruskan.
func TestSaveFailures(t *testing.T) {
	ctx := context.Background()

	service, _ := newSampleService(t)
	_, err := service.Save(ctx, portalAlias, "1", mastertipesparepart.Input{}, actor(), nil)
	var validation *mastertipesparepart.ValidationError
	require.ErrorAs(t, err, &validation)

	findFails := newStubService(t, &stubStore{Repo: memory.NewSampleRepo(), findErr: errStorage})
	_, err = findFails.Save(ctx, portalAlias, "1", newInput("FUEL FILTER"), actor(), nil)
	require.ErrorIs(t, err, errStorage)

	updateFails := newStubService(t, &stubStore{Repo: memory.NewSampleRepo(), updateErr: errStorage})
	_, err = updateFails.Save(ctx, portalAlias, "1", newInput("FUEL FILTER"), actor(), nil)
	require.ErrorIs(t, err, errStorage)
}

// Kegagalan penetapan status diteruskan.
func TestDecideForwardsStorageFailure(t *testing.T) {
	service, repo := newSampleService(t)
	repo.SetError(errStorage)
	_, err := service.Decide(context.Background(), portalAlias, []string{"4"},
		mastertipesparepart.StatusApproved, actor(), nil)
	require.ErrorIs(t, err, errStorage)
}

// Penambahan, penyimpanan, dan keputusan tercatat di log beserta pelakunya; keputusan yang
// tidak menyentuh seluruh baris memberi peringatan.
func TestEventsAreLogged(t *testing.T) {
	service, _ := newSampleService(t)
	buffer := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buffer, nil))
	ctx := context.Background()

	_, err := service.Create(ctx, portalAlias, newInput("SWING MOTOR"), actor(), logger)
	require.NoError(t, err)
	require.Contains(t, buffer.String(), `"peristiwa":"tipe sparepart baru diajukan"`)
	require.Contains(t, buffer.String(), `"oleh":"PNCADMIN"`)

	_, err = service.Save(ctx, portalAlias, "1", newInput("FUEL FILTER"), actor(), logger)
	require.NoError(t, err)
	require.Contains(t, buffer.String(), `"peristiwa":"tipe sparepart diubah"`)

	changed, err := service.Decide(ctx, portalAlias, []string{"4", "404"},
		mastertipesparepart.StatusRejected, actor(), logger)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	require.Contains(t, buffer.String(), `"level":"WARN"`)
	require.Contains(t, buffer.String(), `"dipilih":2`)
	require.Contains(t, buffer.String(), `"status":"2"`)
}
