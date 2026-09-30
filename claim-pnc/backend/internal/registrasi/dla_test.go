package registrasi

import (
	"math/big"
	"testing"
	"time"
)

func pct(v int64) Percent { return Percent(v * 10_000) }

// COINS: hanya bila Sinar Mas leader; anggota lain ber-share > 0 mendapat
// round2(Gross × share / 100). Sinar Mas sendiri, terhapus, share nol, dan 10000000262 tidak.
func TestCoinsDLAFollowsDLACoinsAct(t *testing.T) {
	members := []PLACoinsMember{
		{ID: "1", Name: "PT ASURANSI SINAR MAS", Leader: true, Share: pct(60), HasShare: true},
		{ID: "2", Name: "ANGGOTA A", Share: pct(25), HasShare: true},
		{ID: "3", Name: "ANGGOTA B", Share: pct(15), HasShare: true, Deleted: true},
		{ID: "4", Name: "ANGGOTA C", Share: 0, HasShare: true},
		{ID: "10000000262", Name: "ANGGOTA D", Share: pct(15), HasShare: true},
	}
	got := CoinsDLAShares("ASM", members, Rupiah(10_000_000))
	if len(got) != 1 {
		t.Fatalf("penerima = %d, ingin 1: %+v", len(got), got)
	}
	d := got[0]
	if d.RecipientCode != "2" || d.Code != DLACodeCoins || d.Type != DLATypeCoins {
		t.Fatalf("penerima salah: %+v", d)
	}
	if d.Value != "2500000.00" || d.Percent != "25.00" || d.ClaimAmount != "10000000.00" {
		t.Fatalf("nilai salah: %+v", d)
	}

	members[0].Leader = false
	members[1].Leader = true
	if got := CoinsDLAShares("ASM", members, Rupiah(10_000_000)); len(got) != 0 {
		t.Fatalf("bukan leader tetap terbit: %+v", got)
	}
}

// BPPDAN dan EQ POOL: penerima tetap per kode; nilai = round5(share × round5(adj/100)).
func TestBPPDANDLAFollowsDLABPPDANAct(t *testing.T) {
	d, ok := BPPDANDLAShare("10010", pct(10), Rupiah(1_234_567))
	if !ok || d.Recipient != "BPPDAN" || d.RecipientCode != "10038311" || d.Type != DLATypeBPPDAN || d.Code != "H" {
		t.Fatalf("BPPDAN salah: %+v", d)
	}
	if d.Value != "123456.70000" || d.Percent != "10.00000" || d.ClaimAmount != "1234567.00000" {
		t.Fatalf("nilai BPPDAN salah: %+v", d)
	}
	eq, ok := BPPDANDLAShare("10023", pct(5), Rupiah(100))
	if !ok || eq.Type != DLATypeEQPool || eq.RecipientCode != "10038726" {
		t.Fatalf("EQ POOL salah: %+v", eq)
	}
	if _, ok := BPPDANDLAShare("10029", pct(5), Rupiah(100)); ok {
		t.Fatal("PFRA tidak boleh ke pengolah BPPDAN")
	}
}

// Penggolongan baris spreading (GenerateDLAList 27.4.7).
func TestRouteSpreading(t *testing.T) {
	cases := []struct {
		treaty, kind string
		want         DLARoute
	}{
		{"10002", "3", DLARouteNone},   // FAC-OBLIG dilewati
		{"10024", "3", DLARouteTreaty}, // FACOBLIGINDT dipaksa treaty
		{"10010", "2", DLARouteBPPDAN},
		{"10023", "1", DLARouteBPPDAN},
		{"10029", "3", DLARouteNone},
		{"10015", "3", DLARouteFacOut},
		{"10061", "3", DLARouteFacOut},
		{"10003", "2", DLARouteTreaty},
		{"10001", "1", DLARouteNone},
	}
	for _, c := range cases {
		if got := RouteSpreading(c.treaty, c.kind); got != c.want {
			t.Errorf("%s/%s = %v, ingin %v", c.treaty, c.kind, got, c.want)
		}
	}
}

