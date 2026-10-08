package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// upToChooseSurveyor membawa klaim Fire sampai Choose Surveyor dengan estimasi klaim dan
// (bila adjuster > 0) estimasi adjuster, keduanya sudah dibuatkan Claim Face Sheet.
func (l environment) upToChooseSurveyor(t *testing.T, claim, adjuster registrasi.Money) registrasi.Task {
	t.Helper()
	ctx := context.Background()
	_, task := l.upToInputEstimate(t)
	command := oneEstimate(task.ID, claim, 1)
	if adjuster > 0 {
		// Estimasi berikutnya pada item yang sama menunggu CFS, sehingga estimasi adjuster
		// diberi item sendiri.
		command.Item[0][0] = append(command.Item[0][0], usecase.ObjectItemInput{
			Name: "Adjuster", Estimation: []usecase.EstimationInput{{
				Type: registrasi.EstimateAdjuster, Currency: "IDR", Value: adjuster,
			}},
		})
	}
	_, err := l.service.SaveEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	_, err = l.service.DownloadFaceSheet(ctx, faceSheet(task), l.caller)
	require.NoError(t, err)
	result, err := l.service.CompleteEstimate(ctx, command, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageChooseSurveyor, result.NextTask.Stage)
	return *result.NextTask
}

func addSettlement(task registrasi.Task, in registrasi.SettlementInput) usecase.SettlementCommand {
	return usecase.SettlementCommand{ClaimID: task.ClaimID, TaskID: task.ID, Object: 1, Coverage: 1, Input: in}
}

// Final: Gross = propose − resiko sendiri; nilai ASM = gross × share (polis tanpa
// koasuransi = 100%). Baris tersimpan dan terbaca kembali bersama klaim.
func TestAddSettlementComputesFinalLine(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)

	claim, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000), Submitted: registrasi.Rupiah(12_000_000),
		RiskType: registrasi.RiskOfClaim, RiskPercent: 100_000, // 10%
	}), l.caller)
	require.NoError(t, err)

	line := claim.InsuredItem[0].Coverage[0].Settlement[0]
	require.Equal(t, registrasi.Rupiah(1_000_000), line.RiskValue)
	require.Equal(t, registrasi.Rupiah(9_000_000), line.Gross)
	require.Equal(t, registrasi.PercentFull, line.ShareASM)
	require.Equal(t, registrasi.Rupiah(9_000_000), line.Value)
	require.Equal(t, "IDR", line.Currency)
	require.Empty(t, line.AcceptanceStatus)

	stored, err := l.store.Get(ctx, claim.ID)
	require.NoError(t, err)
	require.Len(t, stored.InsuredItem[0].Coverage[0].Settlement, 1)
}

// ValidationTypePaymentAdj: tipe pembayaran, nilai propose, dan tipe resiko wajib.
func TestAddSettlementRequiresFields(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)

	_, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{}), l.caller)
	violation(t, err, registrasi.ViolationSettlementPaymentType)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{PaymentType: registrasi.PaymentInterim}), l.caller)
	violation(t, err, registrasi.ViolationSettlementPropose)
	violation(t, err, registrasi.ViolationSettlementSubmitted)
	violation(t, err, registrasi.ViolationSettlementRisk)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(60_000_000), RiskType: registrasi.RiskOther, Submitted: registrasi.Rupiah(1),
	}), l.caller)
	violation(t, err, registrasi.ViolationSettlementEstimate)

	// Tolak Klaim menunggu daftar Alasan Tolak Klaim (GetDataPenolakanKlaimMas tidak ada).
	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{PaymentType: registrasi.PaymentReject}), l.caller)
	violation(t, err, registrasi.ViolationSettlementPaymentType)

	// Salvage ditambahkan dari modul Salvage, bukan dari grid ini.
	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{PaymentType: registrasi.PaymentSalvage}), l.caller)
	violation(t, err, registrasi.ViolationSettlementPaymentType)
}

// ValidTotalEstimation: seluruh adjustment jaminan tidak boleh melampaui estimasinya.
func TestAddSettlementRejectsTotalOverEstimate(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)
	in := registrasi.SettlementInput{PaymentType: registrasi.PaymentInterim, Propose: registrasi.Rupiah(30_000_000), RiskType: registrasi.RiskOther, Submitted: registrasi.Rupiah(1)}

	_, err := l.service.AddSettlement(ctx, addSettlement(task, in), l.caller)
	require.NoError(t, err)
	_, err = l.service.AddSettlement(ctx, addSettlement(task, in), l.caller)
	violation(t, err, registrasi.ViolationSettlementEstimate)
}

