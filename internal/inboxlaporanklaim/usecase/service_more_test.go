package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/repo/memory"
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/clock"
)

// faultyRepo membungkus repo memori dan menggagalkan satu metode tertentu, supaya jalur
// galat yang tidak dapat dipicu repo memori sendiri tetap teruji.
type faultyRepo struct {
	*memory.Repo
	summarizeErr error
	updateErr    error
	findErr      error
	registered   bool
}

func (f *faultyRepo) Summarize(ctx context.Context, filter inboxlaporanklaim.Filter) (inboxlaporanklaim.Summary, error) {
	if f.summarizeErr != nil {
		return inboxlaporanklaim.Summary{}, f.summarizeErr
	}
	return f.Repo.Summarize(ctx, filter)
}

func (f *faultyRepo) Update(ctx context.Context, report inboxlaporanklaim.ClaimReport) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	return f.Repo.Update(ctx, report)
}

func (f *faultyRepo) FindPolicy(ctx context.Context, number string) (inboxlaporanklaim.Policy, bool, error) {
	if f.findErr != nil {
		return inboxlaporanklaim.Policy{}, false, f.findErr
	}
	return f.Repo.FindPolicy(ctx, number)
}

func (f *faultyRepo) Get(ctx context.Context, id string) (inboxlaporanklaim.ClaimReport, error) {
	report, err := f.Repo.Get(ctx, id)
	if err == nil && f.registered {
		report.ClaimNumber = "PNCN.26.1"
	}
	return report, err
}

var errPortal = errors.New("portal tidak tersedia")

func newFaultyService(t *testing.T, resolver inboxlaporanklaim.BranchResolver) (*usecase.Service, *faultyRepo) {
	t.Helper()
	fixed := clock.FixedAt(time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC))
	repo := &faultyRepo{Repo: memory.NewRepo(memory.SampleOptions(fixed))}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxlaporanklaim.Repo, error) {
			if alias != portalAlias {
				return nil, errPortal
			}
			return repo, nil
		},
		BranchResolver: resolver,
		Clock:          fixed,
	})
	require.NoError(t, err)
	return service, repo
}

func TestServiceRefusesToStartWithoutRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(time.Now())})
	require.ErrorContains(t, err, "RepoSelector")
}

func TestServiceNowFollowsItsClock(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxlaporanklaim.Repo, error) { return nil, nil },
		Clock:        clock.FixedAt(at),
	})
	require.NoError(t, err)
	require.Equal(t, at, service.Now())
}

// Portal yang tidak tersedia ditolak pada setiap operasi, tidak dilayani portal utama.
func TestEveryOperationRefusesUnknownPortal(t *testing.T) {
	service, _ := newFaultyService(t, memory.NewBranchResolver(memory.SampleBranchOfLogin()))
	ctx := context.Background()

	_, err := service.Regions(ctx, "XXX")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "XXX", "RCV-0001")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "XXX", adminJakarta)
	require.ErrorIs(t, err, errPortal)
	_, err = service.Save(ctx, "XXX", "RCV-0001", adminJakarta, inboxlaporanklaim.Detail{})
	require.ErrorIs(t, err, errPortal)
	_, err = service.LookupPolicy(ctx, "XXX", "12600000000001")
	require.ErrorIs(t, err, errPortal)
}

func TestRegionsAndGetReadFromThePortalRepo(t *testing.T) {
	service, _ := newFaultyService(t, nil)
	ctx := context.Background()

	region, err := service.Regions(ctx, portalAlias)
	require.NoError(t, err)
	require.Equal(t, memory.SampleRegions(), region)

	report, err := service.Get(ctx, portalAlias, "RCV-0004")
	require.NoError(t, err)
	require.Equal(t, "POL-FIR-0004", report.PolicyNumber)
}

// Rakitan tanpa penerjemah cabang tidak boleh diam-diam membuka batas data.
func TestMissingBranchResolverIsUnreadableNotUnbounded(t *testing.T) {
	service, _ := newFaultyService(t, nil)
	_, err := service.List(context.Background(), portalAlias, adminJakarta, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	require.ErrorIs(t, err, inboxlaporanklaim.ErrBranchUnreadable)
}

func TestListReportsSummaryFailure(t *testing.T) {
	service, repo := newFaultyService(t, memory.NewBranchResolver(memory.SampleBranchOfLogin()))
	boom := errors.New("pencacah gagal")
	repo.summarizeErr = boom

	_, err := service.List(context.Background(), portalAlias, adminJakarta, usecase.Query{
		Category: inboxlaporanklaim.CategoryAll,
	})
	require.ErrorIs(t, err, boom)
}

func TestSaveRefusesReportAlreadyRegistered(t *testing.T) {
	service, repo := newFaultyService(t, memory.NewBranchResolver(memory.SampleBranchOfLogin()))
	created, err := service.Create(context.Background(), portalAlias, adminJakarta)
	require.NoError(t, err)
	repo.registered = true

	_, err = service.Save(context.Background(), portalAlias, created.ID, adminJakarta, inboxlaporanklaim.Detail{})
	require.ErrorIs(t, err, inboxlaporanklaim.ErrAlreadyRegistered)
}

func TestSaveReportsPolicyAndUpdateFailures(t *testing.T) {
	service, repo := newFaultyService(t, memory.NewBranchResolver(memory.SampleBranchOfLogin()))
	created, err := service.Create(context.Background(), portalAlias, adminJakarta)
	require.NoError(t, err)

	findBoom := errors.New("polis tak terbaca")
	repo.findErr = findBoom
	_, err = service.Save(context.Background(), portalAlias, created.ID, adminJakarta,
		inboxlaporanklaim.Detail{PolicyNumber: "12600000000001"})
	require.ErrorIs(t, err, findBoom)

	repo.findErr = nil
	updateBoom := errors.New("simpan gagal")
	repo.updateErr = updateBoom
	_, err = service.Save(context.Background(), portalAlias, created.ID, adminJakarta, inboxlaporanklaim.Detail{})
	require.ErrorIs(t, err, updateBoom)
}

func TestLookupPolicyBlankNumberSkipsTheRepo(t *testing.T) {
	service, repo := newFaultyService(t, nil)
	repo.findErr = errors.New("tidak boleh dipanggil")

	got, err := service.LookupPolicy(context.Background(), portalAlias, " .. ")
	require.NoError(t, err)
	require.Equal(t, usecase.PolicyLookup{}, got)
}

func TestLookupPolicyReportsRepoFailure(t *testing.T) {
	service, repo := newFaultyService(t, nil)
	boom := errors.New("polis tak terbaca")
	repo.findErr = boom

	_, err := service.LookupPolicy(context.Background(), portalAlias, "12600000000001")
	require.ErrorIs(t, err, boom)
}
