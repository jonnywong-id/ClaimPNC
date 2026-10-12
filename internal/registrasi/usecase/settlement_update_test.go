package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Perubahan isian baris yang sudah ada dihitung, diperiksa, lalu disimpan (SetNilaiResikoSendiri);
// pelanggaran tidak menyimpan apa pun; baris yang sudah diproses tidak dapat diubah.
func TestUpdateSettlement(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)

	_, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}), l.caller)
	require.NoError(t, err)

	cmd := addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(20_000_000), Submitted: registrasi.Rupiah(20_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	})
	cmd.Adjustment = 1
	claim, err := l.service.UpdateSettlement(ctx, cmd, l.caller)
	require.NoError(t, err)
	lines := claim.InsuredItem[0].Coverage[0].Settlement
	require.Len(t, lines, 1)
	require.Equal(t, registrasi.Rupiah(18_000_000), lines[0].Gross)

	// Pelanggaran: tidak tersimpan.
	bad := cmd
	bad.Input.Propose = 0
	_, err = l.service.UpdateSettlement(ctx, bad, l.caller)
	violation(t, err, registrasi.ViolationSettlementPropose)
	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Equal(t, registrasi.Rupiah(18_000_000), stored.InsuredItem[0].Coverage[0].Settlement[0].Gross)

	// Baris yang tidak ada.
	missing := cmd
	missing.Adjustment = 2
	_, err = l.service.UpdateSettlement(ctx, missing, l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}
