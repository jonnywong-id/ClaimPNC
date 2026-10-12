package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/mastertipesurveyors/usecase"
)

// be4ScriptRepo adalah repo yang perilaku tiap method-nya diatur per uji.
type be4ScriptRepo struct {
	list    []mastertipesurveyors.SurveyorType
	listErr error
	getErr  error
	insErr  error
	updErr  error
}

func (r *be4ScriptRepo) List(context.Context) ([]mastertipesurveyors.SurveyorType, error) {
	return r.list, r.listErr
}

func (r *be4ScriptRepo) Get(_ context.Context, code string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: code}, r.getErr
}

func (r *be4ScriptRepo) Insert(_ context.Context, d string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: "1001", Description: d}, r.insErr
}

func (r *be4ScriptRepo) Update(_ context.Context, c, d string) (mastertipesurveyors.SurveyorType, error) {
	return mastertipesurveyors.SurveyorType{Code: c, Description: d}, r.updErr
}

func be4Service(t *testing.T, repo *be4ScriptRepo) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastertipesurveyors.Repo, error) { return repo, nil },
	})
	require.NoError(t, err)
	return service
}

// TestGetWrapsStorageFailure membuktikan galat selain ErrNotFound dibungkus.
func TestGetWrapsStorageFailure(t *testing.T) {
	boom := errors.New("putus")
	service := be4Service(t, &be4ScriptRepo{getErr: boom})
	_, err := service.Get(context.Background(), portalASM, "1001")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca tipe surveyor")
}

// TestCreatePassesDomainInsertErrorsThrough memeriksa galat penegakan basis data diteruskan.
func TestCreatePassesDomainInsertErrorsThrough(t *testing.T) {
	for _, sentinel := range []error{
		mastertipesurveyors.ErrDescriptionTaken,
		mastertipesurveyors.ErrCodeTaken,
		mastertipesurveyors.ErrNoSite,
	} {
		service := be4Service(t, &be4ScriptRepo{insErr: sentinel})
		_, err := service.Create(context.Background(), portalASM, "BARU")
		require.Equal(t, sentinel, err)
	}

	boom := errors.New("lain")
	service := be4Service(t, &be4ScriptRepo{insErr: boom})
	_, err := service.Create(context.Background(), portalASM, "BARU")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menambah tipe surveyor")
}

// TestCreateFailsWhenUniquenessCannotBeChecked membuktikan galat List dibungkus.
func TestCreateFailsWhenUniquenessCannotBeChecked(t *testing.T) {
	boom := errors.New("daftar rusak")
	service := be4Service(t, &be4ScriptRepo{listErr: boom})
	_, err := service.Create(context.Background(), portalASM, "BARU")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "memeriksa keunikan")
}

// TestUpdateErrorPaths memeriksa galat baca awal dan galat simpan.
func TestUpdateErrorPaths(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("putus")

	service := be4Service(t, &be4ScriptRepo{getErr: boom})
	_, err := service.Update(ctx, portalASM, "1001", "X")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "membaca tipe surveyor")

	service = be4Service(t, &be4ScriptRepo{listErr: boom})
	_, err = service.Update(ctx, portalASM, "1001", "X")
	require.ErrorIs(t, err, boom)

	for _, sentinel := range []error{mastertipesurveyors.ErrNotFound, mastertipesurveyors.ErrDescriptionTaken} {
		service = be4Service(t, &be4ScriptRepo{updErr: sentinel})
		_, err = service.Update(ctx, portalASM, "1001", "X")
		require.Equal(t, sentinel, err)
	}

	service = be4Service(t, &be4ScriptRepo{updErr: boom})
	_, err = service.Update(ctx, portalASM, " 1001 ", " X ")
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "mengubah tipe surveyor")

	service = be4Service(t, &be4ScriptRepo{})
	saved, err := service.Update(ctx, portalASM, " 1001 ", " X ")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1001", Description: "X"}, saved)
}
