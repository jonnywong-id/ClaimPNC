package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasalai"
)

// be4RecordingRepo merekam penyaring yang diterima repo.
type be4RecordingRepo struct {
	got masterpasalai.Filter
}

func (r *be4RecordingRepo) List(
	_ context.Context,
	filter masterpasalai.Filter,
) (masterpasalai.Page, error) {
	r.got = filter
	return masterpasalai.Page{Total: 7, Number: filter.Page}, nil
}

// TestNewServiceRequiresSelector membuktikan rakitan tanpa pemilih repo ditolak.
func TestNewServiceRequiresSelector(t *testing.T) {
	service, err := NewService(Options{})
	require.Error(t, err)
	require.Nil(t, service)
}

// TestListCleansFilterBeforeRepo membuktikan kata kunci dipangkas dan halaman dinormalkan.
func TestListCleansFilterBeforeRepo(t *testing.T) {
	repo := &be4RecordingRepo{}
	var alias string
	service, err := NewService(Options{RepoSelector: func(a string) (masterpasalai.Repo, error) {
		alias = a
		return repo, nil
	}})
	require.NoError(t, err)

	page, err := service.List(context.Background(), "ASM",
		masterpasalai.Filter{Keyword: "  banjir ", Page: -3})
	require.NoError(t, err)
	require.Equal(t, "ASM", alias)
	require.Equal(t, masterpasalai.Filter{Keyword: "banjir", Page: 1}, repo.got)
	require.Equal(t, 7, page.Total)
}

// TestListPropagatesSelectorError membuktikan galat portal diteruskan apa adanya.
func TestListPropagatesSelectorError(t *testing.T) {
	sentinel := errors.New("portal belum siap")
	service, err := NewService(Options{RepoSelector: func(string) (masterpasalai.Repo, error) {
		return nil, sentinel
	}})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "XXX", masterpasalai.Filter{})
	require.ErrorIs(t, err, sentinel)
	require.ErrorIs(t, service.EnsurePortalReady("XXX"), sentinel)
}

// TestEnsurePortalReadySucceeds membuktikan portal yang dapat dipilih dinyatakan siap.
func TestEnsurePortalReadySucceeds(t *testing.T) {
	service, err := NewService(Options{RepoSelector: func(string) (masterpasalai.Repo, error) {
		return &be4RecordingRepo{}, nil
	}})
	require.NoError(t, err)
	require.NoError(t, service.EnsurePortalReady("ASM"))
}
