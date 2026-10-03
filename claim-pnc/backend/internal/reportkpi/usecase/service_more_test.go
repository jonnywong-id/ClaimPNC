package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	reportkpimemory "claim-pnc/internal/reportkpi/repo/memory"
	"claim-pnc/internal/reportkpi/usecase"
)

// Uji tambahan orkestrasi: tab Adjuster, tab Admin, jejak log, dan jalur galat.

var errRepo = errors.New("penyimpanan gagal")
var errNoPortal = errors.New("portal tidak siap")

// faultyRepo meneruskan ke penyimpan memori, kecuali operasi yang diminta gagal.
//
// Kunci `fail` berbentuk nama operasi; untuk Bands dan ThresholdDays kuncinya
// `Bands:<job>:<note>` dan `ThresholdDays:<job>:<note>`.
type faultyRepo struct {
	*reportkpimemory.Store
	fail string
}

func (r faultyRepo) failing(key string) error {
	if r.fail == key {
		return errRepo
	}
	return nil
}

func (r faultyRepo) Summary(ctx context.Context, q reportkpi.Query) ([]reportkpi.AdjusterSummary, error) {
	if err := r.failing("Summary"); err != nil {
		return nil, err
	}
	return r.Store.Summary(ctx, q)
}

func (r faultyRepo) Detail(
	ctx context.Context, q reportkpi.Query, p reportkpi.Pagination,
) (reportkpi.DetailPage, error) {
	if err := r.failing("Detail"); err != nil {
		return reportkpi.DetailPage{}, err
	}
	return r.Store.Detail(ctx, q, p)
}

func (r faultyRepo) Adjusters(ctx context.Context, q reportkpi.Query) ([]string, error) {
	if err := r.failing("Adjusters"); err != nil {
		return nil, err
	}
	return r.Store.Adjusters(ctx, q)
}

func (r faultyRepo) AdminTotals(ctx context.Context, q reportkpi.AdminQuery) (reportkpi.AdminTotals, error) {
	if err := r.failing("AdminTotals"); err != nil {
		return reportkpi.AdminTotals{}, err
	}
	return r.Store.AdminTotals(ctx, q)
}

func (r faultyRepo) AdminDetail(
	ctx context.Context, q reportkpi.AdminQuery, p reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	if err := r.failing("AdminDetail"); err != nil {
		return reportkpi.AdminDetailPage{}, err
	}
	return r.Store.AdminDetail(ctx, q, p)
}

func (r faultyRepo) PICs(ctx context.Context, line reportkpi.BusinessLine) ([]reportkpi.PICProfile, error) {
	if err := r.failing("PICs"); err != nil {
		return nil, err
	}
	return r.Store.PICs(ctx, line)
}

func (r faultyRepo) Holidays(ctx context.Context, from, to time.Time) ([]time.Time, error) {
	if err := r.failing("Holidays"); err != nil {
		return nil, err
	}
	return r.Store.Holidays(ctx, from, to)
}

func (r faultyRepo) Bands(ctx context.Context, job, note string) ([]reportkpi.Band, error) {
	if err := r.failing("Bands:" + job + ":" + note); err != nil {
		return nil, err
	}
	return r.Store.Bands(ctx, job, note)
}

func (r faultyRepo) ThresholdDays(ctx context.Context, job, note string) (reportkpi.Score, error) {
	if err := r.failing("ThresholdDays:" + job + ":" + note); err != nil {
		return reportkpi.Score{}, err
	}
	return r.Store.ThresholdDays(ctx, job, note)
}

func (r faultyRepo) ProgressCounts(ctx context.Context, q reportkpi.PICQuery) ([]reportkpi.ProgressCount, error) {
	if err := r.failing("ProgressCounts"); err != nil {
		return nil, err
	}
	return r.Store.ProgressCounts(ctx, q)
}

func (r faultyRepo) AnalysisSpans(ctx context.Context, q reportkpi.PICQuery) ([]reportkpi.DateSpan, error) {
	if err := r.failing("AnalysisSpans"); err != nil {
		return nil, err
	}
	return r.Store.AnalysisSpans(ctx, q)
}

