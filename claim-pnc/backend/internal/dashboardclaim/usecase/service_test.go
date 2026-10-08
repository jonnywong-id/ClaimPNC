package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/memory"
	"claim-pnc/internal/dashboardclaim/usecase"
)

// faultyRepo gagal pada operasi yang disebut, dan berhasil pada sisanya.
type faultyRepo struct {
	failOutstanding bool
	failSurveyKind  dashboardclaim.SurveyorType
	failList        bool
}

var errStore = errors.New("ora-03113")

func (f faultyRepo) CountOutstanding(context.Context, dashboardclaim.Filter) (int, error) {
	if f.failOutstanding {
		return 0, errStore
	}
	return 1, nil
}

func (f faultyRepo) CountSurvey(_ context.Context, kind dashboardclaim.SurveyorType, _ dashboardclaim.Filter) (int, error) {
	if kind == f.failSurveyKind {
		return 0, errStore
	}
	return 1, nil
}

func (f faultyRepo) ListOutstanding(context.Context, dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	if f.failList {
		return dashboardclaim.ClaimPage{}, errStore
	}
	return dashboardclaim.ClaimPage{}, nil
}

func (f faultyRepo) ListSurvey(context.Context, dashboardclaim.SurveyorType, dashboardclaim.Filter) (dashboardclaim.SurveyPage, error) {
	if f.failList {
		return dashboardclaim.SurveyPage{}, errStore
	}
	return dashboardclaim.SurveyPage{}, nil
}

func (f faultyRepo) ListHolding(context.Context, dashboardclaim.Filter) (dashboardclaim.HoldingPage, error) {
	if f.failList {
		return dashboardclaim.HoldingPage{}, errStore
	}
	return dashboardclaim.HoldingPage{}, nil
}

// faultyClosed selalu gagal.
type faultyClosed struct{}

func (faultyClosed) Count(context.Context, string, dashboardclaim.Filter) (int, error) {
	return 0, errStore
}

func (faultyClosed) List(context.Context, string, dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	return dashboardclaim.ClaimPage{}, errStore
}

func newService(t *testing.T, repo dashboardclaim.Repo, closed dashboardclaim.ClosedClaimReader) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) { return repo, nil },
		ClosedClaim:  closed,
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRequiresBothSeams(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{ClosedClaim: memory.NewClosedReader(nil)})
	require.EqualError(t, err, "dashboardclaim/usecase: RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) { return nil, nil },
	})
	require.EqualError(t, err, "dashboardclaim/usecase: ClosedClaim wajib diisi")
}

// Keempat kartu dihitung dengan penyaring hitung (tanpa paginasi).
func TestCountsFromSample(t *testing.T) {
	service := newService(t,
		memory.NewRepo(memory.SampleOutstanding(), memory.SampleSurveys()),
		memory.NewClosedReader(memory.SampleClosed()))

	result, err := service.Counts(context.Background(), usecase.Query{
		PortalAlias: "ASM",
		Filter:      dashboardclaim.Filter{Limit: 10, Offset: 5},
	})
	require.NoError(t, err)
	require.Equal(t, dashboardclaim.Counts{
		Outstanding: 5, CloseClaim: 2, LossAdjuster: 2, InternalSurveyor: 2,
	}, result.Counts)
	require.Equal(t, dashboardclaim.Filter{Business: dashboardclaim.BusinessAll}, result.Filter)
}

func TestCountsErrors(t *testing.T) {
	selectorErr := errors.New("portal belum siap")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) { return nil, selectorErr },
		ClosedClaim:  memory.NewClosedReader(nil),
	})
	require.NoError(t, err)
	_, err = service.Counts(context.Background(), usecase.Query{})
	require.ErrorIs(t, err, selectorErr)

	cases := map[string]struct {
		repo   dashboardclaim.Repo
		closed dashboardclaim.ClosedClaimReader
		msg    string
	}{
		"berjalan": {faultyRepo{failOutstanding: true}, memory.NewClosedReader(nil),
			"dashboardclaim/usecase: menghitung klaim berjalan: ora-03113"},
		"tutup": {faultyRepo{}, faultyClosed{},
			"dashboardclaim/usecase: menghitung klaim tutup: ora-03113"},
		"adjuster": {faultyRepo{failSurveyKind: dashboardclaim.SurveyorAdjuster}, memory.NewClosedReader(nil),
			"dashboardclaim/usecase: menghitung survei loss adjuster: ora-03113"},
		"internal": {faultyRepo{failSurveyKind: dashboardclaim.SurveyorInternal}, memory.NewClosedReader(nil),
			"dashboardclaim/usecase: menghitung survei internal: ora-03113"},
	}
	for label, c := range cases {
		_, err := newService(t, c.repo, c.closed).Counts(context.Background(), usecase.Query{})
		require.ErrorIs(t, err, errStore, label)
		require.EqualError(t, err, c.msg, label)
	}
}

func TestListPerTile(t *testing.T) {
	service := newService(t,
		memory.NewRepo(memory.SampleOutstanding(), memory.SampleSurveys()),
		memory.NewClosedReader(memory.SampleClosed()))
	ctx := context.Background()

	outstanding, err := service.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileOutstanding})
	require.NoError(t, err)
	require.Equal(t, dashboardclaim.ShapeClaim, outstanding.Shape)
	require.Equal(t, 5, outstanding.Claims.Total)
	require.Equal(t, dashboardclaim.DefaultLimit, outstanding.Filter.Limit)

	closed, err := service.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileCloseClaim})
	require.NoError(t, err)
	require.Equal(t, 2, closed.Claims.Total)

	survey, err := service.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileInternalSurveyor})
	require.NoError(t, err)
	require.Equal(t, dashboardclaim.ShapeSurvey, survey.Shape)
	require.Equal(t, 2, survey.Surveys.Total)
	require.Empty(t, survey.Claims.Rows)
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()

	_, err := newService(t, faultyRepo{}, memory.NewClosedReader(nil)).
		List(ctx, usecase.ListQuery{Tile: "tidak-ada"})
	require.ErrorIs(t, err, dashboardclaim.ErrTileNotFound)

	_, err = newService(t, faultyRepo{}, faultyClosed{}).
		List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileCloseClaim})
	require.EqualError(t, err, "dashboardclaim/usecase: membaca daftar klaim tutup: ora-03113")

	selectorErr := errors.New("portal belum siap")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (dashboardclaim.Repo, error) { return nil, selectorErr },
		ClosedClaim:  memory.NewClosedReader(nil),
	})
	require.NoError(t, err)
	_, err = service.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileOutstanding})
	require.ErrorIs(t, err, selectorErr)

	failing := newService(t, faultyRepo{failList: true}, memory.NewClosedReader(nil))
	_, err = failing.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileLossAdjuster})
	require.EqualError(t, err, "dashboardclaim/usecase: membaca daftar survei: ora-03113")
	_, err = failing.List(ctx, usecase.ListQuery{Tile: dashboardclaim.TileOutstanding})
	require.EqualError(t, err, "dashboardclaim/usecase: membaca daftar klaim berjalan: ora-03113")
}
