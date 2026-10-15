package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxlaporanklaim"
)

// fixedClock adalah jam tetap supaya isi contoh dan nomor terbit deterministik.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

var sampleNow = time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)

func newSampleRepo() *Repo {
	return NewRepo(SampleOptions(fixedClock{now: sampleNow}))
}

func ids(page inboxlaporanklaim.Page) []string {
	result := make([]string, 0, len(page.Report))
	for _, r := range page.Report {
		result = append(result, r.ID)
	}
	return result
}

func list(t *testing.T, repo *Repo, filter inboxlaporanklaim.Filter) inboxlaporanklaim.Page {
	t.Helper()
	page, err := repo.List(context.Background(), filter, inboxlaporanklaim.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)
	return page
}

// Tab bawaan mengecualikan berkas selesai dan ditolak, diurutkan aging terbaru dulu.
func TestListAllExcludesResolvedAndSortsByAgingDesc(t *testing.T) {
	page := list(t, newSampleRepo(), inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll})

	require.Equal(t, 11, page.Total)
	require.Equal(t, []string{
		"RCV-0012", "RCV-0008", "RCV-0006", "RCV-0005", "RCV-0003", "RCV-0002",
		"RCV-0001", "RCV-0009", "RCV-0007", "RCV-0004", "RCV-0010",
	}, ids(page))
	for _, r := range page.Report {
		require.Empty(t, r.LastMessage, "pesan terakhir hanya diisi pada tab komunikasi")
	}
}

func TestListPaginationWindowsAndClampsOffset(t *testing.T) {
	repo := newSampleRepo()
	filter := inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll}

	second, err := repo.List(context.Background(), filter, inboxlaporanklaim.Pagination{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Equal(t, []string{"RCV-0010"}, ids(second))
	require.Equal(t, 11, second.Total)

	beyond, err := repo.List(context.Background(), filter, inboxlaporanklaim.Pagination{Page: 9, Size: 10})
	require.NoError(t, err)
	require.Empty(t, beyond.Report)
	require.Equal(t, 11, beyond.Total)
}

// Pemutus seri: aging yang sama diurutkan nomor register menurun.
func TestListBreaksTiesByIDDescending(t *testing.T) {
	same := sampleNow
	repo := NewRepo(Options{
		Rows: []inboxlaporanklaim.ClaimReport{
			{ID: "A", AgingAt: same}, {ID: "C", AgingAt: same}, {ID: "B", AgingAt: same},
		},
		Clock: fixedClock{now: sampleNow},
	})
	page := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll})
	require.Equal(t, []string{"C", "B", "A"}, ids(page))
}

func TestListCategories(t *testing.T) {
	repo := newSampleRepo()
	cases := map[inboxlaporanklaim.Category][]string{
		inboxlaporanklaim.CategoryOutstanding:       {"RCV-0001", "RCV-0007", "RCV-0004", "RCV-0010"},
		inboxlaporanklaim.CategoryUnregistered:      {"RCV-0006", "RCV-0002", "RCV-0009"},
		inboxlaporanklaim.CategoryNotTransferred:    {"RCV-0012", "RCV-0008", "RCV-0005", "RCV-0003"},
		inboxlaporanklaim.CategoryAccepted:          {"RCV-0004", "RCV-0010"},
		inboxlaporanklaim.CategoryRejected:          {"RCV-0011", "RCV-0014"},
		inboxlaporanklaim.Category("tidak-dikenal"): {},
	}
	for category, want := range cases {
		page := list(t, repo, inboxlaporanklaim.Filter{Category: category})
		require.Equalf(t, want, ids(page), "tab %q", category)
	}
}

