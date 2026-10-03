package registrasi_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

var at = time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC)

func violationCodes(t *testing.T, err error) []registrasi.ViolationCode {
	t.Helper()
	var v *registrasi.ValidationError
	require.True(t, errors.As(err, &v), "galat = %v", err)
	out := []registrasi.ViolationCode{}
	for _, p := range v.Violation {
		out = append(out, p.Code)
	}
	return out
}

// Tugas worklist langsung bertuan; tugas workbasket menunggu diambil.
func TestTaskLifecycle(t *testing.T) {
	claim := registrasi.Claim{ID: "K1", Number: "PNCN.26.1"}
	wb := registrasi.NewTask("T1", claim, registrasi.Stage{ID: "S", Queue: registrasi.QueueWorkbasket, Workbasket: "WB"},
		registrasi.Assignee{Operator: "ABAIKAN"}, at)
	require.Equal(t, "WB", wb.Workbasket)
	require.Empty(t, wb.Owner)
	require.Nil(t, wb.ClaimedAt)
	require.True(t, wb.Open())
	require.False(t, wb.Owned())

	// Tugas tanpa pemilik tidak dapat ditutup sebelum diambil.
	require.ErrorIs(t, wb.Complete("NIK1", "x", at), registrasi.ErrNotTaskOwner)
	require.NoError(t, wb.Get("NIK1", at))
	require.Equal(t, "NIK1", wb.Owner)
	require.NoError(t, wb.Get("NIK1", at), "mengambil ulang tugas sendiri sah")
	require.ErrorIs(t, wb.Get("LAIN", at), registrasi.ErrTaskAlreadyClaimed)
	require.ErrorIs(t, wb.Complete("LAIN", "x", at), registrasi.ErrNotTaskOwner)
	require.NoError(t, wb.Complete("NIK1", "Selesai", at))
	require.False(t, wb.Open())
	require.Equal(t, "Selesai", wb.CompletionReason)
	require.ErrorIs(t, wb.Complete("NIK1", "x", at), registrasi.ErrTaskAlreadyDone)
	require.ErrorIs(t, wb.Get("NIK1", at), registrasi.ErrTaskAlreadyDone)

	wl := registrasi.NewTask("T2", claim, registrasi.Stage{ID: "S", Queue: registrasi.QueueWorklist},
		registrasi.Assignee{Operator: "NIK1"}, at)
	require.Equal(t, "NIK1", wl.Owner)
	require.NotNil(t, wl.ClaimedAt)
	require.Equal(t, "PNCN.26.1", wl.ClaimNumber)
}

func TestInboxEntry(t *testing.T) {
	flow := registrasi.RegisterFlow()
	task := registrasi.Task{Stage: registrasi.StageViewPolicy, Owner: "NIK1"}
	e := registrasi.NewInboxEntry(registrasi.Claim{Number: "PNCN.26.1"}, &task, flow)
	require.Equal(t, "PNCN.26.1", e.Key())
	require.NotEmpty(t, e.StageName)
	require.Equal(t, "NIK1", e.AssignedOperator())
	require.Equal(t, registrasi.WorkStatusNew, e.WorkStatus())
	require.Equal(t, "1", e.Active())

	unknown := registrasi.NewInboxEntry(registrasi.Claim{}, &registrasi.Task{Stage: "TIDAK-ADA"}, flow)
	require.Empty(t, unknown.StageName)

	none := registrasi.NewInboxEntry(registrasi.Claim{ProcessStatus: registrasi.ProcessDone, DeletedAt: &at}, nil, flow)
	require.Empty(t, none.AssignedOperator())
	require.Equal(t, registrasi.WorkStatusCompleted, none.WorkStatus())
	require.Equal(t, "0", none.Active())
	rejected := registrasi.NewInboxEntry(registrasi.Claim{ProcessStatus: registrasi.ProcessRejected}, nil, flow)
	require.Equal(t, registrasi.WorkStatusRejected, rejected.WorkStatus())
}