// TREATY: penerima non-FAC hanya bila nilai terkonversi × share melampaui limit RP.
func TestTreatyDLAFollowsDLATreatyAct(t *testing.T) {
	arr := TreatyArrangement{
		Reinsurers: []TreatyReinsurer{
			{Name: "REAS A", ID: "R1", PctShare: "40", TypeName: "QS"},
			{Name: "REAS B", ID: "R2", PctShare: "60", TypeName: "FAC-OBLIG"},
		},
		Limit: "1000000",
	}
	// share spreading 50%, adj 10.000.000; total terkonversi 2.000.000 → 1.000.000 (tidak lebih).
	got := TreatyDLAShares("10005", pct(50), Rupiah(10_000_000), big.NewRat(2_000_000, 1), arr)
	if len(got) != 1 || got[0].RecipientCode != "R2" {
		t.Fatalf("hanya FAC-OBLIG yang lolos: %+v", got)
	}
	fac := got[0]
	// FCBL: (adj/100) × share = 100.000 × 50.
	if fac.Value != "5000000.00" || fac.Code != DLACodeTreatyFac || fac.Type != DLATypeFacOblig || fac.QSRI != "0" {
		t.Fatalf("FAC-OBLIG salah: %+v", fac)
	}
	got = TreatyDLAShares("10005", pct(50), Rupiah(10_000_000), big.NewRat(4_000_000, 1), arr)
	if len(got) != 2 {
		t.Fatalf("treaty melampaui limit harus terbit: %+v", got)
	}
	// non-QS: round5(40/100) × adj × round5(50/100) = 0.4 × 10.000.000 × 0.5.
	if got[0].Value != "2000000.00" || got[0].Code != DLACodeTreaty || got[0].Type != DLATypeTreaty ||
		got[0].ShareSpread != "40.000" || got[0].Percent != "50.000" || got[0].QSPQS != "QS" {
		t.Fatalf("treaty salah: %+v", got[0])
	}
	arr.QSPct = "0.25"
	got = TreatyDLAShares("10005", pct(50), Rupiah(10_000_000), big.NewRat(4_000_000, 1), arr)
	if got[0].Value != "500000.00" || got[0].QSRI != "25" {
		t.Fatalf("treaty QS salah: %+v", got[0])
	}
	if got := TreatyDLAShares("10005", pct(50), Rupiah(1), big.NewRat(9, 1), TreatyArrangement{Reinsurers: arr.Reinsurers}); got != nil {
		t.Fatalf("tanpa limit, treaty bukan FAC tidak terbit: %+v", got)
	}
}

// FAC OUT Aneka: bagian = ShareOffered × adj / TSI, dasar TSI yang rasionya cocok persen.
func TestFacOutDLAAneka(t *testing.T) {
	policy := DLAPolicy{
		CaseID: "ASM01", SumOfTSI: big.NewRat(500_000_000, 1),
		Coins: []PLACoinsMember{{ID: "1", Name: "PT ASURANSI SINAR MAS", Share: pct(100), HasShare: true}},
		FacOffer: []FacOffer{
			{ReinsurerName: "FAC A", ReinsurerID: "F1", Aneka: []FacObject{{ObjectName: "MESIN", Coverage: []FacCoverage{
				{TSI: "200000000", ShareOffered: "50000000", Percent: "25"},
			}}}},
			{ReinsurerName: "DIHAPUS", ReinsurerID: "F2", Deleted: true},
			{ReinsurerName: "KOSONG", ReinsurerID: "F3", OfferedMissing: true},
		},
	}
	got := FacOutDLAShares(FacOutInput{
		Portal: "ASM", TreatyType: "10015", GroupPanel: "003", Policy: policy,
		ObjectName: "MESIN", CoverageSeq: 1, Amount: Rupiah(8_000_000), InitialBase: big.NewRat(1, 1),
	})
	if len(got) != 1 {
		t.Fatalf("penerima = %+v", got)
	}
	d := got[0]
	// 50.000.000 × 8.000.000 / 200.000.000 = 2.000.000.
	if d.Value != "2000000.000" || d.Percent != "200000000.000" || d.ShareSpread != "50000000.000" || d.Code != "H" {
		t.Fatalf("FAC OUT salah: %+v", d)
	}

	srb := FacOutDLAShares(FacOutInput{
		Portal: "ASM", TreatyType: "10061", GroupPanel: "003",
		Policy:      DLAPolicy{SumOfTSI: big.NewRat(100_000_000, 1), TSISpreaded: map[string]*big.Rat{"10061": big.NewRat(10_000_000, 1)}},
		CoverageSeq: 1, Amount: Rupiah(1_000_000),
	})
	if len(srb) != 1 || srb[0].Code != DLACodeFacOutSRB || srb[0].RecipientCode != "10038290" || srb[0].Value != "100000.000" {
		t.Fatalf("FACOBSRB salah: %+v", srb)
	}
}

