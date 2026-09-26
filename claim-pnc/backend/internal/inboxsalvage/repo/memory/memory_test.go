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

	// FindAnyTab, bukan NewQuery: sebagian uji di berkas ini memeriksa penyaring daftar
	// yang TIDAK ditawarkan layar. Penyaringnya tetap harus benar — tiga di antaranya
	// kembali begitu kewenangan berbasis peran ada — dan penyaring yang tidak diuji akan
	// berhenti benar tanpa ada yang tahu.
	//
	// Penolakan kode yang tidak ditawarkan diuji terpisah, pada NewQuery.
	found, known := inboxsalvage.FindAnyTab(tab)
	require.Truef(t, known, "daftar %q tidak ada sama sekali", tab)

	query, err := inboxsalvage.NewQueryForTab(
		found, inboxsalvage.QueryInput{Tab: tab, Search: search}, callerPIC())
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

	// Klaim yang belum punya pengajuan sama sekali juga masuk — penandanya kosong, dan
	// itulah satu-satunya yang disaring daftar ini.
	require.Contains(t, numbers, memory.SampleClaimWithoutSalvage)

	// PNC-2043 penandanya kosong PULA, tetapi status kerjanya sudah selesai.
	require.NotContains(t, numbers, "PNC-2043",
		"klaim yang sudah selesai dikeluarkan dari daftar ini")

	require.Len(t, page.Items, 3)
}

// Inilah selisih yang DIREPLIKASI (`P-5`), dan uji ini menjaganya tetap ada.
//
// Pencacah "Outstanding" menghitung `STSSALVAGE` 3 atau 5, sementara daftarnya menampilkan
// yang penandanya KOSONG. Keduanya menghitung populasi yang BERBEDA, dan pada data contoh
// keduanya kebetulan berjumlah sama — sehingga yang diuji adalah barisnya, bukan angkanya.
//
// Bila seseorang "memperbaikinya", uji inilah yang gagal lebih dulu, bukan pengguna yang
// melaporkannya.
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
	require.Equal(t, 3, listed, "daftarnya menampilkan STSSALVAGE yang kosong")

	// Angkanya sama, ISINYA tidak. Inilah yang membuktikan keduanya menghitung populasi
	// yang berbeda — dan yang akan gagal bila salah satunya diam-diam disamakan dengan
	// yang lain.
	numbers := claimNumbers(list(t, store, inboxsalvage.TabOutstanding, ""))
	require.NotContains(t, numbers, "PNC-2044",
		"klaim ber-STSSALVAGE 3 dihitung pencacah tetapi TIDAK ditampilkan daftarnya")
	require.Contains(t, numbers, memory.SampleClaimWithoutSalvage,
		"klaim tanpa penanda ditampilkan daftarnya tetapi TIDAK dihitung pencacah")
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
	requestTab, known := inboxsalvage.FindAnyTab(inboxsalvage.TabRequestBalai)
	require.True(t, known)

	other, err := inboxsalvage.NewQueryForTab(requestTab,
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

	// SELURUH daftar yang ditawarkan diperiksa, bukan dua yang dipilih — kolom "Catatan"
	// digambar pada lima daftar, dan kosongnya berlaku pada semuanya.
	for _, tab := range inboxsalvage.Tabs() {
		for _, row := range list(t, store, tab.Code, "").Items {
			require.Empty(t, row.Note, "kolom Catatan pada daftar %s", tab.Code)
		}
	}
}

// Pencarian pada daftar yang ditawarkan MENGANDUNG, bukan cocok persis.
func TestSearchOnOfferedListsMatchesPartially(t *testing.T) {
	store := memory.NewSampleStore()

	require.NotEmpty(t, list(t, store, inboxsalvage.TabHistori, "PNC-204").Items,
		"separuh nomor klaim cocok di tab Histori")
}