func TestReceiverHelpers(t *testing.T) {
	r := registrasi.DefaultReceiver(registrasi.Policy{QQName: " QQ ", InsuredName: "Tertanggung", DeliveryAddress: " JL "})
	require.Equal(t, registrasi.Receiver{ID: "1", Name: "QQ", Address: "JL"}, r)
	r = registrasi.DefaultReceiver(registrasi.Policy{InsuredName: " Tertanggung "})
	require.Equal(t, "Tertanggung", r.Name)

	r.ApplyAccount(registrasi.BankAccount{Number: " 123 ", Name: " PT A ", BankName: " BANK ", Address: " ALAMAT "})
	require.Equal(t, registrasi.Receiver{ID: "1", Name: "PT A", Address: "ALAMAT", BankName: "BANK", AccountNo: "123"}, r)

	require.Equal(t, "1", registrasi.NextReceiverID(nil))
	require.Equal(t, "8", registrasi.NextReceiverID([]registrasi.Receiver{{ID: " 7 "}, {ID: "x"}, {ID: "2"}}))
}

func TestPaymentTypeNameAndShare(t *testing.T) {
	for code, name := range map[string]string{
		registrasi.PaymentFinal: "Final", registrasi.PaymentInterim: "Interim", registrasi.PaymentSalvage: "Salvage",
		registrasi.PaymentAdjusterFee: "Adjuster Fee", registrasi.PaymentAdjustment: "Adjustment",
		registrasi.PaymentReject: "Tolak Klaim", "9": "9",
	} {
		require.Equal(t, name, registrasi.PaymentTypeName(code))
	}
	require.Equal(t, registrasi.PercentFull, registrasi.ShareASMOf(registrasi.Policy{}))
	require.Equal(t, registrasi.Percent(600_000),
		registrasi.ShareASMOf(registrasi.Policy{Coinsurance: registrasi.Coinsurance{ShareASM: 600_000, HasShare: true}}))

	require.False(t, registrasi.SettlementLine{}.Transferred())
	require.True(t, registrasi.SettlementLine{CommitteeCaseID: " KM "}.Transferred())
}

func TestAdjusterFeeGross(t *testing.T) {
	fee := registrasi.AdjusterFee{ProfessionalFee: 1_000_00, SurveyExpenses: 500_00, VAT: 100_000}
	require.Equal(t, registrasi.Money(1_500_00), fee.Gross(), "tanpa tipe VAT")
	fee.VATType = registrasi.VATOfProfessionalFee
	require.Equal(t, registrasi.Money(1_600_00), fee.Gross())
	fee.VATType = registrasi.VATOfSubtotal
	require.Equal(t, registrasi.Money(1_650_00), fee.Gross())
}

func coverageWithEstimates() registrasi.Coverage {
	return registrasi.Coverage{
		TSI: registrasi.Rupiah(100_000_000),
		Item: []registrasi.ObjectItem{{Estimation: []registrasi.Estimation{
			{Type: registrasi.EstimateClaim, Value: registrasi.Rupiah(10_000_000), Converted: registrasi.Rupiah(10_000_000), FaceSheet: true},
			{Type: registrasi.EstimateClaim, Value: registrasi.Rupiah(5_000_000), Converted: registrasi.Rupiah(5_000_000)},
			{Type: registrasi.EstimateAdjuster, Value: registrasi.Rupiah(1_000_000), Converted: registrasi.Rupiah(1_000_000)},
		}}},
		Settlement: []registrasi.SettlementLine{
			{PaymentType: registrasi.PaymentInterim, AcceptanceStatus: "1", Gross: registrasi.Rupiah(1_000_000)},
			{PaymentType: registrasi.PaymentInterim, AcceptanceStatus: "0", Gross: registrasi.Rupiah(9)},
			{PaymentType: registrasi.PaymentFinal, AcceptanceStatus: "1", Gross: registrasi.Rupiah(9)},
		},
	}
}

func TestCoverageEstimateAndInterim(t *testing.T) {
	c := coverageWithEstimates()
	v, idr := registrasi.CoverageEstimate(c, registrasi.EstimateClaim)
	require.Equal(t, registrasi.Rupiah(15_000_000), v)
	require.Equal(t, registrasi.Rupiah(15_000_000), idr)
	v, _ = registrasi.CoverageEstimate(c, registrasi.EstimateAdjuster)
	require.Equal(t, registrasi.Rupiah(1_000_000), v)
	require.Equal(t, registrasi.Rupiah(1_000_000), registrasi.InterimPaid(c))
	require.Equal(t, registrasi.Rupiah(15_000_000), c.Item[0].SumConverted())
}

