package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
)

// listTab mengambil SELURUH baris sebuah tab dari penyimpanan contoh.
//
// Ukuran halamannya sengaja dibuat besar supaya uji penyaring tidak ikut menguji paginasi —
// dua hal yang gagal karena sebab yang berbeda.
func listTab(t *testing.T, code string) inboxrclpucl.Page {
	t.Helper()

	tab, found := inboxrclpucl.FindTab(code)
	require.Truef(t, found, "tab %s tidak ditemukan", code)

	store := memory.NewSampleStore()
	page, err := store.List(
		context.Background(),
		inboxrclpucl.Query{
			Tab:    tab,
			Caller: inboxrclpucl.Caller{Login: "PETUGASCONTOH"},
		},
		inboxrclpucl.Pagination{Page: 1, Size: inboxrclpucl.MaxPageSize},
	)
	require.NoError(t, err)
	return page
}

// caseIDs mengumpulkan nomor case sebuah halaman.
func caseIDs(page inboxrclpucl.Page) []string {
	ids := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.CaseID)
	}
	return ids
}

// ---------------------------------------------------------------------------
// Isi tiap tab
// ---------------------------------------------------------------------------

func TestCetakSuratHoldsOnlyClaimsWithoutALetter(t *testing.T) {
	page := listTab(t, inboxrclpucl.TabCetakSurat)

	require.ElementsMatch(t, []string{"PNC-700001", "PNC-700002"}, caseIDs(page))

	// Kolom "Tanggal Cetak Surat" SELALU kosong di sini. Itu bukan data hilang — justru
	// itulah arti tab ini, dan penyaringnya `IS NULL`.
	for _, item := range page.Items {
		require.Emptyf(t, item.LetterPrintedAt,
			"%s: tab Cetak Surat tidak boleh memuat klaim bersurat", item.CaseID)
	}
}

func TestCetakSuratExcludesADifferentCaseStatus(t *testing.T) {
	// PNC-700003 suratnya belum dicetak, tetapi `STATUSCASE_1`-nya bukan '0'. Ia tidak
	// muncul di tab mana pun — dan itulah yang membuktikan penyaring STATUSCASE_1
	// benar-benar dipakai, bukan sekadar ikut tertulis di kueri.
	require.NotContains(t, caseIDs(listTab(t, inboxrclpucl.TabCetakSurat)), "PNC-700003")
	require.NotContains(t,
		caseIDs(listTab(t, inboxrclpucl.TabKelengkapanDokumen)), "PNC-700003")
	require.NotContains(t, caseIDs(listTab(t, inboxrclpucl.TabKlaimMSIG)), "PNC-700003")
}

func TestKelengkapanDokumenHoldsPrintedUnapprovedNonMSIG(t *testing.T) {
	page := listTab(t, inboxrclpucl.TabKelengkapanDokumen)

	require.ElementsMatch(t, []string{"PNC-700004", "PNC-700005"}, caseIDs(page))

	for _, item := range page.Items {
		require.NotEmptyf(t, item.LetterPrintedAt,
			"%s: tab ini hanya memuat klaim yang suratnya sudah dicetak", item.CaseID)
	}
}

func TestKlaimMSIGHoldsOnlyTheMSIGTrack(t *testing.T) {
	page := listTab(t, inboxrclpucl.TabKlaimMSIG)
	require.Equal(t, []string{"PNC-700008"}, caseIDs(page))
}

func TestTheMSIGClaimDoesNotLeakIntoKelengkapanDokumen(t *testing.T) {
	// Penyaring `MSIG_1 IS NULL` versus `= 'MSIG'` adalah SATU-SATUNYA yang membedakan
	// kedua tab. Kalau ia hilang di salah satunya, klaim MSIG muncul di keduanya — dan
	// karena kolom kedua tab IDENTIK, tidak ada apa pun di layar yang menandakannya.
	require.NotContains(t,
		caseIDs(listTab(t, inboxrclpucl.TabKelengkapanDokumen)), "PNC-700008")
}

