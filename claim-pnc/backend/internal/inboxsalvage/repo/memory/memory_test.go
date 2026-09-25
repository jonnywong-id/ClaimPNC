package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
	"claim-pnc/internal/inboxsalvage/repo/memory"
)

func callerPIC() inboxsalvage.Caller {
	return inboxsalvage.Caller{Login: memory.SampleCallerPIC}
}

// list adalah pembantu yang menyusun permintaan dan menjalankannya sekaligus, supaya tiap
// uji di bawah hanya berisi hal yang benar-benar diujinya.
func list(t *testing.T, store *memory.Store, tab, search string) inboxsalvage.Page {
	t.Helper()

	query, err := inboxsalvage.NewQuery(
		inboxsalvage.QueryInput{Tab: tab, Search: search}, callerPIC())
	require.NoError(t, err)

	page, err := store.List(context.Background(), query,
		inboxsalvage.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)
	return page
}

// Setiap daftar wajib PUNYA ISI pada data contoh. Daftar yang kosong di sini berarti
// penyaringnya salah, bukan datanya habis — dan tanpa uji ini keduanya terlihat sama.
func TestEveryTabHasRowsInTheSampleData(t *testing.T) {
	store := memory.NewSampleStore()

	for _, tab := range inboxsalvage.Tabs() {
		page := list(t, store, tab.Code, "")
		require.NotEmpty(t, page.Items,
			"daftar %q kosong pada data contoh", tab.Name)
	}
}

// Daftar Salvage Outstanding menyaring `STSSALVAGE` yang KOSONG, dan MENGELUARKAN klaim
// yang sudah selesai. Hanya daftar ini yang menyaring status kerja.
func TestOutstandingListsOnlyClaimsWithNoSalvageMarkAndExcludesFinishedOnes(t *testing.T) {
	store := memory.NewSampleStore()
	page := list(t, store, inboxsalvage.TabOutstanding, "")

	numbers := claimNumbers(page)
	require.Contains(t, numbers, "PNC-2041")
	require.Contains(t, numbers, "PNC-2042")

	// PNC-2043 penandanya kosong PULA, tetapi status kerjanya sudah selesai.
	require.NotContains(t, numbers, "PNC-2043",
		"klaim yang sudah selesai dikeluarkan dari daftar ini")

	require.Len(t, page.Items, 2)
}

// Inilah selisih yang DIREPLIKASI (`P-5`), dan uji ini menjaganya tetap ada.
//
// Pencacah "Outstanding" menghitung `STSSALVAGE` 3 atau 5 — tiga baris pada data contoh —
// sementara daftarnya menampilkan yang penandanya KOSONG, dua baris. Bila seseorang
// "memperbaikinya", uji inilah yang gagal lebih dulu, bukan pengguna yang melaporkannya.
func TestOutstandingCounterAndItsListDisagreeOnPurpose(t *testing.T) {
	store := memory.NewSampleStore()

	counts, err := store.Counts(context.Background(), callerPIC())
	require.NoError(t, err)

	var counter int
	for _, row := range counts {
		if row.Label == "Outstanding" {
			counter = row.Total
		}
	}

	listed := list(t, store, inboxsalvage.TabOutstanding, "").Total

	require.Equal(t, 3, counter, "pencacah menghitung STSSALVAGE 3 atau 5")
	require.Equal(t, 2, listed, "daftarnya menampilkan STSSALVAGE yang kosong")
	require.NotEqual(t, counter, listed,
		"selisih ini ADA di Pega dan sengaja dipertahankan")
}

// Daftar "Request Balai Lelang" menyaring menurut PIC. Data contoh memuat baris milik DUA
// PIC yang berbeda, sehingga penyaring yang lupa dipasang terlihat sebagai baris tambahan —
// bukan sebagai daftar kosong, yang dapat lolos tanpa disadari.
func TestRequestBalaiLelangShowsOnlyTheCallersOwnRows(t *testing.T) {
	store := memory.NewSampleStore()

	page := list(t, store, inboxsalvage.TabRequestBalai, "")
	require.Len(t, page.Items, 1)
	require.Equal(t, memory.SampleCallerPIC, page.Items[0].PIC)

	// PIC lain melihat baris yang BERBEDA, bukan baris yang sama.
	other, err := inboxsalvage.NewQuery(
		inboxsalvage.QueryInput{Tab: inboxsalvage.TabRequestBalai},
		inboxsalvage.Caller{Login: "BUDISANTOSO"})
	require.NoError(t, err)

	otherPage, err := store.List(context.Background(), other,
		inboxsalvage.Pagination{Page: 1, Size: 100})
	require.NoError(t, err)
	require.Len(t, otherPage.Items, 1)
	require.NotEqual(t, page.Items[0].SalvageID, otherPage.Items[0].SalvageID)
}

