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

// Kirim ke Inputor menutup tugas berjalan, melompat ke Input Register milik pembuat klaim,
// dan mencatat catatan analis sebagai komunikasi berkanal SENDTOINPUTOR.
func TestSendToInputorJumpsToInputRegister(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToInputor(ctx, usecase.SendToInputorCommand{
		TaskID: task.ID, Note: "  Lengkapi KTP tertanggung.  ",
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, result.Claim.CurrentStage)
	require.NotNil(t, result.NextTask)
	require.Equal(t, registrasi.StageInputRegister, result.NextTask.Stage)
	require.Equal(t, testOperator, result.NextTask.Owner, "PNCAdminRouter: pembuat klaim")

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, stored.CurrentStage)

	comms, err := l.records.Communications(ctx, stored.Keys())
	require.NoError(t, err)
	require.Len(t, comms, 1)
	require.Equal(t, "Lengkapi KTP tertanggung.", comms[0].Message)
	require.Equal(t, registrasi.ChannelSendToInputor, comms[0].Channel)
	require.Equal(t, registrasi.CommunicationStatusOpen, comms[0].Status)
	require.Equal(t, testOperator, comms[0].Sender)

	trail := l.store.AuditTrail()
	require.Equal(t, "KIRIM_KE_INPUTOR", trail[len(trail)-1].Event)
	require.Equal(t, registrasi.StageChooseSurveyor+" → "+registrasi.StageInputRegister, trail[len(trail)-1].Note)

	// Tugas lama sudah ditutup: mengirim lagi lewat tugas yang sama ditolak.
	_, err = l.service.SendToInputor(ctx, usecase.SendToInputorCommand{TaskID: task.ID}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyDone)
}

// Catatan kosong diperbolehkan (pyRequired=false), tetapi tidak menulis baris komunikasi.
func TestSendToInputorWithoutNoteWritesNoCommunication(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	result, err := l.service.SendToInputor(ctx, usecase.SendToInputorCommand{TaskID: task.ID, Note: "   "}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, result.Claim.CurrentStage)

	comms, err := l.records.Communications(ctx, result.Claim.Keys())
	require.NoError(t, err)
	require.Empty(t, comms)
}

// Catatan melebihi kolom MESSAGE ditolak sebelum apa pun berubah.
func TestSendToInputorRejectsTooLongNote(t *testing.T) {
	l := setup(t)
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)

	_, err := l.service.SendToInputor(context.Background(), usecase.SendToInputorCommand{
		TaskID: task.ID, Note: strings.Repeat("a", 4001),
	}, l.caller)
	var invalid *registrasi.ValidationError
	require.True(t, errors.As(err, &invalid))
	require.Equal(t, registrasi.ViolationNoteTooLong, invalid.Violation[0].Code)

	stored, err := l.store.Get(context.Background(), task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, stored.CurrentStage)
}

// Tombol hanya ada di layar ClaimSurvey_sect; Input Register tidak termasuk.
func TestSendToInputorOnlyFromClaimSurveyStages(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	_, err := l.service.SendToInputor(context.Background(), usecase.SendToInputorCommand{TaskID: task.ID}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}
