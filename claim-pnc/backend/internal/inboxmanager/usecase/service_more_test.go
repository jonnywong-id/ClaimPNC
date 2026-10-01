package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/repo/memory"
	"claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/clock"
)

var fixedNow = time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)

// faultyRepo meneruskan ke penyimpanan memori, kecuali operasi yang diberi galat.
type faultyRepo struct {
	*memory.Store
	countersErr  error
	dashboardErr error
	queueErr     error
	decideErr    error
	lineErr      error
}

func (f faultyRepo) Counters(ctx context.Context, c inboxmanager.Caller) ([]inboxmanager.Counter, error) {
	if f.countersErr != nil {
		return nil, f.countersErr
	}
	return f.Store.Counters(ctx, c)
}

func (f faultyRepo) Dashboard(ctx context.Context, q inboxmanager.Query) (inboxmanager.DashboardView, error) {
	if f.dashboardErr != nil {
		return inboxmanager.DashboardView{}, f.dashboardErr
	}
	return f.Store.Dashboard(ctx, q)
}

func (f faultyRepo) Queue(ctx context.Context, q inboxmanager.Query) ([]inboxmanager.QueueRow, error) {
	if f.queueErr != nil {
		return nil, f.queueErr
	}
	return f.Store.Queue(ctx, q)
}

func (f faultyRepo) Decide(ctx context.Context, d inboxmanager.Decision) (int, error) {
	if f.decideErr != nil {
		return 0, f.decideErr
	}
	return f.Store.Decide(ctx, d)
}

func (f faultyRepo) LineBusinessFor(ctx context.Context, login string) (string, error) {
	if f.lineErr != nil {
		return "", f.lineErr
	}
	return f.Store.LineBusinessFor(ctx, login)
}

type harness struct {
	service *usecase.Service
	logs    *bytes.Buffer
}

// newHarness merakit layanan dengan logger; portal "RUSAK" memilih repo yang gagal dipilih.
func newHarness(t *testing.T, repo faultyRepo) harness {
	t.Helper()
	logs := &bytes.Buffer{}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxmanager.Repo, error) {
			if alias == "RUSAK" {
				return nil, errors.New("koneksi portal mati")
			}
			return repo, nil
		},
		LineBusinessSelector: func(alias string) (inboxmanager.LineBusinessRepo, error) {
			if alias == "TANPALINI" {
				return nil, errors.New("koneksi lini mati")
			}
			return repo, nil
		},
		Clock:  clock.FixedAt(fixedNow),
		Logger: slog.New(slog.NewTextHandler(logs, nil)),
	})
	require.NoError(t, err)
	return harness{service: service, logs: logs}
}

var manager = inboxmanager.Caller{Login: "MGR"}

func TestNewServiceRequiresEveryDependency(t *testing.T) {
	selector := func(string) (inboxmanager.Repo, error) { return nil, nil }
	lines := func(string) (inboxmanager.LineBusinessRepo, error) { return nil, nil }

	_, err := usecase.NewService(usecase.Options{})
	require.EqualError(t, err, "inboxmanager/usecase: RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{RepoSelector: selector})
	require.EqualError(t, err, "inboxmanager/usecase: LineBusinessSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{RepoSelector: selector, LineBusinessSelector: lines})
	require.EqualError(t, err, "inboxmanager/usecase: Clock wajib diisi")
}

func TestMetadataForCaller(t *testing.T) {
	h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})

	meta, err := h.service.Metadata(context.Background(), "ASM", manager)
	require.NoError(t, err)
	require.Equal(t, inboxmanager.LineNonMBU, meta.LineBusiness)
	require.Equal(t, inboxmanager.DefaultTab, meta.DefaultTab)
	require.Len(t, meta.Tabs, 13, "NONMBU membuka tab Nomor Rangka juga")
	require.Equal(t, inboxmanager.PlannedDifferences, meta.PlannedDifferences)

	_, err = h.service.Metadata(context.Background(), "ASM", inboxmanager.Caller{Login: "  "})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	_, err = h.service.Metadata(context.Background(), "TANPALINI", manager)
	require.EqualError(t, err, "koneksi lini mati")
}

