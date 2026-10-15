package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/repo/memory"
)

// Uji pengisi memori tab KPI PIC Teknik: penyaringnya harus meniru kueri SQL-nya.

func day(d int) time.Time { return time.Date(2026, 3, d, 0, 0, 0, 0, time.UTC) }

func marchSpan(line reportkpi.BusinessLine) reportkpi.PICQuery {
	return reportkpi.PICQuery{Line: line, From: day(1), To: day(31)}
}

// picStore berisi bahan dengan baris yang sengaja jatuh di luar lini dan periode.
func picStore() *memory.Store {
	return memory.NewStore().WithPICTeknik(memory.PICTeknikData{
		PICs: []reportkpi.PICProfile{
			{OperatorID: "ZPIC"}, {OperatorID: "APIC", Leader: true},
		},
		Bands: []reportkpi.Band{
			{Job: reportkpi.JobSLA, Value: 5, Note: reportkpi.TeamLeader},
			{Job: reportkpi.JobSLA, Value: 4, Note: reportkpi.TeamMember},
			{Job: reportkpi.JobSLA, Value: 3},
			{Job: reportkpi.JobAnalysis, Value: 1},
		},
		Thresholds: []memory.ThresholdRow{
			{Job: reportkpi.JobSLA, Note: reportkpi.TeamLeader, Days: 390},
			{Job: reportkpi.JobSLA, Note: reportkpi.TeamMember, Days: 399},
		},
		// Sabtu 7 Maret, Rabu 18 Maret, dan 1 April (di luar rentang).
		Holidays: []time.Time{day(18), day(7), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), day(4)},
		Progress: []memory.ProgressRow{
			{PIC: "APIC", Line: reportkpi.LineNonMBU, At: day(2), OnTime: true},
			// Pukul 23.00 pada hari terakhir tetap masuk.
			{PIC: "APIC", Line: reportkpi.LineNonMBU, At: time.Date(2026, 3, 31, 23, 0, 0, 0, time.UTC)},
			{PIC: "ZPIC", Line: reportkpi.LineNonMBU, At: day(3), OnTime: true},
			{PIC: "APIC", Line: reportkpi.LinePA, At: day(3), OnTime: true},
			{PIC: "APIC", Line: reportkpi.LineNonMBU, At: time.Time{}},
			{PIC: "APIC", Line: reportkpi.LineNonMBU, At: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
		},
		Analysis: []memory.SpanRow{
			{PIC: "APIC", Line: reportkpi.LineNonMBU, Registered: day(2), Start: day(2), End: day(5)},
			{PIC: "APIC", Line: reportkpi.LineNonMBU, Registered: day(2), Start: day(2)},
			{PIC: "APIC", Line: reportkpi.LinePA, Registered: day(2), Start: day(2), End: day(5)},
		},
		Acceptance: []memory.AcceptanceRow{
			{PIC: "APIC", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamLeader,
				ReceiveLOD: day(9), Committee: day(2), Accepted: day(10)},
			{PIC: "APIC", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamLeader,
				Accepted: time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC)},
		},
		Closure: []memory.ClosureRow{
			{PIC: "APIC", Line: reportkpi.LineNonMBU, Team: reportkpi.TeamLeader,
				Registered: day(2), Closed: day(20)},
			{PIC: "  ", Line: reportkpi.LineNonMBU, Registered: day(2), Closed: day(20)},
			{PIC: "APIC", Line: reportkpi.LineTravel, Registered: day(2), Closed: day(20)},
		},
	})
}

// Daftar PIC terurut menurut Operator ID, dan berupa salinan.
func TestMemoryPICsAreSortedCopies(t *testing.T) {
	store := picStore()

	pics, err := store.PICs(context.Background(), reportkpi.LineNonMBU)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.PICProfile{
		{OperatorID: "APIC", Leader: true}, {OperatorID: "ZPIC"},
	}, pics)

	pics[0].OperatorID = "DIUBAH"
	again, err := store.PICs(context.Background(), reportkpi.LineNonMBU)
	require.NoError(t, err)
	require.Equal(t, "APIC", again[0].OperatorID)
}

