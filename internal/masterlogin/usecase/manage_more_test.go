package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/repo/memory"
	"claim-pnc/internal/masterlogin/usecase"
)

var errStore = errors.New("penyimpanan rusak")

func jsonLogger() (*slog.Logger, *bytes.Buffer) {
	logs := &bytes.Buffer{}
	return slog.New(slog.NewJSONHandler(logs, nil)), logs
}

func TestNewServiceRequiresASelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestGetTrimsTheLoginAndRefusesAnEmptyOne(t *testing.T) {
	service, _ := layanan(t, tim())
	ctx := context.Background()

	got, err := service.Get(ctx, portalUji, " RinaAyu ")
	require.NoError(t, err)
	require.Equal(t, "Rina Ayu", got.Name)

	_, err = service.Get(ctx, portalUji, "  ")
	require.ErrorIs(t, err, masterlogin.ErrNotFound)

	_, err = service.Get(ctx, "ASI", "RinaAyu")
	require.EqualError(t, err, "portal tidak dikenal")
}

func TestCreateAndSaveAreLogged(t *testing.T) {
	service, _ := layanan(t, tim())
	logger, logs := jsonLogger()
	ctx := context.Background()

	saved, err := service.Create(ctx, portalUji, isianSah(), usecase.Actor{Login: "RinaAyu"}, logger)
	require.NoError(t, err)
	require.Contains(t, logs.String(), "akun aplikasinya tidak diterbitkan")
	require.NotContains(t, logs.String(), "login leader tidak dapat diturunkan")

	logs.Reset()
	change := isianSah()
	change.Phone = "021-9999"
	updated, err := service.Save(ctx, portalUji, saved.Login, change, usecase.Actor{Login: "RinaAyu"}, logger)
	require.NoError(t, err)
	require.Equal(t, "021-9999", updated.Phone)
	require.Contains(t, logs.String(), "login surveyor diubah")
}

func TestCreateWarnsWhenTheLeaderCannotBeDerived(t *testing.T) {
	service, _ := layanan(t, tim())
	logger, logs := jsonLogger()

	saved, err := service.Create(context.Background(), portalUji, isianSah(),
		usecase.Actor{Login: "TIDAKADA"}, logger)
	require.NoError(t, err)
	require.Empty(t, saved.LeaderLogin)
	require.Contains(t, logs.String(), "login leader tidak dapat diturunkan")
}

func TestStorageFailuresArePassedOn(t *testing.T) {
	service, repo := layanan(t, tim())
	repo.SetError(errStore)
	ctx := context.Background()

	_, err := service.Create(ctx, portalUji, isianSah(), usecase.Actor{Login: "RinaAyu"}, nil)
	require.ErrorIs(t, err, errStore, "galat pencarian leader diteruskan")

	_, err = service.Save(ctx, portalUji, "RinaAyu", isianSah(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, errStore)
}

func TestSaveAndCreateRefusals(t *testing.T) {
	service, _ := layanan(t, tim())
	ctx := context.Background()

	_, err := service.Save(ctx, "ASI", "RinaAyu", isianSah(), usecase.Actor{}, nil)
	require.EqualError(t, err, "portal tidak dikenal")

	_, err = service.Save(ctx, portalUji, " ", isianSah(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterlogin.ErrNotFound)

	_, err = service.Save(ctx, portalUji, "RinaAyu", masterlogin.Input{}, usecase.Actor{}, nil)
	var validation *masterlogin.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.Create(ctx, "ASI", isianSah(), usecase.Actor{}, nil)
	require.EqualError(t, err, "portal tidak dikenal")

	require.NoError(t, service.EnsurePortalReady(portalUji))
	require.ErrorContains(t, service.EnsurePortalReady("ASI"), `portal "ASI" tidak dapat dilayani`)
}

func TestOneViolationReadsLikeAnyOtherValidationError(t *testing.T) {
	err := masterlogin.OneViolation("login", "sudah dipakai")
	var validation *masterlogin.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "masterlogin: isian tidak sah (login: sudah dipakai)", err.Error())
}

func TestTheSampleRepoIsUsable(t *testing.T) {
	rows, err := memory.NewSampleRepo().List(context.Background(), masterlogin.Filter{})
	require.NoError(t, err)
	require.NotEmpty(t, rows)
}
