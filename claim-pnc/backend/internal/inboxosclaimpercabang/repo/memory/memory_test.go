package memory_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/repo/memory"
	"claim-pnc/internal/platform/money"
)

// cilegon adalah permintaan untuk cabang contoh yang paling banyak isinya.
func cilegon() inboxosclaimpercabang.Query {
	return inboxosclaimpercabang.Query{BranchCode: "100099", BranchName: "CILEGON"}
}

func allPages(t *testing.T, store *memory.Store) []inboxosclaimpercabang.WorkItem {
	t.Helper()

	page, err := store.List(t.Context(), cilegon(),
		inboxosclaimpercabang.Pagination{Page: 1, Size: inboxosclaimpercabang.MaxPageSize})
	require.NoError(t, err)
	return page.Items
}

func TestSampleStoreKeepsOnlyOutstandingClaimsOfTheBranch(t *testing.T) {
	store := memory.NewSampleStore()
	items := allPages(t, store)

	numbers := []string{}
	for _, item := range items {
		numbers = append(numbers, item.ClaimNumber)
	}

	// PNC-9004 cabang lain · PNC-9005 sudah selesai · PNC-9006 tanpa tanggal registrasi.
	require.Equal(t, []string{"PNC-9001", "PNC-9002", "PNC-9003"}, numbers)
}

func TestBranchIsAHardBoundaryNotAFilterHint(t *testing.T) {
	// Kebocoran di sini tidak menghasilkan satu pun galat: layar terisi, angkanya masuk
	// akal, dan yang salah hanya MILIK SIAPA datanya (`R-20`).
	store := memory.NewSampleStore()

	other, err := store.List(t.Context(),
		inboxosclaimpercabang.Query{BranchCode: "100059"},
		inboxosclaimpercabang.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	for _, item := range other.Items {
		require.Equal(t, "100059", item.BranchCode,
			"baris cabang lain ikut terbawa")
	}
	require.Equal(t, 1, other.Total)
}

func TestUnknownBranchReturnsNothingWithoutError(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(t.Context(),
		inboxosclaimpercabang.Query{BranchCode: "999999"},
		inboxosclaimpercabang.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Zero(t, page.Total)
}

func TestRowsAreOrderedByRegistrationDateAscending(t *testing.T) {
	// Urutan ini SETARA dengan `ORDER BY "AgingKlaim" DESC` milik kueri lama: umur menurun
	// sama artinya dengan tanggal registrasi menaik. Itulah yang membuat umur dapat dihitung
	// di Go tanpa memindahkan pengurutannya.
	items := allPages(t, memory.NewSampleStore())
	require.Len(t, items, 3)

	for i := 1; i < len(items); i++ {
		previous, current := items[i-1].RegisterDate, items[i].RegisterDate
		require.Falsef(t, current.Before(*previous),
			"baris %d terdaftar lebih dulu daripada baris %d", i, i-1)
	}
}

func TestPaginationDoesNotRepeatOrSkipRows(t *testing.T) {
	store := memory.NewSampleStore()

	seen := map[string]bool{}
	total := 0

	for page := 1; page <= 3; page++ {
		result, err := store.List(t.Context(), cilegon(),
			inboxosclaimpercabang.Pagination{Page: page, Size: 1})
		require.NoError(t, err)
		require.Equal(t, 3, result.Total, "jumlah seluruhnya berubah antar halaman")

		for _, item := range result.Items {
			require.Falsef(t, seen[item.ClaimNumber],
				"%s muncul di lebih dari satu halaman", item.ClaimNumber)
			seen[item.ClaimNumber] = true
			total++
		}
	}

	require.Equal(t, 3, total, "ada baris yang terlewat saat dipaginasi")
}

func TestBranchNameIsKnownOnlyForRegisteredBranches(t *testing.T) {
	store := memory.NewSampleStore()

	name, known, err := store.BranchName(t.Context(), "100099")
	require.NoError(t, err)
	require.True(t, known)
	require.Equal(t, "CILEGON", name)

	_, known, err = store.BranchName(t.Context(), "999999")
	require.NoError(t, err, "kode yang tidak dikenal bukan galat")
	require.False(t, known)
}

func TestStalledMarkerIsIndependentFromAging(t *testing.T) {
	// Kedua syarat merah harus dapat muncul SENDIRI-SENDIRI. Bila keduanya selalu bersama,
	// uji tidak akan pernah membuktikan syarat kedua benar-benar dibaca.
	items := allPages(t, memory.NewSampleStore())

	stalled := map[string]bool{}
	for _, item := range items {
		stalled[item.ClaimNumber] = item.ProgressStalled
	}

	require.True(t, stalled["PNC-9002"],
		"contoh progres mandek hilang; syarat merah kedua tidak teruji")
	require.False(t, stalled["PNC-9001"],
		"contoh klaim tua tanpa progres mandek hilang; kedua syarat tidak terpisah")
}

func TestClaimWithoutProgressLeavesDateEmptyInsteadOfZeroTime(t *testing.T) {
	// Kosong berbeda artinya dari tanggal mana pun. Menggambarnya sebagai waktu nol akan
	// menampilkan 1 Januari tahun 1 di kolom "Tgl Update Progress Terakhir".
	items := allPages(t, memory.NewSampleStore())

	var found bool
	for _, item := range items {
		if item.ClaimNumber != "PNC-9003" {
			continue
		}
		found = true
		require.Nil(t, item.LastProgressAt)
		require.Empty(t, item.ProgressStatus1)
		require.Empty(t, item.ProgressNote)
	}
	require.True(t, found, "contoh klaim tanpa catatan progres hilang")
}

func TestDominantFactorsAreScopedToTheBranchAndKeepTheirOrder(t *testing.T) {
	store := memory.NewSampleStore()

	factors, err := store.DominantFactors(t.Context(), cilegon())
	require.NoError(t, err)

	require.Equal(t,
		[]string{"Kelalaian Tertanggung", "Dokumen Tidak Lengkap"},
		factors["CLAIM-0002"],
		"urutan faktor berubah; ia harus mengikuti idx_dominanfactor apa adanya")

	other, err := store.DominantFactors(t.Context(),
		inboxosclaimpercabang.Query{BranchCode: "100059"})
	require.NoError(t, err)
	require.NotContains(t, other, "CLAIM-0002",
		"faktor cabang lain ikut terbawa")
}

func TestExportRowsCarryTheWiderColumns(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.ListForExport(t.Context(), cilegon(),
		inboxosclaimpercabang.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)

	first := page.Items[0]
	require.Equal(t, "PNC-9001", first.ClaimNumber)
	require.Equal(t, "CLAIM-0001", first.ClaimKey)
	require.Equal(t, money.FromRupiah(200_000), first.ReserveClaimASM)

	// Nilai grid dan nilai ASM memang berbeda pada contoh ini, dan itu disengaja: ia yang
	// membuktikan keduanya tidak diam-diam disamakan (lihat WorkItem.EstimationValue).
	require.NotEqual(t, first.EstimationValue, first.ReserveClaimASM,
		"contoh kehilangan selisih antara nilai grid dan porsi ASM")
}
