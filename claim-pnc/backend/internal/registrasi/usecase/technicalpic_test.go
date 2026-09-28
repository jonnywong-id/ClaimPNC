package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// toChooseSurveyorWithPIC membawa klaim Fire sampai Choose Surveyor dengan PIC Teknik yang
// diisi di Input Register.
func (l environment) toChooseSurveyorWithPIC(t *testing.T, pic string) (registrasi.Claim, registrasi.Task) {
	t.Helper()
	ctx := context.Background()
	_, task := l.upToInputRegister(t, firePolicy)
	input := validInput(task.ID)
	input.TechnicalPIC = pic
	registered, err := l.service.SaveRegister(ctx, input, l.caller)
	require.NoError(t, err)
	require.NotNil(t, registered.NextTask)

	estimate := *registered.NextTask
	command := oneEstimate(estimate.ID, registrasi.Rupiah(5_000_000), 1)
	_, err = l.service.SaveEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(estimate), l.caller)
	require.NoError(t, err)
	result, err := l.service.CompleteEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, result.NextTask.Stage)
	return result.Claim, *result.NextTask
}

// Klaim tanpa PIC Teknik: router memilih, dan pilihannya tercatat di klaim — sehingga
// PICTEKNIK dan USERTEKNIS_1 berisi pemegang tugas, seperti baris Pega.
func TestChosenTechnicalPICIsStoredOnClaim(t *testing.T) {
	l := setup(t)
	claim, task := l.toChooseSurveyorWithPIC(t, "")

	require.Equal(t, testOperator, task.Owner)
	require.Equal(t, task.Owner, claim.TechnicalPIC, "PIC terpilih harus tersimpan di klaim")

	stored, err := l.store.Get(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Equal(t, task.Owner, stored.TechnicalPIC)

	e, ok := l.inbox.Get(claim.Number)
	require.True(t, ok)
	require.Equal(t, task.Owner, e.Claim.TechnicalPIC, "USERTEKNIS_1 = pemegang tugas")
	require.Equal(t, task.Owner, e.AssignedOperator())
}

// Klaim yang sudah punya PIC Teknik: tahap teknis diberikan kepadanya, bukan ke petugas
// dengan beban paling ringan (320 dari 338 baris Choose Surveyor Pega).
func TestExistingTechnicalPICReceivesTechnicalStage(t *testing.T) {
	l := setup(t)
	claim, task := l.toChooseSurveyorWithPIC(t, "PICLAIN")

	require.Equal(t, "PICLAIN", task.Owner)
	require.Equal(t, "PICLAIN", claim.TechnicalPIC, "PIC yang sudah ada tidak ditimpa")
}

func TestAdoptTechnicalPICOnlyForTechnicalRouterAndEmptyPIC(t *testing.T) {
	technical := registrasi.Stage{Router: registrasi.RouterPNCTechnical}
	admin := registrasi.Stage{Router: registrasi.RouterPNCAdmin}

	k := registrasi.Claim{}
	registrasi.AdoptTechnicalPIC(&k, admin, registrasi.Assignee{Operator: "ADMIN"})
	require.Empty(t, k.TechnicalPIC, "tahap admin tidak menentukan PIC Teknik")

	registrasi.AdoptTechnicalPIC(&k, technical, registrasi.Assignee{Operator: "PIC1"})
	require.Equal(t, "PIC1", k.TechnicalPIC)

	registrasi.AdoptTechnicalPIC(&k, technical, registrasi.Assignee{Operator: "PIC2"})
	require.Equal(t, "PIC1", k.TechnicalPIC, "PIC yang sudah ada tidak pernah ditimpa")

	require.Equal(t, "PIC1", registrasi.AssignedTechnicalPIC(technical, k))
	require.Empty(t, registrasi.AssignedTechnicalPIC(admin, k))
}
