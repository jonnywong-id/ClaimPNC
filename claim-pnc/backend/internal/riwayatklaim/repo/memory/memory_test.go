package memory_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/repo/memory"
)

func criteria(code, text string, date *time.Time) riwayatklaim.Criteria {
	searchType, found := riwayatklaim.FindSearchType(code)
	if !found {
		searchType = riwayatklaim.SearchType{Code: code}
	}
	return riwayatklaim.Criteria{Type: searchType, Text: text, SearchDate: date}
}

func day(year int, month time.Month, d int) *time.Time {
	value := time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
	return &value
}

func numbers(page riwayatklaim.Page) []string {
	out := []string{}
	for _, claim := range page.Claims {
		out = append(out, claim.Number)
	}
	return out
}

func search(t *testing.T, c riwayatklaim.Criteria) riwayatklaim.Page {
	t.Helper()
	page, err := memory.NewRepo(memory.SampleClaims()...).
		Search(context.Background(), c, riwayatklaim.Pagination{Page: 1, Size: 50})
	require.NoError(t, err)
	return page
}

func TestSearchByEachTextType(t *testing.T) {
	cases := map[string]struct {
		code, text string
		want       []string
	}{
		"polis persis":     {riwayatklaim.TypePolicyNumber, " pol-contoh-0002 ", []string{"PNC-9002"}},
		"nama sebagian":    {riwayatklaim.TypeInsuredName, "contoh jaya", []string{"PNC-9003"}},
		"nama objek":       {riwayatklaim.TypeInsuredItemName, "kontainer", []string{"PNC-9002"}},
		"PLA":              {riwayatklaim.TypePLANumber, "PLA-CONTOH-0001", []string{"PNC-9001"}},
		"DLA":              {riwayatklaim.TypeDLANumber, "DLA-CONTOH-0003", []string{"PNC-9003"}},
		"nomor klaim baru": {riwayatklaim.TypeClaimNumber, "pncn.26.0001", []string{"PNCN.26.0001"}},
		"akseptasi":        {riwayatklaim.TypeAcceptanceNumber, "AKS-CONTOH-0001", []string{"PNC-9001"}},
		"survei":           {riwayatklaim.TypeSurveyNumber, "srv-contoh-0004", []string{"PNC-9004"}},
		"balai lelang":     {riwayatklaim.TypeAuctionHouseID, "BL-CONTOH-77", []string{"PNC-9003"}},
		"tidak dikenal":    {"99", "PNC", []string{}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, numbers(search(t, criteria(tc.code, tc.text, nil))))
		})
	}
}

func TestSearchByLossDateMatchesCalendarDay(t *testing.T) {
	page := search(t, criteria(riwayatklaim.TypeLossDate, "", day(2026, time.March, 12)))
	require.Equal(t, []string{"PNC-9001", "PNC-9002"}, numbers(page))
	require.Equal(t, 2, page.Total)

	// Tanggal kosong pada tipe tanggal tidak mencocokkan apa pun.
	require.Empty(t, search(t, criteria(riwayatklaim.TypeLossDate, "", nil)).Claims)
}

func TestSearchByBirthDateDecoratesPersonColumns(t *testing.T) {
	page := search(t, criteria(riwayatklaim.TypeBirthDate, "", day(1990, time.July, 17)))
	require.Equal(t, []string{"PNC-9004"}, numbers(page))
	require.Equal(t, "Peserta Contoh Satu", page.Claims[0].InsuredItemName)
	require.Equal(t, day(1990, time.July, 17), page.Claims[0].BirthDate)

	// Baris tanpa tanggal lahir tidak pernah cocok.
	require.Empty(t, search(t, criteria(riwayatklaim.TypeBirthDate, "", day(2001, 1, 1))).Claims)
}

func TestSearchByAccountNumberDateNeverMatches(t *testing.T) {
	page := search(t, criteria(riwayatklaim.TypeAccountNumber, "", day(2026, time.March, 12)))
	require.Empty(t, page.Claims)
}

func TestSearchDecorationForAuctionAndItemName(t *testing.T) {
	auction := search(t, criteria(riwayatklaim.TypeAuctionHouseID, "BL-CONTOH-77", nil))
	require.Equal(t, "AKS-CONTOH-0003", auction.Claims[0].AcceptanceNumber)
	require.Equal(t, "BL-CONTOH-77", auction.Claims[0].AuctionHouseID)

	item := search(t, criteria(riwayatklaim.TypeInsuredItemName, "gudang", nil))
	require.Empty(t, item.Claims[0].ClaimPosition)
}

func TestSearchPaginatesAndReturnsEmptyBeyondEnd(t *testing.T) {
	repo := memory.NewRepo(memory.SampleClaims()...)
	c := criteria(riwayatklaim.TypeInsuredName, "CONTOH", nil)

	first, err := repo.Search(context.Background(), c, riwayatklaim.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Len(t, first.Claims, 2)
	require.Equal(t, 5, first.Total)
	// Diurutkan menurut Reference.
	require.Equal(t, []string{"PNC-9001", "PNC-9002"}, numbers(first))

	far, err := repo.Search(context.Background(), c, riwayatklaim.Pagination{Page: 9, Size: 2})
	require.NoError(t, err)
	require.Empty(t, far.Claims)
	require.NotNil(t, far.Claims)
	require.Equal(t, 5, far.Total)
}

func TestProtectionRepoFindCountAndRecord(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewProtectionRepo(memory.SampleProtections()...)

	found, registered, err := repo.Find(ctx, " ADMINPNC ", riwayatklaim.ModuleKey)
	require.NoError(t, err)
	require.True(t, registered)
	require.Equal(t, 50, found.SearchQuota)

	_, registered, err = repo.Find(ctx, "profilbolong", riwayatklaim.ModuleKey)
	require.NoError(t, err)
	require.False(t, registered)

	require.NoError(t, repo.RecordUsage(ctx, riwayatklaim.Usage{
		Login: "adminpnc", Module: riwayatklaim.ModuleKey, ConsumesQuota: true,
	}))
	require.NoError(t, repo.RecordUsage(ctx, riwayatklaim.Usage{
		Login: "adminpnc", Module: riwayatklaim.ModuleKey, ConsumesQuota: false,
	}))
	require.NoError(t, repo.RecordUsage(ctx, riwayatklaim.Usage{
		Login: "adminpnc", Module: "LAIN", ConsumesQuota: true,
	}))
	require.NoError(t, repo.RecordUsage(ctx, riwayatklaim.Usage{
		Login: "lain", Module: riwayatklaim.ModuleKey, ConsumesQuota: true,
	}))

	used, err := repo.CountUsage(ctx, "AdminPNC", strings.ToLower(riwayatklaim.ModuleKey))
	require.NoError(t, err)
	require.Equal(t, 1, used)

	usage := repo.Usage()
	require.Len(t, usage, 4)
	usage[0].Login = "rusak"
	require.Equal(t, "adminpnc", repo.Usage()[0].Login)
}
