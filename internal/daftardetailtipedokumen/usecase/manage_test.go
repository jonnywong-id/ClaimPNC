package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetailtipedokumen"
	"claim-pnc/internal/daftardetailtipedokumen/repo/memory"
	"claim-pnc/internal/daftardetailtipedokumen/usecase"
	"claim-pnc/internal/platform/clock"
)

var errPortal = errors.New("portal tidak dikenal")

// recordingRepo membungkus repo memory dan mencatat Editor yang diterima.
type recordingRepo struct {
	*memory.Repo
	editors []daftardetailtipedokumen.Editor
	inputs  []daftardetailtipedokumen.Input
}

func (r *recordingRepo) InsertNew(ctx context.Context, input daftardetailtipedokumen.Input, by daftardetailtipedokumen.Editor) (daftardetailtipedokumen.DetailType, error) {
	r.editors = append(r.editors, by)
	r.inputs = append(r.inputs, input)
	return r.Repo.InsertNew(ctx, input, by)
}

func (r *recordingRepo) Update(ctx context.Context, id string, input daftardetailtipedokumen.Input, by daftardetailtipedokumen.Editor) (daftardetailtipedokumen.DetailType, error) {
	r.editors = append(r.editors, by)
	r.inputs = append(r.inputs, input)
	return r.Repo.Update(ctx, id, input, by)
}

// partialReferences gagal hanya pada daftar yang ditandai.
type partialReferences struct {
	failDocument bool
}

func (p partialReferences) ListDocumentTypes(context.Context) ([]daftardetailtipedokumen.DocumentTypeOption, error) {
	if p.failDocument {
		return nil, errPortal
	}
	return memory.SampleDocumentTypeList(), nil
}

var fixedNow = time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC)

func newService(t *testing.T, repo daftardetailtipedokumen.Repo, references daftardetailtipedokumen.ReferenceRepo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (daftardetailtipedokumen.Repo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return repo, nil
		},
		ReferenceSelector: func(alias string) (daftardetailtipedokumen.ReferenceRepo, error) {
			if alias != "ASM" {
				return nil, errPortal
			}
			return references, nil
		},
		Clock: clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRejectsMissingParts(t *testing.T) {
	repoSelector := func(string) (daftardetailtipedokumen.Repo, error) { return nil, nil }
	referenceSelector := func(string) (daftardetailtipedokumen.ReferenceRepo, error) { return nil, nil }
	fixed := clock.FixedAt(fixedNow)

	_, err := usecase.NewService(usecase.Options{ReferenceSelector: referenceSelector, Clock: fixed})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")
	_, err = usecase.NewService(usecase.Options{RepoSelector: repoSelector, Clock: fixed})
	require.ErrorContains(t, err, "ReferenceSelector wajib diisi")
	_, err = usecase.NewService(usecase.Options{RepoSelector: repoSelector, ReferenceSelector: referenceSelector})
	require.ErrorContains(t, err, "Clock wajib diisi")
}

func TestServiceListAndGet(t *testing.T) {
	service := newService(t, memory.NewRepo(memory.SampleList()...), memory.NewSampleReferenceRepo())
	ctx := context.Background()

	list, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, list, 5)

	row, err := service.Get(ctx, "ASM", "100004")
	require.NoError(t, err)
	require.Equal(t, "Surat Keterangan Dokter", row.Detail)

	_, err = service.Get(ctx, "ASM", "nope")
	require.ErrorIs(t, err, daftardetailtipedokumen.ErrNotFound)
}

func TestServiceCreateCleansAndStampsEditor(t *testing.T) {
	repo := &recordingRepo{Repo: memory.NewRepo(memory.SampleList()...)}
	service := newService(t, repo, memory.NewSampleReferenceRepo())

	created, err := service.Create(context.Background(), "ASM", daftardetailtipedokumen.Input{
		DocumentTypeID: " 10001 ",
		Detail:         "  Kwitansi ",
	}, "  adminpnc ")
	require.NoError(t, err)
	require.Equal(t, "100006", created.ID)
	require.Equal(t, "Kwitansi", created.Detail)

	require.Equal(t, []daftardetailtipedokumen.Editor{{Identity: "adminpnc", At: fixedNow}}, repo.editors)
	require.Equal(t, "10001", repo.inputs[0].DocumentTypeID)
}

func TestServiceUpdateCleansAndStampsEditor(t *testing.T) {
	repo := &recordingRepo{Repo: memory.NewRepo(memory.SampleList()...)}
	service := newService(t, repo, memory.NewSampleReferenceRepo())

	updated, err := service.Update(context.Background(), "ASM", "100003", daftardetailtipedokumen.Input{Detail: " Foto Baru "}, "pic")
	require.NoError(t, err)
	require.Equal(t, "Foto Baru", updated.Detail)
	require.Equal(t, "pic", repo.editors[0].Identity)

	_, err = service.Update(context.Background(), "ASM", "999", daftardetailtipedokumen.Input{}, "pic")
	require.ErrorIs(t, err, daftardetailtipedokumen.ErrNotFound)
}

func TestServiceRejectsInvalidInputBeforeRepo(t *testing.T) {
	repo := &recordingRepo{Repo: memory.NewRepo()}
	service := newService(t, repo, memory.NewSampleReferenceRepo())
	long := make([]byte, daftardetailtipedokumen.MaxRiskLength+1)
	for i := range long {
		long[i] = 'x'
	}

	var validation *daftardetailtipedokumen.ValidationError
	_, err := service.Create(context.Background(), "ASM", daftardetailtipedokumen.Input{Risk: string(long)}, "u")
	require.ErrorAs(t, err, &validation)
	require.Equal(t, daftardetailtipedokumen.FieldRisk, validation.Violation[0].Field)

	_, err = service.Update(context.Background(), "ASM", "1", daftardetailtipedokumen.Input{Risk: string(long)}, "u")
	require.ErrorAs(t, err, &validation)
	require.Empty(t, repo.editors)
}

func TestServiceUnknownPortal(t *testing.T) {
	service := newService(t, memory.NewRepo(), memory.NewSampleReferenceRepo())
	ctx := context.Background()

	_, err := service.List(ctx, "X")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "X", "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "X", daftardetailtipedokumen.Input{}, "u")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "X", "1", daftardetailtipedokumen.Input{}, "u")
	require.ErrorIs(t, err, errPortal)
	_, err = service.References(ctx, "X")
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("X")
	require.ErrorIs(t, err, errPortal)
	require.ErrorContains(t, err, `portal "X" tidak dapat dilayani`)
	require.NoError(t, service.EnsurePortalReady("ASM"))
}

func TestReferencesAllAvailable(t *testing.T) {
	service := newService(t, memory.NewRepo(), partialReferences{})
	got, err := service.References(context.Background(), "ASM")
	require.NoError(t, err)
	require.Len(t, got.DocumentTypes, 6)
	require.Empty(t, got.Unavailable)
}

func TestReferencesReportEachUnavailableListWithoutFailing(t *testing.T) {
	service := newService(t, memory.NewRepo(), partialReferences{failDocument: true})
	got, err := service.References(context.Background(), "ASM")
	require.NoError(t, err)
	require.Equal(t, []string{usecase.ReferenceDocumentType}, got.Unavailable)
	require.Nil(t, got.DocumentTypes)

	// Master yang terbaca tidak ikut masuk daftar yang hilang.
	service = newService(t, memory.NewRepo(), partialReferences{})
	got, err = service.References(context.Background(), "ASM")
	require.NoError(t, err)
	require.Empty(t, got.Unavailable)
	require.Len(t, got.DocumentTypes, 6)
}