// Perhitungan baris: tiga tipe resiko sendiri, interim Non-MBU, dan aturan Travel.
func TestComputeSettlementLine(t *testing.T) {
	cov := coverageWithEstimates()
	fire := registrasi.Claim{Policy: registrasi.Policy{Line: registrasi.LineFire, Currency: "IDR"}}
	sc := registrasi.SettlementContext{Claim: fire, Coverage: cov, Rate: registrasi.ExchangeRateOne, Now: at}

	in := registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), LOC: 100_000,
		SalvageA: registrasi.Rupiah(1_000_000), SalvageB: registrasi.Rupiah(500_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000, Chronology: " k ", Notes: " n ",
	}
	line := registrasi.ComputeSettlementLine(in, sc)
	// LOC 10% = 1.000.000; resiko = (10jt − 1jt − 1jt) × 10% = 800.000.
	require.Equal(t, registrasi.Rupiah(800_000), line.RiskValue)
	require.Equal(t, registrasi.Rupiah(1_000_000), line.Interim)
	// Gross = 10jt − 1jt − 1jt − 800rb − 500rb salvage B − 1jt interim.
	require.Equal(t, registrasi.Rupiah(5_700_000), line.Gross)
	require.Equal(t, line.Gross, line.Value)
	require.Equal(t, "IDR", line.Currency)
	require.Equal(t, "k", line.Chronology)
	require.Equal(t, registrasi.Rupiah(15_000_000), line.Estimation)

	in.RiskType = registrasi.RiskOfTSI
	line = registrasi.ComputeSettlementLine(in, sc)
	require.Equal(t, registrasi.Rupiah(10_000_000), line.RiskValue)

	in.RiskType = registrasi.RiskOther
	in.RiskValue = registrasi.Rupiah(123)
	in.PaymentType = registrasi.PaymentAdjustment
	line = registrasi.ComputeSettlementLine(in, sc)
	require.Equal(t, registrasi.Rupiah(123), line.RiskValue)
	// Adjustment tidak dikurangi salvage B maupun interim.
	require.Equal(t, registrasi.Rupiah(10_000_000-1_000_000-1_000_000-123), line.Gross)

	// Travel: resiko melebihi propose membuat gross nol; Travel bukan Non-MBU, tanpa interim.
	travel := sc
	travel.Claim.Policy.Line = registrasi.LineTravel
	line = registrasi.ComputeSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(100), RiskType: registrasi.RiskOther,
		RiskValue: registrasi.Rupiah(500), Currency: "USD",
	}, travel)
	require.Zero(t, line.Gross)
	require.Zero(t, line.Interim)
	require.Equal(t, "USD", line.Currency)

	// Fee adjuster dan koasuransi tipe 1: nilai akseptasi mengikuti bagian ASM.
	coins := sc
	coins.Claim.Policy.TypeOfCoins = "1"
	coins.Claim.Policy.Coinsurance = registrasi.Coinsurance{ShareASM: 500_000, HasShare: true}
	line = registrasi.ComputeSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjusterFee,
		Fee:         registrasi.AdjusterFee{ProfessionalFee: registrasi.Rupiah(200)},
	}, coins)
	require.Equal(t, registrasi.Rupiah(200), line.Gross)
	require.Equal(t, registrasi.Rupiah(100), line.Value)
	require.Equal(t, line.Value, line.Accepted)
	require.Equal(t, registrasi.Rupiah(1_000_000), line.Estimation)
}

func faceSheetClaim(line registrasi.LineOfBusiness, cov registrasi.Coverage) registrasi.Claim {
	return registrasi.Claim{
		Policy:      registrasi.Policy{Line: line, Currency: "IDR"},
		InsuredItem: []registrasi.InsuredItem{{Coverage: []registrasi.Coverage{cov}}},
	}
}

