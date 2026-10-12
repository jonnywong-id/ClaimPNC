package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/repo/memory"
	"claim-pnc/internal/masterpanel/usecase"
)

// storeTerganggu membungkus penyimpanan memori dan menggagalkan operasi tertentu.
type storeTerganggu struct {
	*memory.Repo
	emptyID   bool
	updateErr error
	statusErr error
	findErr   error
}

func (s *storeTerganggu) NextID(ctx context.Context) (string, error) {
	if s.emptyID {
		return "  ", nil
	}
	return s.Repo.NextID(ctx)
}

func (s *storeTerganggu) Update(ctx context.Context, p masterpanel.Panel) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.Repo.Update(ctx, p)
}

func (s *storeTerganggu) SetStatus(
	ctx context.Context, id []string, status masterpanel.ApprovalStatus, reason string,
) (int, error) {
	if s.statusErr != nil {
		return 0, s.statusErr
	}
	return s.Repo.SetStatus(ctx, id, status, reason)
}

func (s *storeTerganggu) FindByName(ctx context.Context, name string) (masterpanel.Panel, error) {
	if s.findErr != nil {
		return masterpanel.Panel{}, s.findErr
	}
	return s.Repo.FindByName(ctx, name)
}

func serviceOver(t *testing.T, store masterpanel.Store) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpanel.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

func bufferLogger() (*slog.Logger, *bytes.Buffer) {
	log := &bytes.Buffer{}
	return slog.New(slog.NewTextHandler(log, nil)), log
}

// Setiap operasi menolak portal yang tidak dikenal.
func TestEveryOperationRejectsAnUnknownPortal(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.Get(ctx, "SMI", "01000001")
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.Create(ctx, "SMI", validInput(), usecase.Actor{}, nil)
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.Save(ctx, "SMI", "01000001", validInput(), usecase.Actor{}, nil)
	require.ErrorContains(t, err, "portal tidak dikenal")
	_, err = service.Decide(ctx, "SMI", []string{"01000001"}, masterpanel.StatusApproved, "",
		usecase.Actor{}, nil)
	require.ErrorContains(t, err, "portal tidak dikenal")
}

// ID kosong dari penerbit adalah galat, bukan baris tanpa kunci.
func TestCreateRejectsAnEmptyIssuedID(t *testing.T) {
	service := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), emptyID: true})

	_, err := service.Create(context.Background(), portalAlias, validInput(), usecase.Actor{}, nil)
	require.EqualError(t, err, "masterpanel/usecase: ID panel yang diterbitkan kosong")
}

// Penambahan dicatat sebagai pemberitahuan yang tidak dikirim.
func TestCreateLogsTheOmittedNotification(t *testing.T) {
	service, _ := newService(t)
	logger, log := bufferLogger()

	created, err := service.Create(context.Background(), portalAlias, validInput(),
		usecase.Actor{Login: "penguji"}, logger)
	require.NoError(t, err)
	require.Equal(t, masterpanel.StatusPending, created.Status)
	require.Contains(t, log.String(), "pemberitahuan master panel tidak dikirim")
	require.Contains(t, log.String(), "oleh=penguji")
}

// Penyimpanan dengan isian salah, baris hilang, atau galat penulisan dihentikan.
func TestSaveStopsOnInvalidInputMissingRowOrWriteFailure(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)

	bad := validInput()
	bad.Name = strings.Repeat("x", 500)
	_, err := service.Save(ctx, portalAlias, "01000001", bad, usecase.Actor{}, nil)
	var validation *masterpanel.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.Save(ctx, portalAlias, "99999999", validInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterpanel.ErrNotFound)

	failure := errors.New("oracle mati")
	broken := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), updateErr: failure})
	_, err = broken.Save(ctx, portalAlias, "01000001", validInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, failure)

	unchecked := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), findErr: failure})
	_, err = unchecked.Save(ctx, portalAlias, "01000001", validInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, failure)
	require.ErrorContains(t, err, "memeriksa nama")
}

// Galat penyimpanan keputusan diteruskan; keputusan yang tidak menyentuh seluruh baris
// diberi peringatan, dan yang tersimpan dicatat.
func TestDecideReportsFailuresAndPartialChanges(t *testing.T) {
	ctx := context.Background()

	failure := errors.New("oracle mati")
	broken := serviceOver(t, &storeTerganggu{Repo: memory.NewSampleRepo(), statusErr: failure})
	_, err := broken.Decide(ctx, portalAlias, []string{"01000002"}, masterpanel.StatusApproved,
		"", usecase.Actor{}, nil)
	require.ErrorIs(t, err, failure)

	service, _ := newService(t)
	logger, log := bufferLogger()
	changed, err := service.Decide(ctx, portalAlias, []string{"01000002", "99999999"},
		masterpanel.StatusApproved, "", usecase.Actor{Login: "penyetuju"}, logger)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	require.Contains(t, log.String(), "tidak menyentuh seluruh baris yang dipilih")
	require.Contains(t, log.String(), "keputusan master panel tersimpan tanpa pemberitahuan")
}
