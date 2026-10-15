package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Uji pembantu yang tidak dapat dicapai dari luar paket dengan contoh bawaan.

// Tangga nilai 1–5 ditiru kata demi kata dari `CASE` kuerinya.
func TestSLAScoreLadder(t *testing.T) {
	cases := []struct {
		percent float64
		want    float64
	}{
		{0, 5}, {0.49, 5}, {0.5, 4}, {1, 4}, {1.2, 2}, {1.5, 2}, {1.7, 1}, {2, 1}, {2.01, 0}, {50, 0},
	}
	for _, c := range cases {
		require.Equalf(t, reportkpi.NewScore(c.want), slaScore(reportkpi.NewScore(c.percent)),
			"persentase %v", c.percent)
	}
	require.Equal(t, reportkpi.EmptyScore(), slaScore(reportkpi.EmptyScore()))
}

func TestSLALabel(t *testing.T) {
	require.Equal(t, "SLA", slaLabel(1))
	require.Equal(t, "TIDAK SLA", slaLabel(1.01))
}

func TestAdminWithinYear(t *testing.T) {
	require.True(t, adminWithinYear("2023-05-10", 2023))
	require.False(t, adminWithinYear("2024-05-10", 2023))
	require.False(t, adminWithinYear("bukan tanggal", 2023))
}

// Rentang yang tidak terbaca dan tanggal baris yang tidak terbaca sama-sama tidak lolos.
func TestWithinRangeRejectsUnreadableDates(t *testing.T) {
	good := reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"}
	require.True(t, withinRange("2026-03-31", good))
	require.False(t, withinRange("2026-04-01", good))
	require.False(t, withinRange("", good))
	require.False(t, withinRange("2026-03-10", reportkpi.DateRange{From: "x", To: "2026-03-31"}))
	require.False(t, withinRange("2026-03-10", reportkpi.DateRange{From: "2026-03-01", To: "y"}))
}

// Nilai yang lebih sedikit daripada komponen hanya mengisi komponen awal.
func TestFullScoresStopsAtShorterInput(t *testing.T) {
	scores := fullScores("1", "2")
	require.Equal(t, map[string]string{
		reportkpi.ComponentSurvey:          "1",
		reportkpi.ComponentImmediateAdvice: "2",
	}, scores)
}

// Hanya kelompok leader yang punya klaim: nilai member kosong, sehingga subtotal dan
// rasio pencapaian tidak dihitung.
func TestAdminTotalsNonMBUWithoutMembersLeavesSubtotalEmpty(t *testing.T) {
	store := NewStore().WithAdminRows([]AdminRow{
		{Group: reportkpi.AdminGroupNonMBU, Leader: true, RegisterDate: "2026-03-02", RegisterAging: 2},
	})
	totals, err := store.AdminTotals(context.Background(), reportkpi.AdminQuery{
		Group: reportkpi.AdminGroupNonMBU,
		Range: reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	})
	require.NoError(t, err)
	require.Equal(t, reportkpi.NewScore(1), totals.LeaderOverSLA)
	require.Equal(t, reportkpi.NewScore(100), totals.LeaderPercent)
	require.Equal(t, reportkpi.NewScore(0), totals.LeaderScore)
	require.False(t, totals.MemberPercent.Present)
	require.False(t, totals.LeaderSubtotal.Present)
	require.False(t, totals.AchievementRatio.Present)
}

// Member yang melewati SLA ikut tercacah pada pencacah member.
func TestAdminTotalsNonMBUCountsMemberOverSLA(t *testing.T) {
	store := NewStore().WithAdminRows([]AdminRow{
		{Group: reportkpi.AdminGroupNonMBU, RegisterDate: "2026-03-02", RegisterAging: 1.5},
		{Group: reportkpi.AdminGroupNonMBU, RegisterDate: "2026-03-03", RegisterAging: 0.5},
	})
	totals, err := store.AdminTotals(context.Background(), reportkpi.AdminQuery{
		Group: reportkpi.AdminGroupNonMBU,
		Range: reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	})
	require.NoError(t, err)
	require.Equal(t, reportkpi.NewScore(1), totals.MemberOverSLA)
	require.Equal(t, reportkpi.NewScore(2), totals.MemberTotal)
	require.Equal(t, reportkpi.NewScore(50), totals.MemberPercent)
}

// Halaman di luar jumlah baris menghasilkan halaman kosong dengan jumlah yang tetap benar.
func TestPagesBeyondTotalAreEmpty(t *testing.T) {
	store := NewSampleStore()
	beyond := reportkpi.Pagination{Page: 99, Size: 10}

	detail, err := store.Detail(context.Background(), reportkpi.Query{
		ReportType: reportkpi.TypeAll,
		Range:      reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	}, beyond)
	require.NoError(t, err)
	require.Empty(t, detail.Rows)
	require.Equal(t, 7, detail.Total)

	admin, err := store.AdminDetail(context.Background(), reportkpi.AdminQuery{
		Group: reportkpi.AdminGroupNonMBU,
		Range: reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	}, beyond)
	require.NoError(t, err)
	require.Empty(t, admin.Rows)
	require.Equal(t, 7, admin.Total)
}
