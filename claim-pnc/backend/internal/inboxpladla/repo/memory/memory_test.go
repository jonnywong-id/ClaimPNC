package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
)

// Uji di berkas ini menegakkan aturan yang SAMA dengan yang ditegakkan penyimpanan SQL.
//
// Satu aturan bahkan HANYA dapat diuji di sini: perbedaan antara daftar yang mencocokkan
// seluruh kode reasuradur dan daftar yang hanya mencocokkan kode tertinggi hidup di dalam
// teks SQL, sebagai `IN` versus `=`.

func codesOf(t *testing.T, store *memory.Store, login string) []string {
	t.Helper()
	codes, err := store.ReinsurerCodes(context.Background(), login)
	require.NoError(t, err)
	return codes
}

// list menjalankan satu daftar dan mengembalikan nomor klaimnya.
func list(t *testing.T, store *memory.Store, login, tab, search string) []string {
	t.Helper()

	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: tab, Search: search},
		inboxpladla.Caller{Login: login},
		codesOf(t, store, login),
	)
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxpladla.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	numbers := []string{}
	for _, row := range page.Items {
		numbers = append(numbers, row.ClaimNo)
	}
	return numbers
}

// rowOf mencari satu baris menurut nomor klaimnya.
func rowOf(t *testing.T, store *memory.Store, login, tab, claimNo string) inboxpladla.Row {
	t.Helper()

	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: tab},
		inboxpladla.Caller{Login: login},
		codesOf(t, store, login),
	)
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxpladla.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)

	for _, row := range page.Items {
		if row.ClaimNo == claimNo {
			return row
		}
	}

	t.Fatalf("baris %s tidak ada di daftar %s", claimNo, tab)
	return inboxpladla.Row{}
}

// Login yang bukan reasuradur tidak punya satu pun kode.
//
// Itulah yang membedakan "bukan untuk Anda" dari "belum ada pekerjaan", dan usecase yang
// menerjemahkannya menjadi pesan.
func TestALoginThatIsNotAReinsurerHasNoCodes(t *testing.T) {
	store := memory.NewSampleStore()

	require.Empty(t, codesOf(t, store, "JONNY"))
	require.NotEmpty(t, codesOf(t, store, memory.SampleReinsurerLogin))
}

// Kode reasuradur diserahkan terurut MENURUN.
//
// Yang PERTAMA di senarai inilah yang dipakai kedua daftar yang hanya memakai satu kode —
// sama dengan `ORDER BY REINSURERID DESC FETCH NEXT 1 ROW ONLY` di SQL.
func TestReinsurerCodesAreReturnedHighestFirst(t *testing.T) {
	store := memory.NewSampleStore()

	require.Equal(t, []string{"R901", "R900"},
		codesOf(t, store, memory.SampleSecondLogin))
}

// Daftar PLA menampilkan klaim yang PLA-nya terkirim dan DLA-nya belum.
func TestThePLAListShowsClaimsWhoseDLAHasNotBeenSentYet(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, memory.SampleReinsurerLogin, "pla", "")

	require.Contains(t, numbers, "PNC-2001")
	require.NotContains(t, numbers, "PNC-2002",
		"klaim yang DLA-nya sudah terkirim berpindah ke daftar DLA")
}

// Daftar DLA menampilkan klaim yang DLA-nya terkirim.
func TestTheDLAListShowsClaimsWhoseDLAHasBeenSent(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, memory.SampleReinsurerLogin, "dla", "")

	require.Contains(t, numbers, "PNC-2002")
	require.NotContains(t, numbers, "PNC-2001",
		"klaim yang baru dikirimi PLA belum masuk daftar DLA")
}

// Daftar Close menampilkan klaim yang sudah selesai dan tidak menunggu penutupan.
func TestTheCloseListShowsCompletedClaimsOnly(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, memory.SampleReinsurerLogin, "close", "")

	require.Contains(t, numbers, "PNC-2003")
	require.NotContains(t, numbers, "PNC-2005",
		"klaim yang masih menunggu penutupan belum masuk daftar Close")
}

// Dokumen yang ditandai terkirim TANPA tanggal kirim tidak dihitung terkirim.
//
// Kueri lama menuntut ketiga syaratnya sekaligus — `iskirim='1'`, tanggal kirim terisi,
// alamat surel terisi. Memeriksa yang pertama saja akan memasukkan dokumen yang tidak
// pernah benar-benar dikirim.
func TestAnAdviceMarkedSentWithoutADateDoesNotCount(t *testing.T) {
	store := memory.NewSampleStore()

	for _, tab := range []string{"pla", "dla", "close"} {
		require.NotContains(t,
			list(t, store, memory.SampleReinsurerLogin, tab, ""), "PNC-2004",
			"%s: dokumen tanpa tanggal kirim seharusnya tidak dihitung", tab)
	}
}

// Klaim yang menunggu penutupan muncul di daftar DLA dengan kode status DIGANTI `1139`.
func TestAClaimPendingCloseAppearsOnTheDLAListWithStatus1139(t *testing.T) {
	store := memory.NewSampleStore()

	numbers := list(t, store, memory.SampleReinsurerLogin, "dla", "")
	require.Contains(t, numbers, "PNC-2005")

	row := rowOf(t, store, memory.SampleReinsurerLogin, "dla", "PNC-2005")
	require.Equal(t, inboxpladla.PendingCloseStatusCode, row.StatusCode,
		"kode status seharusnya diganti 1139")
	require.Equal(t, "Pending Close Claim", row.StatusLabel,
		"label harus mengikuti kode PENGGANTI, bukan kode aslinya")
}