func TestMemoryBandsFilterByJobAndNote(t *testing.T) {
	store := picStore()

	all, err := store.Bands(context.Background(), reportkpi.JobSLA, "")
	require.NoError(t, err)
	require.Len(t, all, 3)

	leader, err := store.Bands(context.Background(), reportkpi.JobSLA, reportkpi.TeamLeader)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.Band{{Job: reportkpi.JobSLA, Value: 5, Note: reportkpi.TeamLeader}}, leader)

	none, err := store.Bands(context.Background(), reportkpi.JobAcceptance, "")
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestMemoryThresholdDays(t *testing.T) {
	store := picStore()

	member, err := store.ThresholdDays(context.Background(), reportkpi.JobSLA, reportkpi.TeamMember)
	require.NoError(t, err)
	require.Equal(t, reportkpi.NewScore(399), member)

	// Tanpa penyaring NOTE, baris pertama yang cocok dipakai.
	first, err := store.ThresholdDays(context.Background(), reportkpi.JobSLA, "")
	require.NoError(t, err)
	require.Equal(t, reportkpi.NewScore(390), first)

	missing, err := store.ThresholdDays(context.Background(), reportkpi.JobProgress, "")
	require.NoError(t, err)
	require.Equal(t, reportkpi.EmptyScore(), missing)
}

// Akhir pekan dan tanggal di luar rentang dibuang; hasilnya terurut.
func TestMemoryHolidaysDropWeekendsAndOutOfRange(t *testing.T) {
	days, err := picStore().Holidays(context.Background(), day(1), day(31))
	require.NoError(t, err)
	require.Equal(t, []time.Time{day(4), day(18)}, days)
}

func TestMemoryProgressCountsFilterLineAndPeriod(t *testing.T) {
	counts, err := picStore().ProgressCounts(context.Background(), marchSpan(reportkpi.LineNonMBU))
	require.NoError(t, err)
	require.Equal(t, []reportkpi.ProgressCount{
		{PIC: "APIC", Total: 2, OnTime: 1},
		{PIC: "ZPIC", Total: 1, OnTime: 1},
	}, counts)
}

// Baris tanpa tanggal akhir dibuang, meniru `IS NOT NULL` pada kuerinya.
func TestMemoryAnalysisSpansSkipMissingEnd(t *testing.T) {
	spans, err := picStore().AnalysisSpans(context.Background(), marchSpan(reportkpi.LineNonMBU))
	require.NoError(t, err)
	require.Equal(t, []reportkpi.DateSpan{{PIC: "APIC", Start: day(2), End: day(5)}}, spans)
}

func TestMemoryAcceptanceSpansFilterByAcceptanceDate(t *testing.T) {
	spans, err := picStore().AcceptanceSpans(context.Background(), marchSpan(reportkpi.LineNonMBU))
	require.NoError(t, err)
	require.Equal(t, []reportkpi.AcceptanceSpan{{
		PIC: "APIC", Team: reportkpi.TeamLeader, ReceiveLOD: day(9),
		CommitteeDate: day(2), AcceptanceDate: day(10),
	}}, spans)
}

// Baris tanpa PIC dibuang, dan lini lain tidak ikut.
func TestMemoryClosureSpansSkipBlankPIC(t *testing.T) {
	spans, err := picStore().ClosureSpans(context.Background(), marchSpan(reportkpi.LineNonMBU))
	require.NoError(t, err)
	require.Equal(t, []reportkpi.ClosureSpan{{
		PIC: "APIC", Team: reportkpi.TeamLeader, RegisterDate: day(2), CloseDate: day(20),
	}}, spans)
}

// Contoh bawaan memang punya bahan PIC Teknik untuk lini NON-MBU.
func TestSampleStoreHasPICTeknikData(t *testing.T) {
	store := memory.NewSampleStore()

	pics, err := store.PICs(context.Background(), reportkpi.LineNonMBU)
	require.NoError(t, err)
	require.Len(t, pics, 2)

	spans, err := store.ClosureSpans(context.Background(), marchSpan(reportkpi.LineNonMBU))
	require.NoError(t, err)
	require.Len(t, spans, 2)
}
