package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/mastercolsimasonline/repo/memory"
	"claim-pnc/internal/mastercolsimasonline/usecase"
)

func tooLongInput() mastercolsimasonline.Input {
	return mastercolsimasonline.Input{
		Description: strings.Repeat("A", mastercolsimasonline.MaxDescriptionLength+1),
	}
}

// Isian yang melanggar batas panjang ditolak sebelum apa pun tersimpan.
func TestCreateRejectsAnInvalidInputWithEveryViolation(t *testing.T) {
	h := newHarness(t)

	_, err := h.service.Create(context.Background(), portalAlias, tooLongInput())
	var validation *mastercolsimasonline.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, mastercolsimasonline.FieldDescription, validation.Violation[0].Field)

	list, err := h.repo.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

// Penyuntingan dengan isian salah ditolak, dan barisnya tidak berubah.
func TestUpdateRejectsAnInvalidInput(t *testing.T) {
	h := newHarness(t, memory.SampleList()...)

	_, err := h.service.Update(context.Background(), portalAlias, "1001", tooLongInput())
	var validation *mastercolsimasonline.ValidationError
	require.ErrorAs(t, err, &validation)

	row, err := h.repo.Get(context.Background(), "1001")
	require.NoError(t, err)
	require.Equal(t, "KEBAKARAN", row.Description)
}

// Daftar bisnis ditolak bila portalnya tidak dapat dilayani master bisnis.
func TestListBusinessRejectsAnUnservedPortal(t *testing.T) {
	h := newHarness(t)

	_, err := h.service.ListBusiness(context.Background(), "SMI")
	require.EqualError(t, err, "portal tidak dikenal")

	list, err := h.service.ListBusiness(context.Background(), portalAlias)
	require.NoError(t, err)
	require.Len(t, list, len(memory.SampleBusinessList()))
}

// Portal yang dapat dilayani lolos pemeriksaan awal; yang tidak, ditolak dengan aliasnya.
func TestEnsurePortalReady(t *testing.T) {
	h := newHarness(t)

	require.NoError(t, h.service.EnsurePortalReady(portalAlias))
	require.EqualError(t, h.service.EnsurePortalReady("SMI"),
		`mastercolsimasonline/usecase: portal "SMI" tidak dapat dilayani: portal tidak dikenal`)
}

// Bila master bisnis tidak dapat dipilih, nama bisnis tetap tersimpan tanpa ID.
func TestUnselectableBusinessMasterStillSavesNamesWithoutIDs(t *testing.T) {
	repo := memory.NewRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastercolsimasonline.Repo, error) { return repo, nil },
		BusinessSelector: func(string) (mastercolsimasonline.BusinessRepo, error) {
			return nil, errors.New("master bisnis belum siap")
		},
	})
	require.NoError(t, err)

	saved, err := service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description: "BANJIR", BusinessNames: []string{"ANEKA"},
	})
	require.NoError(t, err)
	require.Equal(t, []mastercolsimasonline.Business{{Name: "ANEKA"}}, saved.Businesses)
}
