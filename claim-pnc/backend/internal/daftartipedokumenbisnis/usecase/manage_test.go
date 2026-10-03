package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumenbisnis"
	"claim-pnc/internal/daftartipedokumenbisnis/repo/memory"
	"claim-pnc/internal/daftartipedokumenbisnis/usecase"
	"claim-pnc/internal/platform/clock"
)

var (
	errPortal = errors.New("portal belum siap")
	fixedAt   = time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)
)

// recordingRepo merekam jejak simpan yang diteruskan Service ke repo.
type recordingRepo struct {
	*memory.Repo
	editors []daftartipedokumenbisnis.Editor
	batches []daftartipedokumenbisnis.BatchInput
	inputs  []daftartipedokumenbisnis.Input
}

func (r *recordingRepo) InsertBatch(
	ctx context.Context, input daftartipedokumenbisnis.BatchInput, by daftartipedokumenbisnis.Editor,
) ([]daftartipedokumenbisnis.DocumentRule, error) {
	r.editors = append(r.editors, by)
	r.batches = append(r.batches, input)
	return r.Repo.InsertBatch(ctx, input, by)
}

func (r *recordingRepo) Update(
	ctx context.Context, id string, input daftartipedokumenbisnis.Input, by daftartipedokumenbisnis.Editor,
) (daftartipedokumenbisnis.DocumentRule, error) {
	r.editors = append(r.editors, by)
	r.inputs = append(r.inputs, input)
	return r.Repo.Update(ctx, id, input, by)
}

func options(repo daftartipedokumenbisnis.Repo) usecase.Options {
	return usecase.Options{
		RepoSelector: func(alias string) (daftartipedokumenbisnis.Repo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return repo, nil
		},
		BusinessSelector: func(alias string) (daftartipedokumenbisnis.BusinessRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return memory.NewBusinessRepo(memory.SampleBusinessList()...), nil
		},
		DocumentTypeSelector: func(alias string) (daftartipedokumenbisnis.DocumentTypeRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return memory.NewReferenceRepo(memory.SampleDocumentTypeList()...), nil
		},
		DetailTypeDocSelector: func(alias string) (daftartipedokumenbisnis.DetailTypeDocRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return memory.NewReferenceRepo(memory.SampleDetailTypeDocList()...), nil
		},
		ObjectDocSelector: func(alias string) (daftartipedokumenbisnis.ObjectDocRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return memory.NewReferenceRepo(memory.SampleObjectDocList()...), nil
		},
		Clock:                        clock.FixedAt(fixedAt),
		BulkSelectExcludedBusinesses: []string{" 10028 ", "", "  "},
	}
}

func newService(t *testing.T) (*usecase.Service, *recordingRepo) {
	t.Helper()
	repo := &recordingRepo{Repo: memory.NewRepo(memory.SampleList()...)}
	service, err := usecase.NewService(options(repo))
	require.NoError(t, err)
	return service, repo
}

func TestNewServiceRejectsMissingDependencies(t *testing.T) {
	cases := map[string]func(*usecase.Options){
		"RepoSelector":          func(o *usecase.Options) { o.RepoSelector = nil },
		"BusinessSelector":      func(o *usecase.Options) { o.BusinessSelector = nil },
		"DocumentTypeSelector":  func(o *usecase.Options) { o.DocumentTypeSelector = nil },
		"DetailTypeDocSelector": func(o *usecase.Options) { o.DetailTypeDocSelector = nil },
		"ObjectDocSelector":     func(o *usecase.Options) { o.ObjectDocSelector = nil },
		"Clock":                 func(o *usecase.Options) { o.Clock = nil },
	}
	for field, clear := range cases {
		t.Run(field, func(t *testing.T) {
			o := options(memory.NewRepo())
			clear(&o)
			service, err := usecase.NewService(o)
			require.Nil(t, service)
			require.ErrorContains(t, err, field+" wajib diisi")
		})
	}
}

func TestEveryOperationRejectsUnknownPortal(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.ListBusinesses(ctx, "X")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListByBusiness(ctx, "X", "001")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "X", "10001")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "X", daftartipedokumenbisnis.BatchInput{BusinessIDs: []string{"001"}}, "u")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "X", "10001", daftartipedokumenbisnis.Input{}, "u")
	require.ErrorIs(t, err, errPortal)
	_, err = service.AddCoverage(ctx, "X", "10001", "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListBusinessChoices(ctx, "X")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListDocumentTypes(ctx, "X")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListDetailTypeDocs(ctx, "X")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListObjectDocs(ctx, "X")
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("X")
	require.ErrorIs(t, err, errPortal)
	require.Contains(t, err.Error(), `portal "X" tidak dapat dilayani`)
	require.NoError(t, service.EnsurePortalReady("ASM"))
}

func TestReadsTrimKeys(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	businesses, err := service.ListBusinesses(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, businesses, 2)

	rules, err := service.ListByBusiness(ctx, "ASM", " 004 ")
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, "10004", rules[0].ID)

	rule, err := service.Get(ctx, "ASM", " 10001 ")
	require.NoError(t, err)
	require.Len(t, rule.Coverages, 1)
}