// ---------------------------------------------------------------------------
// Penyaring yang mengeluarkan baris
// ---------------------------------------------------------------------------

func TestApprovedClaimsLeaveEveryTab(t *testing.T) {
	for _, code := range []string{
		inboxrclpucl.TabCetakSurat,
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		require.NotContainsf(t, caseIDs(listTab(t, code)), "PNC-700006",
			"klaim yang sudah disetujui muncul di tab %s", code)
	}
}

func TestAnEmptyApprovalFlagAlsoHidesTheRow(t *testing.T) {
	// Baris paling penting di seluruh berkas uji ini.
	//
	// `NULL <> '1'` bernilai UNKNOWN di Oracle maupun PostgreSQL — bukan TRUE — sehingga
	// klaim yang penanda persetujuannya belum pernah diisi TIDAK muncul, meski suratnya
	// sudah dicetak dan ia jelas belum disetujui.
	//
	// Perilakunya terasa keliru dan mungkin memang keliru. Uji ini ada supaya perilakunya
	// TIDAK berubah diam-diam: memperbaikinya akan menambah baris yang di Pega tidak
	// pernah terlihat, dan itu keputusan yang belum diambil siapa pun.
	require.NotContains(t,
		caseIDs(listTab(t, inboxrclpucl.TabKelengkapanDokumen)), "PNC-700007")
	require.NotContains(t, caseIDs(listTab(t, inboxrclpucl.TabKlaimMSIG)), "PNC-700007")
}

func TestCompletedWorkLeavesEveryTab(t *testing.T) {
	for _, code := range []string{
		inboxrclpucl.TabCetakSurat,
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		require.NotContainsf(t, caseIDs(listTab(t, code)), "PNC-700009",
			"klaim yang sudah selesai muncul di tab %s", code)
	}
}

func TestAnotherWorkbasketNeverAppears(t *testing.T) {
	// PNC-700010 berada di antrean komite. Tanpa penyaring akun antrean, layar ini akan
	// menampilkan SELURUH klaim yang belum selesai — pada basis data berisi puluhan juta
	// baris (`D-10`), hasilnya bukan sekadar keliru melainkan tidak dapat dipakai.
	for _, code := range []string{
		inboxrclpucl.TabCetakSurat,
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		require.NotContainsf(t, caseIDs(listTab(t, code)), "PNC-700010",
			"klaim antrean lain muncul di tab %s", code)
	}
}

func TestAPersonalAccidentClaimOutsideTheQueueNeverAppearsInAnyTab(t *testing.T) {
	// PNC-700011 hanya ada di laporan, lewat cabang kedua UNION. Ia TIDAK boleh muncul di
	// tab mana pun — antreannya bukan RCLPUCL.
	for _, code := range []string{
		inboxrclpucl.TabCetakSurat,
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		require.NotContainsf(t, caseIDs(listTab(t, code)), "PNC-700011",
			"klaim PA di luar antrean muncul di tab %s", code)
	}
}

// ---------------------------------------------------------------------------
// Penerjemahan dan urutan
// ---------------------------------------------------------------------------

func TestTrackIsDerivedNotStored(t *testing.T) {
	// Baris contoh menyimpan kode MENTAH; teksnya diturunkan lewat penerjemah milik
	// domain. Menyimpannya sudah jadi akan membuat penerjemah yang salah tetap lulus uji.
	page := listTab(t, inboxrclpucl.TabCetakSurat)

	tracks := map[string]string{}
	for _, item := range page.Items {
		tracks[item.CaseID] = item.Track
	}

	require.Equal(t, inboxrclpucl.TrackRCL, tracks["PNC-700001"])
	require.Equal(t, inboxrclpucl.TrackPUCL, tracks["PNC-700002"])
}

