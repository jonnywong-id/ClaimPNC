package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
)

// Konteks yang sudah dibatalkan ditolak oleh setiap operasi, sebelum menyentuh data.
func TestEveryOperationHonoursACancelledContext(t *testing.T) {
	store := memory.NewSampleStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.List(ctx, inboxsalvage.Query{}, inboxsalvage.Pagination{})
	require.ErrorIs(t, err, context.Canceled)

	_, err = store.Counts(ctx, callerPIC())
	require.ErrorIs(t, err, context.Canceled)

	_, err = store.Detail(ctx, "1")
	require.ErrorIs(t, err, context.Canceled)

	_, err = store.DetailByClaim(ctx, "PNC-2041")
	require.ErrorIs(t, err, context.Canceled)

	_, err = store.Create(ctx, inboxsalvage.Form{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestBlankReferencesAreNotFound(t *testing.T) {
	store := memory.NewSampleStore()

	_, err := store.Detail(context.Background(), "  ")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)

	_, err = store.DetailByClaim(context.Background(), "  ")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
}

// Daftar tanpa penyaring status apa pun menampilkan seluruh klaim yang ada.
func TestATabWithoutAnyClaimFilterListsEveryClaim(t *testing.T) {
	store := memory.NewStore()
	store.Seed([]memory.Claim{{ClaimNo: "B"}, {ClaimNo: "A"}}, nil)

	q, err := inboxsalvage.NewQueryForTab(
		inboxsalvage.Tab{Code: "semua", Family: inboxsalvage.FamilyClaimObject},
		inboxsalvage.QueryInput{}, callerPIC())
	require.NoError(t, err)

	page, err := store.List(context.Background(), q, inboxsalvage.Pagination{})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Equal(t, "A", page.Items[0].ClaimNo, "diurutkan menurut nomor klaim")
}

// ID salvage yang bukan angka dibandingkan sebagai teks, dan pengajuan atas klaim yang
// tidak ada tetap dapat dibuka dengan nama bisnis kosong.
func TestNonNumericIDsAndOrphanSubmissions(t *testing.T) {
	store := memory.NewStore()
	store.SetNow(func() string { return "2026-09-25" })
	store.Seed(
		[]memory.Claim{{ClaimNo: "PNC-1", BusinessName: "Fire"}},
		[]memory.Salvage{
			{SalvageID: "A1", ClaimNo: "PNC-1", InputDate: "2026-09-01", TransferStatus: "1"},
			{SalvageID: "B2", ClaimNo: "PNC-1", InputDate: "2026-09-01", TransferStatus: "6"},
			{SalvageID: "Z9", ClaimNo: "YATIM", InputDate: "2026-09-02"},
		},
	)

	detail, err := store.DetailByClaim(context.Background(), "PNC-1")
	require.NoError(t, err)
	require.Equal(t, "B2", detail.SalvageID, "teks \"B2\" lebih besar dari \"A1\"")
	require.Equal(t, []string{"B2", "A1"},
		[]string{detail.History[0].SalvageID, detail.History[1].SalvageID})

	orphan, err := store.Detail(context.Background(), "Z9")
	require.NoError(t, err)
	require.Empty(t, orphan.BusinessName)

	// Tanpa data contoh, ID berikutnya dimulai dari 1 karena tidak ada ID berupa angka.
	id, err := store.Create(context.Background(), inboxsalvage.Form{ClaimNo: "PNC-1"})
	require.NoError(t, err)
	require.Equal(t, "1", id)
}

// Penyimpanan baru memakai tanggal hari ini menurut WIB bila uji tidak menggantinya.
func TestNewStoreAgesAgainstTheRealToday(t *testing.T) {
	store := memory.NewStore()
	store.Seed(nil, []memory.Salvage{{SalvageID: "1", ClaimNo: "PNC-1", InputDate: "2000-01-01"}})

	tab, _ := inboxsalvage.FindTab(inboxsalvage.TabHistori)
	q, err := inboxsalvage.NewQueryForTab(tab, inboxsalvage.QueryInput{}, callerPIC())
	require.NoError(t, err)

	page, err := store.List(context.Background(), q, inboxsalvage.Pagination{})
	require.NoError(t, err)
	// Angka pastinya bergantung hari ini; yang dipastikan adalah ia terhitung terhadap
	// tanggal sungguhan (lebih dari 9.000 hari sejak tahun 2000), bukan kosong.
	aging := page.Items[0].Aging
	require.Regexp(t, `^\d{4,} day$`, aging)
}
