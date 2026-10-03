package inboxpladlapredla_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
)

func caller() inboxpladlapredla.Caller {
	return inboxpladlapredla.Caller{Login: "JONNY"}
}

func date(year int, month time.Month, day int) *time.Time {
	moment := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &moment
}

// Ketiga tab layar lama ada, dengan kode yang dipakai alamat.
func TestTheScreenOffersExactlyThreeLists(t *testing.T) {
	tabs := inboxpladlapredla.Tabs()
	require.Len(t, tabs, 3)

	codes := []string{}
	for _, tab := range tabs {
		codes = append(codes, tab.Code)
	}
	require.Equal(t, []string{"pla", "dla", "pre-dla"}, codes)
}

// Setiap tab punya kolom, judul pencarian, dan judul rentang tanggalnya sendiri.
//
// Judul rentang tanggal BERBEDA per tab meski kotaknya sama-sama berjudul "Dari"/"Sampai"
// di Pega: yang disaring adalah tanggal PLA, tanggal DLA, atau tanggal Pre-DLA.
func TestEveryListDescribesItsOwnColumnsAndFilters(t *testing.T) {
	seen := map[string]bool{}

	for _, tab := range inboxpladlapredla.Tabs() {
		require.NotEmpty(t, tab.Name, "%s tanpa judul", tab.Code)
		require.NotEmpty(t, tab.Description, "%s tanpa keterangan", tab.Code)
		require.Len(t, tab.Columns, 7, "%s kolomnya bergeser", tab.Code)
		require.Equal(t, "No Klaim", tab.SearchLabel, "%s", tab.Code)

		require.False(t, seen[tab.DateLabel],
			"judul rentang tanggal %q dipakai dua tab", tab.DateLabel)
		seen[tab.DateLabel] = true
	}
}

// Hanya tab PLA dan DLA yang menggambar grid rincian.
//
// `Section/PNCInboxPreDLA_sect-Section.xml` hanya memuat SATU grid, sementara kedua
// section lain memuat dua. Uji ini menjaga temuan itu tetap tercermin di kode.
func TestOnlyPLAAndDLAHaveADocumentGrid(t *testing.T) {
	withGrid := map[string]bool{}
	for _, tab := range inboxpladlapredla.Tabs() {
		withGrid[tab.Code] = tab.HasDocuments()
	}

	require.True(t, withGrid["pla"])
	require.True(t, withGrid["dla"])
	require.False(t, withGrid["pre-dla"],
		"tab Pre DLA tidak punya grid rincian di Pega")
}

// Kolom `REVISI` hanya digambar PLA, dan `NOAKSEP` hanya digambar DLA.
//
// Keduanya berasal dari kueri yang berbeda: `GetDLAList` tidak mengambil `REVISI` sama
// sekali, dan `GetPLAList` tidak mengambil `NOAKSEP`.
func TestRevisionBelongsToPLAAndAcceptanceNumberToDLA(t *testing.T) {
	keys := func(code string) map[string]bool {
		tab, found := inboxpladlapredla.FindTab(code)
		require.True(t, found)

		out := map[string]bool{}
		for _, column := range tab.DocumentColumns {
			out[column.Key] = true
		}
		return out
	}

	pla, dla := keys("pla"), keys("dla")

	require.True(t, pla[inboxpladlapredla.FieldRevision])
	require.False(t, pla[inboxpladlapredla.FieldAcceptanceNo])

	require.True(t, dla[inboxpladlapredla.FieldAcceptanceNo])
	require.False(t, dla[inboxpladlapredla.FieldRevision])
}

// Penyaring keanggotaan tab Pre DLA BUKAN penanda terkirim.
//
// Ini perbedaan yang paling mudah "diseragamkan" menjadi salah; uji ini yang akan gagal
// bila seseorang menyamakan ketiganya.
func TestPreDLAIsFilteredByAcceptanceNumberNotBySentMarker(t *testing.T) {
	pre, found := inboxpladlapredla.FindTab("pre-dla")
	require.True(t, found)

	require.False(t, pre.SentFilterApplies,
		"Pre DLA tidak menyaring ISKIRIM")
	require.True(t, pre.RequiresNoAcceptance,
		"Pre DLA disaring oleh Nomor Akseptasi")

	for _, code := range []string{"pla", "dla"} {
		tab, ok := inboxpladlapredla.FindTab(code)
		require.True(t, ok)
		require.True(t, tab.SentFilterApplies, "%s", code)
		require.False(t, tab.RequiresNoAcceptance, "%s", code)
	}
}