// Pencacah "Request Balai Lelang" ikut disaring PIC pula — kueri lama membandingkan
// `B.PIC = {TempClaimAttach.UserAdmin}` di dalam `SUM(CASE WHEN …)`.
func TestRequestBalaiLelangCounterIsAlsoFilteredByCaller(t *testing.T) {
	store := memory.NewSampleStore()

	counts, err := store.Counts(context.Background(), callerPIC())
	require.NoError(t, err)

	for _, row := range counts {
		if row.Tab == inboxsalvage.TabRequestBalai {
			require.Equal(t, 1, row.Total,
				"dua baris berstatus 7 di data contoh, satu di antaranya milik PIC lain")
		}
	}
}

// "Checker" dan "Salvage Diterima" menyaring `STSTRANSFER` yang SAMA, sehingga isinya sama
// persis. Itu keadaan di Pega, bukan kekeliruan penyalinan.
func TestCheckerAndSalvageDiterimaListTheSameRows(t *testing.T) {
	store := memory.NewSampleStore()

	checker := claimNumbers(list(t, store, inboxsalvage.TabChecker, ""))
	diterima := claimNumbers(list(t, store, inboxsalvage.TabSalvageDiterima, ""))

	require.Equal(t, checker, diterima)
	require.Len(t, checker, 2)
}

// Daftar "Salvage Ditolak" BERISI baris di sini, dan di Pega selalu kosong — kode tabnya
// tidak pernah disebut langkah pengambilan data mana pun. Ia diperbaiki, dan uji ini
// menjaga perbaikannya tetap berlaku.
func TestSalvageDitolakHasRowsHereEvenThoughLegacyNeverRanItsQuery(t *testing.T) {
	store := memory.NewSampleStore()
	page := list(t, store, inboxsalvage.TabSalvageDitolak, "")

	require.Len(t, page.Items, 1)
	require.Equal(t, "5", page.Items[0].TransferStatus)
}

// Kolom "Status Lelang" membaca NILAINYA, bukan sekadar keberadaannya. Data contoh memuat
// satu baris bernilai NOL justru untuk membuktikannya.
func TestAuctionStatusDistinguishesZeroFromAnActualAmount(t *testing.T) {
	store := memory.NewSampleStore()
	page := list(t, store, inboxsalvage.TabBalaiLelang, "")

	byClaim := map[string]string{}
	for _, row := range page.Items {
		byClaim[row.ClaimNo] = row.AuctionStatus
	}

	require.Equal(t, inboxsalvage.AuctionSold, byClaim["PNC-2044"])
	require.Equal(t, inboxsalvage.AuctionUnsold, byClaim["PNC-2045"],
		"nilai akseptasi NOL berarti belum terjual")
}

// "Tipe Pengajuan" diturunkan dari terisi-tidaknya catatan request.
func TestSubmissionTypeFollowsWhetherTheRequestNoteIsFilled(t *testing.T) {
	store := memory.NewSampleStore()
	page := list(t, store, inboxsalvage.TabChecker, "")

	byClaim := map[string]string{}
	for _, row := range page.Items {
		byClaim[row.ClaimNo] = row.SubmissionType
	}

	require.Equal(t, inboxsalvage.SubmissionNew, byClaim["PNC-2046"])
	require.Equal(t, inboxsalvage.SubmissionRequest, byClaim["PNC-2047"])
}

// Kolom "Catatan" SELALU kosong, dan itu keadaan di Pega pula — tidak ada satu pun penulis
// di jalur pemuat daftar.
func TestTheCatatanColumnIsAlwaysEmpty(t *testing.T) {
	store := memory.NewSampleStore()

	for _, tab := range []string{
		inboxsalvage.TabChecker,
		inboxsalvage.TabRejectedChecker,
		inboxsalvage.TabRequestBalai,
	} {
		for _, row := range list(t, store, tab, "").Items {
			require.Empty(t, row.Note, "kolom Catatan pada daftar %s", tab)
		}
	}
}

// Pencarian COCOK PERSIS pada tab Checker, MENGANDUNG pada tab lain.
func TestSearchMatchesExactlyOnCheckerAndPartiallyElsewhere(t *testing.T) {
	store := memory.NewSampleStore()

	require.Empty(t, list(t, store, inboxsalvage.TabChecker, "PNC-204").Items,
		"separuh nomor klaim tidak cocok di tab Checker")
	require.Len(t, list(t, store, inboxsalvage.TabChecker, "PNC-2046").Items, 1)

	require.NotEmpty(t, list(t, store, inboxsalvage.TabHistori, "PNC-204").Items,
		"separuh nomor klaim cocok di tab Histori")
}

// Satu kotak pencarian tab Checker mencari DUA kolom sekaligus.
func TestCheckerSearchAlsoMatchesThePIC(t *testing.T) {
	store := memory.NewSampleStore()

	page := list(t, store, inboxsalvage.TabChecker, memory.SampleCallerPIC)
	require.Len(t, page.Items, 1)
	require.Equal(t, memory.SampleCallerPIC, page.Items[0].PIC)
}

