package registrasi

import "testing"

func TestKeysCoverAllLegacyForms(t *testing.T) {
	k := Claim{ID: "abc", Number: "PNCN.26.0012"}.Keys()
	if k.Number != "PNCN.26.0012" || k.ID != "abc" || k.Prefixed != "ASM-FW-GCNMFW-WORK PNCN.26.0012" {
		t.Fatalf("kunci salah: %+v", k)
	}
}

func checklistClaim(line LineOfBusiness, item ObjectItem, coverageID string) Claim {
	return Claim{
		Policy:      Policy{Line: line},
		InsuredItem: []InsuredItem{{Coverage: []Coverage{{ID: coverageID, Item: []ObjectItem{item}}}}},
	}
}

func TestDocumentRequiredFollowsItemGroup(t *testing.T) {
	types := []DocumentType{
		{Category: "REGISTER", ID: "1", Name: "B", RequiredRaw: "1", ObjectDocID: "BUILDING(S)"},
		{Category: "REGISTER", ID: "2", Name: "A", RequiredRaw: "1", ObjectDocID: "OTHERS"},
		{Category: "SURVEY", ID: "3", Name: "C", RequiredRaw: "0", ObjectDocID: "BUILDING(S)"},
	}
	claim := checklistClaim(LineFire, ObjectItem{Name: "BUILDING", Group: "BUILDING(S)"}, "X")
	got := DocumentChecklist(claim, types, nil)

	if len(got) != len(DocumentCategories) || got[0].Code != "REGISTER" {
		t.Fatalf("urutan kategori salah: %+v", got)
	}
	reg := got[0].Row
	if reg[0].Type.Name != "A" || reg[0].Required {
		t.Fatalf("OTHERS tidak boleh wajib tanpa item OTHERS: %+v", reg[0])
	}
	if !reg[1].Required {
		t.Fatalf("BUILDING(S) harus wajib: %+v", reg[1])
	}
	if got[1].Row[0].Required {
		t.Fatalf("STS_WAJIB 0 tidak pernah wajib")
	}
}

func TestOthersNameWithoutGroupCountsAsOthers(t *testing.T) {
	types := []DocumentType{{Category: "REGISTER", ID: "2", RequiredRaw: "1", ObjectDocID: "OTHERS"}}
	claim := checklistClaim(LineMiscellaneous, ObjectItem{Name: "Others equipment"}, "X")
	if !DocumentChecklist(claim, types, nil)[0].Row[0].Required {
		t.Fatalf("item bernama OTHERS tanpa kelompok harus dianggap OTHERS")
	}
}

func TestPersonalAccidentRequiredFollowsCoverage(t *testing.T) {
	types := []DocumentType{
		{Category: "REGISTER", ID: "1", Name: "A", RequiredRaw: "1", Coverage: []string{"10004"}},
		{Category: "REGISTER", ID: "2", Name: "B", RequiredRaw: "1", ObjectDocID: "OTHERS", Coverage: []string{"10003"}},
	}
	claim := checklistClaim(LinePersonalAccident, ObjectItem{Name: "OTHERS", Group: "OTHERS"}, "10004")
	rows := DocumentChecklist(claim, types, nil)[0].Row
	if !rows[0].Required || rows[1].Required {
		t.Fatalf("PA wajib menurut coverage, bukan kelompok item: %+v", rows)
	}
}

func TestUploadedCountsOnlyStoredFilesBySubCategory(t *testing.T) {
	types := []DocumentType{{Category: "PAYMENT", ID: "14904", Name: "KWITANSI"}}
	files := []Attachment{
		{SubCategory: "14904", ImageID: "A"},
		{SubCategory: "14904", ImageID: "B"},
		{SubCategory: "14904"},
		{SubCategory: "99999", ImageID: "C"},
	}
	got := DocumentChecklist(Claim{}, types, files)
	for _, c := range got {
		if c.Code == "PAYMENT" && c.Row[0].Uploaded != 2 {
			t.Fatalf("terunggah = %d, mau 2", c.Row[0].Uploaded)
		}
	}
}
