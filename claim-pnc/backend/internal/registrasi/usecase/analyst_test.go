package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// upToEstimationPA mendaftarkan klaim PA sampai tahap Estimation (Assignment4), dengan dua jaminan
// pada objek pertama.
func (l environment) upToEstimationPA(t *testing.T) (registrasi.Claim, registrasi.Task) {
	t.Helper()
	_, task := l.upToInputRegister(t, paPolicy)
	input := validInput(task.ID)
	input.InsuredItem[0].Coverage[0].CauseOfLoss = registrasi.CauseOfLossPA
	second := input.InsuredItem[0].Coverage[0]
	second.ID = "CVG-2"
	input.InsuredItem[0].Coverage = append(input.InsuredItem[0].Coverage, second)
	registered, err := l.service.SaveRegister(context.Background(), input, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimatePA, registered.NextTask.Stage)
	return registered.Claim, *registered.NextTask
}

// Kirim Analyst (setTicketToAnalyst) menutup Estimation, melompat ke Send To Analis lewat Ticket
// SendtoAnalysator, mengisi StatusClaim 1151 dan tanggal transfer Analyst.
func TestTransferToAnalystJumpsToSendToAnalyst(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.upToEstimationPA(t)
	item := claim.InsuredItem[0]
	coverage := item.Coverage[len(item.Coverage)-1]

	result, err := l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: coverage.ID,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageSendToAnalyst, result.Claim.CurrentStage)
	require.NotNil(t, result.NextTask)
	require.Equal(t, registrasi.StageSendToAnalyst, result.NextTask.Stage)

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, usecase.StatusClaimAnalyst, stored.ClaimStatus)
	require.False(t, stored.AnalystTransferredAt.IsZero())
	require.True(t, stored.InsuredItem[0].Coverage[len(item.Coverage)-1].AnalystTransferred)

	trail := l.store.AuditTrail()
	require.Equal(t, "TRANSFER_KE_ANALYST", trail[len(trail)-1].Event)

	_, err = l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: coverage.ID,
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyDone)
}

// Jaminan yang BUKAN terakhir hanya ditandai (setTicketToAnalyst step 5 dan 9): klaim dan tugasnya
// tetap di Estimation, dan jaminan itu tidak dapat ditransfer dua kali. Jaminan terakhir kemudian
// memindahkan klaim.
func TestTransferToAnalystOnEarlierCoverageOnlyMarks(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.upToEstimationPA(t)
	item := claim.InsuredItem[0]
	require.Len(t, item.Coverage, 2)

	result, err := l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: item.Coverage[0].ID,
	}, l.caller)
	require.NoError(t, err)
	require.Nil(t, result.NextTask)
	require.Equal(t, registrasi.StageEstimatePA, result.Claim.CurrentStage)

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimatePA, stored.CurrentStage)
	require.NotEqual(t, usecase.StatusClaimAnalyst, stored.ClaimStatus)
	require.True(t, stored.InsuredItem[0].Coverage[0].AnalystTransferred)
	require.False(t, stored.InsuredItem[0].Coverage[1].AnalystTransferred)
	require.False(t, stored.AnalystTransferredAt.IsZero())

	_, err = l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: item.Coverage[0].ID,
	}, l.caller)
	var invalid *registrasi.ValidationError
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, registrasi.ViolationAnalystTransferred, invalid.Violation[0].Code)

	moved, err := l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: item.Coverage[1].ID,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageSendToAnalyst, moved.Claim.CurrentStage)
}

// Jaminan yang tidak ada ditolak sebelum apa pun berubah.
func TestTransferToAnalystRejectsUnknownCoverage(t *testing.T) {
	l := setup(t)
	claim, task := l.upToEstimationPA(t)

	_, err := l.service.TransferToAnalyst(context.Background(), usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: claim.InsuredItem[0].ID, CoverageID: "tidak-ada",
	}, l.caller)
	var invalid *registrasi.ValidationError
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, registrasi.ViolationAnalystNotAllowed, invalid.Violation[0].Code)

	stored, err := l.store.Get(context.Background(), task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimatePA, stored.CurrentStage)
	require.True(t, stored.AnalystTransferredAt.IsZero())
}

// Tombol hanya ada pada Estimation PA; tahap lain ditolak.
func TestTransferToAnalystOnlyOnEstimationPA(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.TransferToAnalyst(context.Background(), usecase.TransferToAnalystCommand{TaskID: task.ID}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrNotAvailableAtStage)
}

// PreClaimComitee_OC langkah 21: klaim PA tanpa PIC Teknik dikirim ke PIC Teknik bawaan
// (PIC_TEKNIK_PA_BAWAAN), dan PIC itu dicatat ke klaim.
func TestTransferToAnalystUsesDefaultPATechnicalPIC(t *testing.T) {
	l := setupWith(t, func(o *usecase.Options) { o.DefaultPATechnicalPIC = "PICBAWAAN" })
	ctx := context.Background()
	claim, task := l.upToEstimationPA(t)
	// Klaim tanpa PIC Teknik — seperti PNCN.26.56 (PICTEKNIK kosong).
	claim.TechnicalPIC = ""
	require.NoError(t, l.store.Save(ctx, claim))
	item := claim.InsuredItem[0]

	result, err := l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: item.Coverage[len(item.Coverage)-1].ID,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "PICBAWAAN", result.NextTask.Owner)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, "PICBAWAAN", stored.TechnicalPIC)
}

// Tanpa PIC bawaan, PIC yang dipilih router tetap dicatat ke klaim (PICTEKNIK).
func TestTransferToAnalystRecordsChosenPIC(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.upToEstimationPA(t)
	item := claim.InsuredItem[0]

	result, err := l.service.TransferToAnalyst(ctx, usecase.TransferToAnalystCommand{
		TaskID: task.ID, ObjectID: item.ID, CoverageID: item.Coverage[len(item.Coverage)-1].ID,
	}, l.caller)
	require.NoError(t, err)
	require.NotEmpty(t, result.NextTask.Owner)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, result.NextTask.Owner, stored.TechnicalPIC)
}
