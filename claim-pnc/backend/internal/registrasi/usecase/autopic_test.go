package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// parkAtServicePNC meniru klaim yang tidak mendapat PIC: PIC Teknik `-` dan tugasnya di
// antrean ServicePNC.
func (l environment) parkAtServicePNC(t *testing.T) (registrasi.Claim, registrasi.Task) {
	t.Helper()
	ctx := context.Background()
	claim, task := l.toChooseSurveyorWithPIC(t, "")
	claim.TechnicalPIC = "-"
	require.NoError(t, l.store.Save(ctx, claim))
	task.Owner = registrasi.OperatorUnassigned
	require.NoError(t, l.store.SaveTask(ctx, task))
	return claim, task
}

// TransferAllCaseNotAssigned: PIC diisi, tugas tahap teknis dipindah ke PIC itu, dan putaran
// berikutnya tidak memproses klaim yang sama lagi.
func TestAutoPICAssignsAndMovesTask(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, task := l.parkAtServicePNC(t)

	result, err := l.service.AssignUnassignedTechnicalPIC(ctx)
	require.NoError(t, err)
	require.Equal(t, usecase.AutoPICResult{Examined: 1, Assigned: 1, Moved: 1}, result)

	stored, err := l.store.Get(ctx, claim.ID)
	require.NoError(t, err)
	require.Equal(t, testOperator, stored.TechnicalPIC)
	require.Equal(t, usecase.AutoPICAgent, stored.UpdatedBy)
	moved, err := l.store.ClaimTask(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, testOperator, moved.Owner)

	again, err := l.service.AssignUnassignedTechnicalPIC(ctx)
	require.NoError(t, err)
	require.Equal(t, usecase.AutoPICResult{}, again)
}

// Klaim yang sudah ber-PIC tidak diproses agent, walau tugasnya masih di ServicePNC.
func TestAutoPICIgnoresClaimWithPIC(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	claim, _ := l.parkAtServicePNC(t)
	claim.TechnicalPIC = "PICLAIN"
	require.NoError(t, l.store.Save(ctx, claim))

	result, err := l.service.AssignUnassignedTechnicalPIC(ctx)
	require.NoError(t, err)
	require.Equal(t, usecase.AutoPICResult{}, result, "klaim ber-PIC tidak masuk daftar")
}