// Baris keluarga C diurutkan menurut tanggal input, yang TERBARU lebih dulu —
// `order by a.tglinput desc` pada kueri lama.
func TestSalvageRowsAreOrderedByInputDateNewestFirst(t *testing.T) {
	store := memory.NewSampleStore()
	page := list(t, store, inboxsalvage.TabHistori, "")

	require.Greater(t, len(page.Items), 1)
	for index := 1; index < len(page.Items); index++ {
		require.GreaterOrEqual(t,
			page.Items[index-1].InputDate, page.Items[index].InputDate,
			"baris tidak terurut menurun menurut tanggal input")
	}
}

// Pencacah "Histori Salvage" menghitung `STSTRANSFER` 1 atau 6, sementara daftarnya tidak
// menyaring sama sekali. Ini selisih ketiga sejenis pada layar ini, dan ia direplikasi.
func TestHistoriCounterAndItsListDisagreeOnPurpose(t *testing.T) {
	store := memory.NewSampleStore()

	counts, err := store.Counts(context.Background(), callerPIC())
	require.NoError(t, err)

	var counter int
	for _, row := range counts {
		if row.Tab == inboxsalvage.TabHistori {
			counter = row.Total
		}
	}

	listed := list(t, store, inboxsalvage.TabHistori, "").Total

	require.Equal(t, 3, counter, "pencacah menghitung STSTRANSFER 1 atau 6")
	require.Equal(t, 9, listed, "daftarnya menampilkan seluruh pengajuan")
}

func TestCreateIssuesAnIncreasingSalvageIDAndTheRowAppearsInChecker(t *testing.T) {
	store := memory.NewSampleStore()

	form, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		ClaimNo:      "PNC-2041",
		ObjectName:   "Gudang Blok C",
		CoverageName: "Property All Risk",
		SalvageType:  "Besi Tua",
		InputDate:    "2026-09-25",
		MinimumValue: "1000000",
	}, callerPIC())
	require.NoError(t, err)

	id, err := store.Create(context.Background(), form)
	require.NoError(t, err)
	require.Equal(t, "110", id, "melanjutkan ID tertinggi pada data contoh")

	// Pengajuan baru LANGSUNG masuk antrean Checker di portal ASM.
	page := list(t, store, inboxsalvage.TabChecker, "")
	found := false
	for _, row := range page.Items {
		if row.SalvageID == id {
			found = true
			require.Equal(t, memory.SampleCallerPIC, row.PIC)
		}
	}
	require.True(t, found, "pengajuan baru tidak muncul di daftar Checker")
}

// Cacat `IDSALVAGE` yang tertimpa kosong saat pembaruan — butir 12 daftar perbaikan `P-5`
// (`D-49` #9) — sudah diperbaiki, dan uji ini menjaganya.
func TestUpdateKeepsTheSalvageIDInsteadOfBlankingIt(t *testing.T) {
	store := memory.NewSampleStore()

	form, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		Mode:         inboxsalvage.FormModeUpdate,
		SalvageID:    "103",
		ClaimNo:      "PNC-2046",
		ObjectName:   "Drum Kosong",
		CoverageName: "Marine Cargo",
		SalvageType:  "Drum Kosong",
		InputDate:    "2026-09-09",
		MinimumValue: "1900000",
	}, callerPIC())
	require.NoError(t, err)

	id, err := store.Create(context.Background(), form)
	require.NoError(t, err)
	require.Equal(t, "103", id)

	page := list(t, store, inboxsalvage.TabChecker, "PNC-2046")
	require.Len(t, page.Items, 1)
	require.Equal(t, "103", page.Items[0].SalvageID,
		"kunci barisnya tidak boleh tertimpa kosong")
	require.Equal(t, "1900000", page.Items[0].EstimateValue,
		"nilainya memang berubah")
}

func TestUpdatingAnUnknownSalvageIsRejected(t *testing.T) {
	store := memory.NewSampleStore()

	form, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		Mode:         inboxsalvage.FormModeUpdate,
		SalvageID:    "999",
		ClaimNo:      "PNC-2046",
		ObjectName:   "Drum Kosong",
		CoverageName: "Marine Cargo",
		SalvageType:  "Drum Kosong",
	}, callerPIC())
	require.NoError(t, err)

	_, err = store.Create(context.Background(), form)
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
}

func TestPaginationCutsThePageAndKeepsTheTotalIntact(t *testing.T) {
	store := memory.NewSampleStore()

	query, err := inboxsalvage.NewQuery(
		inboxsalvage.QueryInput{Tab: inboxsalvage.TabHistori}, callerPIC())
	require.NoError(t, err)

	first, err := store.List(context.Background(), query,
		inboxsalvage.Pagination{Page: 1, Size: 4})
	require.NoError(t, err)
	require.Len(t, first.Items, 4)
	require.Equal(t, 9, first.Total)
	require.Equal(t, 3, first.TotalPages())

	last, err := store.List(context.Background(), query,
		inboxsalvage.Pagination{Page: 3, Size: 4})
	require.NoError(t, err)
	require.Len(t, last.Items, 1)
	require.Equal(t, 9, last.Total)
}

func claimNumbers(page inboxsalvage.Page) []string {
	numbers := make([]string, 0, len(page.Items))
	for _, row := range page.Items {
		numbers = append(numbers, row.ClaimNo)
	}
	return numbers
}
