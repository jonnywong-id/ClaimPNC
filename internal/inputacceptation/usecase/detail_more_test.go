package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/inputacceptation/repo/memory"
	"claim-pnc/internal/inputacceptation/usecase"
)

var errStore = errors.New("penyimpanan rusak")

// scriptedRepo menjawab Find dan Save dengan galat yang ditentukan.
type scriptedRepo struct {
	*memory.Store
	findErr error
	saveErr error
	saved   []inputacceptation.SaveCommand
}

func (s *scriptedRepo) Find(ctx context.Context, q inputacceptation.Query) (inputacceptation.Detail, error) {
	if s.findErr != nil {
		return inputacceptation.Detail{}, s.findErr
	}
	return s.Store.Find(ctx, q)
}

func (s *scriptedRepo) Save(_ context.Context, cmd inputacceptation.SaveCommand) error {
	s.saved = append(s.saved, cmd)
	return s.saveErr
}

func serviceOver(t *testing.T, repo inputacceptation.Repo, logs *bytes.Buffer) *usecase.Service {
	t.Helper()
	var logger *slog.Logger
	if logs != nil {
		logger = slog.New(slog.NewJSONHandler(logs, nil))
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inputacceptation.Repo, error) {
			if alias != portalUtama {
				return nil, errPortalAsing
			}
			return repo, nil
		},
		Logger: logger,
	})
	require.NoError(t, err)
	return service
}

func TestFindIsLoggedAndWrapsStorageFailures(t *testing.T) {
	logs := &bytes.Buffer{}
	repo := &scriptedRepo{Store: memory.NewSampleStore()}
	service := serviceOver(t, repo, logs)

	detail, err := service.Find(context.Background(), portalUtama, caller, "CLMNP-1001")
	require.NoError(t, err)
	require.Equal(t, "CLMNP-1001", detail.ClaimID)
	require.Contains(t, logs.String(), "akseptasi klaim treaty non-prop dibuka")

	repo.findErr = errStore
	_, err = service.Find(context.Background(), portalUtama, caller, "CLMNP-1001")
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "membuka akseptasi CLMNP-1001")
}

func TestSubmitStoresAValidCommandAndLogsIt(t *testing.T) {
	logs := &bytes.Buffer{}
	repo := &scriptedRepo{Store: memory.NewSampleStore()}
	service := serviceOver(t, repo, logs)

	err := service.Submit(context.Background(), portalUtama, caller, "CLMNP-1001",
		map[string]string{"dla_no_ceding": "DLA/2026/0002"}, nil)
	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	require.Equal(t, "DLA/2026/0002", repo.saved[0].Values["dla_no_ceding"])
	require.NotEmpty(t, repo.saved[0].Reference, "rujukan diambil dari rincian yang ada")
	require.Contains(t, logs.String(), "akseptasi klaim treaty non-prop disimpan")

	// Tanpa pencatat, penyimpanan tetap berjalan.
	quiet := serviceOver(t, repo, nil)
	require.NoError(t, quiet.Submit(context.Background(), portalUtama, caller, "CLMNP-1001", nil, nil))
}

func TestSubmitWrapsStorageFailures(t *testing.T) {
	repo := &scriptedRepo{Store: memory.NewSampleStore(), findErr: errStore}
	service := serviceOver(t, repo, nil)

	err := service.Submit(context.Background(), portalUtama, caller, "CLMNP-1001", nil, nil)
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "sebelum menyimpan")

	repo.findErr = nil
	repo.saveErr = errStore
	err = service.Submit(context.Background(), portalUtama, caller, "CLMNP-1001", nil, nil)
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "menyimpan akseptasi CLMNP-1001")

	err = service.Submit(context.Background(), "ASI", caller, "CLMNP-1001", nil, nil)
	require.ErrorIs(t, err, errPortalAsing)

	err = service.Submit(context.Background(), portalUtama, inputacceptation.Caller{}, "CLMNP-1001", nil, nil)
	require.ErrorIs(t, err, inputacceptation.ErrCallerUnknown)
}
