package registrasi_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func idr(v int64) []registrasi.FaceSheetAmount {
	return []registrasi.FaceSheetAmount{{Currency: "IDR", Value: registrasi.Rupiah(v)}}
}

// Angka dari baris T_PLALIST FACOUT Pega (PNC-770): reserve 25.505.000 dengan bagian ASM 50%
// (ASM leader) → EstimationValue 12.752.500; SharePLA 314.000 / PercentPLA 1.570.000 →
// ResultPLA 2.550.500 (PLAFacout_Act 4.8).
func TestFacOutPLAAmountsMatchPega(t *testing.T) {
	r := registrasi.PLARecipient{Type: registrasi.PLATypeFacOut, Share: big.NewRat(314_000, 1), Base: big.NewRat(1_570_000, 1)}
	got := registrasi.PLAReinsuranceAmounts(idr(25_505_000), map[string]string{"IDR": "10026"}, r, big.NewRat(1, 2), true)

	require.Len(t, got, 1)
	require.Equal(t, registrasi.Rupiah(25_505_000), got[0].Reserve)
	require.Equal(t, registrasi.Rupiah(12_752_500), got[0].Base)
	require.Equal(t, registrasi.Rupiah(2_550_500), got[0].Result)
	require.Equal(t, registrasi.Rupiah(314_000), got[0].FacShare)
	require.Equal(t, registrasi.Rupiah(1_570_000), got[0].FacBase)
	require.Equal(t, "10026", got[0].CurrencyID)
}

// Bukan leader: HandingFee = reserve penuh (GeneratePLAList 19.2); dasar nol → hasil nol.
func TestFacOutPLAAmountsNotLeaderAndZeroBase(t *testing.T) {
	r := registrasi.PLARecipient{Type: registrasi.PLATypeFacOut, Share: big.NewRat(50, 1), Base: big.NewRat(100, 1)}
	got := registrasi.PLAReinsuranceAmounts(idr(1_000), nil, r, big.NewRat(1, 2), false)
	require.Equal(t, registrasi.Rupiah(1_000), got[0].Base)
	require.Equal(t, registrasi.Rupiah(500), got[0].Result)

	r.Base = new(big.Rat)
	require.Zero(t, registrasi.PLAReinsuranceAmounts(idr(1_000), nil, r, nil, false)[0].Result)
}

// Fire (006), PLAFacout_Act 3.1: objek lewat ObjectNo, coverage lewat kode CoverageOldID,
// bagian = ShareOffered × Sublimit/100, dasar = TSISublimit (cadangan TSI) × percentASM.
// FacOffer ber-FlagDelete dilewati; FacOffer yang objeknya tidak cocok memakai dasar
// FacOffer sebelumnya dengan bagian nol.
func TestFacOutPLARecipientsFire(t *testing.T) {
	offers := []registrasi.FacOffer{
		{ReinsurerName: "REAS SATU", ReinsurerID: " R1 ", Property: []registrasi.FacObject{
			{ObjectNo: "9", Coverage: []registrasi.FacCoverage{{Code: "C1", ShareOffered: "1", TSI: "1"}}},
			{ObjectNo: "35", Coverage: []registrasi.FacCoverage{
				{Code: "LAIN", ShareOffered: "999"},
				{Code: "C1", ShareOffered: "400", Sublimit: "50", TSISublimit: "2000"},
			}},
		}},
		{ReinsurerName: "DIHAPUS", ReinsurerID: "R2", Deleted: true},
		{ReinsurerName: "REAS TIGA", ReinsurerID: "R3", Property: []registrasi.FacObject{
			{ObjectNo: "7", Coverage: []registrasi.FacCoverage{{Code: "C1", ShareOffered: "100"}}},
		}},
		{ReinsurerName: "REAS EMPAT", ReinsurerID: "R4", Property: []registrasi.FacObject{
			{ObjectNo: "35", Coverage: []registrasi.FacCoverage{{Code: "C1", ShareOffered: "100", TSI: "3000"}}},
		}},
	}
	got := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: registrasi.TreatyFacOut, GroupPanel: "006", ObjectID: "35", Coverage: "C1",
		PercentASM: big.NewRat(1, 2), Offers: offers,
	})
	require.Len(t, got, 3)
	require.Equal(t, "R1", got[0].ID)
	require.Equal(t, registrasi.PLACodeFacOut, got[0].Code)
	require.Equal(t, big.NewRat(200, 1), got[0].Share, "400 × 50%")
	require.Equal(t, big.NewRat(1_000, 1), got[0].Base, "TSISublimit 2000 × 50%")
	require.Equal(t, "R3", got[1].ID)
	require.Zero(t, got[1].Share.Sign())
	require.Equal(t, big.NewRat(1_000, 1), got[1].Base, "dasar FacOffer sebelumnya terbawa")
	require.Equal(t, big.NewRat(1_500, 1), got[2].Base, "TSISublimit kosong → TSI 3000 × 50%")
}

