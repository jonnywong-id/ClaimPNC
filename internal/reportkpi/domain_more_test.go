package reportkpi_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Uji tambahan lapisan domain: tab KPI Admin, penyaring PIC Teknik, dan pembantu kecil.

func validAdminInput(group string) reportkpi.AdminQueryInput {
	return reportkpi.AdminQueryInput{Group: group, From: "2026-03-01", To: "2026-03-31"}
}

func TestAdminGroupsReturnsCopy(t *testing.T) {
	groups := reportkpi.AdminGroups()
	require.Len(t, groups, 2)
	require.Equal(t, reportkpi.AdminGroupNonMBU, groups[0].Code)
	require.Equal(t, "KLAIM NON MBU", groups[0].Label)
	require.Equal(t, reportkpi.AdminGroupPA, groups[1].Code)

	groups[0].Label = "DIUBAH"
	require.Equal(t, "KLAIM NON MBU", reportkpi.AdminGroups()[0].Label)
}

func TestFindAdminGroupIgnoresCase(t *testing.T) {
	group, found := reportkpi.FindAdminGroup("  pa ")
	require.True(t, found)
	require.Equal(t, reportkpi.AdminGroupPA, group)

	group, found = reportkpi.FindAdminGroup("MBU")
	require.False(t, found)
	require.Empty(t, group)
}

func TestAdminIdentityFor(t *testing.T) {
	nonMBU := reportkpi.AdminIdentityFor(reportkpi.AdminGroupNonMBU)
	// Nama yang DIGAMBAR adalah yang tertulis di teks kueri, bukan yang ditulis activity.
	// Dibalik pada 2026-10-09 setelah layar Pega yang berjalan diperiksa; lihat admin.go.
	require.Equal(t, "YUSMIARSIH DYAHPUSPITA S", nonMBU.Coordinator)
	require.Equal(t, "KLAIM NON MBU", nonMBU.Category)
	require.NotEqual(t, reportkpi.CoordinatorInActivity, nonMBU.Coordinator)

	pa := reportkpi.AdminIdentityFor(reportkpi.AdminGroupPA)
	require.Equal(t, "KOORDINASI PA", pa.WorkUnit)

	require.Equal(t, reportkpi.AdminIdentity{}, reportkpi.AdminIdentityFor("LAIN"))
}

// Kartu NON-MBU memuat 14 metrik berurutan, bobot literal, dan kesimpulan.
func TestBuildScorecardNonMBU(t *testing.T) {
	rng := reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"}
	totals := reportkpi.AdminTotals{
		LeaderOverSLA:    reportkpi.NewScore(1),
		AchievementRatio: reportkpi.NewScore(1.05),
	}

	card := reportkpi.BuildScorecard(reportkpi.AdminGroupNonMBU, rng, totals)
	require.Equal(t, reportkpi.AdminGroupNonMBU, card.Group)
	require.Equal(t, "01/03/2026 - 31/03/2026", card.EffectiveOn)
	require.Equal(t, "YUSMIARSIH DYAHPUSPITA S", card.Identity.Coordinator)
	require.Len(t, card.Metrics, 15)
	require.Equal(t, reportkpi.MetricLeaderOverSLA, card.Metrics[0].Code)
	require.Equal(t, reportkpi.NewScore(1), card.Metrics[0].Value)
	require.Equal(t, reportkpi.FormatCount, card.Metrics[0].Format)
	require.Equal(t, reportkpi.MetricLeaderWeight, card.Metrics[4].Code)
	require.Equal(t, reportkpi.NewScore(reportkpi.AdminLeaderWeight), card.Metrics[4].Value)
	require.Equal(t, reportkpi.NewScore(reportkpi.AdminMemberWeight), card.Metrics[10].Value)
	// ACHIEVEMENT adalah tetapan `3/5*85`, dan ia berada SEBELUM rasio pencapaian.
	require.Equal(t, reportkpi.MetricAchievementTarget, card.Metrics[13].Code)
	require.Equal(t, reportkpi.NewScore(51), card.Metrics[13].Value)
	require.Equal(t, reportkpi.MetricAchievementRatio, card.Metrics[14].Code)
	require.Equal(t, reportkpi.AchievementReached, card.Achievement)

	missed := reportkpi.BuildScorecard(reportkpi.AdminGroupNonMBU, rng,
		reportkpi.AdminTotals{AchievementRatio: reportkpi.NewScore(0.99)})
	require.Equal(t, reportkpi.AchievementMissed, missed.Achievement)

	// Rasio yang tidak dapat dihitung menghasilkan kesimpulan kosong.
	empty := reportkpi.BuildScorecard(reportkpi.AdminGroupNonMBU, rng, reportkpi.AdminTotals{})
	require.Empty(t, empty.Achievement)
}

