package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/repo/memory"
)

// Uji urutan dan penyaring yang tidak terwakili data contoh.

const komiteUji = "KOMITEUJI"

func pada(day int) *time.Time {
	at := time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC)
	return &at
}

func kueriUji(t *testing.T, tab, cari string) inboxbandinghargasalvage.Query {
	t.Helper()
	q, err := inboxbandinghargasalvage.NewQuery(
		inboxbandinghargasalvage.QueryInput{Tab: tab, Keyword: cari},
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)
	return q
}

// Umur yang sama diurutkan menurut id detail salvage.
func TestRequestRowsWithTheSameAgeAreOrderedByDetail(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{ClaimNo: "K1", DetailObject: "B", CommitteeName: komiteUji,
			RequestPrice: "1", RequestDate: pada(1), InDetailSalvage: true},
		memory.CheckerRecord{ClaimNo: "K2", DetailObject: "A", CommitteeName: komiteUji,
			RequestPrice: "1", RequestDate: pada(1), InDetailSalvage: true},
	).WithNow(time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC))

	page, err := store.List(context.Background(),
		kueriUji(t, inboxbandinghargasalvage.TabRequest, ""), inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"},
		[]string{page.Items[0].DetailObject, page.Items[1].DetailObject})
	require.Equal(t, 10, page.Items[0].AgingDays)
}

// Riwayat menyaring kata kunci, dan satu klaim berpengajuan ganda diurutkan menurut id.
func TestHistoryRowsFilterTheKeywordAndOrderBySalvage(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{ClaimNo: "K1", CommitteeName: komiteUji, ApprovalStatus: "1",
			ApprovedAt: pada(2)},
		memory.CheckerRecord{ClaimNo: "K2", CommitteeName: komiteUji, ApprovalStatus: "0",
			ApprovedAt: pada(3)},
	).WithSalvages(
		memory.SalvageRecord{ClaimNo: "K1", SalvageID: "20"},
		memory.SalvageRecord{ClaimNo: "K1", SalvageID: "10"},
		memory.SalvageRecord{ClaimNo: "K2", SalvageID: "30"},
	)

	page, err := store.List(context.Background(),
		kueriUji(t, inboxbandinghargasalvage.TabHistory, "k1"), inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, "K1", page.Items[0].ClaimNo)

	all, err := store.Count(context.Background(),
		kueriUji(t, inboxbandinghargasalvage.TabHistory, ""))
	require.NoError(t, err)
	require.Equal(t, 3, all)
}

// Panel rincian: tanggal terbaru di atas, tanggal kosong di bawah, seri menurut detail.
func TestDecisionsOrderNewestFirstAndEmptyDatesLast(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "N2", CommitteeName: komiteUji,
			ApprovalStatus: "1"},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "N1", CommitteeName: komiteUji,
			ApprovalStatus: "1"},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "D2", CommitteeName: komiteUji,
			ApprovalStatus: "1", ApprovedAt: pada(5)},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "D1", CommitteeName: komiteUji,
			ApprovalStatus: "0", ApprovedAt: pada(5)},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "D0", CommitteeName: komiteUji,
			ApprovalStatus: "1", ApprovedAt: pada(9)},
		// Klaim lain dan keputusan yang belum diambil tidak masuk panel.
		memory.CheckerRecord{ClaimNo: "LAIN", DetailObject: "X", CommitteeName: komiteUji,
			ApprovalStatus: "1"},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "Y", CommitteeName: komiteUji},
	)

	q, err := inboxbandinghargasalvage.NewDecisionQuery("k",
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)

	items, err := store.ListDecisions(context.Background(), q)
	require.NoError(t, err)

	details := make([]string, 0, len(items))
	for _, item := range items {
		details = append(details, item.DetailObject)
	}
	require.Equal(t, []string{"D0", "D1", "D2", "N1", "N2"}, details)
}