func (r faultyRepo) AcceptanceSpans(
	ctx context.Context, q reportkpi.PICQuery,
) ([]reportkpi.AcceptanceSpan, error) {
	if err := r.failing("AcceptanceSpans"); err != nil {
		return nil, err
	}
	return r.Store.AcceptanceSpans(ctx, q)
}

func (r faultyRepo) ClosureSpans(ctx context.Context, q reportkpi.PICQuery) ([]reportkpi.ClosureSpan, error) {
	if err := r.failing("ClosureSpans"); err != nil {
		return nil, err
	}
	return r.Store.ClosureSpans(ctx, q)
}

// serviceWith membentuk layanan di atas repo tertentu; portal selain ASM ditolak.
func serviceWith(t *testing.T, repo reportkpi.Repo, logs *bytes.Buffer) *usecase.Service {
	t.Helper()

	var logger *slog.Logger
	if logs != nil {
		logger = slog.New(slog.NewJSONHandler(logs, nil))
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (reportkpi.Repo, error) {
			if alias != samplePortal {
				return nil, errNoPortal
			}
			return repo, nil
		},
		Logger: logger,
	})
	require.NoError(t, err)
	return service
}

func adjusterInput() reportkpi.QueryInput {
	return reportkpi.QueryInput{ReportType: "final", From: "2026-03-01", To: "2026-03-31"}
}

func adminInput(group string) reportkpi.AdminQueryInput {
	return reportkpi.AdminQueryInput{Group: group, From: "2026-03-01", To: "2026-03-31"}
}

func TestNewServiceRequiresRepoSelector(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
	require.Nil(t, service)
}

func TestMetadataListsEverything(t *testing.T) {
	meta := serviceWith(t, reportkpimemory.NewStore(), nil).Metadata()

	require.Equal(t, reportkpi.DefaultTabKPI, meta.DefaultTab)
	require.Len(t, meta.Tabs, len(reportkpi.Tabs()))
	require.Len(t, meta.Components, 9)
	require.Len(t, meta.ReportTypes, 3)
	require.Len(t, meta.AdminGroups, 2)
	require.Equal(t, reportkpi.CoordinatorInQuery, meta.CoordinatorInQuery)
	require.Equal(t, reportkpi.PlannedDifferences, meta.PlannedDifferences)
	require.Equal(t, reportkpi.AdminPlannedDifferences, meta.AdminPlannedDifferences)
	require.Equal(t, reportkpi.PICTeknikPlannedDifferences, meta.PICTeknikPlannedDifferences)
	require.Equal(t, reportkpi.SLAExcludedPICs(), meta.SLAExcludedPICs)
	require.Len(t, meta.PICComponents, len(reportkpi.PICComponents()))
	require.Len(t, meta.BusinessLines, len(reportkpi.BusinessLines()))
}

// Ringkasan dibuka, tipe dibakukan, dan jejaknya tercatat atas nama pemanggil.
func TestSummaryReturnsRowsAndRecordsTrail(t *testing.T) {
	var logs bytes.Buffer
	service := serviceWith(t, reportkpimemory.NewSampleStore(), &logs)

	result, err := service.Summary(context.Background(), samplePortal, picCaller(), adjusterInput())
	require.NoError(t, err)
	require.Equal(t, reportkpi.TypeFinal, result.Query.ReportType)
	require.NotEmpty(t, result.Rows)

	require.Contains(t, logs.String(), "ringkasan KPI adjuster dibuka")
	require.Contains(t, logs.String(), `"adjuster":"(seluruhnya)"`)
	require.Contains(t, logs.String(), `"pemanggil":"contohpenyelia"`)
}

func TestSummaryErrors(t *testing.T) {
	store := reportkpimemory.NewSampleStore()

	_, err := serviceWith(t, store, nil).Summary(context.Background(), samplePortal, picCaller(),
		reportkpi.QueryInput{})
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = serviceWith(t, store, nil).Summary(context.Background(), "LAIN", picCaller(), adjusterInput())
	require.ErrorIs(t, err, errNoPortal)

	_, err = serviceWith(t, faultyRepo{store, "Summary"}, nil).
		Summary(context.Background(), samplePortal, picCaller(), adjusterInput())
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "mengambil ringkasan KPI adjuster")
}