// Klaim yang DITOLAK tidak muncul di daftar mana pun.
//
// Akibatnya reasuradur kehilangan jejak klaim yang pernah diberitahukan kepadanya, tanpa
// satu pun pemberitahuan. Itu perilaku Pega (`P-5`), dan uji ini menjaganya tertiru
// sekaligus membuatnya terlihat oleh pembaca berikutnya.
func TestARejectedClaimDisappearsFromEveryList(t *testing.T) {
	store := memory.NewSampleStore()

	for _, tab := range []string{"pla", "dla", "close"} {
		require.NotContains(t,
			list(t, store, memory.SampleReinsurerLogin, tab, ""), "PNC-2006",
			"%s: klaim ditolak seharusnya tidak muncul", tab)
	}
}

// Klaim TANPA baris tabel kerja Pega muncul di daftar PLA, tetapi tidak di daftar DLA.
//
// Perbedaannya datang dari bentuk kuerinya: sub-kueri (setara LEFT JOIN) pada daftar PLA,
// gabungan INNER pada daftar DLA. Menyeragamkan keduanya akan menghilangkan baris.
func TestAClaimWithoutAWorkRowAppearsOnPLAButNotOnDLA(t *testing.T) {
	store := memory.NewSampleStore()

	require.Contains(t, list(t, store, memory.SampleReinsurerLogin, "pla", ""),
		"PNC-2007", "gabungan LEFT pada daftar PLA harus mempertahankannya")

	require.NotContains(t, list(t, store, memory.SampleReinsurerLogin, "dla", ""),
		"PNC-2007", "gabungan INNER pada daftar DLA harus menyaringnya keluar")
}

// Daftar Close mencocokkan SELURUH kode reasuradur; daftar PLA hanya yang tertinggi.
//
// PNC-2008 dikirimi PLA ke kode `R900`, sementara kode tertinggi login itu `R901`. Ia
// karena itu muncul di daftar Close dan tidak di daftar PLA.
//
// Ini satu-satunya uji yang membuktikan perbedaan `IN` versus `=`, dan ia tidak dapat
// dibuktikan di tempat lain tanpa Oracle.
func TestOnlyTheCloseListMatchesEveryReinsurerCodeOfTheLogin(t *testing.T) {
	store := memory.NewSampleStore()

	require.Contains(t, list(t, store, memory.SampleSecondLogin, "close", ""),
		"PNC-2008", "daftar Close mencocokkan SELURUH kode")

	require.NotContains(t, list(t, store, memory.SampleSecondLogin, "pla", ""),
		"PNC-2008", "daftar PLA hanya mencocokkan kode tertinggi")
}

// Kolom "No PLA" mengambil nomor berrevisi TERTINGGI.
func TestTheAdviceNumberComesFromTheHighestRevision(t *testing.T) {
	store := memory.NewSampleStore()

	row := rowOf(t, store, memory.SampleReinsurerLogin, "pla", "PNC-2001")
	require.Equal(t, "PLA/2026/2001-R1", row.AdviceNo)
}

// Satu reasuradur tidak melihat klaim milik reasuradur lain.
func TestAReinsurerNeverSeesAnotherReinsurersClaims(t *testing.T) {
	store := memory.NewSampleStore()

	first := list(t, store, memory.SampleReinsurerLogin, "close", "")
	require.NotContains(t, first, "PNC-2008",
		"klaim milik login lain bocor ke daftar ini")

	second := list(t, store, memory.SampleSecondLogin, "close", "")
	require.NotContains(t, second, "PNC-2003",
		"klaim milik login lain bocor ke daftar ini")
}

// Pencarian mencocokkan kunci klaim, dan tidak peka huruf besar-kecil.
func TestSearchFindsTheClaimByItsWorkKey(t *testing.T) {
	store := memory.NewSampleStore()

	require.Equal(t, []string{"PNC-2001"},
		list(t, store, memory.SampleReinsurerLogin, "pla", "pnc-2001"))

	require.Empty(t,
		list(t, store, memory.SampleReinsurerLogin, "pla", "PNC-9999"))
}

// Tabel ringkas dihitung dari populasi yang SAMA dengan daftarnya.
//
// Angka yang tidak cocok dengan tabel di bawahnya adalah hal pertama yang dilaporkan
// pengguna sebagai kerusakan.
func TestTheSummaryCountsTheSamePopulationAsTheList(t *testing.T) {
	store := memory.NewSampleStore()

	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: "pla"},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin},
		codesOf(t, store, memory.SampleReinsurerLogin),
	)
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxpladla.Pagination{Page: 1, Size: 1})
	require.NoError(t, err)

	counts, err := store.Counts(context.Background(), query)
	require.NoError(t, err)

	summed := 0
	for _, count := range counts {
		summed += count.Total
	}

	require.Equal(t, page.Total, summed,
		"jumlah tabel ringkas harus sama dengan total daftar, bukan dengan "+
			"jumlah baris yang sedang tampil")
}

// Tabel ringkas mengikuti pencarian yang sedang aktif.
func TestTheSummaryFollowsTheActiveSearch(t *testing.T) {
	store := memory.NewSampleStore()

	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: "pla", Search: "PNC-2001"},
		inboxpladla.Caller{Login: memory.SampleReinsurerLogin},
		codesOf(t, store, memory.SampleReinsurerLogin),
	)
	require.NoError(t, err)

	counts, err := store.Counts(context.Background(), query)
	require.NoError(t, err)

	summed := 0
	for _, count := range counts {
		summed += count.Total
	}
	require.Equal(t, 1, summed)
}
