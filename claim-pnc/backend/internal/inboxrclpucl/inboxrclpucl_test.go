package inboxrclpucl_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
)

// caller adalah pemanggil yang sah, dipakai hampir seluruh uji di berkas ini.
func caller() inboxrclpucl.Caller {
	return inboxrclpucl.Caller{Login: "PETUGASCONTOH"}
}

// ---------------------------------------------------------------------------
// Bentuk layar
// ---------------------------------------------------------------------------

func TestThereAreExactlyThreeTabs(t *testing.T) {
	// Ketiganya dari `Section/InputPUCL-RCL_Section-Section.xml`, yang menyertakan tepat
	// tiga SUB_SECTION. Tab keempat berarti ada yang dikarang; tab yang hilang berarti ada
	// antrean yang tidak dapat dibuka siapa pun.
	tabs := inboxrclpucl.Tabs()
	require.Len(t, tabs, 3)

	require.Equal(t, "Cetak Surat", tabs[0].Name)
	require.Equal(t, "Kelengkapan Dokumen", tabs[1].Name)
	require.Equal(t, "Klaim MSIG", tabs[2].Name)
}

func TestTabsReturnsACopy(t *testing.T) {
	// Pemanggil tidak boleh dapat mengubah daftar tab dengan menulisi hasilnya.
	first := inboxrclpucl.Tabs()
	first[0].Name = "Diubah"

	second := inboxrclpucl.Tabs()
	require.Equal(t, "Cetak Surat", second[0].Name)
}

func TestEveryTabDrawsTheSameNineColumns(t *testing.T) {
	// Kesembilan kolom dibaca dari sel grid ketiga section, dan ketiganya IDENTIK kecuali
	// judul kolom keenam. Kolom yang bertambah atau berkurang di salah satu tab berarti
	// salah satu section dibaca keliru.
	wanted := []string{
		inboxrclpucl.FieldCaseID,
		inboxrclpucl.FieldPolicyNumber,
		inboxrclpucl.FieldInsuredName,
		inboxrclpucl.FieldInboxEntryAt,
		inboxrclpucl.FieldAnalystNote,
		inboxrclpucl.FieldTrack,
		inboxrclpucl.FieldLetterPrintedAt,
		inboxrclpucl.FieldClaimAge,
		inboxrclpucl.FieldExpiryStatus,
	}

	for _, tab := range inboxrclpucl.Tabs() {
		keys := []string{}
		for _, column := range tab.Columns {
			keys = append(keys, column.Key)
		}
		require.Equalf(t, wanted, keys, "kolom tab %s berbeda", tab.Name)
	}
}

func TestOnlyTheMSIGTabRenamesTheTrackColumn(t *testing.T) {
	// `Section/InboxAJSMSIG_Section-Section.xml` menuliskan judulnya "Status" saja, tanpa
	// "RCL/PUCL". Ia dibawa apa adanya (`D-13`) meski kolom sumbernya sama persis di
	// ketiga tab.
	titles := map[string]string{}
	for _, tab := range inboxrclpucl.Tabs() {
		for _, column := range tab.Columns {
			if column.Key == inboxrclpucl.FieldTrack {
				titles[tab.Code] = column.Title
			}
		}
	}

	require.Equal(t, "Status RCL/PUCL", titles[inboxrclpucl.TabCetakSurat])
	require.Equal(t, "Status RCL/PUCL", titles[inboxrclpucl.TabKelengkapanDokumen])
	require.Equal(t, "Status", titles[inboxrclpucl.TabKlaimMSIG])
}

func TestOnlyTheCetakSuratTabHasADateRangeReport(t *testing.T) {
	// Hanya `Activity/ExportCetakSurat_act` yang menjalankan kueri rentang tanggal; dua
	// activity ekspor lain menjalankan Report Definition grid-nya sendiri.
	//
	// Kalau tab lain ikut ditandai punya laporan, layar akan menampilkan dua isian tanggal
	// yang tidak menyetir apa pun — dan pengguna akan mengisinya lalu bertanya-tanya
	// mengapa tidak terjadi apa-apa.
	for _, tab := range inboxrclpucl.Tabs() {
		if tab.Code == inboxrclpucl.TabCetakSurat {
			require.True(t, tab.HasDateRangeReport)
			continue
		}
		require.Falsef(t, tab.HasDateRangeReport,
			"tab %s seharusnya tidak punya laporan rentang tanggal", tab.Name)
	}
}