// Tanpa jam yang digeser, penulis memakai jam proses sebagai tanggal putusan, dan harga
// hanya diterapkan pada pengajuan salvage yang sama.
func TestWriterWithTheDefaultClockAppliesPriceOnlyToTheSameSalvage(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "D", SalvageID: "1",
			CommitteeName: komiteUji, ItemPrice: "100", RequestPrice: "150",
			InDetailSalvage: true},
		memory.CheckerRecord{ClaimNo: "K", DetailObject: "D", SalvageID: "2",
			CommitteeName: "LAIN", ItemPrice: "100", RequestPrice: "150",
			InDetailSalvage: true},
	)
	writer := memory.NewWriter(store)

	result, err := writer.Decide(context.Background(), inboxbandinghargasalvage.DecisionCommand{
		DetailObject: "D", SalvageID: "1", RequestPrice: "150",
		Status:     inboxbandinghargasalvage.DecisionApproved,
		Reviewer:   inboxbandinghargasalvage.Reviewer{Name: komiteUji},
		ApplyPrice: true,
	})
	require.NoError(t, err)
	require.True(t, result.PriceApplied)

	q, err := inboxbandinghargasalvage.NewDecisionQuery("K",
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)
	items, err := store.ListDecisions(context.Background(), q)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NotNil(t, items[0].ApprovedAt)
	require.Equal(t, "150", items[0].ItemPrice)

	// Baris pengajuan salvage lain tetap berharga lama.
	other, err := store.ListDecisions(context.Background(), inboxbandinghargasalvage.DecisionQuery{
		ClaimNo: "K", Reviewer: inboxbandinghargasalvage.Reviewer{Name: "LAIN"}})
	require.NoError(t, err)
	require.Empty(t, other, "baris komite lain belum diputus")
}

// Penulis berjam tetap mencatat tanggal putusan tepat pada jam itu.
func TestWriterWithAFixedClockRecordsThatDate(t *testing.T) {
	store := memory.NewStore(memory.CheckerRecord{ClaimNo: "K", DetailObject: "D",
		CommitteeName: komiteUji, RequestPrice: "1", InDetailSalvage: true})
	at := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)

	_, err := memory.NewWriter(store).WithNow(at).Decide(context.Background(),
		inboxbandinghargasalvage.DecisionCommand{
			DetailObject: "D", Status: inboxbandinghargasalvage.DecisionRejected,
			Reviewer: inboxbandinghargasalvage.Reviewer{Name: komiteUji},
		})
	require.NoError(t, err)

	q, err := inboxbandinghargasalvage.NewDecisionQuery("K",
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)
	items, err := store.ListDecisions(context.Background(), q)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, at, *items[0].ApprovedAt)
	require.Equal(t, inboxbandinghargasalvage.DecisionRejected, items[0].Status)
}

// Dokumen: terbaru di atas, tanpa tanggal di bawah, seri menurut id menurun; kepemilikan
// menuntut pengajuan salvage yang sama.
func TestDocumentOrderingAndOwnership(t *testing.T) {
	store := memory.NewStore(
		memory.CheckerRecord{DetailObject: "D", SalvageID: "9", CommitteeName: "LAIN"},
		memory.CheckerRecord{DetailObject: "D", SalvageID: "1", CommitteeName: komiteUji},
	)
	documents := memory.NewDocumentStore(store,
		memory.DocumentRecord{ID: "1", DetailObject: "D", SalvageID: "1"},
		memory.DocumentRecord{ID: "2", DetailObject: "D", SalvageID: "1"},
		memory.DocumentRecord{ID: "3", DetailObject: "D", SalvageID: "1", UploadedAt: pada(4)},
		memory.DocumentRecord{ID: "4", DetailObject: "D", SalvageID: "1", UploadedAt: pada(4)},
		memory.DocumentRecord{ID: "5", DetailObject: "D", SalvageID: "1", UploadedAt: pada(8)},
	)

	q, err := inboxbandinghargasalvage.NewDocumentQuery("D", "1",
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)

	rows, err := documents.ListDocuments(context.Background(), q)
	require.NoError(t, err)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	require.Equal(t, []string{"5", "4", "3", "2", "1"}, ids)

	// Pengajuan salvage yang hanya dimiliki komite lain tidak terbaca.
	foreign, err := inboxbandinghargasalvage.NewDocumentQuery("D", "9",
		inboxbandinghargasalvage.Caller{Login: komiteUji})
	require.NoError(t, err)
	none, err := documents.ListDocuments(context.Background(), foreign)
	require.NoError(t, err)
	require.Empty(t, none)
}
