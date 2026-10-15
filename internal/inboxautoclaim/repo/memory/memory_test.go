package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
	"claim-pnc/internal/inboxautoclaim/repo/memory"
)

const success = inboxautoclaim.MessageSuccess

func newRepo() *memory.Repo {
	return memory.NewRepo(
		map[string]string{" BRI ": "Bank Rakyat", "BNI": "Bank Negara"},
		map[string]memory.PolicyRow{
			" p1 ": {CompanyCode: "BRI", ProductSeq: "1", OpenProtection: []string{"5"},
				Detail: inboxautoclaim.PolicyDetail{BusinessCode: "01", SourceOfBusiness: "BRI"}},
			"P2": {CompanyCode: "", ProductSeq: ""},
			"P3": {CompanyCode: "BNI", ProductSeq: "2",
				Detail: inboxautoclaim.PolicyDetail{BusinessCode: "01", SourceOfBusiness: "BRI"}},
		},
		inboxautoclaim.Line{CompanyCode: "BRI", BatchNumber: "2", PolicyNo: "B", Message: success, UploadedBy: "u", ProcessedDate: "01/09/2026"},
		inboxautoclaim.Line{CompanyCode: "BRI", BatchNumber: "2", PolicyNo: "A", Message: "Gagal", UploadedBy: "u", ProcessedDate: "01/09/2026"},
		inboxautoclaim.Line{CompanyCode: "BRI", BatchNumber: "2", PolicyNo: "A", UploadedBy: "u", ProcessedDate: "01/09/2026", DateOfLoss: "x"},
		inboxautoclaim.Line{CompanyCode: "BRI", BatchNumber: "10", PolicyNo: "C", Message: success, UploadedBy: "u"},
		inboxautoclaim.Line{CompanyCode: "BNI", BatchNumber: "1", PolicyNo: "D", Message: success, UploadedBy: "v"},
		inboxautoclaim.Line{CompanyCode: "BNI", BatchNumber: "X", PolicyNo: "E", UploadedBy: "v"},
	)
}

func TestListBatchGroupsCountsAndPages(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	page, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka})
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	// Nomor angka diurutkan menurun; nomor bukan angka jatuh ke urutan teks.
	numbers := []string{}
	for _, b := range page.Item {
		numbers = append(numbers, b.BatchNumber)
	}
	require.Equal(t, []string{"X", "10", "2", "1"}, numbers)
	two := page.Item[2]
	require.Equal(t, inboxautoclaim.Batch{CompanyCode: "BRI", CompanyName: "Bank Rakyat", BatchNumber: "2",
		ProcessedDate: "01/09/2026", Uploaded: 3, Processed: 2, Succeeded: 1, Failed: 1, UploadedBy: "u"}, two)

	byCompany, _ := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka, CompanyCode: " BNI "})
	require.Equal(t, 2, byCompany.Total)

	second, _ := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		Page: inboxautoclaim.PageRequest{Number: 2, Size: 3}})
	require.Len(t, second.Item, 1)
	beyond, _ := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: inboxautoclaim.SourceAneka,
		Page: inboxautoclaim.PageRequest{Number: 9, Size: 3}})
	require.Empty(t, beyond.Item)
	require.Equal(t, 4, beyond.Total)
}

func TestSummarizeAndListCompany(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	summary, err := repo.SummarizeCompany(ctx, inboxautoclaim.SourceAneka)
	require.NoError(t, err)
	require.Equal(t, 4, summary.Total)
	require.Equal(t, []inboxautoclaim.CompanySummary{
		{Code: "BNI", Name: "Bank Negara", BatchCount: 2},
		{Code: "BRI", Name: "Bank Rakyat", BatchCount: 2},
	}, summary.Company)

	empty, _ := repo.SummarizeCompany(ctx, inboxautoclaim.SourceTravel)
	require.Zero(t, empty.Total)
	require.NotNil(t, empty.Company)

	companies, err := repo.ListCompany(ctx)
	require.NoError(t, err)
	require.Equal(t, []inboxautoclaim.Company{{Code: "BNI", Name: "Bank Negara"}, {Code: "BRI", Name: "Bank Rakyat"}}, companies)
}

