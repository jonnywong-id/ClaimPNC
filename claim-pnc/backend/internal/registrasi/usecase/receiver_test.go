package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tombol Tambah lalu Simpan: penerima baru mendapat IDRECEIVER berikutnya, dan Nama, Nama
// Bank, serta Alamat diisi dari Master Rekening — ketiganya baca-saja di InputReceiver.
func TestSaveReceiverAddsReceiverFromMasterAccount(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.toChooseSurveyorWithPIC(t, "")
	require.Len(t, claim.Receiver, 1, "penerima bawaan dari Input Register")

	saved, err := l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: claim.ID, TaskID: task.ID, AccountNo: "1234567890", Email: "penerima@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	require.Len(t, saved.Receiver, 2)
	require.Equal(t, registrasi.Receiver{
		ID: "2", AccountNo: "1234567890", Name: "PT CONTOH PENERIMA", BankName: "BANK CONTOH", Address: "JL. CONTOH NO. 1",
	}, saved.Receiver[1])

	stored, err := l.store.Get(ctx, claim.ID)
	require.NoError(t, err)
	require.Equal(t, saved.Receiver, stored.Receiver)
}

// Membuka baris penerima lalu mengganti No Rekening mengubah penerima itu, bukan menambah.
func TestSaveReceiverUpdatesExistingReceiver(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.toChooseSurveyorWithPIC(t, "")

	saved, err := l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: claim.ID, TaskID: task.ID, ReceiverID: "1", AccountNo: "9876543210", Email: "a@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	require.Len(t, saved.Receiver, 1)
	require.Equal(t, "1", saved.Receiver[0].ID)
	require.Equal(t, "CV CONTOH KEDUA", saved.Receiver[0].Name)
	require.Equal(t, "9876543210", saved.Receiver[0].AccountNo)

	_, err = l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: claim.ID, TaskID: task.ID, ReceiverID: "7", AccountNo: "9876543210", Email: "a@contoh.internal",
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction, "penerima yang tidak ada tidak diciptakan diam-diam")
}

// Pelanggaran dikumpulkan sekaligus: No Rekening dan Email kosong ditolak bersama.
func TestSaveReceiverRejectsIncompleteInput(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.toChooseSurveyorWithPIC(t, "")

	_, err := l.service.SaveReceiver(ctx, usecase.ReceiverCommand{ClaimID: claim.ID, TaskID: task.ID}, l.caller)
	var validation *registrasi.ValidationError
	require.True(t, errors.As(err, &validation))
	codes := []registrasi.ViolationCode{}
	for _, v := range validation.Violation {
		codes = append(codes, v.Code)
	}
	require.ElementsMatch(t, []registrasi.ViolationCode{
		registrasi.ViolationReceiverAccountEmpty, registrasi.ViolationReceiverEmailEmpty,
	}, codes)

	_, err = l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: claim.ID, TaskID: task.ID, AccountNo: "000", Email: "a@contoh.internal",
	}, l.caller)
	require.True(t, errors.As(err, &validation))
	require.Equal(t, registrasi.ViolationReceiverAccountUnknown, validation.Violation[0].Code)

	stored, err := l.store.Get(ctx, claim.ID)
	require.NoError(t, err)
	require.Len(t, stored.Receiver, 1, "isian yang ditolak tidak mengubah penerima")
}

func TestFindAccountReadsMasterAccount(t *testing.T) {
	l := setup(t)
	a, err := l.service.FindAccount(context.Background(), " 1234567890 ")
	require.NoError(t, err)
	require.Equal(t, "JAKARTA", a.Branch)
	require.False(t, a.CommitteeApprovedAt.IsZero())

	_, err = l.service.FindAccount(context.Background(), "")
	require.ErrorIs(t, err, registrasi.ErrAccountNotFound)
}

func TestNextReceiverIDFollowsHighestNumber(t *testing.T) {
	require.Equal(t, "1", registrasi.NextReceiverID(nil))
	require.Equal(t, "4", registrasi.NextReceiverID([]registrasi.Receiver{{ID: "1"}, {ID: "3"}, {ID: "x"}}))
}
