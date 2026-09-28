package registrasi

import (
	"testing"
	"time"
)

func faceSheetClaim() Claim {
	return Claim{
		Number: "PNCN.26.0001",
		Policy: Policy{
			Number: "P-1", Line: LineFire, BusinessType: "IAR", BusinessName: "INDUSTRIAL ALL RISK", Currency: "10026",
			Coinsurance: Coinsurance{Role: "LEADER", ShareASM: 570_000, HasShare: true},
		},
		InsuredItem: []InsuredItem{{ID: "7", Name: "PABRIK", Coverage: []Coverage{{
			ID: "100", Name: "4.3", TSI: Rupiah(1_000_000_000),
			Spreading: []Spreading{
				{TreatyKind: "10001", Name: "or", Share: 70_000},
				{TreatyKind: TreatyFacOut, Name: "fac-out", Share: 930_000},
				{TreatyKind: "10002", Name: "dihapus", Share: 1, Removed: true},
			},
			Item: []ObjectItem{{Estimation: []Estimation{
				{Type: EstimateClaim, Currency: "10026", Value: Rupiah(30_000_000)},
				{Type: EstimateAdjuster, Currency: "10026", Value: Rupiah(5_000_000)},
				{Type: EstimateClaim, Currency: "10026", Value: Rupiah(20_000_000)},
			}}},
		}}}},
	}
}

func TestFaceSheetFollowsTheSampleArithmetic(t *testing.T) {
	f := BuildFaceSheet(FaceSheetInput{
		Claim: faceSheetClaim(), Now: time.Now(), CurrencyName: map[string]string{"10026": "IDR"},
		CoMember: []CoinsuranceRow{
			{CoinsName: "ASURANSI SINAR MAS", PercentShare: 570_000, HasShare: true},
			{CoinsName: "ANGGOTA LAIN", PercentShare: 430_000, HasShare: true},
		},
		Reinsurer: []FacReinsurer{{Name: "BROKER A", Share: 600_000, HasShare: true}, {Name: "BROKER B", Share: 400_000, HasShare: true}},
	})

	if f.Title != "INDUSTRIAL ALL RISK" || f.Class != "IAR" || f.SumInsured.Currency != "IDR" {
		t.Fatalf("kepala salah: %+v", f)
	}
	// Reserve hanya estimasi tipe 1: 30 jt + 20 jt.
	if len(f.Reserve) != 1 || f.Reserve[0].Value != Rupiah(50_000_000) {
		t.Fatalf("reserve = %+v", f.Reserve)
	}
	// Anggota koasuransi: share × reserve.
	if f.CoMember[0].Value != Rupiah(28_500_000) || f.CoMember[1].Value != Rupiah(21_500_000) {
		t.Fatalf("co member = %+v", f.CoMember)
	}
	// Spreading: reserve × share ASM × share treaty; baris terhapus tidak dicetak.
	if len(f.Spreading) != 2 || f.Spreading[0].Name != "OR" ||
		f.Spreading[0].Value != Rupiah(1_995_000) || f.Spreading[1].Value != Rupiah(26_505_000) {
		t.Fatalf("spreading = %+v", f.Spreading)
	}
	// Fac out dibagi menurut PctShareForAllObj: 93% × 60% dan 93% × 40%.
	if f.Reinsurer[0].Share != 558_000 || f.Reinsurer[1].Share != 372_000 {
		t.Fatalf("reasuradur = %+v", f.Reinsurer)
	}
}

// Share ASM hanya dikenakan bila Sinar Mas leader.
func TestFaceSheetMemberDoesNotApplyASMShare(t *testing.T) {
	k := faceSheetClaim()
	k.Policy.Coinsurance.Role = "MEMBER"
	f := BuildFaceSheet(FaceSheetInput{Claim: k, Now: time.Now()})
	if f.Spreading[0].Value != Rupiah(3_500_000) {
		t.Fatalf("spreading member = %+v", f.Spreading[0])
	}
}

func TestCheckNewEstimates(t *testing.T) {
	item := ObjectItem{Estimation: []Estimation{{FaceSheet: false}, {}}}
	if CheckNewEstimates(item, 2) != nil {
		t.Fatalf("estimasi lama tidak ikut diperiksa")
	}
	if CheckNewEstimates(item, 1) == nil {
		t.Fatalf("estimasi baru harus menunggu CFS estimasi sebelumnya")
	}
	item.Estimation[0].FaceSheet = true
	if CheckNewEstimates(item, 1) != nil {
		t.Fatalf("estimasi sebelumnya sudah CFS")
	}
}

func TestFaceSheetFileNameMatchesPega(t *testing.T) {
	if got := FaceSheetFileName(1, 1, 0); got != "ClaimFaceSheetHTML1#1/Revisi0.pdf" {
		t.Fatalf("nama = %s", got)
	}
}
