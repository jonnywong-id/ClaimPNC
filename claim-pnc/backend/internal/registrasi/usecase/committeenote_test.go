package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Isian modal "Transfer Claim ke Komite" tersimpan pada jaminan, dan penyimpanan klaim
// berikutnya tidak mengosongkannya.
func TestSaveCommitteeNoteStoresOnCoverage(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	note := registrasi.CommitteeNote{Circumstances: "Kebakaran gudang", ExtentOfLoss: "Rp 10 jt", Remarks: "Layak dibayar"}

	claim, err := l.service.SaveCommitteeNote(ctx, usecase.CommitteeNoteCommand{
		TaskID: task.ID, ClaimID: task.ClaimID, Object: 1, Coverage: 1, Note: note,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, note, claim.InsuredItem[0].Coverage[0].Committee)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, note, stored.InsuredItem[0].Coverage[0].Committee)
}

func TestSaveCommitteeNoteRejects(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	base := usecase.CommitteeNoteCommand{TaskID: task.ID, ClaimID: task.ClaimID, Object: 1, Coverage: 1}

	long := base
	long.Note = registrasi.CommitteeNote{DiagnoseCode: strings.Repeat("X", 21), Remarks: strings.Repeat("r", 4001)}
	_, err := l.service.SaveCommitteeNote(ctx, long, l.caller)
	var v *registrasi.ValidationError
	require.ErrorAs(t, err, &v)
	require.Len(t, v.Violation, 2)

	missing := base
	missing.Coverage = 9
	_, err = l.service.SaveCommitteeNote(ctx, missing, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)

	_, err = l.service.SaveCommitteeNote(ctx, base, usecase.Caller{Identity: "ORANGLAIN"})
	require.ErrorIs(t, err, registrasi.ErrNotTaskOwner)
}

// Travel: Kirim Komite menolak bila Penerima Klaim belum dipilih atau data banknya belum lengkap
// (ValidationTypePayment step 9–12), lalu berhasil setelah lengkap.
func TestTransferCommitteeChecksTravelReceiver(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	_, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)

	k, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	k.Policy.Line = registrasi.LineTravel
	k.Receiver = []registrasi.Receiver{{ID: "1", Name: "Budi"}, {ID: "2", Name: "Ani", BankName: "BCA", AccountNo: "123"}}
	require.NoError(t, l.store.Save(ctx, k))

	messages := func(err error) []string {
		var v *registrasi.ValidationError
		require.ErrorAs(t, err, &v)
		var out []string
		for _, x := range v.Violation {
			out = append(out, x.Message)
		}
		return out
	}

	_, err = l.service.TransferCommittee(ctx, transfer(task), l.caller)
	require.Equal(t, []string{"Receiver Claim harus di isi"}, messages(err))

	incomplete := transfer(task)
	incomplete.ReceiverID = "1"
	_, err = l.service.TransferCommittee(ctx, incomplete, l.caller)
	require.Equal(t, []string{"Nama Bank Belum Di isi", "No Rekening Belum Di isi"}, messages(err))

	// Penerima tersimpan di isian modal (TEMPRECEIVER) dipakai bila permintaan tidak membawanya.
	_, err = l.service.SaveCommitteeNote(ctx, usecase.CommitteeNoteCommand{
		TaskID: task.ID, ClaimID: task.ClaimID, Object: 1, Coverage: 1, Note: registrasi.CommitteeNote{Receiver: "2"},
	}, l.caller)
	require.NoError(t, err)
	_, err = l.service.TransferCommittee(ctx, transfer(task), l.caller)
	require.NoError(t, err)
}
