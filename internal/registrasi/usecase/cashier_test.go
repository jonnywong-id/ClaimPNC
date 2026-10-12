package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

func cashierCommand(task registrasi.Task) usecase.CashierCommand {
	return usecase.CashierCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1}
}

// readyForCashier membawa adjustment 1 sampai diakseptasi dengan penerima berekening lengkap.
func (l environment) readyForCashier(t *testing.T) registrasi.Task {
	t.Helper()
	task, _ := l.acceptedForDLA(t)
	_, err := l.service.SaveReceiver(context.Background(), usecase.ReceiverCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, ReceiverID: "1", AccountNo: "1234567890", Email: "a@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	l.cashier.Banks["BANK CONTOH"] = "001"
	return task
}

// Berhasil: muatan terkirim, log ditulis, adjustment ditandai, Status Klaim 1162.
func TestTransferCashierMarksAdjustment(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.readyForCashier(t)
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}

	preview, err := l.service.PreviewCashierTransfer(ctx, cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Empty(t, preview.Problem)
	require.Equal(t, "PT CONTOH PENERIMA", preview.Receiver.Name)

	claim, err := l.service.TransferCashier(ctx, cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StatusClaimTransferCashier, claim.ClaimStatus)
	require.Len(t, l.cashier.Payloads, 1)
	p := l.cashier.Payloads[0].TAllPaymentData[0]
	require.Equal(t, claim.InsuredItem[0].Coverage[0].Settlement[0].AcceptedNo, p.NoTrans)
	require.Equal(t, "1234567890", p.AccountNo)
	require.Equal(t, "001", p.LbgID)
	require.Equal(t, "D0031", p.LjtdId)
	require.Equal(t, []string{registrasi.CashierServicePaid}, l.cashier.Services)
	require.Len(t, l.cashier.Logs, 1)
	require.Equal(t, "Sedang Transfer Kasir", l.cashier.Logs[0].Reason)
	require.Len(t, l.cashier.Marked, 1)
	require.Equal(t, "ECR-1", l.cashier.Marked[0].CaseID)
}

// Kasir tanpa CaseIDCashier: tidak ada yang ditandai; pesannya diteruskan.
func TestTransferCashierRejected(t *testing.T) {
	l := setup(t)
	task := l.readyForCashier(t)
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "Rekening diblokir"}
	_, err := l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	var rejected *usecase.CashierRejectedError
	require.True(t, errors.As(err, &rejected), err)
	require.Equal(t, "Rekening diblokir", rejected.Message)
	require.Empty(t, l.cashier.Marked)
	require.Empty(t, l.cashier.Logs)
}

// Kasir tidak menjawab: galat layanan, tidak ada yang ditandai.
func TestTransferCashierUnavailable(t *testing.T) {
	l := setup(t)
	task := l.readyForCashier(t)
	l.cashier.Err = errors.New("timeout")
	_, err := l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	require.ErrorIs(t, err, usecase.ErrCashierUnavailable)
	require.Empty(t, l.cashier.Marked)
}

// Validasi Pega berhenti pada galat pertama; di sini email rekening master kosong.
func TestTransferCashierRequiresEmail(t *testing.T) {
	l := setup(t)
	task, _ := l.acceptedForDLA(t) // rekening 9876543210 tanpa email
	l.cashier.Banks["BANK CONTOH"] = "001"
	preview, err := l.service.PreviewCashierTransfer(context.Background(), cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "Harap Isi Email Terlebih Dahulu", preview.Problem)
	_, err = l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	violation(t, err, registrasi.ViolationCashierIncomplete)
	require.Empty(t, l.cashier.Payloads)
}

// DLA yang belum dicetak menahan transfer.
func TestTransferCashierRequiresPrintedDLA(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.readyForCashier(t)
	_, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller) // menerbitkan DLA COINS, belum dicetak
	require.NoError(t, err)
	_, err = l.service.TransferCashier(ctx, cashierCommand(task), l.caller)
	violation(t, err, registrasi.ViolationCashierNeedsDLA)
}

// Adjustment yang belum diakseptasi tidak dapat ditransfer.
func TestTransferCashierRequiresAcceptance(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	_, err := l.service.PreviewCashierTransfer(context.Background(), cashierCommand(task), l.caller)
	violation(t, err, registrasi.ViolationCashierNotAllowed)
}

// Setiap pemanggilan Kasir dicatat ke CLAIM_SERVICE_LOG — jawaban utuh sebagai JSONOUT.
func TestTransferCashierWritesServiceLog(t *testing.T) {
	l := setup(t)
	task := l.readyForCashier(t)
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1", Body: `{"ResponseMessage":"SUCCESS"}`}
	claim, err := l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Len(t, l.cashier.ServiceLogs, 1)
	log := l.cashier.ServiceLogs[0]
	require.Equal(t, claim.Number, log.ClaimNumber)
	require.Equal(t, claim.InsuredItem[0].Coverage[0].Settlement[0].AcceptedNo, log.AcceptedNo)
	require.Equal(t, `{"ResponseMessage":"SUCCESS"}`, log.Response)
	require.Len(t, log.Request.TAllPaymentData, 1)
}

// Kasir tidak menjawab: tetap dicatat, JSONOUT berisi galatnya.
func TestTransferCashierLogsUnavailable(t *testing.T) {
	l := setup(t)
	task := l.readyForCashier(t)
	l.cashier.Err = errors.New("timeout")
	_, err := l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	require.ErrorIs(t, err, usecase.ErrCashierUnavailable)
	require.Len(t, l.cashier.ServiceLogs, 1)
	require.Contains(t, l.cashier.ServiceLogs[0].Response, "timeout")
}

// Log yang gagal ditulis TIDAK membatalkan transfer yang sudah diterima Kasir.
func TestTransferCashierSurvivesServiceLogFailure(t *testing.T) {
	l := setup(t)
	task := l.readyForCashier(t)
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}
	l.cashier.ServiceLogErr = errors.New("ORA-00001")
	_, err := l.service.TransferCashier(context.Background(), cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Len(t, l.cashier.Marked, 1)
}

// Dialog menampilkan DLA FAC OUT; Fronting dengan satu centang menambah satu baris negatif.
func TestTransferCashierUnpaidFacOut(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.readyForCashier(t)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	l.dla.Saved = append(l.dla.Saved, registrasi.DLA{
		ClaimID: claim.ID, ObjectID: claim.InsuredItem[0].ID, CoverageSeq: 1, AdjustmentSeq: 1,
		Number: "H261000000000000001", Type: registrasi.DLATypeFacOut, Recipient: "REAS UJI", Value: "2500.00", Printed: true,
	})
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-1"}

	preview, err := l.service.PreviewCashierTransfer(ctx, cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Len(t, preview.FacOut, 1)

	cmd := cashierCommand(task)
	cmd.TransferType, cmd.UnpaidFacOut = registrasi.CashierTransferFronting, []string{"H261000000000000001"}
	_, err = l.service.TransferCashier(ctx, cmd, l.caller)
	require.NoError(t, err)
	rows := l.cashier.Payloads[0].TAllPaymentData
	require.Len(t, rows, 2)
	require.Equal(t, "H261000000000000001", rows[1].NoTrans)
	require.Equal(t, "-2500.00", string(rows[1].Nett))
}
