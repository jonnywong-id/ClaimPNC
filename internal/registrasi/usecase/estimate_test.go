package usecase_test

import (
	"context"
	"errors"
	"strings"
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

	choices, err := l.service.ItemOptions(context.Background(), claim.ID, claim.InsuredItem[0].ID, "")
	require.NoError(t, err)
	require.NotEmpty(t, choices.Option)
	require.Equal(t, "BUILDING AND CONTENTS", choices.Option[0].Name)
	require.Empty(t, choices.Default, "Fire tidak punya nama item bawaan")
}

// Lini tanpa daftar item (Aneka, Marine Cargo, PA): satu-satunya pilihan "Others", seperti
// item bawaan `GetObjectFromTable`.
func TestItemOptionsOutsideFireAndTravelAreOthers(t *testing.T) {
	l := setup(t)
	claim, _ := l.upToInputRegister(t, paPolicy)

	choices, err := l.service.ItemOptions(context.Background(), claim.ID, "1", "")
	require.NoError(t, err)
	require.Equal(t, []registrasi.ItemOption{{Name: "Others"}}, choices.Option)
	require.Equal(t, "Others", choices.Default)
}

// Lini Travel: pilihan Objek adalah manfaat plan, plan = kode coverage jaminan.
func TestItemOptionsForTravelComeFromThePlan(t *testing.T) {
	l := setup(t)
	claim, _ := l.upToInputRegister(t, "POL-TRV-0003")

	choices, err := l.service.ItemOptions(context.Background(), claim.ID, "1", "10001")
	require.NoError(t, err)
	require.Equal(t, "OTHERS", choices.Default)
	require.Len(t, choices.Option, 2)
	require.Equal(t, "10919", choices.Option[0].ID)

	choices, err = l.service.ItemOptions(context.Background(), claim.ID, "1", "99999")
	require.NoError(t, err)
	require.Empty(t, choices.Option, "plan tanpa manfaat")
}

func TestDefaultItemNamePerLine(t *testing.T) {
	cases := map[string]registrasi.Policy{
		"":       {Line: registrasi.LineFire},
		"OTHERS": {Line: registrasi.LineTravel},
		"Others": {Line: registrasi.LineMarineCargo},
	}
	for want, p := range cases {
		require.Equal(t, want, registrasi.DefaultItemName(p), "%+v", p)
	}
	require.Equal(t, "Others", registrasi.DefaultItemName(registrasi.Policy{BusinessType: "Aneka"}))
	require.Equal(t, "Others", registrasi.DefaultItemName(registrasi.Policy{Line: registrasi.LinePersonalAccident}))
	require.True(t, registrasi.ItemFromTravelPlan(registrasi.Policy{BusinessType: " travel "}))
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

// Section InputEstimasiDetail juga ditanam di layar InputSurveyor (`ClaimSurvey_sect`):
// estimasi dapat ditambah dan disimpan di tahap Choose Surveyor tanpa memindahkan tahap,
// sedangkan Kirim PIC Teknik tetap hanya milik Input Estimasi.
func TestSaveEstimateAlsoWorksAtInputSurveyor(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	command := oneEstimate(task.ID, registrasi.Rupiah(5_000_000), 2)
	claim, err := l.service.SaveEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, claim.CurrentStage)
	require.Len(t, claim.InsuredItem[0].Coverage[0].Item[0].Estimation, 2)

	_, err = l.service.CompleteEstimate(ctx, command, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}

// Catatan ke PIC Teknis tersimpan bersama isian estimasi (T_CLAIM_PNC.REMARK); permintaan
// tanpa catatan (nil) membiarkan catatan yang tersimpan.
func TestEstimateSavesTechnicalPICNote(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	command := oneEstimate(task.ID, registrasi.Rupiah(10), 1)
	note := "  Mohon cek dokumen survei  "
	command.TechnicalPICNote = &note
	_, err := l.service.SaveEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, "Mohon cek dokumen survei", stored.TechnicalPICNote)

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(10), 1), l.caller)
	require.NoError(t, err)
	stored, err = l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, "Mohon cek dokumen survei", stored.TechnicalPICNote)
}

// Catatan yang tidak muat di kolom REMARK (4000) ditolak.
func TestEstimateRejectsTooLongTechnicalPICNote(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)
	command := oneEstimate(task.ID, registrasi.Rupiah(10), 1)
	note := strings.Repeat("a", registrasi.TechnicalPICNoteMax+1)
	command.TechnicalPICNote = &note
	_, err := l.service.SaveEstimate(context.Background(), command, l.caller)
	violation(t, err, registrasi.ViolationTechnicalPICNoteTooLong)
}