// Sejak daftar dibatasi sembilan, TIDAK ADA satu pun daftar yang ditawarkan memakai
// pencarian cocok persis maupun pencarian menurut PIC.
//
// Ketiganya — Checker, Salvage Diterima, Salvage Ditolak — hanya tampil bagi dua operator
// bernama di sistem lama, dan ketiganya pula satu-satunya yang mencocokkan persis.
//
// Uji ini ADA supaya kenyataan itu tidak tertinggal di dokumen: selisih terencana tentang
// "pencarian cocok persis" dan keterangannya di layar tidak berlaku bagi satu pun daftar
// yang sekarang ditawarkan. Bila kelak ketiganya kembali, uji inilah yang gagal lebih dulu
// dan mengingatkan keterangan itu harus dihidupkan kembali.
func TestNoOfferedListUsesExactSearchOrSearchesByPIC(t *testing.T) {
	for _, tab := range inboxsalvage.Tabs() {
		require.Falsef(t, tab.SearchExact,
			"daftar %q mencocokkan persis — keterangannya di layar harus dihidupkan", tab.Code)
		require.Falsef(t, tab.SearchByPIC,
			"daftar %q mencari menurut PIC", tab.Code)
	}

	// Dan ketiganya memang masih tersimpan, hanya tidak ditawarkan.
	exact := 0
	for _, tab := range inboxsalvage.AllTabs() {
		if tab.SearchExact {
			exact++
			require.NotEmpty(t, tab.HiddenReason)
		}
	}
	require.Equal(t, 3, exact)
}

