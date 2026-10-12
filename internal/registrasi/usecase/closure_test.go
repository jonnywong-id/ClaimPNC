package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// closableTask membawa klaim ke InputSurveyor dengan pemanggil sebagai PIC Teknisnya.
func (l environment) closableTask(t *testing.T) registrasi.Task {
	t.Helper()
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(5_000_000), 0)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.TechnicalPIC = l.caller.Identity
	require.NoError(t, l.store.Save(ctx, claim))
	return task
}

func closure(note string) registrasi.Closure {
	return registrasi.Closure{Note: note, Proposal: "Usulan", Effort: "Effort", Obstacle: "Kendala"}
}

// Tutup permanen: Status Klaim 1143, alur selesai, tugas ditutup, dan akibat di tabel warisan.
func TestCloseClaimPermanently(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.closableTask(t)

	result, err := l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("  Selesai.  ")}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StatusClosed, result.Claim.ClaimStatus)
	require.Equal(t, registrasi.ProcessDone, result.Claim.ProcessStatus)
	require.Empty(t, result.Claim.CurrentStage)
	require.Nil(t, result.NextTask)

	saved := l.records.Closure.Saved[task.ClaimID]
	require.Equal(t, "Selesai.", saved.Note)
	require.False(t, saved.Temporary)
	require.False(t, l.records.Closure.ClosedAt[task.ClaimID].IsZero())
	require.Equal(t, []string{l.caller.Identity}, l.records.Closure.ReleasedPIC)
	require.Len(t, l.records.Closure.DashboardClosed, 1)
	require.Equal(t, registrasi.ClosureActionClose, l.records.Closure.Logs[0].Action)

	trail := l.store.AuditTrail()
	require.Equal(t, "TUTUP_KLAIM", trail[len(trail)-1].Event)

	_, err = l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("lagi")}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyDone)
}

// Tutup sementara: penanda dan log saja — status klaim dan tugas tidak disentuh.
func TestCloseClaimTemporarily(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.closableTask(t)
	c := closure("Menunggu dokumen.")
	c.Temporary = true

	result, err := l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: c}, l.caller)
	require.NoError(t, err)
	require.True(t, result.Claim.PendingClose)
	require.Equal(t, registrasi.StageChooseSurveyor, result.Claim.CurrentStage)
	require.NotEqual(t, registrasi.StatusClosed, result.Claim.ClaimStatus)
	require.Empty(t, l.records.Closure.ReleasedPIC)
	require.Equal(t, registrasi.ClosureActionTemporary, l.records.Closure.Logs[0].Action)

	// Layar klaim membaca penandanya; tombol tidak dapat dipakai lagi.
	view, err := l.service.ViewClaim(ctx, task.ClaimID, l.caller)
	require.NoError(t, err)
	require.True(t, view.Claim.PendingClose)
	_, err = l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("x")}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}

func violationMessage(t *testing.T, err error) string {
	t.Helper()
	v, ok := err.(*registrasi.ValidationError)
	require.True(t, ok, "galat harus ValidationError, bukan %T: %v", err, err)
	require.Len(t, v.Violation, 1)
	return v.Violation[0].Message
}

func TestCloseClaimRequiresNote(t *testing.T) {
	l := setup(t)
	task := l.closableTask(t)
	_, err := l.service.CloseClaim(context.Background(), usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("  ")}, l.caller)
	require.Equal(t, registrasi.MsgClosureNote, violationMessage(t, err))
	require.Empty(t, l.records.Closure.Logs)
}

// Hanya PIC Teknis klaim, atau identitas lamanya (T_ACCESS_GROUP_PNC).
func TestCloseClaimOnlyByTechnicalPIC(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.closableTask(t)
	claim, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	claim.TechnicalPIC = "PIC-LAMA"
	require.NoError(t, l.store.Save(ctx, claim))

	_, err = l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("tutup")}, l.caller)
	require.Equal(t, registrasi.MsgClosureNotPIC, violationMessage(t, err))

	l.records.Closure.LegacyOperators = map[string]string{l.caller.Identity: "PIC-LAMA"}
	_, err = l.service.CloseClaim(ctx, usecase.CloseClaimCommand{TaskID: task.ID, Closure: closure("tutup")}, l.caller)
	require.NoError(t, err)
}

func TestValidateClosure(t *testing.T) {
	k := registrasi.Claim{Portal: "ASM", TechnicalPIC: "PIC", Policy: registrasi.Policy{Line: registrasi.LineFire}}
	c := registrasi.Closure{Note: "n"}
	require.Nil(t, registrasi.ValidateClosure(k, c, "pic", ""))

	// Salvage TBA menahan, kecuali tutup sementara atau Travel/PA.
	k.SalvageStatus = registrasi.SalvageStatusTBA
	require.Equal(t, registrasi.MsgClosureSalvageTBA, registrasi.ValidateClosure(k, c, "PIC", "").Message)
	c.Temporary = true
	require.Nil(t, registrasi.ValidateClosure(k, c, "PIC", ""))
	c.Temporary = false
	k.Policy.Line = registrasi.LineTravel
	require.Nil(t, registrasi.ValidateClosure(k, c, "PIC", ""))
	k.Policy.Line = registrasi.LineFire
	k.SalvageStatus = ""

	// Adjustment yang masih di komite (AcceptanceStatus 0).
	k.InsuredItem = []registrasi.InsuredItem{{Coverage: []registrasi.Coverage{{
		Settlement: []registrasi.SettlementLine{{AcceptanceStatus: registrasi.DecisionPending}},
	}}}}
	require.Equal(t, registrasi.MsgClosureCommittee, registrasi.ValidateClosure(k, c, "PIC", "").Message)
}
