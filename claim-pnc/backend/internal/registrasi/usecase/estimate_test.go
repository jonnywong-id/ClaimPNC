package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// upToInputEstimate membawa klaim Fire (Non-MBU) sampai tahap Input Estimasi.
func (l environment) upToInputEstimate(t *testing.T) (registrasi.Claim, registrasi.Task) {
	t.Helper()
	_, task := l.upToInputRegister(t, firePolicy)
	result, err := l.service.SaveRegister(context.Background(), validInput(task.ID), l.caller)
	require.NoError(t, err)
	require.NotNil(t, result.NextTask)
	require.Equal(t, registrasi.StageEstimateAdmin, result.Claim.CurrentStage)
	return result.Claim, *result.NextTask
}

func oneEstimate(taskID string, value registrasi.Money, rows int) usecase.EstimateCommand {
	item := usecase.ObjectItemInput{Name: "Gudang", Description: "Atap rusak"}
	for n := 0; n < rows; n++ {
		item.Estimation = append(item.Estimation, usecase.EstimationInput{
			Type: registrasi.EstimateClaim, Currency: "IDR", Value: value,
		})
	}
	return usecase.EstimateCommand{TaskID: taskID, Item: [][][]usecase.ObjectItemInput{{{item}}}}
}

func TestSaveEstimateKeepsTheStageAndComputesRupiah(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)

	claim, err := l.service.SaveEstimate(context.Background(), oneEstimate(task.ID, registrasi.Rupiah(8_000_000), 1), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimateAdmin, claim.CurrentStage)

	e := claim.InsuredItem[0].Coverage[0].Item[0].Estimation[0]
	require.Equal(t, registrasi.ExchangeRateOne, e.Rate)
	require.Equal(t, registrasi.Rupiah(8_000_000), e.Converted)
	require.False(t, e.Date.IsZero(), "tanggal estimasi kosong diisi hari ini")
}

// Kirim PIC Teknik (`finishAssignment`) membawa klaim Non-MBU ke Choose Surveyor.
func TestSendToTechnicalPICMovesToChooseSurveyor(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(1_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	result, err := l.service.CompleteEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(1_000_000), 1), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, result.Claim.CurrentStage)
	require.NotNil(t, result.NextTask)
	require.Equal(t, registrasi.StageChooseSurveyor, result.NextTask.Stage)
}

// Tombolnya `pyDisabledWhen !isCFS`; server menolaknya juga.
func TestSendToTechnicalPICNeedsFaceSheet(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)

	_, err := l.service.CompleteEstimate(context.Background(), oneEstimate(task.ID, registrasi.Rupiah(1_000_000), 1), l.caller)
	violation(t, err, registrasi.ViolationSendNeedsFaceSheet)
}

func TestNextRequiresAnEstimate(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)

	_, err := l.service.CompleteEstimate(context.Background(), oneEstimate(task.ID, 0, 0), l.caller)
	var validation *registrasi.ValidationError
	require.True(t, errors.As(err, &validation), "galat = %v", err)
	require.True(t, validation.Has(registrasi.ViolationEstimateMissing))
}

// Back kembali ke Input Register, dan estimasinya tidak hilang saat Input Register
// disimpan lagi.
func TestBackKeepsTheEstimate(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	command := oneEstimate(task.ID, registrasi.Rupiah(5_000_000), 1)
	command.Return = true
	back, err := l.service.CompleteEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, back.Claim.CurrentStage)

	again, err := l.service.SaveRegister(ctx, validInput(back.NextTask.ID), l.caller)
	require.NoError(t, err)
	require.Len(t, again.Claim.InsuredItem[0].Coverage[0].Item[0].Estimation, 1)
}

// T_CLAIM_ESTIMASI tidak punya penanda hapus; estimasi yang tersimpan tidak dapat
// dikurangi.
func TestSavedEstimateCannotBeRemoved(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(1), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(1), 2), l.caller)
	require.NoError(t, err)

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(1), 1), l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}

// Tahap Input Estimasi tidak dapat ditutup lewat jalur umum yang melewatkan isiannya.
func TestGenericCompletionRefusesInputEstimate(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)

	_, err := l.service.CompleteStage(context.Background(), usecase.CompleteCommand{
		TaskID: task.ID, Action: registrasi.ActionInputEstimate,
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}

// Lini Fire: pilihan Objek item estimasi berasal dari item properti polis.
func TestItemOptionsComeFromThePolicy(t *testing.T) {
	l := setup(t)
	claim, _ := l.upToInputEstimate(t)

	options, err := l.service.ItemOptions(context.Background(), claim.ID, claim.InsuredItem[0].ID)
	require.NoError(t, err)
	require.NotEmpty(t, options)
	require.Equal(t, "BUILDING AND CONTENTS", options[0].Name)
}

// Status Klaim dibawa dengan namanya dari master, seperti layar Pega ("Register").
func TestClaimCarriesItsStatusName(t *testing.T) {
	l := setup(t)
	claim, task := l.upToInputEstimate(t)
	require.Equal(t, registrasi.StatusRegistered, claim.ClaimStatus)

	saved, err := l.service.SaveEstimate(context.Background(), oneEstimate(task.ID, registrasi.Rupiah(1), 1), l.caller)
	require.NoError(t, err)
	view, err := l.service.ViewClaim(context.Background(), saved.ID, l.caller)
	require.NoError(t, err)
	require.Equal(t, "Register", view.Claim.ClaimStatusName)
}

// InputRegister_act: submit Input Register membentuk satu penerima klaim bawaan dari polis
// (IDReceiver "1", nama tertanggung), dan penerima tidak menumpuk saat disubmit ulang.
func TestRegisterCreatesDefaultReceiver(t *testing.T) {
	l := setup(t)
	claim, _ := l.upToInputEstimate(t)
	require.Len(t, claim.Receiver, 1)
	require.Equal(t, "1", claim.Receiver[0].ID)
	require.NotEmpty(t, claim.Receiver[0].Name)

	stored, err := l.store.Get(context.Background(), claim.ID)
	require.NoError(t, err)
	require.Equal(t, claim.Receiver, stored.Receiver)
}