func TestNewSettlementLineRejections(t *testing.T) {
	cov := coverageWithEstimates()
	sc := registrasi.SettlementContext{Claim: faceSheetClaim(registrasi.LineFire, cov), Coverage: cov,
		Rate: registrasi.ExchangeRateOne, TSIRate: registrasi.ExchangeRateOne}

	for _, pt := range []string{"", registrasi.PaymentSalvage, registrasi.PaymentReject, "9"} {
		_, err := registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: pt}, sc)
		require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationSettlementPaymentType}, violationCodes(t, err), pt)
	}

	noSheet := sc
	noSheet.Claim = faceSheetClaim(registrasi.LineFire, registrasi.Coverage{})
	_, err := registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: registrasi.PaymentFinal}, noSheet)
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationSettlementFaceSheet}, violationCodes(t, err))

	// Isian kosong: propose, pengajuan, dan tipe resiko sekaligus.
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: registrasi.PaymentFinal}, sc)
	codes := violationCodes(t, err)
	require.Contains(t, codes, registrasi.ViolationSettlementPropose)
	require.Contains(t, codes, registrasi.ViolationSettlementSubmitted)
	require.Contains(t, codes, registrasi.ViolationSettlementRisk)

	// Persen resiko kosong, resiko melebihi propose, dan melebihi estimasi/TSI.
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(1), Submitted: 1, RiskType: registrasi.RiskOfTSI,
	}, sc)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementRisk)
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(1), Submitted: 1, RiskType: registrasi.RiskOther,
		RiskValue: registrasi.Rupiah(5),
	}, sc)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementRisk)
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(200_000_000), Submitted: 1, RiskType: registrasi.RiskOther,
	}, sc)
	codes = violationCodes(t, err)
	require.Contains(t, codes, registrasi.ViolationSettlementEstimate)
	require.Contains(t, codes, registrasi.ViolationSettlementTSI)

	// Adjustment negatif melebihi estimasi.
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjustment, Propose: registrasi.Rupiah(1), Submitted: 1, RiskType: registrasi.RiskOther,
		RiskValue: registrasi.Rupiah(0), SalvageA: registrasi.Rupiah(20_000_000),
	}, sc)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementEstimate)

	// Fee adjuster: fee nol, dan total fee melebihi estimasi adjuster (koasuransi tipe 2 dibagi share).
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: registrasi.PaymentAdjusterFee}, sc)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementFee)
	over := sc
	over.Claim.Policy.TypeOfCoins = "2"
	over.Coverage.Settlement = append(over.Coverage.Settlement,
		registrasi.SettlementLine{PaymentType: registrasi.PaymentAdjusterFee, Value: registrasi.Rupiah(900_000), Rate: registrasi.ExchangeRateOne})
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjusterFee, Fee: registrasi.AdjusterFee{ProfessionalFee: registrasi.Rupiah(5_000_000)},
	}, over)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementEstimate)

	// Baris sah.
	line, err := registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(5_000_000), Submitted: 1,
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000,
	}, sc)
	require.NoError(t, err)
	require.Equal(t, registrasi.Rupiah(4_500_000-1_000_000), line.Gross)
}

func TestEstimationValidation(t *testing.T) {
	empty := registrasi.Claim{}
	require.Equal(t, 0, empty.EstimationCount())
	require.False(t, empty.HasFaceSheet())
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationEstimateMissing}, violationCodes(t, registrasi.ValidateEstimate(empty)))

	noSheet := registrasi.Claim{InsuredItem: []registrasi.InsuredItem{{Coverage: []registrasi.Coverage{{
		Spreading: []registrasi.Spreading{{TreatyKind: "10001", Share: registrasi.PercentFull}},
		Item:      []registrasi.ObjectItem{{Estimation: []registrasi.Estimation{{Value: 0}}}},
	}}}}}
	codes := violationCodes(t, registrasi.ValidateEstimate(noSheet))
	require.Contains(t, codes, registrasi.ViolationSendNeedsFaceSheet)
	require.Contains(t, codes, registrasi.ViolationEstimateMissing)

	noSpreading := registrasi.Claim{InsuredItem: []registrasi.InsuredItem{{Coverage: []registrasi.Coverage{{
		Item: []registrasi.ObjectItem{{Estimation: []registrasi.Estimation{{Value: 1, FaceSheet: true}}}},
	}}}}}
	require.True(t, noSpreading.HasFaceSheet())
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationNoSpreading}, violationCodes(t, registrasi.ValidateEstimate(noSpreading)))

	ok := noSpreading
	ok.InsuredItem[0].Coverage[0].Spreading = []registrasi.Spreading{{TreatyKind: "10001", Share: registrasi.PercentFull}}
	require.NoError(t, registrasi.ValidateEstimate(ok))
}