func TestListMessageTabsFilterOwnReportsAndLastMessage(t *testing.T) {
	repo := newSampleRepo()

	// Tanpa operator, tab komunikasi selalu kosong.
	empty := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryMessageWaiting})
	require.Empty(t, empty.Report)

	waiting := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageWaiting, Operator: "adminpnc",
	})
	require.Equal(t, []string{"RCV-0012", "RCV-0002"}, ids(waiting))
	require.Equal(t, "Berkas menunggu konfirmasi polis.", waiting.Report[0].LastMessage)

	replied := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageReplied, Operator: "adminpnc",
	})
	require.Equal(t, []string{"RCV-0012", "RCV-0003"}, ids(replied))
	require.Equal(t, "Polis sedang kami telusuri.", replied.Report[0].LastMessage)

	unanswered := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageUnanswered, Operator: "adminpnc",
	})
	require.Equal(t, []string{"RCV-0006"}, ids(unanswered))

	// Berkas milik orang lain tidak pernah tampil di tab komunikasi.
	other := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageReplied, Operator: "pictekniks",
	})
	require.Empty(t, other.Report)
}

// Dari beberapa pesan yang cocok, yang paling baru yang ditampilkan.
func TestLastMessagePicksNewest(t *testing.T) {
	repo := NewRepo(Options{
		Rows: []inboxlaporanklaim.ClaimReport{{ID: "R1", CreatedBy: "op"}},
		Message: map[string][]messageRow{
			"R1": {
				{Status: "0", Sender: "op", Text: "lama", SentAt: sampleNow.Add(-2 * time.Hour)},
				{Status: "0", Sender: "op", Text: "baru", SentAt: sampleNow},
				{Status: "0", Sender: "op", Text: "tengah", SentAt: sampleNow.Add(-time.Hour)},
			},
		},
		Clock: fixedClock{now: sampleNow},
	})
	page := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageWaiting, Operator: "op",
	})
	require.Len(t, page.Report, 1)
	require.Equal(t, "baru", page.Report[0].LastMessage)
}

func TestListScopeFilters(t *testing.T) {
	repo := newSampleRepo()

	branch := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, BranchCode: "2001"})
	require.Equal(t, []string{"RCV-0007"}, ids(branch))

	region := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, RegionCode: "03"})
	require.Equal(t, []string{"RCV-0008", "RCV-0009"}, ids(region))

	// Pencarian kata kunci persis, tanpa membedakan huruf besar-kecil.
	keyword := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, Keyword: "rcv-0005"})
	require.Equal(t, []string{"RCV-0005"}, ids(keyword))

	partial := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, Keyword: "RCV-00"})
	require.Empty(t, partial.Report)
}

func TestListBusinessLineFilters(t *testing.T) {
	repo := newSampleRepo()
	cases := map[inboxlaporanklaim.BusinessLine][]string{
		inboxlaporanklaim.BusinessLinePA:           {"RCV-0012", "RCV-0001", "RCV-0007"},
		inboxlaporanklaim.BusinessLineTravel:       {"RCV-0008", "RCV-0002"},
		inboxlaporanklaim.BusinessLineSpecialGroup: {"RCV-0006", "RCV-0009"},
		// Non-MBU: Group Panel 003/004/006/009, kecuali kelompok bisnis khusus.
		inboxlaporanklaim.BusinessLineNonMBU: {"RCV-0005", "RCV-0003", "RCV-0004", "RCV-0010"},
	}
	for line, want := range cases {
		page := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, BusinessLine: line})
		require.Equalf(t, want, ids(page), "lini %q", line)
	}
}

func TestSummarizeCountsEveryCounter(t *testing.T) {
	repo := newSampleRepo()

	summary, err := repo.Summarize(context.Background(), inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryRejected, Operator: "adminpnc",
	})
	require.NoError(t, err)
	// Kategori diabaikan; pencacah komunikasi memakai pasangan milik pencacah lama.
	require.Equal(t, inboxlaporanklaim.Summary{
		Total: 11, NotTransferred: 4, Unregistered: 3, Outstanding: 4, Accepted: 2,
		MessageUnanswered: 2, MessageWaiting: 2, MessageReplied: 0,
	}, summary)

	scoped, err := repo.Summarize(context.Background(), inboxlaporanklaim.Filter{BranchCode: "1002"})
	require.NoError(t, err)
	require.Equal(t, inboxlaporanklaim.Summary{
		Total: 3, NotTransferred: 1, Unregistered: 1, Outstanding: 1, Accepted: 1,
	}, scoped)
}