// Dua pengecualian tambahan hanya berlaku di tab DLA.
func TestBranchAndBusinessGroupExclusionsBelongToDLAOnly(t *testing.T) {
	for _, tab := range inboxpladlapredla.Tabs() {
		if tab.Code == "dla" {
			require.True(t, tab.ExcludeASNET)
			require.Equal(t, "10008", tab.ExcludeBusinessGroup)
			continue
		}
		require.False(t, tab.ExcludeASNET, "%s", tab.Code)
		require.Empty(t, tab.ExcludeBusinessGroup, "%s", tab.Code)
	}
}

// Kode daftar dicocokkan tanpa memedulikan huruf besar-kecil dan spasi di ujung.
//
// Ia datang dari alamat, dan `?daftar=PLA` maupun `?daftar=pla ` adalah permintaan yang
// sama.
func TestListCodeIsMatchedLoosely(t *testing.T) {
	for _, code := range []string{"pla", "PLA", " Pla "} {
		tab, found := inboxpladlapredla.FindTab(code)
		require.True(t, found, "%q", code)
		require.Equal(t, "pla", tab.Code)
	}

	_, found := inboxpladlapredla.FindTab("tidak-ada")
	require.False(t, found)
}

// Permintaan tanpa daftar jatuh ke daftar bawaan, bukan ditolak.
func TestAnEmptyListCodeFallsBackToTheDefault(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{}, caller())

	require.NoError(t, err)
	require.Equal(t, inboxpladlapredla.DefaultTab, query.Tab.Code)
}

// Daftar yang tidak dikenal DITOLAK, bukan dijatuhkan ke bawaan.
//
// Menjatuhkannya akan menampilkan antrean yang BUKAN yang diminta, dan pengguna tidak
// punya cara mengetahuinya.
func TestAnUnknownListIsRejected(t *testing.T) {
	_, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Tab: "salvage"}, caller())

	var validation *inboxpladlapredla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxpladlapredla.FieldTab, validation.Violations[0].Field)
}

// Permintaan tanpa identitas pemanggil ditolak.
//
// Identitas tidak menyaring apa pun di layar ini; yang membutuhkannya adalah jejak. Setiap
// baris memuat nama tertanggung dan nomor polis, dan `D-59` menjadikan jejak audit
// satu-satunya kontrol pengimbang selama pemeriksaan peran belum ada.
func TestAQueryWithoutACallerIsRejected(t *testing.T) {
	_, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{}, inboxpladlapredla.Caller{Login: "   "})

	require.ErrorIs(t, err, inboxpladlapredla.ErrCallerUnknown)
}

// Batas atas rentang digeser satu hari dan menjadi EKSKLUSIF.
//
// Itu yang membuat kuerinya tidak perlu `TRUNC` pada kolom tanggal. Dokumen yang terbit
// pukul berapa pun pada tanggal akhir tetap ikut.
func TestTheUpperBoundBecomesExclusiveByAddingOneDay(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		From: date(2026, time.January, 1),
		To:   date(2026, time.January, 31),
	}, caller())
	require.NoError(t, err)

	require.Equal(t, *date(2026, time.January, 1), *query.From)
	require.Equal(t, *date(2026, time.February, 1), *query.To,
		"batas atas belum digeser menjadi eksklusif")

	// Tepat pada tengah malam tanggal akhir masih masuk; tengah malam sehari sesudahnya
	// sudah tidak.
	require.True(t, query.WithinRange(*date(2026, time.January, 31)))
	require.False(t, query.WithinRange(*date(2026, time.February, 1)))
	require.False(t, query.WithinRange(*date(2025, time.December, 31)))
}