// Aneka (003), PLAFacout_Act 3.2: objek lewat ObjectName, coverage urutan jaminan; dasar TSI
// × percentASM bila ShareOffered = TSI × Percent / 100, selain itu SumTSISpreaded.
func TestFacOutPLARecipientsAneka(t *testing.T) {
	offers := []registrasi.FacOffer{
		{ReinsurerName: "A", ReinsurerID: "R1", Aneka: []registrasi.FacObject{{ObjectName: "GEDUNG", Coverage: []registrasi.FacCoverage{
			{ShareOffered: "1"},
			{ShareOffered: "250", TSI: "1000", Percent: "25"},
		}}}},
		{ReinsurerName: "B", ReinsurerID: "R2", Aneka: []registrasi.FacObject{{ObjectName: "GEDUNG", Coverage: []registrasi.FacCoverage{
			{ShareOffered: "1"},
			{ShareOffered: "300", TSI: "1000", Percent: "25", SumTSISpreaded: "6000"},
		}}}},
	}
	got := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: registrasi.TreatyFacOut, GroupPanel: "003", ObjectName: "GEDUNG", CoverageSeq: 2,
		PercentASM: big.NewRat(1, 1), Offers: offers,
	})
	require.Equal(t, big.NewRat(250, 1), got[0].Share)
	require.Equal(t, big.NewRat(1_000, 1), got[0].Base)
	require.Equal(t, big.NewRat(6_000, 1), got[1].Base)
}

// Cargo (004): bagian nol mengeluarkan reasuradur; PA (002) tidak punya cabang → bagian 0.
func TestFacOutPLARecipientsCargoAndPA(t *testing.T) {
	cargo := []registrasi.FacOffer{
		{ReinsurerName: "NOL", ReinsurerID: "R1", Cargo: []registrasi.FacObject{
			{GoodNote: "KOPI", IndexObject: "1", Coverage: []registrasi.FacCoverage{{ShareOffered: "0"}}}}},
		{ReinsurerName: "ADA", ReinsurerID: "R2", Cargo: []registrasi.FacObject{
			{GoodNote: "KOPI", IndexObject: "1", Coverage: []registrasi.FacCoverage{{ShareOffered: "12.5", TSI: "500"}}}}},
	}
	got := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: registrasi.TreatyFacOut, GroupPanel: "004", ObjectID: "1", ObjectName: "KOPI", CoverageSeq: 1,
		Policy: registrasi.DLAPolicy{CaseID: "ASM-1"}, Offers: cargo,
	})
	require.Len(t, got, 1)
	require.Equal(t, "R2", got[0].ID)
	require.Equal(t, big.NewRat(500, 1), got[0].Base)

	pa := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: registrasi.TreatyFacOut, GroupPanel: "002", ObjectID: "1",
		Offers: []registrasi.FacOffer{{ReinsurerName: "P", ReinsurerID: "R9"}},
	})
	require.Len(t, pa, 1)
	require.Zero(t, pa[0].Share.Sign())
}

// FACOBSRB (10061), PLAFacout_Act 2: satu penerima SRB berhuruf M, bagian TSISpreaded polis,
// dasar SumOfTSI × percentASM.
func TestFacOutPLARecipientsFACOBSRB(t *testing.T) {
	got := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: "10061", PercentASM: big.NewRat(1, 2),
		Policy: registrasi.DLAPolicy{SumOfTSI: big.NewRat(10_000, 1),
			TSISpreaded: map[string]*big.Rat{"10061": big.NewRat(700, 1)}},
	})
	require.Equal(t, []registrasi.PLARecipient{{Type: registrasi.PLATypeFacOut, Code: registrasi.PLACodeFacOutSRB,
		Name: "SIMAS REINSURANCE BROKER", ID: "10038290", Share: big.NewRat(700, 1), Base: big.NewRat(5_000, 1)}}, got)
}

// Baris T_FACOFFER tanpa JSONDATA: PCT_SHAREREAS persen dengan dasar sendiri (100).
func TestFacOutPLARecipientsFlatFallback(t *testing.T) {
	got := registrasi.FacOutPLARecipients(registrasi.PLAReinsuranceInput{
		TreatyType: registrasi.TreatyFacOut, GroupPanel: "006", ObjectID: "35", PercentASM: big.NewRat(1, 1),
		Offers: []registrasi.FacOffer{{ReinsurerName: "AON", ReinsurerID: "10038693", FlatShare: "3"}},
	})
	require.Len(t, got, 1)
	amounts := registrasi.PLAReinsuranceAmounts(idr(192_000), nil, got[0], big.NewRat(1, 1), false)
	require.Equal(t, registrasi.Rupiah(5_760), amounts[0].Result, "3% × 192.000")
}

