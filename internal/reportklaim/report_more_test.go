package reportklaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

func TestGroupLabelNamesEveryGroup(t *testing.T) {
	require.Equal(t, "Klaim", reportklaim.GroupLabel(reportklaim.GroupKlaim))
	require.Equal(t, "PLA / DLA", reportklaim.GroupLabel(reportklaim.GroupReasuransi))
	require.Equal(t, "Akseptasi & Penyelesaian", reportklaim.GroupLabel(reportklaim.GroupPenyelesaian))
	require.Equal(t, "Lini Bisnis & Mitra Kerja Sama", reportklaim.GroupLabel(reportklaim.GroupLiniBisnis))
	require.Equal(t, "Operasional", reportklaim.GroupLabel(reportklaim.GroupOperasional))
	require.Equal(t, "lain", reportklaim.GroupLabel("lain"), "grup tak dikenal ditampilkan apa adanya")
}

func TestHeadersAndFieldsKeepColumnOrder(t *testing.T) {
	columns := []reportklaim.Column{{Header: "No Klaim", Field: "CaseID"}, {Header: "Polis", Field: "ClaimNo"}}
	require.Equal(t, []string{"No Klaim", "Polis"}, reportklaim.Headers(columns))
	require.Equal(t, []string{"CaseID", "ClaimNo"}, reportklaim.Fields(columns))
	require.Empty(t, reportklaim.Headers(nil))
}

func TestVariantReasonFollowsSelectedColumns(t *testing.T) {
	single, multi := 0, 0
	for _, report := range reportklaim.CatalogInPegaOrder() {
		if report.HasSingleColumnSet() {
			single++
			continue
		}
		multi++
		// Laporan bervarian memilih susunan kolom menurut penyaring; setiap pilihan yang
		// menghasilkan kolom juga menyebut alasannya.
		for _, line := range reportklaim.BusinessLineOptions() {
			for _, detail := range []bool{false, true} {
				filter := reportklaim.Filter{BusinessLine: line.Value, Detail: detail}
				if len(report.Columns(filter)) > 0 {
					require.NotPanics(t, func() { _ = report.VariantReason(filter) })
				}
			}
		}
	}
	require.Positive(t, single)
	require.Positive(t, multi, "katalog memuat laporan bervarian")
	require.Empty(t, reportklaim.Report{}.VariantReason(reportklaim.Filter{}), "laporan tanpa varian tidak punya alasan")
	require.Nil(t, reportklaim.Report{}.Columns(reportklaim.Filter{}))
}

func TestParseBusinessLineAcceptsKnownCodesOnly(t *testing.T) {
	for _, line := range []reportklaim.BusinessLine{reportklaim.BusinessLineAll, reportklaim.BusinessLinePA,
		reportklaim.BusinessLineTravel, reportklaim.BusinessLineNonMBU, reportklaim.BusinessLineBonding} {
		parsed, ok := reportklaim.ParseBusinessLine(" " + string(line) + " ")
		require.True(t, ok, string(line))
		require.Equal(t, line, parsed)
	}
	parsed, ok := reportklaim.ParseBusinessLine("999")
	require.False(t, ok)
	require.Equal(t, reportklaim.BusinessLineAll, parsed)
}

func TestBusinessLineOptionsAreNamed(t *testing.T) {
	names := map[reportklaim.BusinessLine]string{}
	for _, option := range reportklaim.BusinessLineOptions() {
		names[option.Value] = option.Name
	}
	require.Equal(t, "Non-MBU", names[reportklaim.BusinessLineNonMBU])
	require.Equal(t, "Bonding", names[reportklaim.BusinessLineBonding])
}

func TestRowValueOnNilRow(t *testing.T) {
	var row reportklaim.Row
	require.Empty(t, row.Value("x"))
	require.Equal(t, "1", reportklaim.Row{"x": "1"}.Value("x"))
}

func TestValidationErrorListsEveryViolation(t *testing.T) {
	err := &reportklaim.ValidationError{Violation: []reportklaim.Violation{
		{Field: "dari", Message: "wajib"}, {Field: "sampai", Message: "salah"},
	}}
	require.EqualError(t, err, "reportklaim: isian tidak sah (dari: wajib; sampai: salah)")
}

func TestMasterCountsOnUnavailableAndLoadedValues(t *testing.T) {
	require.Zero(t, reportklaim.UnavailableFeeScale().Count())
	require.Equal(t, 2, reportklaim.NewFeeScale([]reportklaim.FeeBand{{Index: 1}, {Index: 2}}).Count())
	require.Zero(t, reportklaim.UnavailableProgressNames().Count())
	require.Equal(t, 1, reportklaim.NewProgressNames(map[string]string{"1": "Register"}).Count())
	require.Zero(t, reportklaim.UnavailableDominantFactors().Count())

	var nilScale *reportklaim.FeeScale
	require.Zero(t, nilScale.Count())
	var nilProgress *reportklaim.ProgressNames
	require.Zero(t, nilProgress.Count())
	var nilFactors *reportklaim.DominantFactors
	require.Zero(t, nilFactors.Count())
}
