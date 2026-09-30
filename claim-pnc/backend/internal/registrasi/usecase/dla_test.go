package usecase_test

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// acceptedForDLA membawa adjustment 1 sampai diakseptasi dengan persetujuan LOD, lalu
// menyiapkan CoinsList polis dengan Sinar Mas sebagai leader.
func (l environment) acceptedForDLA(t *testing.T) (registrasi.Task, registrasi.Claim) {
	t.Helper()
	task, _ := l.approvedClaim(t)
	claim, err := l.service.AcceptSettlement(context.Background(), acceptance(task, registrasi.LODAgreed), l.caller)
	require.NoError(t, err)
	l.dla.Policies[claim.Policy.Number] = registrasi.DLAPolicy{Coins: []registrasi.PLACoinsMember{
		{ID: "1", Name: "PT ASURANSI SINAR MAS", Leader: true, Share: registrasi.Percent(70 * 10_000), HasShare: true},
		{ID: "77", Name: "KOASURADUR UJI", Share: registrasi.Percent(30 * 10_000), HasShare: true},
	}}
	return task, claim
}

func dlaCommand(task registrasi.Task) usecase.DLACommand {
	return usecase.DLACommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Adjustment: 1}
}

// Membuka dialog menerbitkan DLA koasuransi sekali; membuka lagi tidak menerbitkan ulang.
func TestListDLAIssuesCoinsOnce(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, claim := l.acceptedForDLA(t)
	line := claim.InsuredItem[0].Coverage[0].Settlement[0]

	list, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 1, list.Issued)
	require.Len(t, list.DLA, 1)
	d := list.DLA[0]
	require.Equal(t, registrasi.DLATypeCoins, d.Type)
	require.Equal(t, "77", d.RecipientCode)
	require.True(t, strings.HasPrefix(d.Number, "J26"), d.Number)
	require.Equal(t, line.AcceptedNo, d.AcceptedNo)
	// round2(GrossValue × 30 / 100), klaim = GrossValue.
	gross := big.NewRat(int64(line.Gross), 100)
	require.Equal(t, new(big.Rat).Mul(gross, big.NewRat(30, 100)).FloatString(2), d.Value)
	require.Equal(t, gross.FloatString(2), d.ClaimAmount)

	again, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller)
	require.NoError(t, err)
	require.Equal(t, 0, again.Issued)
	require.Len(t, again.DLA, 1)
}

// PRINT pertama menyimpan REMARKS + tipe pembayaran dan menandai ISDLA; hasilnya PDF.
func TestPrintDLAMarksPrinted(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, _ := l.acceptedForDLA(t)
	list, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller)
	require.NoError(t, err)
	number := list.DLA[0].Number

	cmd := dlaCommand(task)
	cmd.Number = number
	cmd.Remarks = map[string]string{number: "Catatan uji"}
	result, err := l.service.PrintDLA(ctx, cmd, l.caller)
	require.NoError(t, err)
	require.Equal(t, "application/pdf", result.ContentType)
	require.True(t, strings.HasPrefix(string(result.Content), "%PDF"))
	require.Equal(t, "DLACOINS"+number+".pdf", result.FileName)

	after, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller)
	require.NoError(t, err)
	require.True(t, after.DLA[0].Printed)
	require.True(t, strings.HasPrefix(after.DLA[0].Note, "Catatan uji\n- "), after.DLA[0].Note)
}

// Klaim ex gratia non-ASO tidak mendapat DLA (GenerateDLAList langkah 14).
func TestListDLASkipsExGratia(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task, claim := l.acceptedForDLA(t)
	claim, err := l.store.Get(ctx, claim.ID)
	require.NoError(t, err)
	claim.ExGratia = true
	require.NoError(t, l.store.Save(ctx, claim))

	list, err := l.service.ListDLA(ctx, dlaCommand(task), l.caller)
	require.NoError(t, err)
	require.True(t, list.ExGratia)
	require.Empty(t, list.DLA)
	require.Empty(t, l.dla.Saved)
}

// Baris yang belum disetujui LOD tidak dapat membuka Print DLA.
func TestListDLARequiresLODAgreement(t *testing.T) {
	l := setup(t)
	task, _ := l.approvedClaim(t)
	_, err := l.service.ListDLA(context.Background(), dlaCommand(task), l.caller)
	violation(t, err, registrasi.ViolationDLANotAllowed)
}
