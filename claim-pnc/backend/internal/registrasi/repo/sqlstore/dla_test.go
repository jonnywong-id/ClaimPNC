package sqlstore

import "testing"

// Pengurai dokumen polis membaca CoinsList, FacOfferList per Group Panel, dan TSISpreaded.
func TestParseDLAPolicy(t *testing.T) {
	body := []byte(`{
	  "CaseID": "ASM-01", "SumOfTSI": "500000000", "TypeOfCoins": "F", "SyariahStatus": "0",
	  "StartDateTime": "20260101T050000.000 GMT", "OfferFacIn": {"PercentShare": "40"},
	  "Quotation": {"BusinessCode": "10140"},
	  "CoinsList": [{"CoinsID": "1", "CoinsName": "PT ASURANSI SINAR MAS", "Leader": "true", "PercentShare": "60"}],
	  "FacOfferList": [{
	    "ReinsurerName": "FAC A", "ReinsurerID": "F1", "TotalOffered": "50000000",
	    "AnekaList": [{"ObjectName": "MESIN", "SumTSIObjectAneka": "300000000",
	      "CoverageList": [{"Coverage": "C1", "TSI": "200000000", "FacOutObjectList": [{"ShareOffered": "50000000", "Percent": "25"}]}]}],
	    "OldFacOffer": [{"SpreadingList": [{"TreatyType": "10061", "TSISpreaded": "1"}]}]
	  }],
	  "ObjectList": [{"CoverageList": [{"SpreadingList": [{"TreatyType": "10061", "TSISpreaded": "7000000"}]}]}]
	}`)
	p, err := parseDLAPolicy(body)
	if err != nil {
		t.Fatal(err)
	}
	if p.CaseID != "ASM-01" || p.SumOfTSI.FloatString(0) != "500000000" || p.TypeOfCoins != "F" ||
		p.OfferFacInShare.FloatString(0) != "40" || p.BusinessCode != "10140" || p.StartYear != 2026 {
		t.Fatalf("kepala polis salah: %+v", p)
	}
	if len(p.Coins) != 1 || !p.Coins[0].Leader || !p.Coins[0].HasShare {
		t.Fatalf("CoinsList salah: %+v", p.Coins)
	}
	if len(p.FacOffer) != 1 || p.FacOffer[0].OfferedMissing || len(p.FacOffer[0].Aneka) != 1 {
		t.Fatalf("FacOfferList salah: %+v", p.FacOffer)
	}
	c := p.FacOffer[0].Aneka[0].Coverage[0]
	if c.ShareOffered != "50000000" || c.Percent != "25" || c.TSI != "200000000" {
		t.Fatalf("CoverageList salah: %+v", c)
	}
	if got := p.TSISpreaded["10061"]; got == nil || got.FloatString(0) != "7000000" {
		t.Fatalf("TSISpreaded salah (salinan FacOffer tidak boleh dipakai): %v", got)
	}
}
