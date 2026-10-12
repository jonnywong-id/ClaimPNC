package monitoringslinkojk_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
)

// Uji di berkas ini melengkapi monitoringslinkojk_test.go: katalog kolom untuk segmen yang
// tidak dikenal, galat validasi, dan aturan penyusunan baris laporan.

func TestSegmentLabel(t *testing.T) {
	require.Equal(t, "Segment Slik D01", monitoringslinkojk.SegmentD01.Label())
	require.Equal(t, "Segment Slik F06", monitoringslinkojk.SegmentF06.Label())
	require.Equal(t, "X99", monitoringslinkojk.Segment("X99").Label())
}

func TestBusinessScopesFollowOldScreenOrder(t *testing.T) {
	require.Equal(t, []monitoringslinkojk.BusinessScope{
		monitoringslinkojk.ScopeCreditInsurance, monitoringslinkojk.ScopeSuretyBond,
	}, monitoringslinkojk.BusinessScopes())
}

func TestCallerClean(t *testing.T) {
	require.Equal(t, monitoringslinkojk.Caller{Login: "PELAPOR"},
		monitoringslinkojk.Caller{Login: "  PELAPOR \t"}.Clean())
}

func TestCatalogForUnknownSegment(t *testing.T) {
	asing := monitoringslinkojk.Segment("X99")
	require.Nil(t, monitoringslinkojk.Columns(asing))
	require.Nil(t, monitoringslinkojk.ExportHeaders(asing))
	require.Nil(t, monitoringslinkojk.ExportSlots(asing))
	require.Equal(t, "Laporan SLIK OJK", monitoringslinkojk.FileName(asing))
}

func TestKeysFollowColumnOrder(t *testing.T) {
	cols := monitoringslinkojk.Columns(monitoringslinkojk.SegmentD01)
	keys := monitoringslinkojk.Keys(cols)
	require.Len(t, keys, len(cols))
	for i, c := range cols {
		require.Equal(t, c.Key, keys[i])
	}
}

func TestNewValidationErrorNilWithoutViolations(t *testing.T) {
	require.NoError(t, monitoringslinkojk.NewValidationError(nil))

	err := monitoringslinkojk.NewValidationError([]monitoringslinkojk.Violation{
		{Field: "a", Message: "satu"}, {Field: "b", Message: "dua"},
	})
	var ve *monitoringslinkojk.ValidationError
	require.True(t, errors.As(err, &ve))
	require.Len(t, ve.Violations, 2)
	require.Equal(t, "monitoringslinkojk: validasi gagal — a: satu; b: dua", err.Error())
}

func TestDataOperationFor(t *testing.T) {
	require.Equal(t, monitoringslinkojk.OperationCreate, monitoringslinkojk.DataOperationFor(0))
	require.Equal(t, monitoringslinkojk.OperationCreate, monitoringslinkojk.DataOperationFor(-1))
	require.Equal(t, monitoringslinkojk.OperationUpdate, monitoringslinkojk.DataOperationFor(3))
}

func TestReportMonthOf(t *testing.T) {
	require.Equal(t, "202609", monitoringslinkojk.ReportMonthOf(time.Date(2026, 9, 30, 23, 0, 0, 0, time.UTC)))
}

func TestNetting(t *testing.T) {
	require.Equal(t, "750.5", monitoringslinkojk.Netting(" 1000.5 ", "250"))
	// Kosong dibaca nol.
	require.Equal(t, "1000", monitoringslinkojk.Netting("1000", ""))
	require.Equal(t, "-5", monitoringslinkojk.Netting("", "5"))
	// Tanpa notasi ilmiah.
	require.Equal(t, "100000000000", monitoringslinkojk.Netting("100000000000", "0"))
	// Nilai yang tidak terbaca dikembalikan apa adanya, bukan menjadi nol.
	require.Equal(t, "abc", monitoringslinkojk.Netting(" abc ", "1"))
	require.Equal(t, "1000", monitoringslinkojk.Netting("1000", "x"))
}

func TestReportEntryCleanTrimsEveryField(t *testing.T) {
	e := monitoringslinkojk.ReportEntry{
		ClaimID: " a ", ContractNo: " b ", FacilityAccountNo: " c ", DebtorCIF: " d ",
		FacilityTypeCode: " e ", FundSource: " f ", PolicyStart: " g ", PolicyEnd: " h ",
		InterestRate: " i ", CurrencyCode: " j ", OriginalCurrencyValue: " k ", Obligation: " l ",
		CollectibilityCode: " m ", DefaultDate: " n ", DefaultReasonCode: " o ", Arrears: " p ",
		ArrearsDays: " q ", ConditionDate: " r ", ConditionCode: " s ", BranchCode: " t ",
		Remark: " u ", ReportMonth: " v ", DataOperation: " w ", Recovery: " x ", IDCardNo: " y ",
		CompanyNPWP: " z ", PolicyNo: " 1 ", ClientID: " 2 ",
	}
	want := monitoringslinkojk.ReportEntry{
		ClaimID: "a", ContractNo: "b", FacilityAccountNo: "c", DebtorCIF: "d",
		FacilityTypeCode: "e", FundSource: "f", PolicyStart: "g", PolicyEnd: "h",
		InterestRate: "i", CurrencyCode: "j", OriginalCurrencyValue: "k", Obligation: "l",
		CollectibilityCode: "m", DefaultDate: "n", DefaultReasonCode: "o", Arrears: "p",
		ArrearsDays: "q", ConditionDate: "r", ConditionCode: "s", BranchCode: "t",
		Remark: "u", ReportMonth: "v", DataOperation: "w", Recovery: "x", IDCardNo: "y",
		CompanyNPWP: "z", PolicyNo: "1", ClientID: "2",
	}
	require.Equal(t, want, e.Clean())
}

func TestReportEntryValidate(t *testing.T) {
	require.NoError(t, monitoringslinkojk.ReportEntry{ClaimID: "A", ContractNo: "B"}.Validate())

	err := monitoringslinkojk.ReportEntry{}.Validate()
	var ve *monitoringslinkojk.ValidationError
	require.True(t, errors.As(err, &ve))
	require.Equal(t, []monitoringslinkojk.Violation{
		{Field: monitoringslinkojk.FieldClaimID, Message: "No Klaim wajib diisi."},
		{Field: monitoringslinkojk.FieldContractNo, Message: "Contract No wajib diisi."},
	}, ve.Violations)
}

func TestWriteOutcomeTotal(t *testing.T) {
	require.Equal(t, 5, monitoringslinkojk.WriteOutcome{Created: 2, Updated: 3}.Total())
}
