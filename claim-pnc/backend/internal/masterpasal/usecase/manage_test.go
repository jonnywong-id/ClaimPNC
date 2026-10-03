package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasal/repo/memory"
	"claim-pnc/internal/masterpasal/usecase"
)

var errNotReady = errors.New("portal tidak siap")

func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpasal.Store, error) {
			if alias != "ASM" {
				return nil, errNotReady
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceRequiresASelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestEveryOperationReportsAPortalThatIsNotReady(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "ASI")
	require.ErrorIs(t, err, errNotReady)
	_, err = service.Get(ctx, "ASI", "1")
	require.ErrorIs(t, err, errNotReady)
	_, err = service.Create(ctx, "ASI", masterpasal.Input{Number: "x"})
	require.ErrorIs(t, err, errNotReady)
	_, err = service.Update(ctx, "ASI", "1", masterpasal.Input{Number: "x"})
	require.ErrorIs(t, err, errNotReady)
	require.ErrorIs(t, service.Delete(ctx, "ASI", "1"), errNotReady)
	_, err = service.SearchBusiness(ctx, "ASI", "travel")
	require.ErrorIs(t, err, errNotReady)
}

func TestListGetAndDelete(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	rows, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, rows, len(memory.SampleList()))

	clause, err := service.Get(ctx, "ASM", " 2 ")
	require.NoError(t, err)
	require.Equal(t, "PSL-002", clause.Number)

	require.NoError(t, service.Delete(ctx, "ASM", " 2 "))
	_, err = service.Get(ctx, "ASM", "2")
	require.ErrorIs(t, err, masterpasal.ErrNotFound)
}

func TestCreateAndUpdateCleanAndValidate(t *testing.T) {
	service, repo := newService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, "ASM", masterpasal.Input{Number: "  "})
	var validation *masterpasal.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "no_pasal", validation.Violation[0].Field)

	created, err := service.Create(ctx, "ASM", masterpasal.Input{
		Number: " PSL-9 ", Text: " teks ",
		Business: []masterpasal.Business{{ID: " 2001 "}, {ID: " ", Name: " "}},
	})
	require.NoError(t, err)
	require.Equal(t, "PSL-9", created.Number)
	require.Equal(t, "teks", created.Text)
	require.Equal(t, []masterpasal.Business{{ID: "2001"}}, created.Business)

	_, err = service.Update(ctx, "ASM", created.ID, masterpasal.Input{})
	require.ErrorAs(t, err, &validation)

	updated, err := service.Update(ctx, "ASM", " "+created.ID+" ", masterpasal.Input{Number: "PSL-9B"})
	require.NoError(t, err)
	require.Equal(t, "PSL-9B", updated.Number)

	stored, _ := repo.Get(ctx, created.ID)
	require.Equal(t, "PSL-9B", stored.Number)
}

func TestSearchBusinessNeedsAMinimumKeyword(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	rows, err := service.SearchBusiness(ctx, "ASM", " t ")
	require.NoError(t, err)
	require.Nil(t, rows, "kata kunci terlalu pendek tidak dicari")

	rows, err = service.SearchBusiness(ctx, "ASM", " travel ")
	require.NoError(t, err)
	require.Equal(t, []masterpasal.Business{{ID: "2002", Name: "Travel"}}, rows)
}
