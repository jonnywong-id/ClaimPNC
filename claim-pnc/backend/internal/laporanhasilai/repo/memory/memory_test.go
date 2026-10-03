package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/laporanhasilai/repo/memory"
)

func day(date int) time.Time {
	return time.Date(2026, time.September, date, 0, 0, 0, 0, time.UTC)
}

// september adalah rentang seluruh bulan September 2026.
var september = laporanhasilai.Filter{From: day(1), To: day(30)}

func sampleRepo() *memory.Repo {
	return memory.NewRepo(
		memory.Row{CommitteeID: "K2", CommitteeStep: "2", ObjectID: "O1", CoverageID: "C1",
			ClaimNumber: "PNC-2", ApproveCode: "2", CommitteeDate: day(5),
			AIResult: laporanhasilai.AIRejected, AIDate: day(4), CaseResolved: true},
		memory.Row{CommitteeID: "K1", CommitteeStep: " 1 ", ObjectID: "O1", CoverageID: "C1",
			ClaimNumber: "PNC-1", ApproveCode: "1", CommitteeDate: day(3),
			AIResult: laporanhasilai.AIAccepted, AIDate: day(2), CaseResolved: true},
		memory.Row{CommitteeID: "K3", CommitteeStep: "1", ObjectID: "O1", CoverageID: "C1",
			ClaimNumber: "PNC-3", ApproveCode: "0", CommitteeDate: day(30),
			AIResult: "", CaseResolved: true},
		// Belum diputus komite — tanpa tanggal.
		memory.Row{CommitteeID: "K4", CommitteeStep: "1", CaseResolved: true},
		// Case belum Resolved-Completed.
		memory.Row{CommitteeID: "K5", CommitteeStep: "1", CommitteeDate: day(10)},
		// Di luar rentang.
		memory.Row{CommitteeID: "K6", CommitteeStep: "1", CommitteeDate: time.Date(
			2026, time.October, 1, 0, 0, 0, 0, time.UTC), CaseResolved: true},
		memory.Row{CommitteeID: "K7", CommitteeStep: "1", CommitteeDate: time.Date(
			2026, time.August, 31, 23, 0, 0, 0, time.UTC), CaseResolved: true},
	)
}

func TestListFiltersSortsAndMapsRows(t *testing.T) {
	page, err := sampleRepo().List(context.Background(), september, laporanhasilai.Pagination{})
	require.NoError(t, err)

	require.Equal(t, 3, page.Total)
	require.Len(t, page.Rows, 3)

	ids := []string{page.Rows[0].ID, page.Rows[1].ID, page.Rows[2].ID}
	// Batas atas inklusif sampai akhir 30 September; urutan menurut kunci komite.
	require.Equal(t, []string{"K1| 1 |O1|C1", "K2|2|O1|C1", "K3|1|O1|C1"}, ids)

	first := page.Rows[0]
	require.Equal(t, "PNC-1", first.ClaimNumber)
	require.Equal(t, laporanhasilai.LabelAccepted, first.CommitteeStatus)
	require.Equal(t, "1", first.CommitteeStatusCode)
	require.Equal(t, day(3), first.CommitteeDate)
	require.Equal(t, laporanhasilai.AIAccepted, first.AIStatus)
	require.Equal(t, day(2), first.AIDate)

	// Nomor klaim hanya pada komite ke-1.
	require.Empty(t, page.Rows[1].ClaimNumber)
	require.Equal(t, laporanhasilai.LabelRejected, page.Rows[1].CommitteeStatus)
	require.Equal(t, laporanhasilai.LabelPending, page.Rows[2].CommitteeStatus)
}

func TestListPagesAndShortensTheLastPage(t *testing.T) {
	page, err := sampleRepo().List(context.Background(), september,
		laporanhasilai.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Rows, 1)
	require.Equal(t, "K3|1|O1|C1", page.Rows[0].ID)
}

func TestListBeyondLastPageIsEmpty(t *testing.T) {
	page, err := sampleRepo().List(context.Background(), september,
		laporanhasilai.Pagination{Page: 5, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Empty(t, page.Rows)
}

func TestSummarizeCountsEveryDecision(t *testing.T) {
	summary, err := sampleRepo().Summarize(context.Background(), september)
	require.NoError(t, err)

	require.Equal(t, laporanhasilai.Tally{
		Subject: laporanhasilai.SubjectCommittee, Accepted: 1, Rejected: 1, Pending: 1,
	}, summary.Committee)
	require.Equal(t, laporanhasilai.Tally{
		Subject: laporanhasilai.SubjectAI, Accepted: 1, Rejected: 1, Pending: 1,
	}, summary.AI)
}

func TestSummarizeEmptyRangeIsAllZero(t *testing.T) {
	summary, err := sampleRepo().Summarize(context.Background(), laporanhasilai.Filter{
		From: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Equal(t, 0, summary.AI.Rows())
	require.Equal(t, 0, summary.Committee.Rows())
}

func TestSampleRowsDropTheRowsThatMustDisappear(t *testing.T) {
	// Contoh bawaan sengaja memuat baris yang wajib hilang: hasilnya harus lebih sedikit
	// daripada seluruh contoh pada rentang yang mencakup semuanya.
	all := memory.SampleRows()
	repo := memory.NewRepo(all...)

	page, err := repo.List(context.Background(), laporanhasilai.Filter{
		From: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2100, time.January, 1, 0, 0, 0, 0, time.UTC),
	}, laporanhasilai.Pagination{Size: laporanhasilai.MaxPageSize})
	require.NoError(t, err)
	require.NotZero(t, page.Total)
	require.Less(t, page.Total, len(all))
}
