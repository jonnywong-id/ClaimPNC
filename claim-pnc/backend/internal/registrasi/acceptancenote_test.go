package registrasi

import (
	"math/big"
	"testing"
)

// Spreading dan rincian QS mengikuti contoh Draft Persetujuan: QS = share × nilai / 100, QS (OR)
// dan QS (R/I) = share × PCT × nilai / 100.
func TestAcceptanceNoteSpreading(t *testing.T) {
	spreading := []Spreading{
		{TreatyKind: treatyQS, Name: "QS", Share: Percent(72_464)}, // 7,2464 %
		{TreatyKind: "10015", Name: "FAC-OUT", Share: Percent(927_536)},
		{TreatyKind: "10001", Name: "OR", Share: Percent(10_000), Removed: true},
	}
	qs := map[string][]TreatyQSPart{treatyQS: {{Name: "QS (OR)", Pct: ".25"}, {Name: "QS (R/I)", Pct: "0.75"}}}
	spread, parts := AcceptanceNoteSpreading(spreading, Money(10_000_000), qs) // 100.000,00
	if len(spread) != 2 || len(parts) != 2 {
		t.Fatalf("baris = %d, QS = %d", len(spread), len(parts))
	}
	if got := spread[0].Amount.FloatString(4); got != "7246.4000" {
		t.Fatalf("QS = %s", got)
	}
	if got := parts[0].Amount.FloatString(4); got != "1811.6000" {
		t.Fatalf("QS (OR) = %s", got)
	}
	sum := new(big.Rat).Add(parts[0].Amount, parts[1].Amount)
	if sum.Cmp(spread[0].Amount) != 0 {
		t.Fatalf("QS (OR) + QS (R/I) = %s, bukan %s", sum.FloatString(4), spread[0].Amount.FloatString(4))
	}
}

// PRINT hanya untuk baris bernomor akseptasi; Travel dan PA belum punya tata letak.
func TestCanPrintAcceptanceNote(t *testing.T) {
	if err := CanPrintAcceptanceNote(SettlementLine{}, Policy{}); err == nil {
		t.Fatal("tanpa nomor akseptasi harus ditolak")
	}
	line := SettlementLine{AcceptedNo: "A26"}
	if err := CanPrintAcceptanceNote(line, Policy{Line: LineMarineCargo}); err != nil {
		t.Fatalf("marine ditolak: %v", err)
	}
	for _, p := range []Policy{{Line: LineTravel}, {Line: LinePersonalAccident}} {
		if err := CanPrintAcceptanceNote(line, p); err == nil {
			t.Fatalf("%s harus ditolak", p.Line)
		}
	}
}

// Tipe 7 dicetak sebagai Adjuster Fee — Pega mengubah tipenya menjadi 4 sebelum template.
func TestAcceptanceNoteBlock(t *testing.T) {
	for payment, want := range map[string]string{
		PaymentFinal: "Claim Accepted", PaymentSalvage: "Salvage", PaymentAdjusterFee: "Adjuster Fee", paymentCollectionFee: "Adjuster Fee",
	} {
		if got := (AcceptanceNote{PaymentType: payment}).AcceptanceNoteBlock(); got != want {
			t.Errorf("tipe %s = %s", payment, got)
		}
	}
}

// DHARMANTO hanya berlaku pada kategori PLA dan DLA, bukan akseptasi.
func TestAcceptanceNoteSignerID(t *testing.T) {
	for name, want := range map[string]string{
		"BAMBANGSETIADJIGUNAWAN": "BAMBANGSG", "LINDA NOVA": "LINDANOVA", "ELLENSUPRIYATI": "ELLENSP",
		"INDRAGUNAWAN": "INDRAGN", "DHARMANTO": "",
	} {
		if got := AcceptanceNoteSignerID(name); got != want {
			t.Errorf("%s = %q, bukan %q", name, got, want)
		}
	}
}

// SetShareAsmWhenPilihAdjustment: 12 → Ex Gratia + 100%; 15 → Ex Gratia + share sendiri di
// CoinsList; tanpa CoinsList → 100%; jenis lain tidak mengubah apa pun.
func TestApplyLODType(t *testing.T) {
	coins := []PLACoinsMember{
		{ID: "1", Name: "PT ASURANSI SINAR MAS", Share: Percent(575_000), HasShare: true},
		{ID: "2", Name: "KOASURADUR UJI", Share: Percent(425_000), HasShare: true},
	}
	base := SettlementLine{Gross: Money(1_000_000), ShareASM: Percent(300_000), Value: Money(300_000)}

	l := base
	if !ApplyLODType(&l, "12", "ASM", "2", coins) || !l.ExGratia || l.ShareASM != PercentFull || l.Value != Money(1_000_000) {
		t.Fatalf("12: %+v", l)
	}
	l = base
	if !ApplyLODType(&l, "15", "ASM", "2", coins) || !l.ExGratia || l.ShareASM != Percent(575_000) || l.Value != Money(575_000) {
		t.Fatalf("15: %+v", l)
	}
	l = base
	if ApplyLODType(&l, "3", "ASM", "2", coins) || l.ExGratia || l.ShareASM != base.ShareASM {
		t.Fatalf("3 tidak boleh mengubah: %+v", l)
	}
	l = base
	if !ApplyLODType(&l, "3", "ASM", "2", nil) || l.ShareASM != PercentFull {
		t.Fatalf("tanpa CoinsList: %+v", l)
	}
	l = base
	ApplyLODType(&l, "12", "ASM", "1", coins)
	if l.Accepted != l.Value {
		t.Fatalf("TYPEOFCOINS 1: NILAIAKSEPTASI = Value, dapat %+v", l)
	}
}
