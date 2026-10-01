package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/masterpenolakan/repo/memory"
	"claim-pnc/internal/masterpenolakan/usecase"
	"claim-pnc/internal/platform/clock"
)

var errPortal = errors.New("portal tidak dikenal")

// fixedNow adalah jam tetap yang mengisi TANGGALKIRIM pada seluruh test di berkas ini.
var fixedNow = time.Date(2026, 9, 20, 4, 30, 0, 0, time.UTC)

// newService merakit layanan dengan satu portal "asm" yang dikenal.
func newService(t *testing.T, repo *memory.Repo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpenolakan.Repo, error) {
			if alias != "asm" {
				return nil, errPortal
			}
			return repo, nil
		},
		Clock: clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRejectsMissingParts(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(fixedNow)})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpenolakan.Repo, error) { return nil, nil },
	})
	require.ErrorContains(t, err, "Clock wajib diisi")
}

func TestServiceReadsFromSelectedPortal(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	service := newService(t, repo)
	ctx := context.Background()

	list, err := service.List(ctx, "asm")
	require.NoError(t, err)
	require.Len(t, list, 4)

	parents, err := service.ListParent(ctx, "asm")
	require.NoError(t, err)
	require.Len(t, parents, 4)
	// Diurutkan menurut nama.
	require.Equal(t, "DOKUMEN TIDAK DILENGKAPI", parents[0].Name)

	row, err := service.Get(ctx, "asm", "3")
	require.NoError(t, err)
	require.Equal(t, "KERUGIAN AKIBAT KEAUSAN", row.Name)

	_, err = service.Get(ctx, "asm", "99")
	require.ErrorIs(t, err, masterpenolakan.ErrNotFound)
}

func TestServiceUnknownPortalFailsEveryMethod(t *testing.T) {
	service := newService(t, memory.NewRepo(nil))
	ctx := context.Background()

	_, err := service.List(ctx, "x")
	require.ErrorIs(t, err, errPortal)
	_, err = service.ListParent(ctx, "x")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Get(ctx, "x", "1")
	require.ErrorIs(t, err, errPortal)
	_, err = service.Create(ctx, "x", "u", masterpenolakan.Input{Name: "A", ParentName: "B"})
	require.ErrorIs(t, err, errPortal)
	_, err = service.Update(ctx, "x", "u", "1", masterpenolakan.Input{Name: "A", ParentName: "B"})
	require.ErrorIs(t, err, errPortal)

	err = service.EnsurePortalReady("x")
	require.ErrorIs(t, err, errPortal)
	require.ErrorContains(t, err, `portal "x" tidak dapat dilayani`)
	require.NoError(t, service.EnsurePortalReady("asm"))
}

func TestServiceCreateCleansStampsAndStores(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	service := newService(t, repo)

	created, err := service.Create(context.Background(), "asm", "adminpnc", masterpenolakan.Input{
		Name:     "  KLAIM GANDA  ",
		ParentID: " 2 ",
	})
	require.NoError(t, err)
	require.Equal(t, "5", created.ID)
	require.Equal(t, "KLAIM GANDA", created.Name)
	require.Equal(t, "2", created.ParentID)
	require.Equal(t, "KERUGIAN DIKECUALIKAN POLIS", created.ParentName)
	require.Equal(t, masterpenolakan.StatusPending, created.Status)
	require.Equal(t, "adminpnc", created.SubmittedBy)
	require.Equal(t, fixedNow, created.SubmittedAt)
}

func TestServiceCreateRejectsInvalidInputBeforeRepo(t *testing.T) {
	repo := memory.NewRepo(nil)
	service := newService(t, repo)

	// Isian spasi saja dibersihkan dulu, lalu gagal pemeriksaan wajib-isi.
	_, err := service.Create(context.Background(), "asm", "u", masterpenolakan.Input{Name: "   ", ParentName: " "})
	var validation *masterpenolakan.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violation, 2)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestServiceUpdateReturnsRowToPending(t *testing.T) {
	repo := memory.NewRepo(memory.SampleParents(), memory.SampleList()...)
	service := newService(t, repo)

	updated, err := service.Update(context.Background(), "asm", "manageradmin", "1", masterpenolakan.Input{
		Name:       "PREMI BELUM DIBAYAR",
		ParentName: "ALASAN BARU",
	})
	require.NoError(t, err)
	require.Equal(t, "1", updated.ID)
	require.Equal(t, "5", updated.ParentID)
	require.Equal(t, "ALASAN BARU", updated.ParentName)
	require.Equal(t, masterpenolakan.StatusPending, updated.Status)
	require.Equal(t, "manageradmin", updated.SubmittedBy)
	require.Equal(t, fixedNow, updated.SubmittedAt)

	_, err = service.Update(context.Background(), "asm", "u", "1", masterpenolakan.Input{Name: ""})
	var validation *masterpenolakan.ValidationError
	require.ErrorAs(t, err, &validation)
}