func TestListRegionsReturnsCopy(t *testing.T) {
	repo := newSampleRepo()
	region, err := repo.ListRegions(context.Background())
	require.NoError(t, err)
	require.Equal(t, SampleRegions(), region)

	region[0].Name = "diubah"
	again, err := repo.ListRegions(context.Background())
	require.NoError(t, err)
	require.Equal(t, "Kanwil Jakarta", again[0].Name)
}

func TestGetFindsTrimmedID(t *testing.T) {
	repo := newSampleRepo()
	report, err := repo.Get(context.Background(), " RCV-0004 ")
	require.NoError(t, err)
	require.Equal(t, "POL-FIR-0004", report.PolicyNumber)

	_, err = repo.Get(context.Background(), "RCV-9999")
	require.ErrorIs(t, err, inboxlaporanklaim.ErrNotFound)
}

func TestInsertIssuesSequentialNumbersWithBranchName(t *testing.T) {
	repo := newSampleRepo()

	first, err := repo.Insert(context.Background(), inboxlaporanklaim.ClaimReport{BranchCode: "2001"})
	require.NoError(t, err)
	require.Equal(t, "RCVN.26.1", first.ID)
	require.Equal(t, inboxlaporanklaim.OriginNew, first.Origin)
	require.Equal(t, inboxlaporanklaim.PositionNotTransferred, first.Position)
	require.Equal(t, "Kanwil Jawa Barat", first.BranchName)

	second, err := repo.Insert(context.Background(), inboxlaporanklaim.ClaimReport{BranchCode: "9999"})
	require.NoError(t, err)
	require.Equal(t, "RCVN.26.2", second.ID)
	require.Empty(t, second.BranchName, "cabang tanpa kanwil tidak diberi nama karangan")

	got, err := repo.Get(context.Background(), "RCVN.26.2")
	require.NoError(t, err)
	require.Equal(t, second, got)
}

func TestUpdateAppliesOnlyFormFields(t *testing.T) {
	repo := newSampleRepo()
	created, err := repo.Insert(context.Background(), inboxlaporanklaim.ClaimReport{
		BranchCode: "1001", CreatedBy: "adminpnc",
	})
	require.NoError(t, err)

	updatedAt := sampleNow.Add(time.Hour)
	err = repo.Update(context.Background(), inboxlaporanklaim.ClaimReport{
		ID: created.ID, PolicyNumber: "POL-X", Chronology: "kronologi",
		BranchCode: "9999", CreatedBy: "penyusup",
		UpdatedBy: "adminpnc", UpdatedAt: updatedAt,
	})
	require.NoError(t, err)

	got, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "POL-X", got.PolicyNumber)
	require.Equal(t, "kronologi", got.Chronology)
	require.Equal(t, "1001", got.BranchCode, "kepala berkas tidak boleh ditimpa form")
	require.Equal(t, "adminpnc", got.CreatedBy)
	require.Equal(t, "adminpnc", got.UpdatedBy)
	require.Equal(t, updatedAt, got.UpdatedAt)
}

func TestUpdateRejectsLegacyAndUnknown(t *testing.T) {
	repo := newSampleRepo()
	require.ErrorIs(t, repo.Update(context.Background(), inboxlaporanklaim.ClaimReport{ID: "RCV-0001"}),
		inboxlaporanklaim.ErrReadOnlyOrigin)
	require.ErrorIs(t, repo.Update(context.Background(), inboxlaporanklaim.ClaimReport{ID: "RCVN.26.9"}),
		inboxlaporanklaim.ErrNotFound)
}