func TestListLineFiltersByResultAndSorts(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()
	base := inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka, CompanyCode: "bri", BatchNumber: " 2 "}

	all, err := repo.ListLine(ctx, base)
	require.NoError(t, err)
	require.Equal(t, 3, all.Total)
	require.Equal(t, []string{"A", "A", "B"}, []string{all.Item[0].PolicyNo, all.Item[1].PolicyNo, all.Item[2].PolicyNo})
	require.Equal(t, "", all.Item[0].DateOfLoss)

	succeeded := base
	succeeded.Result = inboxautoclaim.ResultSucceeded
	got, _ := repo.ListLine(ctx, succeeded)
	require.Equal(t, 1, got.Total)
	require.Equal(t, "B", got.Item[0].PolicyNo)

	failed := base
	failed.Result = inboxautoclaim.ResultFailed
	got, _ = repo.ListLine(ctx, failed)
	require.Equal(t, 1, got.Total, "baris yang belum diproses bukan gagal")
	require.Equal(t, "Gagal", got.Item[0].Message)

	paged := base
	paged.Page = inboxautoclaim.PageRequest{Number: 5, Size: 2}
	got, _ = repo.ListLine(ctx, paged)
	require.Empty(t, got.Item)

	exported, err := repo.ExportLine(ctx, base)
	require.NoError(t, err)
	require.Len(t, exported, 3)
}

func TestLineSortFallsBackToProcessedDate(t *testing.T) {
	repo := memory.NewRepo(nil, nil,
		inboxautoclaim.Line{CompanyCode: "A", BatchNumber: "1", PolicyNo: "P", ProcessedDate: "02"},
		inboxautoclaim.Line{CompanyCode: "A", BatchNumber: "1", PolicyNo: "P", ProcessedDate: "01"},
	)
	got, err := repo.ExportLine(context.Background(), inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka, CompanyCode: "A", BatchNumber: "1"})
	require.NoError(t, err)
	require.Equal(t, "01", got[0].ProcessedDate)
}

func TestBatchExists(t *testing.T) {
	repo := newRepo()
	found, err := repo.BatchExists(context.Background(), inboxautoclaim.SourceAneka, " bri ", " 10 ")
	require.NoError(t, err)
	require.True(t, found)
	found, _ = repo.BatchExists(context.Background(), inboxautoclaim.SourceAneka, "BRI", "99")
	require.False(t, found)
}

func TestPolicyLookups(t *testing.T) {
	repo := newRepo()
	ctx := context.Background()

	company, found, err := repo.ResolveReceiver(ctx, " P1 ")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, inboxautoclaim.Company{Code: "BRI", Name: "Bank Rakyat"}, company)
	_, found, _ = repo.ResolveReceiver(ctx, "P2")
	require.False(t, found)
	_, found, _ = repo.ResolveReceiver(ctx, "TIDAK")
	require.False(t, found)

	seq, found, _ := repo.FindPolicyProductSeq(ctx, "p1")
	require.True(t, found)
	require.Equal(t, "1", seq)
	_, found, _ = repo.FindPolicyProductSeq(ctx, "P2")
	require.False(t, found)

	detail, found, _ := repo.FindPolicyDetail(ctx, "P1", " 1 ")
	require.True(t, found)
	require.Equal(t, "01", detail.BusinessCode)
	_, found, _ = repo.FindPolicyDetail(ctx, "P1", "2")
	require.False(t, found)
	_, found, _ = repo.FindPolicyDetail(ctx, "P2", "")
	require.False(t, found)

	id, found, _ := repo.CurrencyID(ctx, " usd ")
	require.True(t, found)
	require.Equal(t, "2", id)
	_, found, _ = repo.CurrencyID(ctx, "JPY")
	require.False(t, found)

	open, _ := repo.HasOpenProtection(ctx, "P1", " 5 ")
	require.True(t, open)
	open, _ = repo.HasOpenProtection(ctx, "P1", "6")
	require.False(t, open)
}

func TestContractClaimedLooksAtKreditLines(t *testing.T) {
	repo := newRepo()
	repo.Seed(inboxautoclaim.SourceKredit,
		inboxautoclaim.Line{CompanyCode: "BRI", CauseOfLoss: "K-1", Message: "Gagal"},
		inboxautoclaim.Line{CompanyCode: "BRI", CauseOfLoss: "K-2", Message: ""},
		inboxautoclaim.Line{CompanyCode: "BRI", CauseOfLoss: "K-3", Message: success},
	)
	ctx := context.Background()
	claimed, err := repo.ContractClaimed(ctx, "BRI", " ")
	require.NoError(t, err)
	require.False(t, claimed)
	claimed, _ = repo.ContractClaimed(ctx, "BRI", "k-1")
	require.False(t, claimed, "baris gagal tidak menandai kontrak terpakai")
	claimed, _ = repo.ContractClaimed(ctx, "BRI", "k-2")
	require.True(t, claimed, "baris belum diproses tetap menandai kontrak terpakai")
	claimed, _ = repo.ContractClaimed(ctx, " BRI ", "K-3")
	require.True(t, claimed)
	claimed, _ = repo.ContractClaimed(ctx, "BNI", "K-3")
	require.False(t, claimed)
}