// Adjuster yang dipilih tercatat apa adanya di jejak.
func TestDetailReturnsPageAndRecordsChosenAdjuster(t *testing.T) {
	var logs bytes.Buffer
	service := serviceWith(t, reportkpimemory.NewSampleStore(), &logs)

	input := adjusterInput()
	input.Adjuster = "PT TEPI CONTOH MANDIRI"
	result, err := service.Detail(context.Background(), samplePortal, picCaller(), input,
		reportkpi.Pagination{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 2, result.Page.Total)
	require.Equal(t, "PT TEPI CONTOH MANDIRI", result.Query.Adjuster)
	require.Contains(t, logs.String(), `"adjuster":"PT TEPI CONTOH MANDIRI"`)
	require.Contains(t, logs.String(), "rincian KPI adjuster dibuka")
}

func TestDetailErrors(t *testing.T) {
	store := reportkpimemory.NewSampleStore()
	page := reportkpi.Pagination{}

	_, err := serviceWith(t, store, nil).Detail(context.Background(), samplePortal, picCaller(),
		reportkpi.QueryInput{}, page)
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = serviceWith(t, store, nil).Detail(context.Background(), "LAIN", picCaller(), adjusterInput(), page)
	require.ErrorIs(t, err, errNoPortal)

	_, err = serviceWith(t, faultyRepo{store, "Detail"}, nil).
		Detail(context.Background(), samplePortal, picCaller(), adjusterInput(), page)
	require.ErrorIs(t, err, errRepo)
}

// Pilihan adjuster yang sedang aktif diabaikan saat mengisi dropdown.
func TestAdjustersIgnoresActiveChoice(t *testing.T) {
	service := serviceWith(t, reportkpimemory.NewSampleStore(), nil)

	input := adjusterInput()
	input.Adjuster = "PT TEPI CONTOH MANDIRI"
	names, err := service.Adjusters(context.Background(), samplePortal, picCaller(), input)
	require.NoError(t, err)
	require.Equal(t, []string{
		"PT ADJUSTER NUSA CONTOH", "PT PENILAI CONTOH PRATAMA", "PT TEPI CONTOH MANDIRI",
	}, names)
}

func TestAdjustersErrors(t *testing.T) {
	store := reportkpimemory.NewSampleStore()

	_, err := serviceWith(t, store, nil).Adjusters(context.Background(), samplePortal, picCaller(),
		reportkpi.QueryInput{})
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = serviceWith(t, store, nil).Adjusters(context.Background(), "LAIN", picCaller(), adjusterInput())
	require.ErrorIs(t, err, errNoPortal)

	_, err = serviceWith(t, faultyRepo{store, "Adjusters"}, nil).
		Adjusters(context.Background(), samplePortal, picCaller(), adjusterInput())
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "mengambil daftar adjuster")
}

// Kartu skor disusun domain, dan nama koordinator ikut tercatat di jejak.
func TestAdminScorecardBuildsCardAndRecordsCoordinator(t *testing.T) {
	var logs bytes.Buffer
	service := serviceWith(t, reportkpimemory.NewSampleStore(), &logs)

	result, err := service.AdminScorecard(context.Background(), samplePortal, picCaller(), adminInput("nonmbu"))
	require.NoError(t, err)
	require.Equal(t, reportkpi.AdminGroupNonMBU, result.Query.Group)
	require.Equal(t, reportkpi.AdminGroupNonMBU, result.Card.Group)
	require.Len(t, result.Card.Metrics, 14)
	require.Equal(t, "01/03/2026 - 31/03/2026", result.Card.EffectiveOn)
	require.Contains(t, logs.String(), "kartu skor KPI admin dibuka")
	require.Contains(t, logs.String(), `"koordinator":"MORASOTARDODOTARIGAN"`)
}

