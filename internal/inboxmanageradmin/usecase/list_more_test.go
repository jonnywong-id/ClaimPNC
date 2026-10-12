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

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/repo/memory"
	"claim-pnc/internal/inboxmanageradmin/usecase"
)

var errStore = errors.New("penyimpanan mati")

// failingLines selalu gagal membaca lini bisnis.
type failingLines struct{}

func (failingLines) LineBusinessFor(context.Context, string) (string, error) { return "", errStore }

// failingRepo selalu gagal membaca antrean.
type failingRepo struct{}

func (failingRepo) List(context.Context, inboxmanageradmin.Query) ([]inboxmanageradmin.WorkItem, error) {
	return nil, errStore
}

type serviceOptions struct {
	repo    inboxmanageradmin.Repo
	repoErr error
	lines   inboxmanageradmin.LineBusinessRepo
	logger  *slog.Logger
}

func buildService(t *testing.T, o serviceOptions) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxmanageradmin.Repo, error) {
			if o.repoErr != nil {
				return nil, o.repoErr
			}
			return o.repo, nil
		},
		LineBusinessSelector: func(string) (inboxmanageradmin.LineBusinessRepo, error) {
			return o.lines, nil
		},
		Clock:  tetapWaktu{at: sekarang},
		Logger: o.logger,
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRequiresLineBusinessSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxmanageradmin.Repo, error) { return nil, nil },
		Clock:        tetapWaktu{},
	})
	require.ErrorContains(t, err, "LineBusinessSelector wajib diisi")
}

func TestLineBusinessReadFailureStopsMetadataAndList(t *testing.T) {
	service := buildService(t, serviceOptions{repo: memory.NewSampleStore(), lines: failingLines{}})

	_, err := service.Metadata(context.Background(), portalUtama, petugas())
	require.ErrorIs(t, err, errStore)
	require.Contains(t, err.Error(), "membaca lini bisnis pemanggil "+loginPetugas)

	_, err = service.List(context.Background(), portalUtama, petugas(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{})
	require.ErrorIs(t, err, errStore)
}

func TestListWithoutLoginIsCallerUnknown(t *testing.T) {
	store := memory.NewSampleStore()
	service := buildService(t, serviceOptions{repo: store, lines: store})

	_, err := service.List(context.Background(), portalUtama, inboxmanageradmin.Caller{Login: "  "},
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{})
	require.ErrorIs(t, err, inboxmanageradmin.ErrCallerUnknown)
}

func TestListRepoSelectorAndRepoFailures(t *testing.T) {
	store := memory.NewSampleStore()

	selectorFails := buildService(t, serviceOptions{repoErr: errStore, lines: store})
	_, err := selectorFails.List(context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{})
	require.ErrorIs(t, err, errStore)

	repoFails := buildService(t, serviceOptions{repo: failingRepo{}, lines: store})
	_, err = repoFails.List(context.Background(), portalUtama, pengembang(),
		inboxmanageradmin.QueryInput{Tab: inboxmanageradmin.TabPA}, inboxmanageradmin.Pagination{})
	require.ErrorIs(t, err, errStore)
	require.Contains(t, err.Error(), "mengambil isi tab")
}

func TestListLogsEveryOpeningAndWarnsOnLargeResults(t *testing.T) {
	rows := make([]memory.Row, 0, inboxmanageradmin.LargeResultWarning+1)
	for i := 0; i <= inboxmanageradmin.LargeResultWarning; i++ {
		at := sekarang.Add(-time.Duration(i) * time.Minute)
		rows = append(rows, memory.Row{
			WorkClass: memory.WorkClassPNC, OrgUnit: inboxmanageradmin.OrgUnitPA, WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{CaseID: fmt.Sprintf("PNC-%05d", i), RegisteredAt: &at},
		})
	}
	store := memory.NewStore(rows...)
	store.SetLineBusiness(loginPetugas, inboxmanageradmin.LinePA)

	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
	service := buildService(t, serviceOptions{repo: store, lines: store, logger: logger})

	listed, err := service.List(context.Background(), portalUtama, petugas(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Equal(t, inboxmanageradmin.LargeResultWarning+1, listed.Page.Total)
	require.Len(t, listed.Page.Items, 10)
	require.Equal(t, "PNC-00010", listed.Page.Items[0].CaseID)

	logged := buffer.String()
	require.Contains(t, logged, "antrean Inbox Manager Admin dibuka")
	require.Contains(t, logged, "pengguna="+loginPetugas)
	require.Contains(t, logged, "menarik sangat banyak baris")
}

func TestListLogsOpeningWithoutWarningOnSmallResults(t *testing.T) {
	store := memory.NewSampleStore()
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))
	service := buildService(t, serviceOptions{repo: store, lines: store, logger: logger})

	_, err := service.List(context.Background(), portalUtama, petugas(),
		inboxmanageradmin.QueryInput{}, inboxmanageradmin.Pagination{})
	require.NoError(t, err)
	require.Contains(t, buffer.String(), "antrean Inbox Manager Admin dibuka")
	require.NotContains(t, buffer.String(), "menarik sangat banyak baris")
}