// Rentang satu hari — "Dari" sama dengan "Sampai" — tetap memilih hari itu.
//
// Ia diuji tersendiri karena inilah kasus yang paling mudah rusak saat penggeseran satu
// hari dipindahkan ke tempat yang salah.
func TestASingleDayRangeStillSelectsThatDay(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		From: date(2026, time.March, 10),
		To:   date(2026, time.March, 10),
	}, caller())
	require.NoError(t, err)

	require.True(t, query.WithinRange(*date(2026, time.March, 10)))
	require.False(t, query.WithinRange(*date(2026, time.March, 9)))
	require.False(t, query.WithinRange(*date(2026, time.March, 11)))
}

// Rentang terbalik DITOLAK, bukan ditukar diam-diam.
//
// Menukarnya akan menampilkan hasil yang benar untuk pertanyaan yang TIDAK diajukan.
func TestAReversedRangeIsRejected(t *testing.T) {
	_, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		From: date(2026, time.March, 10),
		To:   date(2026, time.March, 1),
	}, caller())

	var validation *inboxpladlapredla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxpladlapredla.FieldTo, validation.Violations[0].Field)
}

// Rentang yang hanya berbatas satu sisi DITERIMA.
//
// Pega tidak mengenal keadaan ini — isian kosong di sana dirangkai menjadi potongan
// berikut lalu ditolak Oracle:
//
//	to_date('','dd/mm/yyyy')
//
// Di sini ia sah, dan itu selisih yang
// sudah dinyatakan.
func TestAHalfOpenRangeIsAccepted(t *testing.T) {
	onlyFrom, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		From: date(2026, time.March, 1),
	}, caller())
	require.NoError(t, err)
	require.NotNil(t, onlyFrom.From)
	require.Nil(t, onlyFrom.To)
	require.True(t, onlyFrom.WithinRange(*date(2030, time.January, 1)))

	onlyTo, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		To: date(2026, time.March, 1),
	}, caller())
	require.NoError(t, err)
	require.Nil(t, onlyTo.From)
	require.NotNil(t, onlyTo.To)
	require.True(t, onlyTo.WithinRange(*date(2000, time.January, 1)))
}

// Tanpa rentang, tanggal apa pun lolos.
func TestWithoutARangeEveryDatePasses(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{}, caller())
	require.NoError(t, err)

	require.True(t, query.WithinRange(*date(1990, time.January, 1)))
	require.True(t, query.WithinRange(*date(2099, time.December, 31)))
}

// Jam pada isian tanggal DIBUANG.
//
// Kotak tanggal di layar tidak punya jam, tetapi API-nya menerima nilai apa pun. Jam yang
// tertinggal akan membuat rentang "1 Januari" melewatkan dokumen yang terbit pagi hari.
func TestTheTimeOfDayIsStrippedFromRangeBounds(t *testing.T) {
	noon := time.Date(2026, time.January, 1, 12, 30, 0, 0, time.UTC)

	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{From: &noon}, caller())
	require.NoError(t, err)

	require.Equal(t, *date(2026, time.January, 1), *query.From)
	require.True(t, query.WithinRange(*date(2026, time.January, 1)),
		"dokumen tengah malam pada hari itu harus ikut")
}

// Pencarian mencocokkan KUNCI KLAIM, bukan nomor klaim — dan tidak peka huruf.
//
// Itu perilaku Pega: `AND B.CLAIMID LIKE '%…%'`. Ia tetap menemukan nomor klaim yang
// diketik pengguna karena kuncinya berbentuk `ASM-FW-GCNMFW-WORK <nomor klaim>`.
func TestSearchMatchesTheWorkKeyCaseInsensitively(t *testing.T) {
	query, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Search: "pnc-1001"}, caller())
	require.NoError(t, err)

	row := inboxpladlapredla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-1001",
		ClaimNo:  "PNC-1001",
	}
	require.True(t, query.Matches(row))

	other := inboxpladlapredla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-2002",
		ClaimNo:  "PNC-2002",
	}
	require.False(t, query.Matches(other))
}

// Kata kunci yang terlalu panjang ditolak sebelum sampai ke basis data.
func TestAnOverlongSearchKeywordIsRejected(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'A'
	}

	_, err := inboxpladlapredla.NewQuery(
		inboxpladlapredla.QueryInput{Search: string(long)}, caller())

	var validation *inboxpladlapredla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxpladlapredla.FieldSearch, validation.Violations[0].Field)
}