// SetValueAdjusterFee: gross = Professional Fee + Survey Expenses + VAT% (tipe 1 dari
// Professional Fee), dan fee tidak melampaui estimasi adjuster.
func TestAddSettlementAdjusterFee(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), registrasi.Rupiah(2_000_000))

	fee := registrasi.AdjusterFee{
		ProfessionalFee: registrasi.Rupiah(1_000_000), SurveyExpenses: registrasi.Rupiah(500_000),
		VAT: 100_000, VATType: registrasi.VATOfProfessionalFee,
	}
	claim, err := l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjusterFee, Fee: fee,
	}), l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.Rupiah(1_600_000), claim.InsuredItem[0].Coverage[0].Settlement[0].Gross)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjusterFee,
	}), l.caller)
	violation(t, err, registrasi.ViolationSettlementFee)

	_, err = l.service.AddSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentAdjusterFee, Fee: fee,
	}), l.caller)
	violation(t, err, registrasi.ViolationSettlementEstimate)
}

// Grid Adjustment hanya milik layar InputSurveyor.
func TestAddSettlementOnlyAtInputSurveyor(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputEstimate(t)
	_, err := l.service.AddSettlement(context.Background(), addSettlement(task, registrasi.SettlementInput{PaymentType: registrasi.PaymentFinal}), l.caller)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)
}

// Final dengan tipe resiko Lainnya (nilai diisi petugas) dan Salvage B: Gross = Total Klaim −
// LOC − Salvage A − Resiko − Salvage B. Pratinjau menghitung tanpa menyimpan.
func TestPreviewSettlementOtherRiskAndSalvageB(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	task := l.upToChooseSurveyor(t, registrasi.Rupiah(50_000_000), 0)

	preview, err := l.service.PreviewSettlement(ctx, addSettlement(task, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(10_000_000),
		LOC: 100_000, SalvageA: registrasi.Rupiah(500_000), SalvageB: registrasi.Rupiah(300_000),
		RiskType: registrasi.RiskOther, RiskValue: registrasi.Rupiah(200_000),
	}), l.caller)
	require.NoError(t, err)
	// 10.000.000 − 1.000.000 (LOC 10%) − 500.000 − 200.000 − 300.000
	require.Equal(t, registrasi.Rupiah(8_000_000), preview.Line.Gross)
	require.Equal(t, registrasi.Rupiah(50_000_000), preview.Line.Estimation)
	require.Equal(t, registrasi.ExchangeRateOne, preview.Line.Rate)

	stored, err := l.store.Get(ctx, task.ClaimID)
	require.NoError(t, err)
	require.Empty(t, stored.InsuredItem[0].Coverage[0].Settlement)
}

// Nilai Interim: gross Interim yang sudah diakseptasi mengurangi gross Final berikutnya.
func TestInterimPaidCountsAcceptedInterimOnly(t *testing.T) {
	c := registrasi.Coverage{Settlement: []registrasi.SettlementLine{
		{PaymentType: registrasi.PaymentInterim, Gross: registrasi.Rupiah(1_000_000), AcceptanceStatus: "1"},
		{PaymentType: registrasi.PaymentInterim, Gross: registrasi.Rupiah(2_000_000)},
		{PaymentType: registrasi.PaymentFinal, Gross: registrasi.Rupiah(4_000_000), AcceptanceStatus: "1"},
	}}
	require.Equal(t, registrasi.Rupiah(1_000_000), registrasi.InterimPaid(c))
}

// Estimation PA (Assignment4) ditutup flow action InputSurveyor — layar dan grid Adjustment yang
// sama dengan Choose Surveyor — sehingga adjustment dapat dihitung di sana. Sebelumnya tahap ini
// tidak terdaftar dan permintaan ditolak sebagai "klaim sudah berpindah tahap".
func TestPreviewSettlementOnEstimationPA(t *testing.T) {
	l := setup(t)
	ctx := context.Background()
	_, task := l.upToInputRegister(t, paPolicy)
	input := validInput(task.ID)
	input.InsuredItem[0].Coverage[0].CauseOfLoss = registrasi.CauseOfLossPA
	registered, err := l.service.SaveRegister(ctx, input, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimatePA, registered.NextTask.Stage)

	_, err = l.service.PreviewSettlement(ctx, addSettlement(*registered.NextTask, registrasi.SettlementInput{
		PaymentType: registrasi.PaymentFinal, Propose: registrasi.Rupiah(1_000_000),
	}), l.caller)
	require.NotErrorIs(t, err, registrasi.ErrStageMismatch)
	require.NotErrorIs(t, err, registrasi.ErrNotAvailableAtStage)
}