// Kartu PA memuat enam metrik dan tidak punya kesimpulan.
func TestBuildScorecardPA(t *testing.T) {
	card := reportkpi.BuildScorecard(reportkpi.AdminGroupPA,
		reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
		reportkpi.AdminTotals{PaymentScore: reportkpi.NewScore(4), AchievementRatio: reportkpi.NewScore(2)})
	require.Len(t, card.Metrics, 6)
	require.Equal(t, reportkpi.MetricRegisterOverSLA, card.Metrics[0].Code)
	require.Equal(t, reportkpi.MetricPaymentScore, card.Metrics[5].Code)
	require.Equal(t, reportkpi.NewScore(4), card.Metrics[5].Value)
	require.Equal(t, reportkpi.FormatScore, card.Metrics[5].Format)
	require.Empty(t, card.Achievement)
}

// Tanggal yang tidak berbentuk ISO tidak menghasilkan teks tanggal efektif.
func TestBuildScorecardEffectiveOnRequiresBothDates(t *testing.T) {
	card := reportkpi.BuildScorecard(reportkpi.AdminGroupPA,
		reportkpi.DateRange{From: "2026-03-01", To: "31/03"}, reportkpi.AdminTotals{})
	require.Empty(t, card.EffectiveOn)
}

func TestNewAdminQueryAcceptsValidInput(t *testing.T) {
	query, err := reportkpi.NewAdminQuery(validAdminInput(" nonmbu "), reportkpi.Caller{Login: " PENYELIA "})
	require.NoError(t, err)
	require.Equal(t, reportkpi.AdminGroupNonMBU, query.Group)
	require.Equal(t, reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"}, query.Range)
	require.Equal(t, "PENYELIA", query.Caller.Login)
}

func TestNewAdminQueryRejectsUnknownCaller(t *testing.T) {
	_, err := reportkpi.NewAdminQuery(validAdminInput("PA"), reportkpi.Caller{})
	require.ErrorIs(t, err, reportkpi.ErrCallerUnknown)
}

// Seluruh pelanggaran dikumpulkan, dan pesannya membedakan kosong dari tidak dikenal.
func TestNewAdminQueryCollectsViolations(t *testing.T) {
	_, err := reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{From: "x"}, caller())
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []reportkpi.Violation{
		{Field: reportkpi.FieldAdminGroup, Message: "Data KPI wajib dipilih."},
		{Field: reportkpi.FieldDateFrom, Message: "Periode \"Dari\" tidak terbaca. Gunakan bentuk " +
			"tahun-bulan-tanggal, misalnya 2026-09-24."},
		{Field: reportkpi.FieldDateTo, Message: "Periode \"Sampai\" wajib diisi."},
	}, validation.Violations)

	_, err = reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{
		Group: "MBU", From: "2026-03-31", To: "2026-03-01",
	}, caller())
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []reportkpi.Violation{
		{Field: reportkpi.FieldAdminGroup, Message: "Data KPI \"MBU\" tidak dikenal. Pilihan: NONMBU, PA."},
		{Field: reportkpi.FieldDateTo, Message: "Tanggal \"Sampai\" tidak boleh lebih awal daripada \"Dari\"."},
	}, validation.Violations)
}

func TestValidationErrorMessage(t *testing.T) {
	require.Equal(t, "reportkpi: isian tidak sah", reportkpi.NewValidationError(nil).Error())
	err := reportkpi.NewValidationError([]reportkpi.Violation{
		{Field: "a", Message: "satu"}, {Field: "b", Message: "dua"},
	})
	require.Equal(t, "reportkpi: a: satu; b: dua", err.Error())
}

func TestComponentCodesAndFindComponent(t *testing.T) {
	codes := reportkpi.ComponentCodes()
	require.Len(t, codes, 9)
	require.Equal(t, reportkpi.ComponentSurvey, codes[0])
	require.Equal(t, reportkpi.ComponentTotal, codes[8])

	component, found := reportkpi.FindComponent(" " + reportkpi.ComponentTotal + " ")
	require.True(t, found)
	require.Equal(t, "NILAI", component.Label)

	_, found = reportkpi.FindComponent("tidak_ada")
	require.False(t, found)
}

