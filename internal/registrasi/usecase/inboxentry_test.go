package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// TestInboxEntryFollowsClaim menjaga baris T_CLAIMLIST_ADMIN klaim PNCN.
//
// Tanpa baris itu klaim yang dibuka aplikasi ini tidak pernah muncul di My Inbox — layarnya
// tampil rapi dan kosong, tanpa satu pun galat. Itulah keadaan sebelum seam ini ada.
func TestInboxEntryFollowsClaim(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)

	e, ok := l.inbox.Get(start.Claim.Number)
	require.True(t, ok, "klaim yang baru dibuka harus langsung punya baris daftar kerja")
	require.Equal(t, start.Claim.Number, e.Key())
	require.Equal(t, registrasi.WorkStatusNew, e.WorkStatus())
	require.Equal(t, "View Polis", e.StageName)
	require.Equal(t, testOperator, e.AssignedOperator(),
		"PXASSIGNEDOPERATORID adalah penyaring My Inbox; tanpa pemilik barisnya tidak terlihat")
	require.Equal(t, "1", e.Active())

	result, err := l.service.CompleteStage(ctx, usecase.CompleteCommand{
		TaskID: start.Task.ID,
		Action: registrasi.ActionViewPolicy,
	}, l.caller)
	require.NoError(t, err)

	e, ok = l.inbox.Get(start.Claim.Number)
	require.True(t, ok)
	require.Equal(t, "Input Register", e.StageName,
		"baris harus berpindah tahap bersama tugasnya, bukan tertinggal di tahap lama")
	require.NotNil(t, e.Task)
	require.Equal(t, result.NextTask.ID, e.Task.ID)
}

func TestInboxEntryStatusFollowsProcess(t *testing.T) {
	cases := map[registrasi.ProcessStatus]string{
		registrasi.ProcessRunning:  registrasi.WorkStatusNew,
		registrasi.ProcessDone:     registrasi.WorkStatusCompleted,
		registrasi.ProcessRejected: registrasi.WorkStatusRejected,
	}
	for process, want := range cases {
		e := registrasi.InboxEntry{Claim: registrasi.Claim{Number: "PNCN.26.0001", ProcessStatus: process}}
		require.Equal(t, want, e.WorkStatus(), "status proses %s", process)
		require.Empty(t, e.AssignedOperator(), "klaim tanpa tugas terbuka tidak bertuan")
	}
}

// My Inbox membuka klaim lewat nomornya. Klaim lama yang pengenalnya acak tetap terbuka.
func TestViewClaimFindsClaimByNumber(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	require.NoError(t, l.store.Save(ctx, registrasi.Claim{
		ID: "pengenal-acak-1", Number: "PNCN.26.9999", Portal: "ASM",
		ProcessStatus: registrasi.ProcessRunning,
	}))

	summary, err := l.service.ViewClaim(ctx, "PNCN.26.9999", l.caller)
	require.NoError(t, err)
	require.Equal(t, "pengenal-acak-1", summary.Claim.ID)

	_, err = l.service.ViewClaim(ctx, "PNCN.26.0000", l.caller)
	require.ErrorIs(t, err, registrasi.ErrClaimNotFound)
}
