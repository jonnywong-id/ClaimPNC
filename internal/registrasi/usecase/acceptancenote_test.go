package usecase_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// PRINT Draft Persetujuan: PDF bernama seperti Pega, jejak audit tertulis, tidak ada kolom diubah.
func TestPrintAcceptanceNote(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, claim := l.acceptedForDLA(t)
	line := claim.InsuredItem[0].Coverage[0].Settlement[0]
	before := len(l.store.AuditTrail())

	result, err := l.service.PrintAcceptanceNote(ctx, usecase.AcceptanceNoteCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "DraftPersetujuan_NO."+line.AcceptedNo+".pdf", result.FileName)
	require.True(t, bytes.HasPrefix(result.Content, []byte("%PDF")))
	trail := l.store.AuditTrail()
	require.Len(t, trail, before+1)
	require.Equal(t, "DRAFT_AKSEPTASI_CETAK", trail[len(trail)-1].Event)
}

// Adjustment tanpa Nomor Akseptasi tidak dapat dicetak.
func TestPrintAcceptanceNoteRequiresAcceptedNo(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	_, err := l.service.PrintAcceptanceNote(context.Background(), usecase.AcceptanceNoteCommand{
		ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1,
	}, l.caller)
	violation(t, err, registrasi.ViolationAcceptanceNoteNotAllowed)
}