func TestFindPolicy(t *testing.T) {
	repo := newSampleRepo()
	policy, found, err := repo.FindPolicy(context.Background(), "12600000000002")
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, policy.Syariah)

	_, found, err = repo.FindPolicy(context.Background(), "000")
	require.NoError(t, err)
	require.False(t, found)
}

// Setelah SetError, setiap metode menjawab galat yang sama.
func TestSetErrorFailsEveryMethod(t *testing.T) {
	repo := newSampleRepo()
	boom := errors.New("boom")
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx, inboxlaporanklaim.Filter{}, inboxlaporanklaim.Pagination{})
	require.ErrorIs(t, err, boom)
	_, err = repo.Summarize(ctx, inboxlaporanklaim.Filter{})
	require.ErrorIs(t, err, boom)
	_, err = repo.ListRegions(ctx)
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "RCV-0001")
	require.ErrorIs(t, err, boom)
	_, err = repo.Insert(ctx, inboxlaporanklaim.ClaimReport{})
	require.ErrorIs(t, err, boom)
	require.ErrorIs(t, repo.Update(ctx, inboxlaporanklaim.ClaimReport{}), boom)
	_, _, err = repo.FindPolicy(ctx, "12600000000001")
	require.ErrorIs(t, err, boom)
}

// NewRepo menyalin bahannya; mengubah bahan asli tidak mengubah isi repo.
func TestNewRepoCopiesOptions(t *testing.T) {
	options := SampleOptions(fixedClock{now: sampleNow})
	repo := NewRepo(options)

	options.Rows[0].PolicyNumber = "diubah"
	options.Branch["1001"] = "99"
	options.Message["RCV-0002"][0].Text = "diubah"
	options.Policy["12600000000001"] = inboxlaporanklaim.Policy{}

	got, err := repo.Get(context.Background(), "RCV-0001")
	require.NoError(t, err)
	require.Equal(t, "POL-PA-0001", got.PolicyNumber)

	region := list(t, repo, inboxlaporanklaim.Filter{Category: inboxlaporanklaim.CategoryAll, RegionCode: "01"})
	require.Contains(t, ids(region), "RCV-0001")

	waiting := list(t, repo, inboxlaporanklaim.Filter{
		Category: inboxlaporanklaim.CategoryMessageWaiting, Operator: "adminpnc",
	})
	require.Equal(t, "Mohon dibantu cek kelengkapan berkas.", waiting.Report[1].LastMessage)

	policy, found, err := repo.FindPolicy(context.Background(), "12600000000001")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "PT CONTOH SEJAHTERA", policy.InsuredName)
}

func TestBranchResolver(t *testing.T) {
	resolver := NewBranchResolver(map[string]string{" AdminPNC ": "1001", "kosong": " "})
	ctx := context.Background()

	code, resolved, err := resolver.Resolve(ctx, "adminpnc")
	require.NoError(t, err)
	require.True(t, resolved)
	require.Equal(t, "1001", code)

	_, resolved, err = resolver.Resolve(ctx, "kosong")
	require.NoError(t, err)
	require.False(t, resolved, "kode kosong bukan cabang yang sah")

	_, resolved, err = resolver.Resolve(ctx, "tidakada")
	require.NoError(t, err)
	require.False(t, resolved)

	boom := errors.New("boom")
	resolver.SetError(boom)
	_, _, err = resolver.Resolve(ctx, "adminpnc")
	require.ErrorIs(t, err, boom)
}

func TestSampleBranchOfLoginIsResolvable(t *testing.T) {
	resolver := NewBranchResolver(SampleBranchOfLogin())
	code, resolved, err := resolver.Resolve(context.Background(), "pictekniks")
	require.NoError(t, err)
	require.True(t, resolved)
	require.Equal(t, "1002", code)
}