func TestAdminScorecardErrors(t *testing.T) {
	store := reportkpimemory.NewSampleStore()

	_, err := serviceWith(t, store, nil).AdminScorecard(context.Background(), samplePortal, picCaller(),
		reportkpi.AdminQueryInput{})
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = serviceWith(t, store, nil).AdminScorecard(context.Background(), "LAIN", picCaller(), adminInput("PA"))
	require.ErrorIs(t, err, errNoPortal)

	_, err = serviceWith(t, faultyRepo{store, "AdminTotals"}, nil).
		AdminScorecard(context.Background(), samplePortal, picCaller(), adminInput("PA"))
	require.ErrorIs(t, err, errRepo)
	require.ErrorContains(t, err, "kartu skor KPI admin")
}

func TestAdminDetailReturnsPage(t *testing.T) {
	var logs bytes.Buffer
	service := serviceWith(t, reportkpimemory.NewSampleStore(), &logs)

	result, err := service.AdminDetail(context.Background(), samplePortal, picCaller(), adminInput("PA"),
		reportkpi.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Equal(t, reportkpi.AdminGroupPA, result.Query.Group)
	require.Len(t, result.Page.Rows, 2)
	require.Equal(t, 4, result.Page.Total)
	require.Contains(t, logs.String(), "rincian KPI admin dibuka")
}

func TestAdminDetailErrors(t *testing.T) {
	store := reportkpimemory.NewSampleStore()
	page := reportkpi.Pagination{}

	_, err := serviceWith(t, store, nil).AdminDetail(context.Background(), samplePortal, picCaller(),
		reportkpi.AdminQueryInput{}, page)
	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = serviceWith(t, store, nil).AdminDetail(context.Background(), "LAIN", picCaller(),
		adminInput("PA"), page)
	require.ErrorIs(t, err, errNoPortal)

	_, err = serviceWith(t, faultyRepo{store, "AdminDetail"}, nil).
		AdminDetail(context.Background(), samplePortal, picCaller(), adminInput("PA"), page)
	require.ErrorIs(t, err, errRepo)
}

// Setiap bahan yang gagal dibaca menggagalkan seluruh tab PIC Teknik.
func TestPICTeknikPropagatesEveryRepoError(t *testing.T) {
	failures := []string{
		"PICs", "Holidays",
		"Bands:" + reportkpi.JobProgress + ":",
		"Bands:" + reportkpi.JobSLA + ":" + reportkpi.TeamLeader,
		"Bands:" + reportkpi.JobSLA + ":" + reportkpi.TeamMember,
		"ThresholdDays:" + reportkpi.JobAnalysis + ":",
		"ThresholdDays:" + reportkpi.JobAcceptance + ":",
		"ThresholdDays:" + reportkpi.JobSLA + ":" + reportkpi.TeamLeader,
		"ThresholdDays:" + reportkpi.JobSLA + ":" + reportkpi.TeamMember,
		"ProgressCounts", "AnalysisSpans", "AcceptanceSpans", "ClosureSpans",
	}
	for _, failure := range failures {
		repo := faultyRepo{reportkpimemory.NewSampleStore(), failure}
		_, err := serviceWith(t, repo, nil).PICTeknik(context.Background(), samplePortal, picInput(), picCaller())
		require.ErrorIsf(t, err, errRepo, "kegagalan %s tidak diteruskan", failure)
	}
}

func TestPICTeknikRejectsUnknownPortal(t *testing.T) {
	_, err := serviceWith(t, reportkpimemory.NewSampleStore(), nil).
		PICTeknik(context.Background(), "LAIN", picInput(), picCaller())
	require.ErrorIs(t, err, errNoPortal)
}

// sparseStore hanya punya pita SLA tanpa NOTE dan TIDAK punya ambang hari sama sekali.
func sparseStore() *reportkpimemory.Store {
	day := func(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }
	return reportkpimemory.NewStore().WithPICTeknik(reportkpimemory.PICTeknikData{
		PICs: []reportkpi.PICProfile{{OperatorID: "CONTOHPIC"}},
		Bands: []reportkpi.Band{
			{Job: reportkpi.JobSLA, Value: 3, Bottom: 0, Top: 100},
			{Job: reportkpi.JobAnalysis, Value: 1, Bottom: 0, Top: 100},
			{Job: reportkpi.JobAcceptance, Value: 1, Bottom: 0, Top: 100},
		},
		Analysis: []reportkpimemory.SpanRow{{
			PIC: "CONTOHPIC", Line: reportkpi.LineNonMBU,
			Registered: day(2), Start: day(2), End: day(3),
		}},
		Acceptance: []reportkpimemory.AcceptanceRow{
			{PIC: "CONTOHPIC", Line: reportkpi.LineNonMBU, Team: "LAINNYA", Accepted: day(4)},
		},
		Closure: []reportkpimemory.ClosureRow{{
			PIC: "CONTOHPIC", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamMember,
			Registered: day(2), Closed: day(3),
		}},
	})
}

// Pita SLA bertanda tim tidak ada → dipakai pita SLA tanpa penyaring. Ambang hari yang
// tidak ada → pembilang nol, pembagi tetap utuh.
func TestPICTeknikFallsBackToUnfilteredSLABandsAndMissingThresholds(t *testing.T) {
	scored, err := serviceWith(t, sparseStore(), nil).
		PICTeknik(context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	card := cardOf(t, scored.Result, "CONTOHPIC")

	sla := rowOf(t, card, reportkpi.PICComponentSLA)
	require.Equal(t, 1.0, sla.Total)
	require.Equal(t, 0.0, sla.Achieved)
	require.Equal(t, reportkpi.NewScore(3), sla.Value, "pita SLA tanpa NOTE dipakai")

	analysis := rowOf(t, card, reportkpi.PICComponentAnalysis)
	require.Equal(t, 1.0, analysis.Total)
	require.Equal(t, 0.0, analysis.Achieved)

	acceptance := rowOf(t, card, reportkpi.PICComponentAcceptance)
	require.Equal(t, 1.0, acceptance.Total)
	require.Equal(t, 0.0, acceptance.Achieved)
}

// Pita SLA tanpa penyaring dibaca hanya bila salah satu pita bertim kosong — dan galatnya
// diteruskan.
func TestPICTeknikFallbackBandsErrorIsPropagated(t *testing.T) {
	repo := faultyRepo{sparseStore(), "Bands:" + reportkpi.JobSLA + ":"}
	_, err := serviceWith(t, repo, nil).PICTeknik(context.Background(), samplePortal, picInput(), picCaller())
	require.ErrorIs(t, err, errRepo)
}

// Tim akseptasi yang tidak dikenal tidak dapat dinilai: ia tidak dihitung tercapai.
func TestPICTeknikUnknownAcceptanceTeamIsNotAchieved(t *testing.T) {
	store := reportkpimemory.NewStore().WithPICTeknik(func() reportkpimemory.PICTeknikData {
		data := reportkpimemory.SamplePICTeknik()
		data.Acceptance = append(data.Acceptance, reportkpimemory.AcceptanceRow{
			PIC: "CONTOHPICSATU", Line: reportkpi.LineNonMBU, Team: "LAINNYA",
			Accepted: time.Date(2026, 3, 11, 0, 0, 0, 0, time.UTC),
		})
		return data
	}())

	scored, err := serviceWith(t, store, nil).
		PICTeknik(context.Background(), samplePortal, picInput(), picCaller())
	require.NoError(t, err)

	acceptance := rowOf(t, cardOf(t, scored.Result, "CONTOHPICSATU"), reportkpi.PICComponentAcceptance)
	require.Equal(t, 2.0, acceptance.Total)
	require.Equal(t, 1.0, acceptance.Achieved)
}

// Tanpa logger, pembukaan laporan tetap berhasil — jejaknya saja yang tidak ditulis.
func TestOperationsSucceedWithoutLogger(t *testing.T) {
	service := serviceWith(t, reportkpimemory.NewSampleStore(), nil)

	summary, err := service.Summary(context.Background(), samplePortal, picCaller(), adjusterInput())
	require.NoError(t, err)
	require.NotEmpty(t, summary.Rows)

	card, err := service.AdminScorecard(context.Background(), samplePortal, picCaller(), adminInput("PA"))
	require.NoError(t, err)
	require.Len(t, card.Card.Metrics, 6)
}
