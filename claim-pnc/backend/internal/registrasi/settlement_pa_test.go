package registrasi_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// PA (InputAdjustment_sect, kontainer IsPA): "Nilai Pengajuan" wajib; "Nilai Propose Adjustment" hanya
// pada baris isAnalistorTransfer (jaminan PHK). Tanpanya Total Klaim dan Nilai Adjustment nol.
func TestNewSettlementLinePA(t *testing.T) {
	cov := coverageWithEstimates()
	cov.ID = "10003"
	sc := registrasi.SettlementContext{Claim: faceSheetClaim(registrasi.LinePersonalAccident, cov), Coverage: cov,
		Rate: registrasi.ExchangeRateOne, TSIRate: registrasi.ExchangeRateOne}

	// Tanpa Nilai Pengajuan: wajib; Nilai Propose Adjustment tidak diminta.
	_, err := registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: registrasi.PaymentInterim, RiskType: registrasi.RiskOther}, sc)
	codes := violationCodes(t, err)
	require.Contains(t, codes, registrasi.ViolationSettlementSubmitted)
	require.NotContains(t, codes, registrasi.ViolationSettlementPropose)

	// Nilai Pengajuan saja: sah; nilainya nol.
	line, err := registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentInterim, Submitted: registrasi.Rupiah(5_000_000),
		RiskType: registrasi.RiskOther, RiskValue: registrasi.Rupiah(2_500_000),
	}, sc)
	require.NoError(t, err)
	require.Zero(t, line.Gross)
	require.Zero(t, line.Value)

	// Analyst: Nilai Pengajuan nonaktif (IsAnalisator), sehingga tidak wajib.
	scAnalyst := sc
	scAnalyst.Analyst = true
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{PaymentType: registrasi.PaymentInterim, RiskType: registrasi.RiskOther}, scAnalyst)
	require.NoError(t, err)

	// Jaminan yang sudah ditransfer ke Analyst (setTicketToAnalyst): Total Klaim wajib.
	transferred := cov
	transferred.AnalystTransferred = true
	scTransferred := sc
	scTransferred.Coverage = transferred
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentInterim, Submitted: registrasi.Rupiah(1_000_000), RiskType: registrasi.RiskOther,
	}, scTransferred)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementPropose)

	// Jaminan PHK: Nilai Propose Adjustment wajib dan tidak boleh melebihi Nilai Pengajuan.
	phk := cov
	phk.ID = "10010"
	scPHK := sc
	scPHK.Coverage = phk
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentInterim, Submitted: registrasi.Rupiah(1_000_000), RiskType: registrasi.RiskOther,
	}, scPHK)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementPropose)
	_, err = registrasi.NewSettlementLine(registrasi.SettlementInput{
		PaymentType: registrasi.PaymentInterim, Submitted: registrasi.Rupiah(1_000_000), Propose: registrasi.Rupiah(2_000_000),
		RiskType: registrasi.RiskOther,
	}, scPHK)
	require.Contains(t, violationCodes(t, err), registrasi.ViolationSettlementPropose)
}
