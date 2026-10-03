package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/repo/memory"
)

var errPortal = errors.New("portal tidak siap")

// newTestService merakit layanan dengan satu portal "ASM"; portal lain ditolak.
func newTestService(t *testing.T) (*Service, *memory.Repo) {
	t.Helper()
	repo := memory.NewRepo(memory.SampleList()...)
	documents := memory.NewDocumentRepo(memory.SampleDocumentList()...)
	plans := memory.NewPlanRepo(memory.SamplePlanList(), memory.SampleCoverageList())

	service, err := NewService(Options{
		RepoSelector: func(alias string) (daftardetaildokumentravel.Repo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return repo, nil
		},
		DocumentSelector: func(alias string) (daftardetaildokumentravel.DocumentRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return documents, nil
		},
		PlanSelector: func(alias string) (daftardetaildokumentravel.PlanRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return plans, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceRejectsMissingSelectors(t *testing.T) {
	repoSelector := func(string) (daftardetaildokumentravel.Repo, error) { return nil, nil }
	documentSelector := func(string) (daftardetaildokumentravel.DocumentRepo, error) { return nil, nil }

	_, err := NewService(Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")

	_, err = NewService(Options{RepoSelector: repoSelector})
	require.ErrorContains(t, err, "DocumentSelector wajib diisi")

	_, err = NewService(Options{RepoSelector: repoSelector, DocumentSelector: documentSelector})
	require.ErrorContains(t, err, "PlanSelector wajib diisi")
}

func TestServiceReadsFromSelectedPortal(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	rows, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, rows, 4)

	row, err := service.Get(ctx, "ASM", "00003")
	require.NoError(t, err)
	require.Len(t, row.Coverages, 2)

	documents, err := service.Documents(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, documents, 6)

	plans, err := service.Plans(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, plans, 3)

	coverages, err := service.Coverages(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, coverages, 7)

	require.NoError(t, service.EnsurePortalReady("ASM"))
}

func TestServiceCleansInputBeforeSaving(t *testing.T) {
	service, repo := newTestService(t)
	ctx := context.Background()

	created, err := service.Create(ctx, "ASM", daftardetaildokumentravel.Input{
		DocumentID:   "  100006 ",
		DocumentName: " Surat ",
		MinUpload:    -4,
		Coverages: []daftardetaildokumentravel.CoverageInput{
			{PlanID: " TP01 ", PlanName: "Silver"},
			{PlanID: "  ", PlanName: "", CoverageID: "", CoverageName: " "},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "100006", created.DocumentID)
	require.Equal(t, "Surat", created.DocumentName)
	require.Equal(t, 0, created.MinUpload)
	// Baris coverage yang seluruhnya kosong dibuang.
	require.Len(t, created.Coverages, 1)
	require.Equal(t, "TP01", created.Coverages[0].PlanID)

	updated, err := service.Update(ctx, "ASM", created.ID, daftardetaildokumentravel.Input{
		DocumentName: " Baru ",
		Mandatory:    true,
		MinUpload:    2,
	})
	require.NoError(t, err)
	require.Equal(t, "Baru", updated.DocumentName)
	require.True(t, updated.Mandatory)
	require.Empty(t, updated.Coverages)

	stored, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, updated, stored)
}

func TestServicePropagatesPortalFailure(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "XX")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "XX", "00001")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "XX", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "XX", "00001", daftardetaildokumentravel.Input{})
	require.ErrorIs(t, err, errPortal)
	_, err = service.Documents(ctx, "XX")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Plans(ctx, "XX")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Coverages(ctx, "XX")
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("XX")
	require.ErrorIs(t, err, errPortal)
	require.ErrorContains(t, err, `portal "XX" tidak dapat dilayani`)
}
