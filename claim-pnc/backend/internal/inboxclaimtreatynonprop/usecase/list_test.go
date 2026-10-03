package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatynonprop/repo/memory"
	"claim-pnc/internal/inboxclaimtreatynonprop/usecase"
)

// failingRepo selalu gagal, untuk menguji pembungkusan galat penyimpanan.
type failingRepo struct{ err error }

func (f failingRepo) List(
	context.Context, inboxclaimtreatynonprop.Query, inboxclaimtreatynonprop.Pagination,
) (inboxclaimtreatynonprop.Page, error) {
	return inboxclaimtreatynonprop.Page{}, f.err
}

func sampleSelector(string) (inboxclaimtreatynonprop.Repo, error) {
	return memory.NewSampleStore(), nil
}

func TestNewServiceRequiresSelector(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{})
	require.Nil(t, service)
	require.EqualError(t, err, "inboxclaimtreatynonprop/usecase: RepoSelector wajib diisi")
}

func TestMetadataListsTabsAndDifferences(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{RepoSelector: sampleSelector})
	require.NoError(t, err)

	meta := service.Metadata()
	require.Len(t, meta.Tabs, 3)
	require.Equal(t, inboxclaimtreatynonprop.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxclaimtreatynonprop.PlannedDifferences, meta.PlannedDifferences)
}

// See All dicatat ke log beserta pemanggil dan portalnya.
func TestListSeeAllIsLogged(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))

	service, err := usecase.NewService(usecase.Options{RepoSelector: sampleSelector, Logger: logger})
	require.NoError(t, err)

	listed, err := service.List(context.Background(), "ASM",
		inboxclaimtreatynonprop.Caller{Login: "ADMINNONPROP1"},
		inboxclaimtreatynonprop.QueryInput{SeeAll: true},
		inboxclaimtreatynonprop.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 4, listed.Page.Total)
	require.True(t, listed.Query.SeeAll)
	require.Contains(t, buffer.String(), "antrean treaty non-prop dibuka tanpa penyaring kepemilikan")
	require.Contains(t, buffer.String(), "pemanggil=ADMINNONPROP1")
	require.Contains(t, buffer.String(), "portal=ASM")
}

// Tanpa See All, tidak ada yang dicatat.
func TestListOwnQueueNotLogged(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buffer, nil))

	service, err := usecase.NewService(usecase.Options{RepoSelector: sampleSelector, Logger: logger})
	require.NoError(t, err)

	listed, err := service.List(context.Background(), "ASM",
		inboxclaimtreatynonprop.Caller{Login: "ADMINNONPROP1"},
		inboxclaimtreatynonprop.QueryInput{}, inboxclaimtreatynonprop.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 3, listed.Page.Total)
	require.Empty(t, buffer.String())
}

// Galat validasi dikembalikan sebelum pemilih portal dipanggil.
func TestListInvalidQueryShortCircuits(t *testing.T) {
	called := false
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxclaimtreatynonprop.Repo, error) {
			called = true
			return memory.NewSampleStore(), nil
		},
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "ASM", inboxclaimtreatynonprop.Caller{},
		inboxclaimtreatynonprop.QueryInput{}, inboxclaimtreatynonprop.Pagination{})
	require.ErrorIs(t, err, inboxclaimtreatynonprop.ErrCallerUnknown)
	require.False(t, called)
}

// Galat pemilih portal diteruskan apa adanya.
func TestListSelectorError(t *testing.T) {
	boom := errors.New("portal belum siap")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxclaimtreatynonprop.Repo, error) { return nil, boom },
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "ASM", inboxclaimtreatynonprop.Caller{Login: "A"},
		inboxclaimtreatynonprop.QueryInput{}, inboxclaimtreatynonprop.Pagination{})
	require.ErrorIs(t, err, boom)
}

// Galat penyimpanan dibungkus dengan kode tab.
func TestListRepoErrorWrapped(t *testing.T) {
	boom := errors.New("ora-01017")
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxclaimtreatynonprop.Repo, error) {
			return failingRepo{err: boom}, nil
		},
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "ASM", inboxclaimtreatynonprop.Caller{Login: "A"},
		inboxclaimtreatynonprop.QueryInput{Tab: inboxclaimtreatynonprop.TabTechnical},
		inboxclaimtreatynonprop.Pagination{})
	require.ErrorIs(t, err, boom)
	require.EqualError(t, err, "mengambil isi tab 2: ora-01017")
}
