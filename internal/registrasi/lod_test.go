package registrasi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Angka contoh PDF LOD dari Work Owner (nama anggota KARANGAN, `D-69`): nilai 77.187.648,30
// dan sembilan anggota — yang dicek di sini bagian dan terbilangnya.
func TestTerbilangMengikutiContohLOD(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		"tujuh puluh tujuh juta seratus delapan puluh tujuh ribu enam ratus empat puluh delapan koma tiga puluh rupiah",
		registrasi.Terbilang(7_718_764_830, "rupiah"))
	require.Equal(t, "seribu sebelas rupiah", registrasi.Terbilang(101_100, "rupiah"))
	require.Equal(t, "satu miliar dua ratus juta rupiah", registrasi.Terbilang(120_000_000_000, "rupiah"))
	require.Equal(t, "nol rupiah", registrasi.Terbilang(0, "rupiah"))
}

func TestBuildLODMembagiNilaiMenurutShareTanpaPembulatan(t *testing.T) {
	t.Parallel()
	line := registrasi.SettlementLine{Gross: 7_718_764_830}
	members := []registrasi.PLACoinsMember{
		{Name: "ANGGOTA A", Share: 575_000, HasShare: true, Leader: true},
		{Name: "ANGGOTA B", Share: 47_500, HasShare: true},
		{Name: "DIHAPUS", Share: 10_000, HasShare: true, Deleted: true},
	}
	doc := registrasi.BuildLOD(line, "3", members, registrasi.LODFacts{Currency: "IDR", PolicyNumber: "POLIS-UJI"}, time.Time{})
	require.Len(t, doc.Members, 2, "anggota bertanda hapus dilewati")
	require.Equal(t, "44382897.7725", doc.Members[0].Value.FloatString(4))
	require.Equal(t, "3666413.29425", doc.Members[1].Value.FloatString(5))

	// Polis tanpa CoinsList: contoh jenis "dengan Co Member" mencetak satu baris
	// "PT. Asuransi Sinar Mas 100 %" senilai seluruh LOD.
	solo := registrasi.BuildLOD(line, "1", nil, registrasi.LODFacts{}, time.Time{})
	require.Len(t, solo.Members, 1)
	require.Equal(t, registrasi.LODInsurer, solo.Members[0].Name)
	require.Equal(t, registrasi.PercentFull, solo.Members[0].Percent)
	require.Equal(t, "77187648.30", solo.Members[0].Value.FloatString(2))
}

func TestAturanPrintLODMengikutiShowAdjustment(t *testing.T) {
	t.Parallel()
	fire := registrasi.Policy{Line: registrasi.LineFire, BusinessType: "Fire"}
	ok := registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, AcceptanceStatus: registrasi.DecisionApprove}
	require.NoError(t, registrasi.CanPrintLOD(ok, fire))

	notYet := ok
	notYet.AcceptanceStatus = registrasi.DecisionPending
	require.Error(t, registrasi.CanPrintLOD(notYet, fire), "belum disetujui komite")

	lod := ok
	lod.AcceptanceLODStatus = "1"
	require.Error(t, registrasi.CanPrintLOD(lod, fire), ".AcceptationStatusLOD != ''")

	fee := ok
	fee.PaymentType = registrasi.PaymentAdjusterFee
	require.Error(t, registrasi.CanPrintLOD(fee, fire), "Type 4")

	require.Error(t, registrasi.CanPrintLOD(ok, registrasi.Policy{Line: registrasi.LineTravel}), "Group Panel 005")
	require.Error(t, registrasi.CanPrintLOD(ok, registrasi.Policy{Line: registrasi.LineFire, BusinessType: "BondingKBG"}))
}

// SetDataEmailTertanggung: ClaimData.Email + "," + UserTeknisEmail, tanpa koma kosong.
func TestIsianEmailLOD(t *testing.T) {
	t.Parallel()
	require.Equal(t, "tertanggung@contoh.internal,pic@contoh.internal",
		registrasi.LODEmailDraft(" tertanggung@contoh.internal ", "pic@contoh.internal"))
	require.Equal(t, "pic@contoh.internal", registrasi.LODEmailDraft("", "pic@contoh.internal"))
	require.Equal(t, "", registrasi.LODEmailDraft("", ""))
}

func TestPilihanTipePDFDanTemplateTersedia(t *testing.T) {
	t.Parallel()
	types := registrasi.LODTypesFor(registrasi.Policy{Line: registrasi.LineFire})
	require.Len(t, types, 16)
	require.Equal(t, "Property - Final dengan Co Member", types[2].Name)
	require.Equal(t, []string{"16", "12", "15"}, []string{types[13].ID, types[14].ID, types[15].ID}, "urutan append Pega")
	require.Empty(t, registrasi.LODTypesFor(registrasi.Policy{Line: registrasi.LinePersonalAccident}))
	for _, ty := range types {
		// Seluruh jenis punya contoh PDF kecuali 16 "Pembayaran Final".
		require.Equal(t, ty.ID != "16", registrasi.LODTemplateReady(ty.ID), ty.ID)
	}
	require.False(t, registrasi.LODTemplateReady("99"))
	require.True(t, registrasi.LODPrintsInsuredAddress("12"))
	require.True(t, registrasi.LODPrintsInsuredAddress("15"))
	require.False(t, registrasi.LODPrintsInsuredAddress("3"))
}
