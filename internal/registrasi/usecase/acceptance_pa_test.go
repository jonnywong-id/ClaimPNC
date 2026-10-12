package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// approvedPAClaim adalah approvedClaim pada lini Personal Accident, dengan rekening penerima terdaftar.
func (l environment) approvedPAClaim(t *testing.T) registrasi.Task {
	t.Helper()
	ctx := context.Background()
	task, claim := l.approvedClaim(t)
	claim.Policy.Line = registrasi.LinePersonalAccident
	require.NoError(t, l.store.Save(ctx, claim))
	_, err := l.service.SaveReceiver(ctx, usecase.ReceiverCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, ReceiverID: "1", AccountNo: "1234567890", Email: "a@contoh.internal",
	}, l.caller)
	require.NoError(t, err)
	l.cashier.Banks["BANK CONTOH"] = "001"
	return task
}

// PA dapat diakseptasi, tetapi Transfer Kasir TIDAK berjalan otomatis (keputusan Work Owner 2026-10-03,
// berbeda dengan Pega langkah 113): baris belum ditransfer sampai tombol Transfer Kasir ditekan.
func TestAcceptSettlementPADoesNotTransferCashier(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.approvedPAClaim(t)
	l.cashier.Reply = registrasi.CashierReply{ResponseMessage: "SUCCESS", CaseIDCashier: "ECR-PA"}

	claim, err := l.service.AcceptSettlement(ctx, acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StatusClaimAccepted, claim.ClaimStatus)
	require.Empty(t, l.cashier.Payloads)
	require.Empty(t, l.cashier.Marked)

	// Transfer manual lewat tombol tetap berjalan.
	transferred, err := l.service.TransferCashier(ctx, cashierCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StatusClaimTransferCashier, transferred.ClaimStatus)
	require.Len(t, l.cashier.Marked, 1)
}