// PLABPPDAN_Act: BPPDAN dan EQ POOL, hasil = HandingFee × persen bagian / 100.
func TestBPPDANPLA(t *testing.T) {
	r, ok := registrasi.BPPDANPLARecipient("10010", 25_000) // 2,5%
	require.True(t, ok)
	require.Equal(t, registrasi.PLATypeBPPDAN, r.Type)
	require.Equal(t, "10038311", r.ID)
	got := registrasi.PLAReinsuranceAmounts(idr(1_000_000), nil, r, nil, false)
	require.Equal(t, registrasi.Rupiah(25_000), got[0].Result)
	require.Equal(t, registrasi.Money(250), got[0].FacShare, "SharePLA 2,50")

	eq, ok := registrasi.BPPDANPLARecipient("10023", 10_000)
	require.True(t, ok)
	require.Equal(t, registrasi.PLATypeEQPool, eq.Type)
	require.Equal(t, "REASURANSI MAIPARK INDONESIA", eq.Name)

	_, ok = registrasi.BPPDANPLARecipient("10001", 1)
	require.False(t, ok)
}

// DownloadPLA 13 dan 19–23: label "Your Share".
func TestPLAShareLabel(t *testing.T) {
	require.Equal(t, "FAC-OUT", registrasi.PLAShareLabel(registrasi.PLA{Type: registrasi.PLATypeFacOut, Number: "H2610001"}))
	require.Equal(t, "FAC-OUT", registrasi.PLAShareLabel(registrasi.PLA{Type: registrasi.PLATypeEQPool, Number: "H2610002"}))
	require.Equal(t, "BPPDAN", registrasi.PLAShareLabel(registrasi.PLA{Type: registrasi.PLATypeBPPDAN, Number: "H2610003"}))
	require.Equal(t, "FACOBLIG", registrasi.PLAShareLabel(registrasi.PLA{Type: registrasi.PLATypeFacOut, Number: "M2610004"}))
	require.Equal(t, "COINS", registrasi.PLAShareLabel(registrasi.PLA{Type: registrasi.PLATypeCoins, Number: "J2610005"}))
}

// Bagian ASM dari CoinsList: tanpa koasuransi 1 dan bukan leader.
func TestASMCoinsShare(t *testing.T) {
	pct, leader := registrasi.ASMCoinsShare("ASM", nil)
	require.Equal(t, big.NewRat(1, 1), pct)
	require.False(t, leader)

	pct, leader = registrasi.ASMCoinsShare("ASM", []registrasi.PLACoinsMember{
		{ID: "1", Name: "PT ASURANSI SINAR MAS", Leader: true, Share: 500_000, HasShare: true},
		{ID: "2", Name: "LAIN", Share: 500_000, HasShare: true},
	})
	require.Equal(t, big.NewRat(1, 2), pct)
	require.True(t, leader)
}

// Reserve PLA menjumlahkan estimasi ber-CFS apa pun tipenya, membuang mata uang bernilai nol,
// dan pada PA PHK memakai nilai pengajuan adjustment terakhir (GeneratePLAList 18–20).
func TestPLAReserve(t *testing.T) {
	name := func(c string) string { return c }
	c := registrasi.Coverage{ID: "100829", Item: []registrasi.ObjectItem{{Estimation: []registrasi.Estimation{
		{Type: registrasi.EstimateClaim, Currency: "IDR", Value: 100, FaceSheet: true},
		{Type: registrasi.EstimateAdjuster, Currency: "IDR", Value: 50, FaceSheet: true},
		{Type: registrasi.EstimateClaim, Currency: "IDR", Value: 999},
		{Type: registrasi.EstimateClaim, Currency: "USD", Value: 0, FaceSheet: true},
	}}}}
	require.Equal(t, []registrasi.FaceSheetAmount{{Currency: "IDR", Value: 150}}, registrasi.PLAReserve(c, name))

	c.ID = "10010"
	c.Settlement = []registrasi.SettlementLine{{Submitted: 7}, {Submitted: 77}}
	// Penimpaan (langkah 18.2) mengenai mata uang TERAKHIR sebelum yang nol dibuang (langkah 20).
	require.Equal(t, []registrasi.FaceSheetAmount{{Currency: "IDR", Value: 150}, {Currency: "USD", Value: 77}},
		registrasi.PLAReserve(c, name))
}