func TestRowsAreSortedNewestFirst(t *testing.T) {
	// `ORDER BY PXCREATEDATETIME DESC`. PNC-700002 dibuat setelah PNC-700001, sehingga ia
	// lebih dulu. Tanpa urutan yang tetap, satu baris dapat muncul di dua halaman begitu
	// hasilnya dipotong.
	require.Equal(t,
		[]string{"PNC-700002", "PNC-700001"},
		caseIDs(listTab(t, inboxrclpucl.TabCetakSurat)))
}

func TestPaginationCutsTheList(t *testing.T) {
	tab, _ := inboxrclpucl.FindTab(inboxrclpucl.TabCetakSurat)
	store := memory.NewSampleStore()

	first, err := store.List(context.Background(),
		inboxrclpucl.Query{Tab: tab, Caller: inboxrclpucl.Caller{Login: "X"}},
		inboxrclpucl.Pagination{Page: 1, Size: 1})
	require.NoError(t, err)

	require.Len(t, first.Items, 1)
	require.Equal(t, 2, first.Total, "total menghitung SELURUH baris, bukan halaman ini")
	require.Equal(t, 2, first.TotalPages())
}

// ---------------------------------------------------------------------------
// Laporan harian
// ---------------------------------------------------------------------------

// report mengambil laporan pada sebuah rentang.
func report(t *testing.T, from, to string) ([]inboxrclpucl.DailyReportRow, int) {
	t.Helper()

	store := memory.NewSampleStore()
	rows, total, err := store.DailyReport(
		context.Background(),
		inboxrclpucl.DateRange{From: from, To: to},
		inboxrclpucl.Pagination{Page: 1, Size: inboxrclpucl.MaxPageSize},
	)
	require.NoError(t, err)
	return rows, total
}

func TestDailyReportIncludesClaimsThatNoTabShows(t *testing.T) {
	// Inilah pembuktian bahwa laporan dan tabel memang berbeda isinya.
	//
	// PNC-700011 adalah klaim Personal Accident DI LUAR antrean RCL/PUCL — ia tidak muncul
	// di tab mana pun, tetapi IKUT di laporan lewat cabang kedua UNION.
	rows, _ := report(t, "2026-09-01", "2026-09-30")

	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.CaseID)
	}
	require.Contains(t, ids, "PNC-700011")
}

func TestDailyReportIgnoresTheWorkStatusFilter(t *testing.T) {
	// Kueri laporan TIDAK menyaring status kerja, sehingga klaim yang sudah selesai ikut.
	// Menyaringnya seperti grid akan menghasilkan laporan yang isinya berbeda dari Oracle.
	rows, _ := report(t, "2026-09-01", "2026-09-30")

	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.CaseID)
	}
	require.Contains(t, ids, "PNC-700009", "klaim selesai harus ikut di laporan")
}

func TestDailyReportIgnoresTheApprovalFilter(t *testing.T) {
	rows, _ := report(t, "2026-09-01", "2026-09-30")

	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.CaseID)
	}
	require.Contains(t, ids, "PNC-700006", "klaim yang sudah disetujui harus ikut")
}

func TestDailyReportExcludesOtherWorkbasketsUnlessPersonalAccident(t *testing.T) {
	// PNC-700010 ada di antrean komite DAN bukan Personal Accident, sehingga ia tidak
	// memenuhi satu pun cabang UNION.
	rows, _ := report(t, "2026-09-01", "2026-09-30")

	for _, row := range rows {
		require.NotEqual(t, "PNC-700010", row.CaseID)
	}
}

func TestDailyReportHonoursTheDateRange(t *testing.T) {
	// Rentang sempit hanya memuat baris yang tanggal kirimnya di dalamnya.
	rows, total := report(t, "2026-09-10", "2026-09-11")

	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.CaseID)
	}
	require.ElementsMatch(t, []string{"PNC-700001", "PNC-700002"}, ids)
	require.Equal(t, 2, total)
}