// SELURUH pelanggaran disampaikan sekaligus, bukan yang pertama saja (`P-5`).
func TestEveryViolationIsReportedAtOnce(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'A'
	}

	_, err := inboxpladlapredla.NewQuery(inboxpladlapredla.QueryInput{
		Search: string(long),
		From:   date(2026, time.March, 10),
		To:     date(2026, time.March, 1),
	}, caller())

	var validation *inboxpladlapredla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violations, 2,
		"kata kunci terlalu panjang DAN rentang terbalik harus disampaikan bersamaan")
}

// Paginasi di luar rentang DIBETULKAN, bukan ditolak.
func TestPaginationIsCorrectedRatherThanRejected(t *testing.T) {
	require.Equal(t,
		inboxpladlapredla.Pagination{
			Page: 1, Size: inboxpladlapredla.DefaultPageSize,
		},
		inboxpladlapredla.Pagination{Page: 0, Size: 0}.Normalize())

	require.Equal(t,
		inboxpladlapredla.Pagination{
			Page: 3, Size: inboxpladlapredla.MaxPageSize,
		},
		inboxpladlapredla.Pagination{Page: 3, Size: 5000}.Normalize())
}

// Ukuran halaman bawaan adalah 10 — angka layar ini sendiri.
//
// `<pyRDLPageSize>10</pyRDLPageSize>` pada ketiga section tabnya. Modul inbox lain memakai
// 20 dan 50; perbedaannya tidak diseragamkan (`D-13`).
func TestTheDefaultPageSizeFollowsThisScreenNotTheOthers(t *testing.T) {
	require.Equal(t, 10, inboxpladlapredla.DefaultPageSize)
}

// Halaman kosong tetap menyatakan satu halaman, bukan nol.
func TestAnEmptyPageStillReportsOnePage(t *testing.T) {
	page := inboxpladlapredla.Page{
		Pagination: inboxpladlapredla.Pagination{Page: 1, Size: 10},
	}
	require.Equal(t, 1, page.TotalPages())
}

// Selisih terencana DIKIRIM ke layar, dan menyebutkan ketiga tombol yang belum ada.
//
// Selisih yang hanya tercatat di komentar akan dilaporkan berulang kali sebagai kerusakan
// oleh orang yang membandingkan layar baru dengan Pega berdampingan.
func TestPlannedDifferencesNameEveryMissingButton(t *testing.T) {
	joined := ""
	for _, item := range inboxpladlapredla.PlannedDifferences {
		joined += item + "\n"
	}

	require.Contains(t, joined, "Send")
	require.Contains(t, joined, "Upload File Penunjang")
	require.Contains(t, joined, "Print Pre DLA")
	require.Contains(t, joined, "Pre DLA")
}

// Setiap tombol yang belum dibangun menjawab alasan yang BERBEDA.
//
// Menjawabnya dengan satu kalimat yang sama akan membuat pengguna yang menekan
// "Kirim Pre DLA" membaca penjelasan tentang email reasuradur — dan menyimpulkan bahwa
// tombol yang ia tekan bukan tombol yang ia kira.
func TestEachUnbuiltButtonAnswersItsOwnReason(t *testing.T) {
	reasons := map[inboxpladlapredla.Action]string{}

	for _, action := range []inboxpladlapredla.Action{
		inboxpladlapredla.ActionSend,
		inboxpladlapredla.ActionUpload,
		inboxpladlapredla.ActionDownloadAttachment,
	} {
		rejected := inboxpladlapredla.NewNotAvailable(string(action))
		require.Equal(t, action, rejected.Action)

		reason := rejected.Reason()
		require.NotEmpty(t, reason, "%s tanpa alasan", action)

		for other, sama := range reasons {
			require.NotEqual(t, sama, reason,
				"%s dan %s menjawab kalimat yang sama", action, other)
		}
		reasons[action] = reason
	}

	// Masing-masing menyebut APA yang belum ada, bukan sekadar "belum tersedia".
	require.Contains(t, reasons[inboxpladlapredla.ActionSend], "email")
	require.Contains(t, reasons[inboxpladlapredla.ActionUpload], "penyimpanan dokumen")
	// Alasan unduh lampiran DIKOREKSI setelah `Activity/PNCGetListPreDla-Act.xml` dibaca:
	// berkasnya diambil `Obj-Open-By-Handle` dari tabel lampiran Pega, BUKAN dari
	// penyimpanan dokumen eksternal (`D-16`). Ia karena itu bukan kemampuan yang belum
	// tersedia, melainkan pekerjaan yang belum diminta — dan kalimatnya harus
	// mengatakannya apa adanya.
	require.Contains(t,
		reasons[inboxpladlapredla.ActionDownloadAttachment], "tabel lampiran Pega")
	require.NotContains(t,
		reasons[inboxpladlapredla.ActionDownloadAttachment], "D-16")

	// Seluruhnya memberi tahu apa yang dapat dilakukan pengguna HARI INI.
	for action, reason := range reasons {
		require.Contains(t, reason, "lewat Pega", "%s", action)
	}
}