// Gerbang ex gratia dan penanda ASO.
func TestDLAExGratiaGate(t *testing.T) {
	if !DLASkipsExGratia(true, false) || DLASkipsExGratia(true, true) || DLASkipsExGratia(false, false) {
		t.Fatal("gerbang ex gratia salah")
	}
	if !FlagASO([]PLACoinsMember{{ID: "10038996"}}) || FlagASO([]PLACoinsMember{{ID: "1"}}) {
		t.Fatal("penanda ASO salah")
	}
}

// Catatan INSERT_PLADLA: DLA sebelumnya lebih dulu, lalu PLA.
func TestDLANote(t *testing.T) {
	day := time.Date(2023, time.December, 4, 10, 0, 0, 0, time.UTC)
	if got := DLANote(PLAPrevious{}, false, DLAPrevious{Number: "J1", Date: day}, true); got != "- Please see our DLA No.:J1 with DD: 04-DEC-23" {
		t.Fatalf("catatan DLA = %q", got)
	}
	if got := DLANote(PLAPrevious{Number: "J9", Date: day}, true, DLAPrevious{}, false); got != "- Please see our PLA No.:J9 with DD: 04/12/2023" {
		t.Fatalf("catatan PLA = %q", got)
	}
	if got := DLANote(PLAPrevious{}, false, DLAPrevious{}, false); got != "" {
		t.Fatalf("catatan kosong = %q", got)
	}
}

// Label "Your Share Under" — DownloadDLA langkah 24–30.
func TestDLAShareUnder(t *testing.T) {
	if got := DLAShareUnder("H2610001", DLATypeFacOut, "", ""); got != "FAC-OUT" {
		t.Errorf("FAC OUT = %s", got)
	}
	if got := DLAShareUnder("H2610001", DLATypeBPPDAN, "", ""); got != "BPPDAN" {
		t.Errorf("BPPDAN = %s", got)
	}
	if got := DLAShareUnder("L2610001", DLATypeTreaty, "QS", "QS (R/I)"); got != "QS" {
		t.Errorf("TREATY = %s", got)
	}
	if got := DLAShareUnder("M2610001", DLATypeFacOblig, "FAC-OBLIG", ""); got != "FACOBLIG" {
		t.Errorf("FAC-OBLIG = %s", got)
	}
}

// Tombol Print DLA hanya untuk baris diakseptasi dengan persetujuan LOD.
func TestCanPrintDLA(t *testing.T) {
	if err := CanPrintDLA(SettlementLine{AcceptanceStatus: "1", AcceptanceLODStatus: LODAgreed}); err != nil {
		t.Fatalf("baris sah ditolak: %v", err)
	}
	if err := CanPrintDLA(SettlementLine{AcceptanceStatus: "1", AcceptanceLODStatus: LODDisagreed}); err == nil {
		t.Fatal("LOD tidak disetujui harus ditolak")
	}
}
