package usecase_test

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

func printPLA(task registrasi.Task) usecase.PLACommand {
	return usecase.PLACommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1}
}

func coinsLeader(l environment) {
	l.pla.Coins[firePolicy] = []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Leader: true, Share: 700_000, HasShare: true},
		{ID: "C1", Name: "ANGGOTA SATU", Share: 200_000, HasShare: true},
		{ID: "C2", Name: "ANGGOTA DUA", Share: 100_000, HasShare: true},
		{ID: "C3", Name: "ANGGOTA TERHAPUS", Share: 50_000, HasShare: true, Deleted: true},
		{ID: "C4", Name: "ANGGOTA NOL", Share: 0, HasShare: true},
	}
}

// Print PLA menunggu Claim Face Sheet (`!isCFS`).
func TestPLANeedsFaceSheet(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(context.Background(), oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)

	_, err = l.service.PrintPLA(context.Background(), printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANeedsFaceSheet)
}

// Tanpa CoinsList (`IsNoCoins`) dan bila Sinar Mas bukan leader, tidak ada PLA.
func TestPLARequiresCoinsuranceLedByUs(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	_, err = l.service.PrintPLA(ctx, printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANoCoins)

	l.pla.Coins[firePolicy] = []registrasi.PLACoinsMember{
		{ID: "C0", Name: "ASURANSI SINAR MAS", Share: 300_000, HasShare: true},
		{ID: "C1", Name: "LEADER LAIN", Leader: true, Share: 700_000, HasShare: true},
	}
	_, err = l.service.PrintPLA(ctx, printPLA(task), l.caller)
	violation(t, err, registrasi.ViolationPLANotLeader)
}

// Satu PLA per anggota (bukan Sinar Mas, bukan terhapus, share > 0); dicetak ulang tanpa
// nomor baru; revisi CFS berikutnya menerbitkan PLA baru dengan catatan rujukan.
func TestPLAIssuesOnePerMemberAndReprints(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	first, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, first.Issued)
	require.Equal(t, "application/zip", first.ContentType)
	zr, err := zip.NewReader(bytes.NewReader(first.Content), int64(len(first.Content)))
	require.NoError(t, err)
	require.Len(t, zr.File, 2)
	require.True(t, strings.HasPrefix(zr.File[0].Name, "PLACOINSJ261"))

	require.Len(t, l.pla.Saved, 2)
	one := l.pla.Saved[0]
	require.Equal(t, "ANGGOTA SATU", one.Recipient)
	require.Equal(t, 0, one.Revision)
	require.Equal(t, registrasi.Rupiah(50_000_000), one.Amount[0].Base)
	require.Equal(t, registrasi.Rupiah(10_000_000), one.Amount[0].Result)
	require.Equal(t, registrasi.Rupiah(5_000_000), l.pla.Saved[1].Amount[0].Result)
	require.Contains(t, one.Note, "Estimation only")

	again, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 0, again.Issued)
	require.Len(t, l.pla.Saved, 2)

	_, err = l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 2), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	next, err := l.service.PrintPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, next.Issued)
	revised := l.pla.Saved[2]
	require.Equal(t, 1, revised.Revision)
	require.Equal(t, registrasi.Rupiah(100_000_000), revised.Amount[0].Reserve)
	require.Contains(t, revised.Note, "Please see our PLA No.:"+one.Number)
}

// Layar PrintPLA_dtl: daftar menerbitkan sekali, REMARKS tersimpan, Print PLA per nomor
// mengembalikan satu PDF.
func TestPLAListNotesAndSinglePrint(t *testing.T) {
	l := setup(t)
	coinsLeader(l)
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	_, err := l.service.SaveEstimate(ctx, oneEstimate(task.ID, registrasi.Rupiah(50_000_000), 1), l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)

	list, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 2, list.Issued)
	again, err := l.service.ListPLA(ctx, printPLA(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 0, again.Issued)
	require.Len(t, again.PLA, 2)

	number := list.PLA[0].Number
	saved, err := l.service.SavePLANotes(ctx, printPLA(task), map[string]string{number: "Catatan baru"}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "Catatan baru", saved.PLA[0].Note)
	_, err = l.service.SavePLANotes(ctx, printPLA(task), map[string]string{"BUKAN-MILIK": "x"}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)

	command := printPLA(task)
	command.Number = number
	one, err := l.service.PrintPLA(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, "application/pdf", one.ContentType)
	require.Equal(t, "PLACOINS"+number+".pdf", one.FileName)
}