func TestReportTypesReturnsCopy(t *testing.T) {
	types := reportkpi.ReportTypes()
	require.Equal(t, []reportkpi.ReportType{
		reportkpi.TypeOutstanding, reportkpi.TypeFinal, reportkpi.TypeAll,
	}, []reportkpi.ReportType{types[0].Code, types[1].Code, types[2].Code})

	types[0].Label = "DIUBAH"
	require.Equal(t, "OUTSTANDING", reportkpi.ReportTypes()[0].Label)
}

func TestFindPICComponentMiss(t *testing.T) {
	component, found := reportkpi.FindPICComponent(reportkpi.PICComponentSLA)
	require.True(t, found)
	require.Equal(t, reportkpi.PICComponentSLA, component.Code)

	_, found = reportkpi.FindPICComponent("tidak_ada")
	require.False(t, found)
}

func TestPICTeknikGridIsScorecard(t *testing.T) {
	require.Equal(t, reportkpi.GridPICScorecard, reportkpi.PICTeknikGrid().Code)
}

func TestFindTabAndAdminGridMiss(t *testing.T) {
	_, found := reportkpi.FindTab("tidak_ada")
	require.False(t, found)

	require.Equal(t, reportkpi.GridAdminDetail, reportkpi.AdminGridFor(reportkpi.GridAdminDetail).Code)
	require.Equal(t, reportkpi.Grid{}, reportkpi.AdminGridFor("tidak_ada"))
}

func TestPaginationNormalizeAndOffset(t *testing.T) {
	require.Equal(t, reportkpi.Pagination{Page: 1, Size: reportkpi.DefaultPageSize},
		reportkpi.Pagination{Page: -3, Size: 0}.Normalize())
	require.Equal(t, reportkpi.Pagination{Page: 2, Size: reportkpi.MaxPageSize},
		reportkpi.Pagination{Page: 2, Size: 500}.Normalize())
	require.Equal(t, 40, reportkpi.Pagination{Page: 3, Size: 20}.Offset())
	require.Equal(t, 0, reportkpi.Pagination{}.Offset())
}

func TestPICTeknikQuerySpanAndDates(t *testing.T) {
	query, err := reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{
		Line: string(reportkpi.LinePA), From: "2026-03-01", To: "2026-03-31",
	}, caller())
	require.NoError(t, err)

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	require.Equal(t, reportkpi.PICQuery{Line: reportkpi.LinePA, From: from, To: to}, query.Span())

	gotFrom, gotTo := query.Dates()
	require.Equal(t, from, gotFrom)
	require.Equal(t, to, gotTo)
}

func TestNewPICTeknikQueryRejections(t *testing.T) {
	_, err := reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{}, reportkpi.Caller{Login: " "})
	require.ErrorIs(t, err, reportkpi.ErrCallerUnknown)

	_, err = reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{
		Line: "MBU", From: "2026-03-31", To: "2026-03-01",
	}, caller())
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
	require.Equal(t, reportkpi.FieldBusinessLine, validation.Violations[0].Field)
	require.Contains(t, validation.Violations[0].Message, "Lini bisnis \"MBU\" tidak dikenal. Pilihan: ")
	require.Equal(t, reportkpi.FieldDateTo, validation.Violations[1].Field)

	_, err = reportkpi.NewPICTeknikQuery(reportkpi.PICQueryInput{To: "2026-02-30"}, caller())
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "Lini bisnis wajib dipilih.", validation.Violations[0].Message)
	require.Len(t, validation.Violations, 3)
}

// Libur bernilai nol diabaikan, dan hasil negatif dibetulkan menjadi nol.
func TestWorkingDaysBetweenClampsAndSkipsZeroHolidays(t *testing.T) {
	friday := time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC)
	saturday := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)

	// Jumat → Sabtu: tidak ada hari kerja; libur pada Sabtu membuatnya negatif → 0.
	require.Equal(t, 0, reportkpi.WorkingDaysBetween(friday, saturday, []time.Time{saturday}))

	monday := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	require.Equal(t, 1, reportkpi.WorkingDaysBetween(friday, monday, []time.Time{{}}))
}