func TestCreateCleansThenValidatesAndStampsEditor(t *testing.T) {
	service, repo := newService(t)

	saved, err := service.Create(context.Background(), "ASM", daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{" 005 ", "005", " "},
		Rules: []daftartipedokumenbisnis.Input{
			{DocumentTypeID: " 20001 ", DetailDocument: " Laporan ", MinDocument: -3},
			{},
		},
	}, " 90000001 ")
	require.NoError(t, err)
	require.Len(t, saved, 1)
	require.Equal(t, "005", saved[0].BusinessID)
	require.Equal(t, "Laporan", saved[0].DetailDocument)
	require.Equal(t, 0, saved[0].MinDocument)
	require.Equal(t, []daftartipedokumenbisnis.Editor{{Identity: "90000001", At: fixedAt}}, repo.editors)
}

func TestCreateWithOnlyBlankBusinessesIsRejected(t *testing.T) {
	service, repo := newService(t)

	_, err := service.Create(context.Background(), "ASM", daftartipedokumenbisnis.BatchInput{
		BusinessIDs: []string{"  "},
		Rules:       []daftartipedokumenbisnis.Input{{DocumentTypeID: "20001"}},
	}, "u")
	require.ErrorIs(t, err, daftartipedokumenbisnis.ErrBusinessRequired)
	require.Empty(t, repo.batches, "penyimpanan tidak boleh mencapai repo")
}

func TestUpdateChecksExistenceThenSavesCleanInput(t *testing.T) {
	service, repo := newService(t)

	got, err := service.Update(context.Background(), "ASM", " 10002 ", daftartipedokumenbisnis.Input{
		DocumentTypeID: " 20004 ", DetailDocument: " Kuitansi ", MinDocument: -1,
	}, "")
	require.NoError(t, err)
	require.Equal(t, "Kuitansi", got.DetailDocument)
	require.Equal(t, daftartipedokumenbisnis.Input{DocumentTypeID: "20004", DetailDocument: "Kuitansi"}, repo.inputs[0])
	require.Equal(t, daftartipedokumenbisnis.Editor{At: fixedAt}, repo.editors[0])

	_, err = service.Update(context.Background(), "ASM", "99999", daftartipedokumenbisnis.Input{}, "u")
	require.ErrorIs(t, err, daftartipedokumenbisnis.ErrNotFound)
	require.Len(t, repo.inputs, 1, "baris yang tidak ada tidak boleh sampai ke Update repo")
}

func TestAddCoverageBlankReturnsRowUnchanged(t *testing.T) {
	service, _ := newService(t)

	got, err := service.AddCoverage(context.Background(), "ASM", " 10001 ", "   ")
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10009"}}, got.Coverages)

	added, err := service.AddCoverage(context.Background(), "ASM", "10001", " 10015 ")
	require.NoError(t, err)
	require.Equal(t, []daftartipedokumenbisnis.Coverage{{ID: "10009"}, {ID: "10015"}}, added.Coverages)
}

func TestListBusinessChoicesMarksConfiguredExclusions(t *testing.T) {
	service, _ := newService(t)

	list, err := service.ListBusinessChoices(context.Background(), "ASM")
	require.NoError(t, err)
	require.Len(t, list, 5)
	for _, business := range list {
		require.Equal(t, business.ID == "10028", business.ExcludedFromBulkSelect, business.ID)
	}
}

// failingBusinessRepo selalu gagal membaca master bisnis.
type failingBusinessRepo struct{}

func (failingBusinessRepo) List(context.Context) ([]daftartipedokumenbisnis.Business, error) {
	return nil, errPortal
}

func TestListBusinessChoicesPropagatesRepoError(t *testing.T) {
	o := options(memory.NewRepo())
	o.BusinessSelector = func(string) (daftartipedokumenbisnis.BusinessRepo, error) {
		return failingBusinessRepo{}, nil
	}
	service, err := usecase.NewService(o)
	require.NoError(t, err)

	_, err = service.ListBusinessChoices(context.Background(), "ASM")
	require.ErrorIs(t, err, errPortal)
}

func TestReferenceListsReturnSamples(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	types, err := service.ListDocumentTypes(ctx, "ASM")
	require.NoError(t, err)
	require.Equal(t, memory.SampleDocumentTypeList(), types)

	details, err := service.ListDetailTypeDocs(ctx, "ASM")
	require.NoError(t, err)
	require.Equal(t, memory.SampleDetailTypeDocList(), details)

	objects, err := service.ListObjectDocs(ctx, "ASM")
	require.NoError(t, err)
	require.Equal(t, memory.SampleObjectDocList(), objects)
}

func TestMayBulkSelectOnlyForNONMBU(t *testing.T) {
	service, _ := newService(t)

	require.True(t, service.MayBulkSelect("NONMBU"))
	require.True(t, service.MayBulkSelect(" nonmbu "))
	require.False(t, service.MayBulkSelect("MBU"))
	require.False(t, service.MayBulkSelect(""))
}