func TestLetterPrintedFilterSplitsTheTabs(t *testing.T) {
	// Inilah penyaring yang memindahkan satu klaim dari tab pertama ke tab kedua begitu
	// suratnya dicetak. Membaliknya menukar isi dua tab tanpa satu pun galat.
	cetak, _ := inboxrclpucl.FindTab(inboxrclpucl.TabCetakSurat)
	lengkap, _ := inboxrclpucl.FindTab(inboxrclpucl.TabKelengkapanDokumen)
	msig, _ := inboxrclpucl.FindTab(inboxrclpucl.TabKlaimMSIG)

	require.False(t, cetak.LetterPrinted, "tab Cetak Surat: surat BELUM dicetak")
	require.True(t, lengkap.LetterPrinted)
	require.True(t, msig.LetterPrinted)

	require.False(t, lengkap.MSIG)
	require.True(t, msig.MSIG)
}

func TestTheMSIGTabIsNotMarkedBlocked(t *testing.T) {
	// Tab Klaim MSIG kemungkinan kosong di produksi, tetapi ia TIDAK terhalang: kuerinya
	// dapat dijalankan, dan kosongnya adalah JAWABAN — bukan ketidakmampuan menjawab.
	//
	// Menandainya terhalang akan menolak permintaannya di NewQuery, sehingga baris yang
	// mungkin memang ada tidak pernah ditampilkan.
	msig, found := inboxrclpucl.FindTab(inboxrclpucl.TabKlaimMSIG)
	require.True(t, found)
	require.False(t, msig.Blocked)
	require.NotEmpty(t, msig.Notice, "kosongnya harus dijelaskan ke pengguna")
}

func TestNoTabIsBlocked(t *testing.T) {
	for _, tab := range inboxrclpucl.Tabs() {
		require.Falsef(t, tab.Blocked, "tab %s tidak seharusnya terhalang", tab.Name)
	}
}

func TestPlannedDifferencesMentionTheThingsMostLikelyReportedAsBugs(t *testing.T) {
	// Tiga hal di layar ini akan dilaporkan sebagai kerusakan oleh orang yang
	// membandingkan kedua layar berdampingan. Ketiganya WAJIB dinyatakan lebih dulu,
	// bukan dijelaskan setelah dilaporkan.
	joined := strings.ToLower(strings.Join(inboxrclpucl.PlannedDifferences, " "))

	require.Contains(t, joined, "msig",
		"tab yang kemungkinan kosong harus dinyatakan")
	require.Contains(t, joined, "tidak menyaring tabel",
		"isian tanggal yang tidak menyaring grid harus dinyatakan")
	require.Contains(t, joined, "laporan harian",
		"isi berkas ekspor yang berbeda dari tabel harus dinyatakan")
}

// ---------------------------------------------------------------------------
// Penerjemah jalur
// ---------------------------------------------------------------------------

func TestTrackOfFollowsTheLegacyCaseWithoutElse(t *testing.T) {
	require.Equal(t, inboxrclpucl.TrackRCL, inboxrclpucl.TrackOf("1"))
	require.Equal(t, inboxrclpucl.TrackPUCL, inboxrclpucl.TrackOf("2"))

	// `CASE` di sistem lama TANPA `ELSE`, sehingga nilai lain menghasilkan kosong — bukan
	// kode mentahnya, dan bukan teks pengganti. Sel kosong adalah jawaban yang benar untuk
	// jalur yang tidak dikenali.
	require.Empty(t, inboxrclpucl.TrackOf("3"))
	require.Empty(t, inboxrclpucl.TrackOf(""))
	require.Empty(t, inboxrclpucl.TrackOf("RCL"))
}

func TestTrackOfIgnoresSurroundingSpaces(t *testing.T) {
	// Kolomnya bertipe teks dan sebagian nilai di Oracle berspasi-rata. Tanpa pemangkasan,
	// jalur yang sah terbaca sebagai tidak dikenali dan selnya kosong.
	require.Equal(t, inboxrclpucl.TrackRCL, inboxrclpucl.TrackOf(" 1 "))
}

// ---------------------------------------------------------------------------
// Paginasi
// ---------------------------------------------------------------------------

func TestPaginationDefaultsToTheLegacyPageSize(t *testing.T) {
	// 50, mengikuti `<pyPageSize>50</pyPageSize>` ketiga section — bukan 25 seperti
	// sebagian modul inbox lain.
	clean := inboxrclpucl.Pagination{}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxrclpucl.DefaultPageSize, clean.Size)
	require.Equal(t, 50, inboxrclpucl.DefaultPageSize)
}

func TestPaginationClampsInsteadOfRejecting(t *testing.T) {
	// Nilai di luar rentang DIBETULKAN, bukan ditolak: keduanya datang dari parameter
	// query yang mudah salah ketik, dan menolak seluruh permintaan karena `halaman=0` akan
	// membuat layar gagal tanpa alasan yang terbaca pengguna.
	clean := inboxrclpucl.Pagination{Page: -4, Size: 9999}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxrclpucl.MaxPageSize, clean.Size)
}

