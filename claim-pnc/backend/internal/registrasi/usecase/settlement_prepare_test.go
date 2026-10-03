package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Tambah pertama pada jaminan PA tanpa adjustment menjalankan NewEstimationPA: satu estimasi klaim
// sebesar TSI, IDR, kurs 1. Estimasi itu membuat Claim Face Sheet PA dapat dibuat.
func TestPrepareSettlementPAAddsEstimate(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToEstimationPA(t)
	cmd := usecase.SettlementCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}

	claim, err := l.service.PrepareSettlement(ctx, cmd, l.caller)
	require.NoError(t, err)
	cov := claim.InsuredItem[0].Coverage[0]
	require.True(t, registrasi.HasUnprintedEstimate(cov))
	last := cov.Item[len(cov.Item)-1].Estimation[0]
	require.Equal(t, registrasi.EstimateClaim, last.Type)
	require.Equal(t, registrasi.CurrencyIDR, last.Currency)
	require.Equal(t, cov.TSI, last.Value)

	// Tambah berulang tanpa simpan tidak menumpuk estimasi.
	again, err := l.service.PrepareSettlement(ctx, cmd, l.caller)
	require.NoError(t, err)
	require.Len(t, again.InsuredItem[0].Coverage[0].Item, len(cov.Item))

	// Claim Face Sheet kini dapat dibuat.
	_, err = l.service.DownloadFaceSheet(ctx, usecase.FaceSheetCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}, l.caller)
	require.NoError(t, err)
}

// Lini selain PA: Tambah tidak mengubah apa pun.
func TestPrepareSettlementNonPAUnchanged(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)
	before, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)

	claim, err := l.service.PrepareSettlement(ctx, usecase.SettlementCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}, l.caller)
	require.NoError(t, err)
	require.Len(t, claim.InsuredItem[0].Coverage[0].Item, len(before.InsuredItem[0].Coverage[0].Item))
}