func TestDailyReportIncludesTheWholeLastDay(t *testing.T) {
	// Batas atasnya setengah terbuka. Baris PNC-700001 dikirim pukul 09.30 pada 10
	// September; rentang yang berakhir pada tanggal itu WAJIB memuatnya.
	//
	// Menuliskannya sebagai `<= akhir` akan membuangnya, dan kesalahan itu hanya terlihat
	// pada data yang punya komponen jam.
	rows, _ := report(t, "2026-09-10", "2026-09-10")

	require.Len(t, rows, 1)
	require.Equal(t, "PNC-700001", rows[0].CaseID)
}

func TestDailyReportDeduplicatesLikeUnion(t *testing.T) {
	// `UNION`, bukan `UNION ALL`. Tidak ada satu pun nomor case yang muncul dua kali,
	// meski sebuah klaim dapat memenuhi kedua cabang sekaligus.
	rows, _ := report(t, "2026-09-01", "2026-09-30")

	seen := map[string]bool{}
	for _, row := range rows {
		require.Falsef(t, seen[row.CaseID], "%s muncul dua kali", row.CaseID)
		seen[row.CaseID] = true
	}
}

func TestDailyReportTranslatesTheTrackCode(t *testing.T) {
	// Kueri lama menuliskan angka mentah ke berkas. Di sini ia diterjemahkan — selisih
	// terencana, supaya berkas dan layar menyebut hal yang sama dengan kata yang sama.
	rows, _ := report(t, "2026-09-10", "2026-09-10")

	require.Len(t, rows, 1)
	require.Equal(t, inboxrclpucl.TrackRCL, rows[0].Track)
}

func TestDailyReportCarriesClaimStatusNotExpiryStatus(t *testing.T) {
	// `STATUSCLAIM_1` (Status Klaim, 33 kode) — BUKAN `STATUSKLAIM_1` yang digambar grid
	// sebagai "Status Kadaluarsa". Kedua nama kolom hanya berbeda satu huruf, dan
	// menukarnya tidak menghasilkan satu pun galat.
	rows, _ := report(t, "2026-09-10", "2026-09-10")

	require.Len(t, rows, 1)
	require.Equal(t, "1142", rows[0].ClaimStatus)
	require.NotEqual(t, "Belum Kadaluarsa", rows[0].ClaimStatus)
}

func TestDailyReportRejectsAnUnreadableRange(t *testing.T) {
	// Rentang yang tidak terbaca seharusnya sudah ditolak NewReportRequest. Sampai di
	// sini, mengembalikan kosong lebih jujur daripada mengembalikan seluruh baris.
	store := memory.NewSampleStore()
	rows, total, err := store.DailyReport(
		context.Background(),
		inboxrclpucl.DateRange{From: "kemarin", To: "besok"},
		inboxrclpucl.Pagination{Page: 1, Size: 50},
	)

	require.NoError(t, err)
	require.Empty(t, rows)
	require.Zero(t, total)
}

func TestSampleStoreCoversEveryFilterInBothDirections(t *testing.T) {
	// Uji yang seluruh barisnya lolos tidak membuktikan penyaringnya bekerja. Susunan
	// contoh ini sengaja memuat baris yang TERTOLAK oleh setiap penyaring — dan uji ini
	// menjaga agar baris-baris itu tidak terhapus kelak karena dikira tidak terpakai.
	rows := memory.SampleRows()
	require.Len(t, rows, 11)

	total := len(rows)
	shown := len(listTab(t, inboxrclpucl.TabCetakSurat).Items) +
		len(listTab(t, inboxrclpucl.TabKelengkapanDokumen).Items) +
		len(listTab(t, inboxrclpucl.TabKlaimMSIG).Items)

	require.Less(t, shown, total,
		"setiap penyaring harus punya baris yang tertolak olehnya")
}