func TestOffsetSkipsWholePages(t *testing.T) {
	page := inboxrclpucl.Pagination{Page: 3, Size: 50}
	require.Equal(t, 100, page.Offset())
}

func TestTotalPagesNeverReportsZero(t *testing.T) {
	// Layar tidak boleh pernah menggambar "halaman 1 dari 0".
	empty := inboxrclpucl.Page{Pagination: inboxrclpucl.Pagination{Page: 1, Size: 50}}
	require.Equal(t, 1, empty.TotalPages())
}

func TestTotalPagesRoundsUp(t *testing.T) {
	page := inboxrclpucl.Page{
		Total:      101,
		Pagination: inboxrclpucl.Pagination{Page: 1, Size: 50},
	}
	require.Equal(t, 3, page.TotalPages())
}

func TestSliceReturnsAnEmptyListBeyondTheLastPage(t *testing.T) {
	// Senarai kosong, bukan nil: `[]` dan `null` ditangani berbeda oleh klien.
	all := []inboxrclpucl.WorkItem{{CaseID: "PNC-1"}, {CaseID: "PNC-2"}}

	page := inboxrclpucl.Slice(all, inboxrclpucl.Pagination{Page: 9, Size: 50})
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)

	// Totalnya tetap benar meski halamannya melewati ujung — layar memakainya untuk
	// menggambar penomoran, dan nol akan membuatnya tampak seperti antrean kosong.
	require.Equal(t, 2, page.Total)
}

// ---------------------------------------------------------------------------
// Permintaan isi tab
// ---------------------------------------------------------------------------

func TestNewQueryFallsBackToTheDefaultTab(t *testing.T) {
	query, err := inboxrclpucl.NewQuery(inboxrclpucl.QueryInput{}, caller())
	require.NoError(t, err)
	require.Equal(t, inboxrclpucl.TabCetakSurat, query.Tab.Code)
}

func TestNewQueryRejectsAnUnknownTab(t *testing.T) {
	_, err := inboxrclpucl.NewQuery(
		inboxrclpucl.QueryInput{Tab: "99"}, caller())

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxrclpucl.FieldTab, validation.Violations[0].Field)
}

func TestNewQueryRequiresACaller(t *testing.T) {
	// Antreannya bersama, sehingga identitas TIDAK dipakai menyaring. Ia tetap wajib:
	// tanpanya pembukaan layar ini tidak dapat dicatat, dan jejak itulah satu-satunya
	// kontrol yang tersisa selama pemeriksaan peran belum ada (`D-59`).
	_, err := inboxrclpucl.NewQuery(
		inboxrclpucl.QueryInput{}, inboxrclpucl.Caller{Login: "   "})

	require.ErrorIs(t, err, inboxrclpucl.ErrCallerUnknown)
}

// ---------------------------------------------------------------------------
// Permintaan laporan harian
// ---------------------------------------------------------------------------

func TestNewReportRequestAcceptsACompleteRange(t *testing.T) {
	request, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
		Tab:  inboxrclpucl.TabCetakSurat,
		From: "2026-09-01",
		To:   "2026-09-30",
	}, caller())

	require.NoError(t, err)
	require.Equal(t, "2026-09-01", request.Range.From)
	require.Equal(t, "2026-09-30", request.Range.To)
}

func TestNewReportRequestAcceptsASingleDay(t *testing.T) {
	// Kedua batas boleh sama. Kuerinya memakai rentang setengah terbuka, sehingga satu
	// tanggal tetap mencakup seluruh jam pada hari itu.
	_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
		From: "2026-09-10",
		To:   "2026-09-10",
	}, caller())

	require.NoError(t, err)
}

func TestNewReportRequestReportsBothMissingDatesAtOnce(t *testing.T) {
	// `P-5` dan `11-CROSSCUTTING.md` §1.1: seluruh pelanggaran dikumpulkan, bukan yang
	// pertama saja. Pengguna yang mengosongkan keduanya diberi tahu keduanya sekaligus.
	_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{}, caller())

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)

	fields := []string{
		validation.Violations[0].Field,
		validation.Violations[1].Field,
	}
	require.ElementsMatch(t,
		[]string{inboxrclpucl.FieldDateFrom, inboxrclpucl.FieldDateTo}, fields)
}

func TestNewReportRequestRejectsADateThatDoesNotExist(t *testing.T) {
	// `time.Parse` menolak 30 Februari; pemeriksaan berbasis pola akan meloloskannya lalu
	// menyerahkannya ke basis data sebagai galat mentah.
	_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
		From: "2026-02-30",
		To:   "2026-03-01",
	}, caller())

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxrclpucl.FieldDateFrom, validation.Violations[0].Field)
}

