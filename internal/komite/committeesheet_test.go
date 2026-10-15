package komite

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/money"
)

// Kasus nyata KMTN-00001 (klaim PNCN, Marine Cargo): gross 2.500.000, share ASM 65%,
// nilai adjustment 1.625.000, spreading ORS 100%, CoinsList dua baris dengan ASM sebagai
// leader. Angkanya dihitung ulang dengan tangan dari rumus `CalculatedSpredingForClaimKomite`.
func sampleSheet() (TransferDetail, AdjustmentLine) {
	line := AdjustmentLine{
		ObjectID: "1", CoverageID: "1", PaymentType: "2", CurrencyCode: "IDR",
		GrossValue:    money.FromRupiah(2_500_000),
		ASMShareValue: money.FromRupiah(1_625_000),
		SalvageValue:  money.FromRupiah(40_000),
		AdjusterFee:   money.FromRupiah(300_000),
	}
	d := TransferDetail{
		Lines: []AdjustmentLine{line},
		Policy: PolicyFacts{Coinsurance: []CoinsuranceShare{
			{Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Leader: true, Percent: "65"},
			{Name: "ASURANSI LAIN", Percent: "35"},
		}},
		Spreading: []SpreadingShare{
			{ObjectID: "1", CoverageID: "1", TreatyType: "10007", TreatyName: "ORS", Percent: "100"},
			{ObjectID: "2", CoverageID: "1", TreatyType: "10001", TreatyName: "OR", Percent: "100"},
		},
	}
	return d, line
}

func TestNilaiKomiteMengikutiTypePembayaran(t *testing.T) {
	t.Parallel()
	_, line := sampleSheet()

	kasus := map[string]money.Money{
		"2": money.FromRupiah(1_625_000), // AdjustmentValue
		"1": money.FromRupiah(1_625_000),
		"3": money.FromRupiah(40_000),  // SalvageValue
		"4": money.FromRupiah(300_000), // AdjusterFeeValue
		"7": money.FromRupiah(300_000),
	}
	for tipe, mau := range kasus {
		l := line
		l.PaymentType = tipe
		require.Equalf(t, mau, l.CommitteeValue(), "Type %s", tipe)
	}
}

func TestSpreadingMemakaiNilaiKomiteDanCoverageYangSama(t *testing.T) {
	t.Parallel()
	d, line := sampleSheet()

	rows := d.SpreadingFor(line)
	require.Len(t, rows, 1, "spreading objek 2 bukan milik baris objek 1")
	require.Equal(t, "ORS", rows[0].TreatyName)
	require.Equal(t, "IDR", rows[0].Currency)
	require.True(t, rows[0].HasValue)
	// KmtSpred = nilai komite × SharePercentage / 100 — bukan gross.
	require.Equal(t, money.FromRupiah(1_625_000), rows[0].Value)
}

func TestCoMemberMemakaiGrossDanHanyaBilaKoasuransiLebihDariSatu(t *testing.T) {
	t.Parallel()
	d, line := sampleSheet()

	rows := d.CoMembersFor(line)
	require.Len(t, rows, 2)
	require.Equal(t, money.FromRupiah(1_625_000), rows[0].Value, "65% × gross 2.500.000")
	require.Equal(t, money.FromRupiah(875_000), rows[1].Value, "35% × gross 2.500.000")

	salvage := line
	salvage.PaymentType = "3"
	require.Empty(t, d.CoMembersFor(salvage), "Local.cekcoas: Type 3 tidak membentuk CoMember")

	tunggal := d
	tunggal.Policy.Coinsurance = tunggal.Policy.Coinsurance[:1]
	require.Empty(t, tunggal.CoMembersFor(line), "CoinsList satu baris tidak membentuk CoMember")
}

func TestLeaderDanBentukSpreadingMengikutiPega(t *testing.T) {
	t.Parallel()
	d, _ := sampleSheet()

	require.Equal(t, Leader{Name: "ASURANSI SINAR MAS - KANTOR PUSAT", Percent: "65"}, d.Leader())
	require.True(t, d.FullSpreading(),
		"nama leader dari CoinsList tidak sama persis dengan 'ASURANSI SINAR MAS'")

	tanpa := TransferDetail{}
	require.Equal(t, Leader{Name: OwnCompany, Percent: "100"}, tanpa.Leader(),
		"polis tanpa koasuransi: ASURANSI SINAR MAS 100")
	require.False(t, tanpa.FullSpreading())
}

func TestShareOfTepatSampaiSen(t *testing.T) {
	t.Parallel()

	v, ok := ShareOf(money.FromRupiah(1_000_000), "6.0945")
	require.True(t, ok)
	require.Equal(t, money.FromMinorUnits(6_094_500), v)

	// 1 sen × 50% = 0,5 sen -> dibulatkan ke atas.
	v, ok = ShareOf(money.FromMinorUnits(1), "50")
	require.True(t, ok)
	require.Equal(t, money.FromMinorUnits(1), v)

	_, ok = ShareOf(money.FromRupiah(100), "")
	require.False(t, ok, "persen kosong bukan nol")
	_, ok = ShareOf(money.FromRupiah(100), "abc")
	require.False(t, ok)
}