func TestLineBusinessReadFailureIsWrapped(t *testing.T) {
	boom := errors.New("ORA-00942")
	h := newHarness(t, faultyRepo{Store: memory.NewSampleStore(), lineErr: boom})

	_, err := h.service.Counters(context.Background(), "ASM", manager)
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca lini bisnis pemanggil MGR")
}

func TestCountersErrors(t *testing.T) {
	h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})

	_, err := h.service.Counters(context.Background(), "ASM", inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	_, err = h.service.Counters(context.Background(), "RUSAK", manager)
	require.EqualError(t, err, "koneksi portal mati")

	_, err = h.service.Counters(context.Background(), "TANPALINI", manager)
	require.EqualError(t, err, "koneksi lini mati")

	boom := errors.New("putus")
	failing := newHarness(t, faultyRepo{Store: memory.NewSampleStore(), countersErr: boom})
	_, err = failing.service.Counters(context.Background(), "ASM", manager)
	require.ErrorIs(t, err, boom)
}

func TestListErrorsAndLogging(t *testing.T) {
	boom := errors.New("putus")

	h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})
	_, err := h.service.List(context.Background(), "RUSAK", inboxmanager.QueryInput{}, manager)
	require.EqualError(t, err, "koneksi portal mati")

	_, err = h.service.List(context.Background(), "TANPALINI", inboxmanager.QueryInput{}, manager)
	require.EqualError(t, err, "koneksi lini mati")

	_, err = h.service.List(context.Background(), "ASM", inboxmanager.QueryInput{Tab: "99"}, manager)
	var validation *inboxmanager.ValidationError
	require.ErrorAs(t, err, &validation)

	dash := newHarness(t, faultyRepo{Store: memory.NewSampleStore(), dashboardErr: boom})
	_, err = dash.service.List(context.Background(), "ASM", inboxmanager.QueryInput{}, manager)
	require.ErrorIs(t, err, boom)

	queue := newHarness(t, faultyRepo{Store: memory.NewSampleStore(), queueErr: boom})
	_, err = queue.service.List(context.Background(), "ASM",
		inboxmanager.QueryInput{Tab: inboxmanager.TabMasterBengkel}, manager)
	require.ErrorIs(t, err, boom)

	// Setiap pembukaan tab tercatat, lengkap dengan portal dan tab.
	require.Contains(t, queue.logs.String(), "inbox manager dibuka")
	require.Contains(t, queue.logs.String(), "portal=ASM")
	require.Contains(t, queue.logs.String(), "tab=5")
}

// Tab ringkasan tidak mengisi dashboard maupun antrean.
func TestOverviewTabHasNoBody(t *testing.T) {
	h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})

	view, err := h.service.List(context.Background(), "ASM",
		inboxmanager.QueryInput{Tab: inboxmanager.TabApprovalMaster}, manager)
	require.NoError(t, err)
	require.Equal(t, inboxmanager.TabApprovalMaster, view.Tab.Code)
	require.Empty(t, view.Dashboard.Panels)
	require.Empty(t, view.Queue.Rows)
}

func TestLargeQueueIsWarned(t *testing.T) {
	store := memory.NewStore()
	rows := make([]inboxmanager.QueueRow, 0, inboxmanager.LargeResultWarning+1)
	for i := 0; i <= inboxmanager.LargeResultWarning; i++ {
		rows = append(rows, inboxmanager.QueueRow{Key: fmt.Sprintf("K%05d", i)})
	}
	store.SetQueue(inboxmanager.TabMasterPanel, rows)
	h := newHarness(t, faultyRepo{Store: store})

	view, err := h.service.List(context.Background(), "ASM",
		inboxmanager.QueryInput{Tab: inboxmanager.TabMasterPanel}, manager)
	require.NoError(t, err)
	require.Equal(t, inboxmanager.LargeResultWarning+1, view.Queue.Total)
	require.Len(t, view.Queue.Rows, inboxmanager.DefaultPageSize)
	require.Contains(t, h.logs.String(), "antrean inbox manager sangat besar")
	require.Contains(t, h.logs.String(), "baris=5001")
}

