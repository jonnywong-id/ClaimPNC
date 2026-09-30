package komite

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/money"
)

// Angka tangkapan layar Pega Komite (Claim Adjustment): Total Claim 97.107.499, estimasi
// 90.000.000, deductible "Lainnya" 1.000.000, nett 96.107.499, ASM 65% = 62.469.874,35.
func TestClaimAdjustmentMengikutiLayarPega(t *testing.T) {
	t.Parallel()
	d := TransferDetail{GroupPanel: "006"}
	l := AdjustmentLine{
		PaymentType: "2", CurrencyCode: "IDR",
		TotalClaim:      money.FromRupiah(97_107_499),
		Estimation:      money.FromRupiah(90_000_000),
		HasEstimation:   true,
		LOCPercent:      "0",
		RiskType:        "3",
		RiskPercent:     "0",
		IndividualRisk:  money.FromRupiah(1_000_000),
		GrossValue:      money.FromRupiah(96_107_499),
		ASMSharePercent: "65",
		ASMShareValue:   money.FromMinorUnits(6_246_987_435),
	}

	b := d.BreakdownFor(l)
	require.True(t, b.Known)
	require.Equal(t, "ADJUSTMENT", b.ValueHeader)
	require.Len(t, b.Rows, 9)

	desc := func(i int) string { return b.Rows[i].Description }
	require.Equal(t, "Total Claim", desc(0))
	require.Equal(t, money.FromRupiah(90_000_000), b.Rows[0].Estimate)
	require.Equal(t, "Lainnya", b.Rows[3].EstimateLabel)
	require.Equal(t, money.FromRupiah(96_107_499), b.Rows[4].Value, "Nett Of Deductible")
	require.False(t, b.Rows[5].HasValue, "Salvage B tanpa sisa dibiarkan kosong")
	require.True(t, b.Rows[6].HasValue, "Interim Payment tampil 0,00")
	require.Equal(t, money.FromRupiah(96_107_499), b.Rows[7].Value)
	require.True(t, b.Rows[8].Total)
	require.Equal(t, money.FromMinorUnits(6_246_987_435), b.Rows[8].Value)
}

func TestSalvageBAdalahSisaSetelahInterim(t *testing.T) {
	t.Parallel()
	d := TransferDetail{GroupPanel: "003"}
	l := AdjustmentLine{
		PaymentType: "1", TotalClaim: money.FromRupiah(1000),
		GrossValue: money.FromRupiah(700), InterimPaid: money.FromRupiah(200),
	}
	b := d.BreakdownFor(l)
	require.Equal(t, money.FromRupiah(100), b.Rows[5].Value, "1000 − 700 − interim 200")
	require.Equal(t, money.FromRupiah(200), b.Rows[6].Value)

	// Di luar Non-MBU interim tidak mengurangi gross (registrasi.ComputeSettlementLine).
	d.GroupPanel = "002"
	b = d.BreakdownFor(l)
	require.Equal(t, money.FromRupiah(300), b.Rows[5].Value)
	require.Equal(t, money.Money(0), b.Rows[6].Value)
}

func TestBentukTabelMenurutTypeDanTravel(t *testing.T) {
	t.Parallel()
	travel := TransferDetail{GroupPanel: "005"}
	b := travel.BreakdownFor(AdjustmentLine{PaymentType: "2"})
	require.Equal(t, "CLAIM ACCEPTED", b.ValueHeader)
	require.Len(t, b.Rows, 5)

	d := TransferDetail{}
	require.Equal(t, "Salvage", d.BreakdownFor(AdjustmentLine{PaymentType: "3"}).Title)
	require.Equal(t, "Adjuster Fee", d.BreakdownFor(AdjustmentLine{PaymentType: "4"}).Title)
	require.False(t, d.BreakdownFor(AdjustmentLine{PaymentType: "6"}).Known)
}

func TestStatusKomiteSepertiPega(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Setuju", CommitteeEntry{Status: "1"}.StatusLabel())
	require.Equal(t, "Tidak Setuju", CommitteeEntry{Status: "2"}.StatusLabel())
	require.Equal(t, "Menunggu", CommitteeEntry{Status: "0"}.StatusLabel())
	require.Equal(t, "Menunggu", CommitteeEntry{}.StatusLabel())
}