// Komite: giliran berjenjang, penolakan menutup seluruh jenjang, keputusan asing ditolak.
func TestCommitteeCaseDecide(t *testing.T) {
	approvers := []registrasi.CommitteeApprover{{OperatorID: " KOMITE01 "}, {OperatorID: "KOMITE02"}}
	line := registrasi.SettlementLine{PaymentType: "1", Currency: "IDR", Value: 900, ShareASM: registrasi.PercentFull}
	c := registrasi.NewCommitteeCase("KM.26.1", "PNCN.26.1", approvers, line, 1_000, at)
	require.Len(t, c.Members, 2)
	require.Equal(t, "KOMITE01", c.Members[0].Operator)
	require.Equal(t, 2, c.Members[1].Level)
	require.Equal(t, registrasi.CommitteeCaseOpen, c.Status())
	require.Equal(t, 1, c.Level())
	require.Empty(t, c.Outcome())
	current, ok := c.Current()
	require.True(t, ok)
	require.Equal(t, "KOMITE01", current.Operator)

	require.ErrorIs(t, c.Decide("KOMITE01", "9", "", at), registrasi.ErrInvalidAction)
	require.ErrorIs(t, c.Decide("KOMITE02", registrasi.DecisionApprove, "", at), registrasi.ErrNotCommitteeTurn)
	require.NoError(t, c.Decide("komite01", registrasi.DecisionApprove, " ok ", at))
	require.Equal(t, "ok", c.Members[0].Note)
	require.Equal(t, 2, c.Level())
	require.NoError(t, c.Decide("KOMITE02", registrasi.DecisionApprove, "", at))
	require.Equal(t, registrasi.DecisionApprove, c.Outcome())
	require.Equal(t, registrasi.CommitteeCaseClosed, c.Status())
	require.Equal(t, 2, c.Level(), "tanpa giliran, jenjang = jumlah anggota")
	_, ok = c.Current()
	require.False(t, ok)

	r := registrasi.NewCommitteeCase("KM.26.2", "PNCN.26.1", approvers, line, 1_000, at)
	require.NoError(t, r.Decide("KOMITE01", registrasi.DecisionReject, "tidak", at))
	require.Equal(t, registrasi.DecisionReject, r.Outcome())
	require.Equal(t, registrasi.DecisionReject, r.Members[1].Decision)
	require.Equal(t, registrasi.CommitteeCaseClosed, r.Members[1].CaseStatus)

	require.Empty(t, registrasi.CommitteeCase{}.Outcome())
}