func TestDecideLogsSuccessStaleAndFailure(t *testing.T) {
	t.Run("berhasil penuh", func(t *testing.T) {
		h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})
		result, err := h.service.Decide(context.Background(), "ASM", usecase.DecideInput{
			Tab: inboxmanager.TabMasterBengkel, Verdict: inboxmanager.VerdictApprove,
			Keys: []string{"BGK-001"},
		}, manager)
		require.NoError(t, err)
		require.Equal(t, inboxmanager.DecisionResult{Requested: 1, Changed: 1}, result)
		require.Contains(t, h.logs.String(), "keputusan inbox manager ditulis")
		require.Contains(t, h.logs.String(), "kunci=[BGK-001]")
	})

	t.Run("sebagian sudah diputuskan", func(t *testing.T) {
		h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})
		result, err := h.service.Decide(context.Background(), "ASM", usecase.DecideInput{
			Tab: inboxmanager.TabMasterBengkel, Verdict: inboxmanager.VerdictApprove,
			Keys: []string{"BGK-001", "BGK-LAMA"},
		}, manager)
		require.NoError(t, err)
		require.Equal(t, 1, result.Stale())
		require.Contains(t, h.logs.String(), "sudah diputuskan lebih dulu")
		require.Contains(t, h.logs.String(), "tidak_berubah=1")
	})

	t.Run("penyimpanan gagal", func(t *testing.T) {
		boom := errors.New("ORA-00054 terkunci")
		h := newHarness(t, faultyRepo{Store: memory.NewSampleStore(), decideErr: boom})
		result, err := h.service.Decide(context.Background(), "ASM", usecase.DecideInput{
			Tab: inboxmanager.TabMasterBengkel, Verdict: inboxmanager.VerdictReject,
			Keys: []string{"BGK-001"}, Reason: "rahasia pengguna",
		}, manager)
		require.ErrorIs(t, err, boom)
		require.Equal(t, inboxmanager.DecisionResult{}, result)
		require.Contains(t, h.logs.String(), "keputusan inbox manager GAGAL")
		require.Contains(t, h.logs.String(), "ORA-00054")
		require.NotContains(t, h.logs.String(), "rahasia pengguna", "alasan tidak disalin ke log")
	})

	t.Run("galat sebelum menulis", func(t *testing.T) {
		h := newHarness(t, faultyRepo{Store: memory.NewSampleStore()})
		_, err := h.service.Decide(context.Background(), "TANPALINI", usecase.DecideInput{}, manager)
		require.EqualError(t, err, "koneksi lini mati")

		_, err = h.service.Decide(context.Background(), "ASM", usecase.DecideInput{
			Tab: inboxmanager.TabOutstanding, Verdict: inboxmanager.VerdictApprove, Keys: []string{"X"},
		}, manager)
		require.ErrorIs(t, err, inboxmanager.ErrQueueNotDecidable)

		_, err = h.service.Decide(context.Background(), "RUSAK", usecase.DecideInput{
			Tab: inboxmanager.TabMasterBengkel, Verdict: inboxmanager.VerdictApprove, Keys: []string{"X"},
		}, manager)
		require.EqualError(t, err, "koneksi portal mati")
		require.Empty(t, h.logs.String(), "tidak ada yang ditulis, tidak ada yang dicatat")
	})

	t.Run("tanpa logger tetap berjalan", func(t *testing.T) {
		service := newService(t, memory.NewSampleStore(), fixedNow)
		result, err := service.Decide(context.Background(), portalUtama, usecase.DecideInput{
			Tab: inboxmanager.TabMasterBengkel, Verdict: inboxmanager.VerdictApprove,
			Keys: []string{"BGK-002"},
		}, manager)
		require.NoError(t, err)
		require.Equal(t, 1, result.Changed)
	})
}
