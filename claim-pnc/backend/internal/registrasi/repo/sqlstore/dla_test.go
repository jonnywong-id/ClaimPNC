package sqlstore

import "testing"

// Pengurai dokumen polis membaca kepala polis dan TSISpreaded; CoinsList dan FacOfferList
// dokumen diabaikan karena sumbernya T_COINSLIST dan T_FACOFFER.
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
	if len(p.Coins) != 0 || len(p.FacOffer) != 0 {
		t.Fatalf("CoinsList/FacOfferList dokumen tidak boleh dipakai: %+v %+v", p.Coins, p.FacOffer)
	}
	if got := p.TSISpreaded["10061"]; got == nil || got.FloatString(0) != "7000000" {
		t.Fatalf("TSISpreaded salah (salinan FacOffer tidak boleh dipakai): %v", got)
	}
}

// JSONDATA setiap baris T_FACOFFER memuat SELURUH FacOfferList; yang diambil hanya entri
// milik REINSURER_ID baris itu.
func TestParseFacOfferRow(t *testing.T) {
	body := []byte(`{"FacOfferList": [
	  {"ReinsurerName": "FAC A", "ReinsurerID": "F1", "TotalOffered": "1"},
	  {"ReinsurerName": "FAC B", "ReinsurerID": "F2", "TotalOffered": "50000000",
	   "AnekaList": [{"ObjectName": "MESIN", "SumTSIObjectAneka": "300000000",
	     "CoverageList": [{"Coverage": "C1", "TSI": "200000000", "FacOutObjectList": [{"ShareOffered": "50000000", "Percent": "25"}]}]}]}
	]}`)
	f, ok, err := parseFacOfferRow("F2", body)
	if err != nil || !ok {
		t.Fatalf("entri F2 tidak ditemukan: %v %v", ok, err)
	}
	if f.ReinsurerName != "FAC B" || f.OfferedMissing || len(f.Aneka) != 1 {
		t.Fatalf("entri salah: %+v", f)
	}
	c := f.Aneka[0].Coverage[0]
	if c.ShareOffered != "50000000" || c.Percent != "25" || c.TSI != "200000000" {
		t.Fatalf("CoverageList salah: %+v", c)
	}
	if _, ok, _ := parseFacOfferRow("F9", body); ok {
		t.Fatal("ReinsurerID yang tidak ada tidak boleh menghasilkan entri")
	}
}
