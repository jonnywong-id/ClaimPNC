package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

func faceSheet(task registrasi.Task) usecase.FaceSheetCommand {
	return usecase.FaceSheetCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}
}

func violation(t *testing.T, err error, code registrasi.ViolationCode) {
	t.Helper()
	var validation *registrasi.ValidationError
	require.True(t, errors.As(err, &validation), "galat = %v", err)
	require.True(t, validation.Has(code), "galat = %v", err)
}

// Tombol menghasilkan PDF bernama seperti Pega, dan revisinya bertambah.
func TestFaceSheetDownloadsPDFAndCountsRevisions(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	first, err := l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "Revisi0.pdf", first.FileName)
	require.True(t, bytes.HasPrefix(first.Content, []byte("%PDF")))

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 2), l.caller)
	require.NoError(t, err)
	second, err := l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, "Revisi1.pdf", second.FileName)
}

// Estimasi yang sudah dibuatkan CFS terkunci: isian layar untuknya diabaikan.
func TestFaceSheetLocksTheEstimate(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(10), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	claim, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(99), 1), l.caller)
	require.NoError(t, err)
	e := claim.InsuredItem[0].Coverage[0].Item[0].Estimation[0]
	require.True(t, e.FaceSheet)
	require.Equal(t, registrasi.Rupiah(10), e.Value)
}

// ValidateInputEstimate_act: estimasi berikutnya menunggu Claim Face Sheet.
func TestNextEstimateWaitsForFaceSheet(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(10), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(10), 2), l.caller)
	violation(t, err, registrasi.ViolationFaceSheetMissing)
}

// Tanpa estimasi baru, tidak ada yang dibuatkan CFS.
func TestFaceSheetNeedsANewEstimate(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)

	_, err := l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	violation(t, err, registrasi.ViolationFaceSheetNothingNew)

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(10), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	violation(t, err, registrasi.ViolationFaceSheetNothingNew)
}

// Tugas satu klaim tidak dapat dipakai mencetak CFS klaim lain.
func TestFaceSheetRejectsForeignClaim(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)
	command := faceSheet(task)
	command.ClaimID = "klaim-lain"
	_, err := l.service.DownloadFaceSheet(context.Background(), command, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}