func TestNewReportRequestDistinguishesEmptyFromMalformed(t *testing.T) {
	// Tindakan penggunanya berbeda: yang satu menuntut ia mengisi, yang lain menuntut ia
	// memperbaiki. Pesan yang sama untuk keduanya membuat pengguna yang sudah mengisi
	// mengira isiannya tidak terkirim.
	_, emptyErr := inboxrclpucl.NewReportRequest(
		inboxrclpucl.ReportInput{To: "2026-09-30"}, caller())
	_, badErr := inboxrclpucl.NewReportRequest(
		inboxrclpucl.ReportInput{From: "30/09/2026", To: "2026-09-30"}, caller())

	var empty, bad *inboxrclpucl.ValidationError
	require.ErrorAs(t, emptyErr, &empty)
	require.ErrorAs(t, badErr, &bad)

	require.Contains(t, empty.Violations[0].Message, "wajib diisi")
	require.Contains(t, bad.Violations[0].Message, "tidak terbaca")
}

func TestNewReportRequestRejectsAReversedRange(t *testing.T) {
	_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
		From: "2026-09-30",
		To:   "2026-09-01",
	}, caller())

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxrclpucl.FieldDateTo, validation.Violations[0].Field)
}

func TestNewReportRequestDoesNotComplainAboutOrderWhenAFormatIsWrong(t *testing.T) {
	// Memeriksa urutan sebelum bentuknya terbaca akan menghasilkan pelanggaran ketiga yang
	// membingungkan — pengguna diberi tahu urutannya salah padahal yang salah bentuknya.
	_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
		From: "bukan-tanggal",
		To:   "juga-bukan",
	}, caller())

	var validation *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
}

func TestNewReportRequestRejectsTabsWithoutAReport(t *testing.T) {
	// Hanya tab "Cetak Surat" yang punya laporan rentang tanggal. Dua tab lain mengekspor
	// grid-nya sendiri, dan memintanya di sana adalah alamat yang salah — bukan isian yang
	// salah.
	for _, code := range []string{
		inboxrclpucl.TabKelengkapanDokumen,
		inboxrclpucl.TabKlaimMSIG,
	} {
		_, err := inboxrclpucl.NewReportRequest(inboxrclpucl.ReportInput{
			Tab:  code,
			From: "2026-09-01",
			To:   "2026-09-30",
		}, caller())

		require.ErrorIsf(t, err, inboxrclpucl.ErrReportNotAvailable,
			"tab %s seharusnya tidak melayani laporan", code)
	}
}

func TestDateRangeIsZeroOnlyWhenBothEndsAreEmpty(t *testing.T) {
	require.True(t, inboxrclpucl.DateRange{}.IsZero())
	require.True(t, inboxrclpucl.DateRange{From: "  ", To: ""}.IsZero())
	require.False(t, inboxrclpucl.DateRange{From: "2026-09-01"}.IsZero())
}

// ---------------------------------------------------------------------------
// Nilai penyaring
// ---------------------------------------------------------------------------

func TestFilterValuesMatchTheLegacyRules(t *testing.T) {
	// Keempatnya dibaca langsung dari Report Definition dan dari SQL hasil generate Pega.
	// Satu nilai yang salah mengosongkan seluruh layar tanpa satu pun galat, dan tidak ada
	// apa pun di antarmuka yang menandakannya.
	require.Equal(t, "RCLPUCL", inboxrclpucl.RCLPUCLWorkbasket)
	require.Equal(t, "Resolved-Completed", inboxrclpucl.WorkStatusCompleted)
	require.Equal(t, "ASM-FW-GCNMFW-Work-PNC", inboxrclpucl.WorkClassClaim)
	require.Equal(t, "0", inboxrclpucl.ExpiryStatusActive)
	require.Equal(t, "1", inboxrclpucl.PUCLApproved)
	require.Equal(t, "MSIG", inboxrclpucl.MSIGMarker)
	require.Equal(t, "002", inboxrclpucl.GroupPanelPA)
}

func TestDailyReportColumnsDifferFromTheGrid(t *testing.T) {
	// Perbedaannya bukan kelalaian: laporan punya "Status Klaim" yang tidak ada di grid,
	// dan TIDAK punya "Lama Klaim" yang ada di setiap grid. Menyamakan keduanya akan
	// membuat berkas ekspor tampak seperti salinan layar, padahal isinya berbeda.
	keys := map[string]bool{}
	for _, column := range inboxrclpucl.DailyReportColumns {
		keys[column.Key] = true
	}

	require.True(t, keys[inboxrclpucl.FieldReportClaimStatus],
		"laporan harus memuat Status Klaim")
	require.False(t, keys[inboxrclpucl.FieldClaimAge],
		"laporan TIDAK memuat Lama Klaim — kueri lama tidak mengambilnya")
	require.True(t, keys[inboxrclpucl.FieldReportSentAt])
}