// Tidak ada daftar yang punya kolom aksi "Rincian".
//
// Rincian PLA dan DLA dibuka dengan MENGKLIK NOMOR KLAIM, persis seperti di Pega — sel
// `.BRANCH_CODE` di sana membawa `pyAction = refresh`. Satu-satunya tombol aksi per baris
// yang tersisa adalah "Print Pre DLA".
//
// Uji ini ada karena kolom aksi mudah dikembalikan tanpa sadar: ia pernah ada, dan
// menambahkannya kembali tidak merusak apa pun yang terlihat.
func TestDetailIsOpenedByClickingTheClaimNumber(t *testing.T) {
	berTombol := map[string]string{}

	for _, tab := range inboxpladlapredla.Tabs() {
		if tab.RowActionLabel != "" {
			berTombol[tab.Code] = tab.RowActionLabel
		}

		if tab.HasDocuments() {
			require.Empty(t, tab.RowActionLabel,
				"daftar %q punya grid rincian, jadi nomor klaimnya yang membukanya "+
					"— bukan tombol", tab.Code)
		}
	}

	require.Equal(t, map[string]string{"pre-dla": "Print Pre DLA"}, berTombol)
}

// "Kirim Pre DLA" BUKAN lagi tindakan yang ditolak.
//
// Tombolnya kini benar-benar menandai Pre-DLA terkirim. Bila namanya kembali masuk daftar
// penolakan, pengguna akan menekan tombol yang bekerja lalu membaca alasan bahwa ia belum
// bekerja — dua pesan yang saling menyangkal dalam satu layar.
func TestSendingAPreDLAIsNotARejectedAction(t *testing.T) {
	rejected := inboxpladlapredla.NewNotAvailable("kirim-pre-dla")

	require.NotEqual(t, inboxpladlapredla.Action("kirim-pre-dla"), rejected.Action,
		"menandai Pre-DLA terkirim masih terdaftar sebagai tindakan yang ditolak")
}

// "Print Pre DLA" BUKAN lagi tindakan yang ditolak.
//
// Tombolnya kini membuka panel yang sudah dibangun. Bila namanya kembali masuk daftar
// penolakan, pengguna akan menekan tombol yang bekerja lalu membaca alasan bahwa ia
// belum bekerja — dua pesan yang saling menyangkal dalam satu layar.
//
// Yang ditolak adalah tombol DI DALAM panelnya, dan keduanya punya nama sendiri.
func TestOpeningThePrintPanelIsNotARejectedAction(t *testing.T) {
	rejected := inboxpladlapredla.NewNotAvailable("cetak-pre-dla")

	require.NotEqual(t, inboxpladlapredla.Action("cetak-pre-dla"), rejected.Action,
		"membuka panel Print Pre DLA masih terdaftar sebagai tindakan yang ditolak")
}

// Tindakan yang tidak dikenal tetap menghasilkan penolakan, bukan galat lain.
//
// Yang dituju pengguna memang tombol yang belum dibangun; nama tindakan yang salah ketik
// di alamat bukan sesuatu yang perlu dibedakan di layar.
func TestAnUnknownActionStillAnswersAsNotAvailable(t *testing.T) {
	rejected := inboxpladlapredla.NewNotAvailable("tidak-dikenal")

	require.Empty(t, rejected.Action)
	require.NotEmpty(t, rejected.Reason())
	require.ErrorIs(t, rejected, inboxpladlapredla.ErrWriteNotAvailable,
		"pemanggil yang hanya memeriksa sentinelnya tidak boleh ikut berubah")
}