// TIDAK ADA satu pun daftar yang ditawarkan menyaring menurut pemanggil.
//
// Sebelumnya ada satu — "Request Balai Lelang" — dan ia menjadi satu-satunya pembatas
// berbasis pengguna di modul ini. Sejak ia tidak lagi ditawarkan, KESEMBILAN daftar
// bersama, dan jejak audit menjadi satu-satunya kontrol yang tersisa (`D-59`).
func TestNoOfferedListIsScopedToTheCaller(t *testing.T) {
	for _, tab := range inboxsalvage.Tabs() {
		require.Falsef(t, tab.OwnedByCaller,
			"daftar %q menyaring menurut pemanggil", tab.Code)
	}
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
	//
	// Diperiksa lewat nilai yang TERSIMPAN, bukan lewat daftar Checker — daftar itu tidak
	// lagi ditawarkan di layar, sementara barisnya tetap masuk antreannya. Keduanya hal
	// yang berbeda, dan yang diuji di sini adalah yang kedua.
	detail, err := store.Detail(context.Background(), id)
	require.NoError(t, err)

	require.Equal(t, "3", detail.TransferStatus,
		"pengajuan baru harus bertanda ke checker")

	// PIC pengaju terbaca dari grid riwayat, bukan dari kepala panel: kepala panel
	// pengajuan memang tidak memuat PIC — kueri lamanya tidak mengambil kolom itu.
	found := false
	for _, row := range detail.History {
		if row.SalvageID == id {
			found = true
			require.Equal(t, memory.SampleCallerPIC, row.PIC)
			require.Equal(t, inboxsalvage.HistoryApprovedByKomite, row.Position,
				"STSTRANSFER 3 pada grid riwayat berbunyi lain dari nama daftarnya")
		}
	}
	require.True(t, found, "pengajuan baru tidak muncul di riwayat klaimnya")
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

	detail, err := store.Detail(context.Background(), "103")
	require.NoError(t, err)
	require.Equal(t, "103", detail.SalvageID,
		"kunci barisnya tidak boleh tertimpa kosong")
	require.Equal(t, "1900000", detail.EstimateValue,
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

// Panel Detail Salvage mengisi kepala panel dari pengajuannya dan lini bisnis dari klaimnya.
//
// Lini bisnis diuji terpisah karena ia SATU-SATUNYA isian panel yang tidak berasal dari
// tabel salvage — di Oracle ia sub-kueri ke tabel klaim, dan sub-kueri yang tidak menemukan
// baris menghasilkan kosong tanpa satu pun galat.
func TestDetailReadsTheHeaderAndTheBusinessLineOfItsClaim(t *testing.T) {
	store := memory.NewSampleStore()

	detail, err := store.Detail(context.Background(), "101")
	require.NoError(t, err)

	require.Equal(t, "101", detail.SalvageID)
	require.Equal(t, "PNC-2044", detail.ClaimNo)
	require.Equal(t, "Besi Tua", detail.SalvageType)
	require.Equal(t, "Gudang Cakung", detail.Location)
	require.NotEmpty(t, detail.BusinessName, "lini bisnis klaim tidak terbaca")
}

// "Posisi Salvage" adalah LABEL, bukan kode.
//
// Yang tersimpan `STSTRANSFER` berupa angka; yang dibaca pengguna nama daftarnya. Keduanya
// dikirim, dan uji ini menjaga supaya kodenya tidak pernah ikut tergambar sebagai posisi.
func TestDetailTranslatesTheTransferCodeIntoTheNameUsersRead(t *testing.T) {
	store := memory.NewSampleStore()

	detail, err := store.Detail(context.Background(), "101")
	require.NoError(t, err)

	require.Equal(t, "1", detail.TransferStatus)
	require.NotEqual(t, detail.TransferStatus, detail.Position,
		"posisi masih berisi kodenya, bukan namanya")
	require.NotEmpty(t, detail.Position)
}

// ID yang tidak ada menghasilkan galat TERSENDIRI, bukan panel kosong.
//
// Bedanya nyata di layar: panel kosong terbaca sebagai pengajuan tanpa isi, sedangkan galat
// ini menyampaikan bahwa pengajuannya tidak ada pada entitas yang sedang dipilih — penyebab
// yang jauh lebih sering (`R-20`).
func TestDetailOfAnUnknownSubmissionIsNotAnEmptyPanel(t *testing.T) {
	store := memory.NewSampleStore()

	_, err := store.Detail(context.Background(), "tidak-ada")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)

	_, err = store.Detail(context.Background(), "   ")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
}

// Pengajuan yang baru disimpan langsung dapat dibuka detailnya, lengkap dengan barangnya.
func TestDetailOfAFreshlySavedSubmissionCarriesItsItems(t *testing.T) {
	store := memory.NewSampleStore()

	form, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		Mode:         inboxsalvage.FormModeInsert,
		ClaimNo:      "PNC-2044",
		ObjectName:   "Mesin Cetak",
		CoverageName: "All Risk",
		SalvageType:  "Mesin",
		Location:     "Gudang Cakung",
		MinimumValue: "1000000",
		Items: []inboxsalvage.DetailItem{
			{Name: "Rotor", Quantity: "2", Unit: "unit"},
			{Name: "Panel", Quantity: "1", Unit: "unit"},
		},
	}, callerPIC())
	require.NoError(t, err)

	id, err := store.Create(context.Background(), form)
	require.NoError(t, err)

	detail, err := store.Detail(context.Background(), id)
	require.NoError(t, err)

	require.Len(t, detail.Items, 2)
	require.Equal(t, "Rotor", detail.Items[0].Name)

	// Barang yang baru disimpan BELUM terjual, dan itu harus terbaca sebagai kalimat —
	// bukan sebagai kode "0" yang tidak berarti apa-apa bagi pembacanya.
	require.NotEqual(t, "0", detail.Items[0].SoldStatus)
	require.NotEmpty(t, detail.Items[0].SoldStatus)
}

// Panel rincian dapat dibuka dari baris yang berupa KLAIM, bukan hanya dari pengajuan.
//
// Keenam daftar berbasis klaim tidak membawa ID pengajuan pada barisnya, sehingga
// pengajuannya dicari dari klaimnya — mengikuti `SetDataDetailSalvage_act` langkah 18.
func TestDetailByClaimFindsTheLatestSubmissionOfThatClaim(t *testing.T) {
	store := memory.NewSampleStore()

	byClaim, err := store.DetailByClaim(context.Background(), "PNC-2044")
	require.NoError(t, err)

	require.True(t, byClaim.HasSubmission)
	require.Equal(t, "PNC-2044", byClaim.ClaimNo)

	// Isian klaim ikut terisi — keduanya TIDAK ada pada kepala panel pengajuan, dan
	// keduanya digambar pada panel yang dibuka dari daftar berbasis klaim.
	require.NotEmpty(t, byClaim.PIC)
	require.NotEmpty(t, byClaim.BusinessName)
}