func TestPremiumCheck(t *testing.T) {
	repo := newRepo()
	repo.SetBusiness(map[string]string{" 01 ": "Fire", "02": "Fire"})
	ctx := context.Background()

	choices, err := repo.PremiumCheckChoices(ctx)
	require.NoError(t, err)
	require.Equal(t, []inboxautoclaim.Choice{{Code: "01", Name: "Fire"}, {Code: "02", Name: "Fire"}}, choices.Business)
	require.Equal(t, []inboxautoclaim.Choice{{Code: "BNI", Name: "Bank Negara"}, {Code: "BRI", Name: "Bank Rakyat"}}, choices.SourceOfBusiness)

	query := inboxautoclaim.PremiumCheckQuery{BusinessCode: "01", SourceOfBusiness: "BRI"}
	total, err := repo.SucceededClaimTotal(ctx, query)
	require.NoError(t, err)
	require.Empty(t, total, "tanpa klaim sukses jumlahnya kosong, bukan nol")

	repo.Seed(inboxautoclaim.SourceKredit,
		inboxautoclaim.Line{PolicyNo: "P1", Message: success, ClaimAmount: "100"},
		inboxautoclaim.Line{PolicyNo: "P3", Message: success, ClaimAmount: "0.25"},
		inboxautoclaim.Line{PolicyNo: "P3", Message: success, ClaimAmount: "bukan angka"},
		inboxautoclaim.Line{PolicyNo: "P1", Message: "Gagal", ClaimAmount: "999"},
		inboxautoclaim.Line{PolicyNo: "P2", Message: success, ClaimAmount: "7"},
	)
	total, _ = repo.SucceededClaimTotal(ctx, query)
	require.Equal(t, "100.25", total)

	repo.Seed(inboxautoclaim.SourceKredit, inboxautoclaim.Line{PolicyNo: "P1", Message: success, ClaimAmount: "0.75"})
	total, _ = repo.SucceededClaimTotal(ctx, query)
	require.Equal(t, "101", total)
}

func TestInsertUploadNumbersBatchAndStoresLines(t *testing.T) {
	repo := newRepo()
	repo.SetProcessedDate("20/09/2026")
	ctx := context.Background()

	_, err := repo.InsertUpload(ctx, inboxautoclaim.SourceAneka, nil, "admin")
	require.ErrorIs(t, err, inboxautoclaim.ErrEmptyUpload)

	result, err := repo.InsertUpload(ctx, inboxautoclaim.SourceAneka, []inboxautoclaim.UploadLine{
		{CompanyCode: "bri", ProductSeq: "1", CurrencyID: "1", Row: inboxautoclaim.UploadRow{PolicyNo: "Z",
			ClaimAmount: "5", CauseOfLoss: "Banjir", Reason: "r"}},
		{CompanyCode: "BRI", Message: "Polis tidak ada", Row: inboxautoclaim.UploadRow{PolicyNo: "Y"}},
	}, "admin")
	require.NoError(t, err)
	require.Equal(t, 2, result.Rows)
	require.Equal(t, inboxautoclaim.BatchRef{CompanyCode: "BRI", CompanyName: "Bank Rakyat", BatchNumber: "11",
		Rows: 2, Succeeded: 1, Failed: 1}, result.Batch[0])

	lines, _ := repo.ExportLine(ctx, inboxautoclaim.LineQuery{Source: inboxautoclaim.SourceAneka, CompanyCode: "BRI", BatchNumber: "11"})
	require.Len(t, lines, 2)
	require.Equal(t, "20/09/2026", lines[0].ProcessedDate)
	require.Equal(t, "Polis tidak ada", lines[0].ClaimID)
	require.Equal(t, "Banjir", lines[1].CauseOfLoss)

	// Tab Kredit menyimpan nomor kontrak (huruf besar) di kolom rujukan.
	_, err = repo.InsertUpload(ctx, inboxautoclaim.SourceKredit, []inboxautoclaim.UploadLine{
		{CompanyCode: "BNI", Row: inboxautoclaim.UploadRow{PolicyNo: "Q", ContractNo: " k-9 "}},
	}, "admin")
	require.NoError(t, err)
	claimed, _ := repo.ContractClaimed(ctx, "BNI", "K-9")
	require.True(t, claimed)
}

func TestSampleRepoHasEveryTab(t *testing.T) {
	repo := memory.NewSampleRepo()
	ctx := context.Background()
	for _, source := range inboxautoclaim.AllSource() {
		page, err := repo.ListBatch(ctx, inboxautoclaim.BatchFilter{Source: source})
		require.NoError(t, err)
		require.NotZero(t, page.Total, string(source))
	}
	require.NotEmpty(t, memory.SampleMaster())
	require.NotEmpty(t, memory.SamplePolicy())
	require.NotEmpty(t, memory.SampleBusiness())
	choices, _ := repo.PremiumCheckChoices(ctx)
	require.Len(t, choices.Business, len(memory.SampleBusiness()))
}