func TestCommitteeLineAndValue(t *testing.T) {
	for kind, want := range map[string]string{"Bonding": "BONDING", "CustomBond": "BONDING", "Travel": "TRAVEL", "PA": "PA"} {
		require.Equal(t, want, registrasi.CommitteeLine(registrasi.Policy{BusinessType: kind}, 1))
	}
	require.Equal(t, "BONDING", registrasi.CommitteeLine(registrasi.Policy{BusinessCode: "10145"}, 1))
	require.Equal(t, registrasi.CommitteeLineNonMBUAB, registrasi.CommitteeLine(registrasi.Policy{}, registrasi.Rupiah(50_000_000)))
	require.Equal(t, registrasi.CommitteeLineNonMBU, registrasi.CommitteeLine(registrasi.Policy{}, registrasi.Rupiah(50_000_001)))

	reject := registrasi.CommitteeValue(registrasi.SettlementLine{PaymentType: registrasi.PaymentReject}, registrasi.Policy{})
	require.Equal(t, registrasi.Rupiah(500_000_001), reject)
	require.Equal(t, reject, registrasi.CommitteeValue(registrasi.SettlementLine{Propose: 1, RiskValue: 2}, registrasi.Policy{}))

	one := registrasi.ExchangeRateOne
	// Nilai dibulatkan ke atas ke rupiah penuh, tanda negatif dibuang.
	require.Equal(t, registrasi.Money(1_100), registrasi.CommitteeValue(registrasi.SettlementLine{Value: -1_050, Rate: one, Propose: 5}, registrasi.Policy{}))
	require.Equal(t, registrasi.Money(700), registrasi.CommitteeValue(registrasi.SettlementLine{PaymentType: registrasi.PaymentAdjusterFee, Gross: 700, Value: 1, Rate: one}, registrasi.Policy{}))
	leader := registrasi.Policy{Coinsurance: registrasi.Coinsurance{Role: " leader "}}
	require.Equal(t, registrasi.Money(800), registrasi.CommitteeValue(registrasi.SettlementLine{Gross: 800, Value: 400, Rate: one, Propose: 5}, leader))
}

func approvedLine() registrasi.SettlementLine {
	return registrasi.SettlementLine{PaymentType: registrasi.PaymentFinal, AcceptanceStatus: registrasi.DecisionApprove, Value: 1_000}
}

func TestCanAcceptAndAttachment(t *testing.T) {
	fire := registrasi.Policy{Line: registrasi.LineFire}
	require.NoError(t, registrasi.CanAccept(approvedLine(), fire))
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationAcceptanceNotAllowed}, violationCodes(t, registrasi.CanAccept(registrasi.SettlementLine{}, fire)))
	disagreed := approvedLine()
	disagreed.AcceptanceLODStatus = registrasi.LODDisagreed
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationAcceptanceNotAllowed}, violationCodes(t, registrasi.CanAccept(disagreed, fire)))
	numbered := approvedLine()
	numbered.AcceptedNo = "A1"
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationAcceptanceNumbered}, violationCodes(t, registrasi.CanAccept(numbered, fire)))
	// PA tidak lagi ditahan: akseptasinya berlanjut ke Transfer Kasir (SetAdjustmentAcceptation langkah 113).
	require.NoError(t, registrasi.CanAccept(approvedLine(), registrasi.Policy{Line: registrasi.LinePersonalAccident}))

	require.True(t, registrasi.AcceptanceAttachmentRequired(approvedLine(), fire))
	for _, p := range []registrasi.Policy{
		{Line: registrasi.LineTravel}, {Line: registrasi.LinePersonalAccident}, {BusinessType: "BondingKBG"},
		{BusinessCode: "10168"}, {BusinessName: " asuransi kredit "},
	} {
		require.False(t, registrasi.AcceptanceAttachmentRequired(approvedLine(), p), "%+v", p)
	}
	for _, pt := range []string{registrasi.PaymentSalvage, registrasi.PaymentAdjusterFee, "7"} {
		require.False(t, registrasi.AcceptanceAttachmentRequired(registrasi.SettlementLine{PaymentType: pt}, fire), pt)
	}
}

func TestAcceptanceComparedValue(t *testing.T) {
	v, ok := registrasi.AcceptanceComparedValue(registrasi.SettlementLine{PaymentType: registrasi.PaymentInterim, Value: 5})
	require.True(t, ok)
	require.Equal(t, registrasi.Money(5), v)
	v, _ = registrasi.AcceptanceComparedValue(registrasi.SettlementLine{PaymentType: registrasi.PaymentSalvage, SalvageB: 6})
	require.Equal(t, registrasi.Money(6), v)
	v, _ = registrasi.AcceptanceComparedValue(registrasi.SettlementLine{PaymentType: "7", Gross: 7})
	require.Equal(t, registrasi.Money(7), v)
	_, ok = registrasi.AcceptanceComparedValue(registrasi.SettlementLine{PaymentType: registrasi.PaymentReject})
	require.False(t, ok)
}

