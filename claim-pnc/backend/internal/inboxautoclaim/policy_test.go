package inboxautoclaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

func TestPremiBelumLunasMengikutiKondisiPega(t *testing.T) {
	// `AgingAmount=="" || @toDecimal(AgingAmount) > 1` (InsertKlaimToTable_Kredit :8382).
	for aging, unpaid := range map[string]bool{
		"":     true,
		"0":    false,
		"1":    false, // TEPAT 1 masih lunas — pembandingnya "lebih dari"
		"1.00": false,
		"1.01": true,
		"30":   true,
		"abc":  true, // tidak dapat dipastikan -> tidak diloloskan
	} {
		require.Equal(t, unpaid, inboxautoclaim.PremiumAnswer{AgingAmount: aging}.Unpaid(),
			"AgingAmount %q", aging)
	}
}

func TestPeriodePolisMencakupHariPertamaDanTerakhir(t *testing.T) {
	polis := inboxautoclaim.PolicyDetail{StartDate: "01/01/2026", EndDate: "31/12/2026"}

	require.True(t, polis.Covers("01/01/2026"))
	require.True(t, polis.Covers("31/12/2026"))
	require.True(t, polis.Covers("15/06/2026"))
	require.False(t, polis.Covers("31/12/2025"))
	require.False(t, polis.Covers("01/01/2027"))

	// Periode yang tidak terbaca tidak menggagalkan baris.
	require.True(t, inboxautoclaim.PolicyDetail{}.Covers("01/01/2026"))
}

func TestPolisBatalButuhKeduaPenanda(t *testing.T) {
	require.True(t, inboxautoclaim.PolicyDetail{StatusBusiness: "3", FlagEdmBatal: "1"}.Cancelled())
	require.False(t, inboxautoclaim.PolicyDetail{StatusBusiness: "3"}.Cancelled())
	require.False(t, inboxautoclaim.PolicyDetail{FlagEdmBatal: "1"}.Cancelled())
}