// Pengajuan yang dipilih adalah yang TERAKHIR, dibandingkan sebagai angka.
//
// Perbandingan teks akan menempatkan "9" di atas "10" — kekeliruan yang tidak terlihat
// sampai nomor pengajuan melewati sepuluh.
func TestDetailByClaimPicksTheHighestSalvageIDNumerically(t *testing.T) {
	store := memory.NewSampleStore()

	form, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		ClaimNo:      "PNC-2044",
		ObjectName:   "Mesin Cetak",
		CoverageName: "All Risk",
		SalvageType:  "Mesin",
		MinimumValue: "1000000",
	}, callerPIC())
	require.NoError(t, err)

	fresh, err := store.Create(context.Background(), form)
	require.NoError(t, err)

	byClaim, err := store.DetailByClaim(context.Background(), "PNC-2044")
	require.NoError(t, err)
	require.Equal(t, fresh, byClaim.SalvageID,
		"pengajuan terbaru yang harus terbaca, bukan yang nomornya terbesar sebagai teks")
}

// Klaim TANPA pengajuan bukan galat — ia panel dengan isian klaim saja.
//
// Ini keadaan yang lazim, bukan kasus tepi: daftar Salvage Outstanding justru berisi klaim
// yang salvage-nya belum ditandai sama sekali. Menjawabnya sebagai "tidak ditemukan" akan
// membuat hampir setiap baris daftar itu terbaca sebagai kerusakan.
func TestDetailByClaimOfAClaimWithoutAnySubmissionIsNotAnError(t *testing.T) {
	store := memory.NewSampleStore()

	empty := memory.SampleClaimWithoutSalvage

	byClaim, err := store.DetailByClaim(context.Background(), empty)
	require.NoError(t, err)

	require.False(t, byClaim.HasSubmission)
	require.Equal(t, empty, byClaim.ClaimNo)
	require.Empty(t, byClaim.SalvageID)
	require.Empty(t, byClaim.Items)
}

// Nomor klaim yang TIDAK ada tetap galat — dan itu bedanya dengan kasus di atas.
func TestDetailByClaimOfAnUnknownClaimIsNotFound(t *testing.T) {
	store := memory.NewSampleStore()

	_, err := store.DetailByClaim(context.Background(), "PNC-tidak-ada")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
}

// Grid riwayat memuat SELURUH pengajuan milik klaim itu, terbaru lebih dulu.
func TestHistoryListsEverySubmissionOfTheClaimNewestFirst(t *testing.T) {
	store := memory.NewSampleStore()

	// PNC-2044 punya DUA pengajuan pada data contoh — itulah sebabnya ia dipakai di sini.
	detail, err := store.DetailByClaim(context.Background(), "PNC-2044")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(detail.History), 2,
		"data contoh harus memuat klaim dengan lebih dari satu pengajuan")

	for index := 1; index < len(detail.History); index++ {
		require.GreaterOrEqual(t,
			detail.History[index-1].InputDate, detail.History[index].InputDate,
			"riwayat harus terurut terbaru lebih dulu")
	}

	for _, row := range detail.History {
		require.Equal(t, "PNC-2044", row.ClaimNo)
		require.NotEmpty(t, row.Position, "posisi harus berupa kalimat, bukan kosong")
	}
}

// Klaim tanpa pengajuan punya riwayat KOSONG — bukan galat, dan bukan nil.
//
// Bedanya nyata di layar: senarai kosong menggambar grid beserta keterangannya, sedangkan
// nil pada JSON menjadi `null` dan membuat grid gagal digambar sama sekali.
func TestHistoryOfAClaimWithoutSubmissionsIsEmptyNotNil(t *testing.T) {
	store := memory.NewSampleStore()

	detail, err := store.DetailByClaim(
		context.Background(), memory.SampleClaimWithoutSalvage)
	require.NoError(t, err)

	require.NotNil(t, detail.History)
	require.Empty(t, detail.History)
}