func TestValidateAcceptance(t *testing.T) {
	fire := registrasi.Policy{Line: registrasi.LineFire}
	receivers := []registrasi.Receiver{{ID: "1", Name: "Budi", AccountNo: "123", BankName: "BANK"}}
	form := registrasi.AcceptanceForm{
		LODStatus: registrasi.LODAgreed, ReceiveDate: at, PayableDate: at, LODValue: 500, HasLODValue: true, ReceiverID: "1",
	}
	r, err := registrasi.ValidateAcceptance(approvedLine(), fire, form, registrasi.AcceptanceCheck{Receivers: receivers, Files: 1, Location: "Jakarta"})
	require.NoError(t, err)
	require.Equal(t, "Budi", r.Name)

	// Isian kosong, LOD melebihi nilai, penerima tidak lengkap, lampiran kosong, DLA lain belum beres.
	bad := registrasi.AcceptanceForm{LODValue: 5_000, HasLODValue: true, ReceiverID: "2"}
	_, err = registrasi.ValidateAcceptance(approvedLine(), fire, bad, registrasi.AcceptanceCheck{
		Receivers: receivers,
		OtherDLA:  []registrasi.AcceptanceDLAState{{Number: "D1"}, {Number: "D2", Printed: true}, {Number: "D3", Printed: true, Sent: true}},
	})
	codes := violationCodes(t, err)
	require.Contains(t, codes, registrasi.ViolationAcceptanceRequired)
	require.Contains(t, codes, registrasi.ViolationAcceptanceLODValue)
	require.Contains(t, codes, registrasi.ViolationAcceptanceReceiver)
	require.Contains(t, codes, registrasi.ViolationAcceptanceAttachment)
	var v *registrasi.ValidationError
	require.True(t, errors.As(err, &v))
	messages := []string{}
	for _, p := range v.Violation {
		messages = append(messages, p.Message)
	}
	require.Contains(t, messages, "No DLA D1 belum diprint")
	require.Contains(t, messages, "No DLA D2 belum dikirim")

	// Non-MBU tanpa nilai LOD, dan Travel tanpa tanggal tetap sah.
	_, err = registrasi.ValidateAcceptance(approvedLine(), fire, registrasi.AcceptanceForm{LODStatus: "1", ReceiveDate: at, PayableDate: at, ReceiverID: "1"},
		registrasi.AcceptanceCheck{Receivers: receivers, Files: 1, Location: "Jakarta"})
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationAcceptanceRequired}, violationCodes(t, err))
	_, err = registrasi.ValidateAcceptance(approvedLine(), registrasi.Policy{Line: registrasi.LineTravel},
		registrasi.AcceptanceForm{LODStatus: "0", ReceiverID: "1", LODValue: 9_999, HasLODValue: true},
		registrasi.AcceptanceCheck{Receivers: receivers})
	require.NoError(t, err)
}

func TestAcceptHelpers(t *testing.T) {
	line := approvedLine()
	line.Accept(registrasi.Acceptance{Form: registrasi.AcceptanceForm{LODStatus: " 1 "}, ReceiverName: "Budi"}, "A26")
	require.Equal(t, "A26", line.AcceptedNo)
	require.Equal(t, "1", line.AcceptanceLODStatus)
	require.Equal(t, "Budi", line.Acceptance.ReceiverName)

	require.Equal(t, "Claim Accepted with No A26", registrasi.AcceptanceHistoryNote("A26"))
	require.Contains(t, registrasi.ReceiverIncompleteMessage(" Budi "), "Atas Nama Budi Data Tidak Lengkap")
	require.Equal(t, []registrasi.ViolationCode{registrasi.ViolationAcceptancePremium}, violationCodes(t, registrasi.ErrAcceptancePremiumUnpaid()))
}

func TestLineIsNonMBU(t *testing.T) {
	for _, l := range []registrasi.LineOfBusiness{registrasi.LineMiscellaneous, registrasi.LineMarineCargo, registrasi.LineFire, "009"} {
		require.True(t, l.IsNonMBU(), l)
	}
	require.False(t, registrasi.LineTravel.IsNonMBU())
	require.False(t, registrasi.LinePersonalAccident.IsNonMBU())
}
