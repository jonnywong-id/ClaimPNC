package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/memory"
)

func claimNumbers(rows []dashboardclaim.ClaimRow) []string {
	out := []string{}
	for _, row := range rows {
		out = append(out, row.ClaimNumber)
	}
	return out
}

// Penyaring lini bisnis meniru `SetDashboardClaim`: Non-MBU hanya 003/004/006 di luar grup
// bonding, Bonding menurut grup bisnis, PA 002, Travel 005.
func TestBusinessLineFilter(t *testing.T) {
	repo := memory.NewRepo(memory.SampleOutstanding(), nil)
	ctx := context.Background()

	cases := map[dashboardclaim.BusinessLine][]string{
		dashboardclaim.BusinessAll: {
			"PNCN.26.0001", "PNCN.26.0002", "PNCN.26.0003", "PNCN.26.0004", "PNCN.26.0005",
		},
		dashboardclaim.BusinessNonMBU:  {"PNCN.26.0001"},
		dashboardclaim.BusinessBonding: {"PNCN.26.0004"},
		dashboardclaim.BusinessPA:      {"PNCN.26.0002"},
		dashboardclaim.BusinessTravel:  {"PNCN.26.0003"},
		"TIDAK-DIKENAL":                {},
	}
	for line, want := range cases {
		page, err := repo.ListOutstanding(ctx, dashboardclaim.Filter{Business: line})
		require.NoError(t, err)
		require.Equal(t, want, claimNumbers(page.Rows), line)
		require.Equal(t, len(want), page.Total, line)

		count, err := repo.CountOutstanding(ctx, dashboardclaim.Filter{Business: line})
		require.NoError(t, err)
		require.Equal(t, len(want), count, line)
	}
}

// Group Panel 006 ikut Non-MBU selama grup bisnisnya bukan bonding.
func TestNonMBUIncludesFireButExcludesBondingGroup(t *testing.T) {
	repo := memory.NewRepo([]memory.ClaimRecord{
		{GroupPanel: "006", BusinessGroupID: "10001", Row: dashboardclaim.ClaimRow{ClaimNumber: "A"}},
		{GroupPanel: "006", BusinessGroupID: "10023", Row: dashboardclaim.ClaimRow{ClaimNumber: "B"}},
	}, nil)

	page, err := repo.ListOutstanding(context.Background(),
		dashboardclaim.Filter{Business: dashboardclaim.BusinessNonMBU})
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, claimNumbers(page.Rows))
}

// Kotak cari tidak peka huruf besar-kecil dan memeriksa nomor polis serta nomor klaim.
func TestSearchAndPagination(t *testing.T) {
	repo := memory.NewRepo(memory.SampleOutstanding(), nil)
	ctx := context.Background()

	page, err := repo.ListOutstanding(ctx, dashboardclaim.Filter{Search: "polis-contoh-0003"})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0003"}, claimNumbers(page.Rows))

	page, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{Search: "pncn.26.000"})
	require.NoError(t, err)
	require.Equal(t, 5, page.Total)

	page, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{Limit: 2, Offset: 4})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0005"}, claimNumbers(page.Rows))
	require.Equal(t, 5, page.Total)

	page, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{Limit: 2, Offset: 10})
	require.NoError(t, err)
	require.Empty(t, page.Rows)
	require.NotNil(t, page.Rows)
	require.Equal(t, 5, page.Total)
}

// Hitungan loss adjuster menghitung KLAIM unik; surveyor internal menghitung baris survei.
func TestCountSurveyByKind(t *testing.T) {
	surveys := append(memory.SampleSurveys(), memory.SurveyRecord{
		Kind: dashboardclaim.SurveyorAdjuster, GroupPanel: "003", BusinessGroupID: "10001",
		Row: dashboardclaim.SurveyRow{SurveyNumber: "SRV-X", ClaimNumber: "PNCN.26.0001"},
	})
	repo := memory.NewRepo(nil, surveys)
	ctx := context.Background()

	adjuster, err := repo.CountSurvey(ctx, dashboardclaim.SurveyorAdjuster, dashboardclaim.Filter{})
	require.NoError(t, err)
	require.Equal(t, 2, adjuster, "survei kedua atas klaim yang sama tidak menambah hitungan")

	page, err := repo.ListSurvey(ctx, dashboardclaim.SurveyorAdjuster, dashboardclaim.Filter{})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total, "daftar tetap menampilkan setiap baris survei")

	internal, err := repo.CountSurvey(ctx, dashboardclaim.SurveyorInternal,
		dashboardclaim.Filter{Business: dashboardclaim.BusinessTravel})
	require.NoError(t, err)
	require.Equal(t, 1, internal)

	searched, err := repo.ListSurvey(ctx, dashboardclaim.SurveyorInternal,
		dashboardclaim.Filter{Search: "srv-contoh-0004"})
	require.NoError(t, err)
	require.Len(t, searched.Rows, 1)
	require.Equal(t, "Denpasar", searched.Rows[0].SurveyLocation)

	none, err := repo.ListSurvey(ctx, dashboardclaim.SurveyorInternal,
		dashboardclaim.Filter{Business: dashboardclaim.BusinessPA})
	require.NoError(t, err)
	require.Empty(t, none.Rows)
}

// Pembaca klaim tutup memakai penyaring yang sama dan mengabaikan alias portal.
func TestClosedReader(t *testing.T) {
	reader := memory.NewClosedReader(memory.SampleClosed())
	ctx := context.Background()

	count, err := reader.Count(ctx, "ASM", dashboardclaim.Filter{})
	require.NoError(t, err)
	require.Equal(t, 2, count)

	page, err := reader.List(ctx, "ASM", dashboardclaim.Filter{Business: dashboardclaim.BusinessPA})
	require.NoError(t, err)
	require.Equal(t, []string{"PNCN.26.0102"}, claimNumbers(page.Rows))
	require.Equal(t, 1, page.Total)

	page, err = reader.List(ctx, "ASM", dashboardclaim.Filter{Search: "tidak-ada"})
	require.NoError(t, err)
	require.Empty(t, page.Rows)
}
