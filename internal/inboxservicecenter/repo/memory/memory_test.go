package memory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
)

// queryFor membentuk Query sah lewat NewQuery, supaya penyaring tab ikut terbentuk seperti
// di jalur sungguhan.
func queryFor(t *testing.T, tab, keyword, login string) inboxservicecenter.Query {
	t.Helper()
	q, err := inboxservicecenter.NewQuery(
		inboxservicecenter.QueryInput{Tab: tab, Keyword: keyword},
		inboxservicecenter.Caller{Login: login},
	)
	require.NoError(t, err)
	return q
}

func claimIDs(page inboxservicecenter.Page) []string {
	result := make([]string, 0, len(page.Items))
	for _, c := range page.Items {
		result = append(result, c.ID)
	}
	return result
}

func TestListFiltersByTabAndOwner(t *testing.T) {
	store := NewSampleStore()

	page, err := store.List(context.Background(),
		queryFor(t, inboxservicecenter.TabRegistration, "", SampleOwner),
		inboxservicecenter.Pagination{})
	require.NoError(t, err)

	// SC-000103 milik PIC lain; SC-000102 tanpa tanggal ditaruh paling bawah.
	require.Equal(t, []string{"SC-000101", "SC-000102"}, claimIDs(page))
	require.Equal(t, 2, page.Total)
	require.True(t, page.Paginated)
	require.Equal(t, inboxservicecenter.DefaultPageSize, page.Pagination.Size)

	// Login dicocokkan tanpa membedakan huruf besar-kecil.
	page, err = store.List(context.Background(),
		queryFor(t, inboxservicecenter.TabRejected, "", "picservicecenter"),
		inboxservicecenter.Pagination{})
	require.NoError(t, err)
	require.Equal(t, []string{"SC-000401", "SC-000402"}, claimIDs(page))
}

func TestListPaginationSlices(t *testing.T) {
	store := NewSampleStore()
	q := queryFor(t, inboxservicecenter.TabRegistration, "", SampleOwner)

	page, err := store.List(context.Background(), q, inboxservicecenter.Pagination{Page: 2, Size: 1})
	require.NoError(t, err)
	require.Equal(t, []string{"SC-000102"}, claimIDs(page))
	require.Equal(t, 2, page.Total)

	// Halaman di luar jangkauan: kosong, tetapi totalnya tetap dilaporkan.
	page, err = store.List(context.Background(), q, inboxservicecenter.Pagination{Page: 5, Size: 1})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.Equal(t, 2, page.Total)
}

// Pencarian adalah IRISAN dua kelompok: ID saja tidak cukup, nomor polis cocok di keduanya.
func TestListSearchIsIntersectionAndUnpaginated(t *testing.T) {
	store := NewSampleStore()

	page, err := store.List(context.Background(),
		queryFor(t, inboxservicecenter.TabRegistration, "sc-000101", SampleOwner),
		inboxservicecenter.Pagination{Page: 3, Size: 1})
	require.NoError(t, err)
	require.Empty(t, page.Items, "ID hanya ada di kelompok pertama")
	require.False(t, page.Paginated)

	page, err = store.List(context.Background(),
		queryFor(t, inboxservicecenter.TabRegistration, "imei-contoh", SampleOwner),
		inboxservicecenter.Pagination{Page: 3, Size: 1})
	require.NoError(t, err)
	// Paginasi mati: seluruh baris yang cocok terbawa meski halaman 3 ukuran 1 diminta.
	require.Equal(t, []string{"SC-000101", "SC-000102"}, claimIDs(page))
	require.Equal(t, 2, page.Total)
}

func TestMatchesApprovalCodes(t *testing.T) {
	claim := inboxservicecenter.ServiceClaim{ApprovalStatus: " 2 "}
	require.True(t, matchesApproval(claim, inboxservicecenter.ApprovalFilter{Codes: []string{"2", "3"}}))
	require.False(t, matchesApproval(claim, inboxservicecenter.ApprovalFilter{Codes: []string{"1"}}))
	require.False(t, matchesApproval(claim, inboxservicecenter.ApprovalFilter{MatchNull: true}))
	require.True(t, matchesApproval(inboxservicecenter.ServiceClaim{},
		inboxservicecenter.ApprovalFilter{MatchNull: true}))
}

func TestSortClaimsOrder(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	claims := []inboxservicecenter.ServiceClaim{
		{ID: "D"},
		{ID: "B", InputDate: &early},
		{ID: "C"},
		{ID: "A", InputDate: &early},
		{ID: "E", InputDate: &late},
	}
	sortClaims(claims)

	got := make([]string, 0, len(claims))
	for _, c := range claims {
		got = append(got, c.ID)
	}
	// Tanggal menurun, sama tanggal urut ID, tanpa tanggal paling bawah urut ID.
	require.Equal(t, []string{"E", "A", "B", "C", "D"}, got)
}

func TestNewStoreHasNoDetails(t *testing.T) {
	store := NewStore(SampleClaims()...)

	_, err := store.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: "SC-000101", Caller: inboxservicecenter.Caller{Login: SampleOwner},
	})
	require.ErrorIs(t, err, inboxservicecenter.ErrNotFound)

	// WithDetails menyerahkan salinan; penyimpanan asal tetap tanpa rincian.
	withDetails := store.WithDetails(SampleDetails(), SampleProgress())
	found, err := withDetails.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: " sc-000101 ", Caller: inboxservicecenter.Caller{Login: "picservicecenter"},
	})
	require.NoError(t, err)
	require.Equal(t, "SC-000101", found.ID)
	require.Equal(t, "SIMAS INSURTECH", found.Insurance)

	_, err = store.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: "SC-000101", Caller: inboxservicecenter.Caller{Login: SampleOwner},
	})
	require.ErrorIs(t, err, inboxservicecenter.ErrNotFound)
}

func TestFindDetailRequiresOwner(t *testing.T) {
	store := NewSampleStore()
	_, err := store.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: "SC-000103", Caller: inboxservicecenter.Caller{Login: SampleOwner},
	})
	require.ErrorIs(t, err, inboxservicecenter.ErrNotFound)

	found, err := store.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: "SC-000103", Caller: inboxservicecenter.Caller{Login: sampleOther},
	})
	require.NoError(t, err)
	require.Equal(t, sampleOther, found.TechnicalPIC)
	// Hanya baris pertama yang diisi rapat.
	require.Empty(t, found.Insurance)
}

func TestListProgressByRepairID(t *testing.T) {
	store := NewSampleStore()

	notes, err := store.ListProgress(context.Background(), " 1000101 ")
	require.NoError(t, err)
	require.Len(t, notes, 2)
	require.Equal(t, SampleOwner, notes[0].RecordedBy)

	notes, err = store.ListProgress(context.Background(), "SC-000101")
	require.NoError(t, err)
	require.Empty(t, notes, "riwayat dikunci REPAIRID, bukan ID")
	require.NotNil(t, notes)

	notes, err = store.ListProgress(context.Background(), "  ")
	require.NoError(t, err)
	require.Empty(t, notes)
	require.NotNil(t, notes)
}

func TestSampleDetailsOnePerClaim(t *testing.T) {
	details := SampleDetails()
	require.Len(t, details, len(SampleClaims()))
	for i, claim := range SampleClaims() {
		require.Equal(t, claim.ID, details[i].ID)
		require.Equal(t, claim.RepairID, details[i].RepairID)
		require.Equal(t, claim.CustomerName, details[i].InsuredName)
	}
	require.Equal(t, "2214500", details[0].TotalFee)
	require.Equal(t, time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC), *details[0].WarrantyStart)
}
