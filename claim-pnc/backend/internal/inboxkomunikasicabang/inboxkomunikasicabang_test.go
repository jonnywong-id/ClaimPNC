package inboxkomunikasicabang_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Nama uji menyebutkan ATURANNYA, bukan nama fungsinya, sehingga daftar uji terbaca sebagai
// daftar aturan bisnis yang selalu mutakhir (`14-TESTING-STRATEGY.md` §3.2).

func TestHeadOfficeIsFilteredByChannelCodeNotByBranchCode(t *testing.T) {
	// Petugas kantor pusat punya kode cabang `100081` di POOLDATA.BRANCH, tetapi kolom yang
	// disaring menyimpan `1`. Menukar keduanya menghasilkan daftar KOSONG tanpa satu pun
	// galat — kelas cacat yang sama dengan yang tercatat pada modul Inbox Laporan Klaim.
	filter := inboxkomunikasicabang.ResolveBranch(
		inboxkomunikasicabang.HeadOfficeBranch, true)

	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, filter.Code)
	require.NotEqual(t, inboxkomunikasicabang.HeadOfficeBranch, filter.Code)
	require.True(t, filter.HeadOffice)
	require.True(t, filter.Resolved)
}

func TestUnreadableBranchIsServedAsHeadOfficeButMarkedUnresolved(t *testing.T) {
	// Keputusan Work Owner 2026-09-24: replikasi apa adanya (`P-5`). Sistem lama menyatukan
	// "petugas kantor pusat" dengan "cabang tidak terbaca" lewat precondition
	// `KodeCabang == "100081" || KodeCabang == ""`.
	filter := inboxkomunikasicabang.ResolveBranch("", false)

	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, filter.Code,
		"penyaringnya harus SAMA dengan petugas kantor pusat")
	require.True(t, filter.HeadOffice)

	// Yang membedakan keduanya hanyalah penanda ini. Tanpanya, pelebaran batas data itu
	// tidak dapat dinyatakan ke pengguna maupun dicatat di log.
	require.False(t, filter.Resolved)
}

func TestBlankResolvedCodeIsTreatedAsUnresolved(t *testing.T) {
	// Adapter dapat melaporkan "ditemukan" dengan nilai kosong bila kolomnya berisi spasi
	// saja. Nilai itu tidak boleh menjadi penyaring: `COMMUNICATE_FROM = ''` tidak pernah
	// cocok, dan daftarnya akan kosong tanpa sebab yang terlihat.
	filter := inboxkomunikasicabang.ResolveBranch("   ", true)

	require.Equal(t, inboxkomunikasicabang.HeadOfficeCode, filter.Code)
	require.False(t, filter.Resolved)
}

func TestBranchOfficeKeepsItsOwnCode(t *testing.T) {
	filter := inboxkomunikasicabang.ResolveBranch("1002", true)

	require.Equal(t, "1002", filter.Code)
	require.False(t, filter.HeadOffice)
	require.True(t, filter.Resolved)
}

func TestSenderOriginKeepsBranchCodeButRecipientOriginDoesNot(t *testing.T) {
	// Asimetri yang terbaca seperti kelalaian penulis rule-nya, dan memang mungkin begitu.
	// Ia direplikasi (`P-5`): memperbaikinya berarti menampilkan kode cabang di tempat
	// pengguna hari ini membaca kata "CABANG".
	//
	// Langkah 5 hanya menimpa nilai `1`; langkah 6 dan 7 menyapu SISANYA menjadi "CABANG".
	require.Equal(t, "1002", inboxkomunikasicabang.OriginOf("1002"))
	require.Equal(t, inboxkomunikasicabang.OriginBranch,
		inboxkomunikasicabang.RecipientOf("1002"))

	// Keduanya sepakat hanya pada kantor pusat.
	require.Equal(t, inboxkomunikasicabang.OriginHeadOffice,
		inboxkomunikasicabang.OriginOf(inboxkomunikasicabang.HeadOfficeCode))
	require.Equal(t, inboxkomunikasicabang.OriginHeadOffice,
		inboxkomunikasicabang.RecipientOf(inboxkomunikasicabang.HeadOfficeCode))
}

func TestBlankRecipientCodeStillReadsAsBranch(t *testing.T) {
	// Langkah 7 berbunyi `.UserTeknisEmail != "PUSAT"`, dan nilai kosong memenuhi syarat itu
	// — sehingga kolom tujuan yang kosong pun tampil sebagai "CABANG".
	//
	// Kolom ASAL berperilaku sebaliknya: kosong tetap kosong.
	require.Equal(t, inboxkomunikasicabang.OriginBranch,
		inboxkomunikasicabang.RecipientOf(""))
	require.Equal(t, "", inboxkomunikasicabang.OriginOf(""))
}

func TestOriginTranslationIgnoresPaddingSpaces(t *testing.T) {
	// Kolom CHAR berlebar memadatkan nilainya dengan spasi tanpa memberi tanda apa pun.
	// `"1 "` dan `"1"` harus menghasilkan teks yang sama.
	require.Equal(t, inboxkomunikasicabang.OriginHeadOffice,
		inboxkomunikasicabang.OriginOf(" 1 "))
	require.Equal(t, inboxkomunikasicabang.OriginHeadOffice,
		inboxkomunikasicabang.RecipientOf(" 1 "))
}

func TestClosedChannelDiffersFromOpenChannelDespiteSharingAPrefix(t *testing.T) {
	// `CABANG SELESAI` BERAWALAN `CABANG`. Penyaring yang ditulis sebagai awalan alih-alih
	// perbandingan persis akan meloloskan percakapan yang justru sudah ditutup.
	require.NotEqual(t, inboxkomunikasicabang.CaseOpen, inboxkomunikasicabang.CaseClosed)
	require.Contains(t, inboxkomunikasicabang.CaseClosed, inboxkomunikasicabang.CaseOpen)
}

func TestBothTabsExistAndOnlyOneIsAnswered(t *testing.T) {
	tabs := inboxkomunikasicabang.Tabs()
	require.Len(t, tabs, 2)

	require.Equal(t, inboxkomunikasicabang.TabNotAnswered, tabs[0].Code)
	require.False(t, tabs[0].Answered)

	require.Equal(t, inboxkomunikasicabang.TabAnswered, tabs[1].Code)
	require.True(t, tabs[1].Answered)
}

func TestDefaultTabIsTheOneHoldingWaitingWork(t *testing.T) {
	tab, found := inboxkomunikasicabang.FindTab(inboxkomunikasicabang.DefaultTab)

	require.True(t, found)
	require.False(t, tab.Answered,
		"tab bawaan harus yang BELUM dijawab — itulah pekerjaan yang menunggu")
}

func TestNotAnsweredTabDrawsFewerDataColumnsThanAnsweredTab(t *testing.T) {
	// Kedua grid section memang berbeda jumlah kolom ISIAN-nya: tiga dan lima. Kolom balasan
	// dan penjawab tidak digambar pada tab pertama karena penyaringnya menjamin keduanya
	// kosong.
	notAnswered, _ := inboxkomunikasicabang.FindTab(inboxkomunikasicabang.TabNotAnswered)
	answered, _ := inboxkomunikasicabang.FindTab(inboxkomunikasicabang.TabAnswered)

	require.Len(t, dataColumns(notAnswered), 3)
	require.Len(t, dataColumns(answered), 5)

	for _, column := range notAnswered.Columns {
		require.NotEqual(t, inboxkomunikasicabang.FieldReply, column.Key)
		require.NotEqual(t, inboxkomunikasicabang.FieldReplier, column.Key)
	}
}

func TestBothTabsDrawTheTwoActionButtonColumns(t *testing.T) {
	// Keduanya ada di KEDUA grid layar lama — terbaca dari offsetnya sendiri, dua kali
	// masing-masing. Versi pertama modul ini melewatkan keduanya dan menggantinya dengan
	// tautan pada sel "Pesan", yang di Pega tidak ada sama sekali.
	for _, tab := range inboxkomunikasicabang.Tabs() {
		keys := []string{}
		for _, column := range tab.Columns {
			if inboxkomunikasicabang.IsAction(column.Key) {
				keys = append(keys, column.Key)
			}
		}

		require.Equalf(t,
			[]string{
				inboxkomunikasicabang.FieldActionDetail,
				inboxkomunikasicabang.FieldActionFinish,
			},
			keys,
			"tab %q harus menggambar kedua kolom tombol, berurutan", tab.Name)
	}
}

func TestActionColumnsKeepTheLiteralPegaHeading(t *testing.T) {
	// `pyCaption Button` pada keempat kolomnya. Dua kolom berjudul sama memang tidak
	// membantu, tetapi `D-13` menetapkan teks layar mengikuti Pega apa adanya — yang
	// ditambahkan adalah nama yang dibaca pembaca layar, bukan judul kolomnya.
	tab, _ := inboxkomunikasicabang.FindTab(inboxkomunikasicabang.TabNotAnswered)

	for _, column := range tab.Columns {
		if inboxkomunikasicabang.IsAction(column.Key) {
			require.Equal(t, "Button", column.Title)
		}
	}
}

// dataColumns menyaring kolom yang benar-benar menggambar isian baris.
func dataColumns(tab inboxkomunikasicabang.Tab) []inboxkomunikasicabang.Column {
	result := []inboxkomunikasicabang.Column{}
	for _, column := range tab.Columns {
		if !inboxkomunikasicabang.IsAction(column.Key) {
			result = append(result, column)
		}
	}
	return result
}

func TestTabsCannotBeMutatedThroughTheReturnedSlice(t *testing.T) {
	first := inboxkomunikasicabang.Tabs()
	first[0].Name = "diubah"

	require.NotEqual(t, "diubah", inboxkomunikasicabang.Tabs()[0].Name)
}

func TestEveryGridDataColumnIsAlsoAnExportColumn(t *testing.T) {
	// Berkas ekspor memuat SELURUH isian, termasuk tiga yang tidak digambar tabel mana pun.
	// Yang tidak boleh terjadi adalah sebaliknya: kolom yang tampil di layar tetapi hilang
	// dari berkas, karena orang yang mencocokkan keduanya akan mengira ada baris yang
	// tergeser.
	exported := map[string]bool{}
	for _, column := range inboxkomunikasicabang.ExportColumns {
		exported[column.Key] = true
	}

	for _, tab := range inboxkomunikasicabang.Tabs() {
		for _, column := range dataColumns(tab) {
			require.True(t, exported[column.Key],
				"kolom %q pada tab %q tidak ada di berkas ekspor", column.Key, tab.Name)
		}
	}
}

func TestActionColumnsAreNeverExported(t *testing.T) {
	// Kolom tombol tidak punya nilai yang dapat ditulis ke berkas. Menuliskannya sebagai sel
	// kosong akan menggeser seluruh kolom sesudahnya — dan itu tidak menghasilkan satu pun
	// galat, hanya berkas yang isinya tidak sejajar dengan judulnya.
	for _, column := range inboxkomunikasicabang.ExportColumns {
		require.Falsef(t, inboxkomunikasicabang.IsAction(column.Key),
			"kolom tombol %q tidak boleh ikut ke berkas ekspor", column.Key)
	}
}

func TestPaginationDefaultsToTheOldScreenPageSize(t *testing.T) {
	// `<pyPageSize>20</pyPageSize>` pada Section/InboxKomunikasi-Section.xml, muncul tiga
	// kali dengan nilai yang sama. Angkanya BERBEDA dari modul inbox lain, dan perbedaan itu
	// disengaja — ukuran halaman menentukan baris mana yang terlihat tanpa menggulir.
	require.Equal(t, 20, inboxkomunikasicabang.DefaultPageSize)

	clean := inboxkomunikasicabang.Pagination{}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, 20, clean.Size)
}

func TestOversizedPageRequestIsCappedRatherThanHonoured(t *testing.T) {
	// `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan dipenuhi diam-diam
	// — memenuhinya membuat batas menjadi saran, bukan batas.
	clean := inboxkomunikasicabang.Pagination{Page: 3, Size: 5_000}.Normalize()

	require.Equal(t, inboxkomunikasicabang.MaxPageSize, clean.Size)
	require.Equal(t, 3, clean.Page)
}

func TestTotalPagesIsNeverZeroSoTheScreenNeverSaysPageOneOfZero(t *testing.T) {
	empty := inboxkomunikasicabang.Page{
		Pagination: inboxkomunikasicabang.Pagination{}.Normalize(),
	}
	require.Equal(t, 1, empty.TotalPages())
}

func TestSummaryTotalIsSentRatherThanInferredBecauseCountersDoNotComplementEachOther(t *testing.T) {
	// Kedua pencacah BUKAN saling melengkapi: percakapan yang salah satu kolom balasannya
	// terisi sendirian tidak terhitung di mana pun. Jumlah keduanya karena itu dapat KURANG
	// dari jumlah seluruh percakapan, dan itu memang yang dinyatakan.
	summary := inboxkomunikasicabang.Summary{NotAnswered: 3, Answered: 4}
	require.Equal(t, 7, summary.Total())
}

func TestAttachmentWithoutUploadDateReadsAsNotYetUploaded(t *testing.T) {
	// Sistem lama menyimpulkannya dari `uploaddate is null`. Kueri lamanya sendiri cacat —
	// cabang pertama `CASE`-nya membandingkan `COUNT(...)` dengan nol secara terbalik,
	// sehingga jawabannya SELALU "Belum Upload". Yang dibawa adalah aturannya, bukan cacatnya.
	require.False(t, inboxkomunikasicabang.Attachment{}.Uploaded())
	require.False(t, inboxkomunikasicabang.Attachment{UploadedAt: "  "}.Uploaded())
	require.True(t,
		inboxkomunikasicabang.Attachment{UploadedAt: "2026-09-03 10:07"}.Uploaded())
}

func TestQueryIsRefusedWhenTheCallerHasNoLogin(t *testing.T) {
	// Batas data layar ini DITURUNKAN dari login. Tanpa login tidak ada batas yang dapat
	// dipakai, dan satu-satunya jalan yang tersisa adalah menampilkan percakapan siapa saja.
	_, err := inboxkomunikasicabang.NewQuery(
		inboxkomunikasicabang.QueryInput{},
		inboxkomunikasicabang.BranchFilter{Code: "1001"},
		inboxkomunikasicabang.Caller{Login: "   "},
	)

	require.ErrorIs(t, err, inboxkomunikasicabang.ErrCallerUnknown)
}

func TestEmptyTabFallsBackToTheDefaultRatherThanBeingRejected(t *testing.T) {
	query, err := inboxkomunikasicabang.NewQuery(
		inboxkomunikasicabang.QueryInput{},
		inboxkomunikasicabang.BranchFilter{Code: "1001"},
		inboxkomunikasicabang.Caller{Login: "pictekniks"},
	)

	require.NoError(t, err)
	require.Equal(t, inboxkomunikasicabang.DefaultTab, query.Tab.Code)
}

func TestUnknownTabIsRejectedWithAFieldLevelViolation(t *testing.T) {
	_, err := inboxkomunikasicabang.NewQuery(
		inboxkomunikasicabang.QueryInput{Tab: "99"},
		inboxkomunikasicabang.BranchFilter{Code: "1001"},
		inboxkomunikasicabang.Caller{Login: "pictekniks"},
	)

	var validation *inboxkomunikasicabang.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxkomunikasicabang.FieldTab, validation.Violations[0].Field)
}

func TestDetailRequestRefusesABlankConversationNumber(t *testing.T) {
	_, err := inboxkomunikasicabang.NewDetailRequest(
		inboxkomunikasicabang.DetailInput{ID: "  "},
		inboxkomunikasicabang.Caller{Login: "pictekniks"},
	)

	var validation *inboxkomunikasicabang.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxkomunikasicabang.FieldConversation, validation.Violations[0].Field)
}

func TestSliceNeverReturnsANilItemSliceSoJSONDrawsAnEmptyArray(t *testing.T) {
	// Senarai nil menjadi `null` di JSON, dan layar yang memetakannya akan gagal alih-alih
	// menggambar tabel kosong.
	page := inboxkomunikasicabang.Slice(
		[]inboxkomunikasicabang.Conversation{},
		inboxkomunikasicabang.Pagination{Page: 9},
	)

	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Equal(t, 0, page.Total)
}

func TestPlannedDifferencesNameTheBranchWideningDecision(t *testing.T) {
	// Pelebaran batas data pada petugas yang cabangnya tidak terbaca WAJIB dinyatakan ke
	// pengguna — ia tidak menghasilkan satu pun galat, dan tanpa pernyataan ini tidak ada
	// apa pun di layar yang menandainya.
	require.NotEmpty(t, inboxkomunikasicabang.PlannedDifferences)

	found := false
	for _, difference := range inboxkomunikasicabang.PlannedDifferences {
		if strings.Contains(difference, "KANTOR PUSAT") &&
			strings.Contains(difference, "TIDAK DAPAT DITURUNKAN") {
			found = true
		}
	}
	require.True(t, found,
		"selisih terencana harus menyebut petugas yang cabangnya tidak terbaca")
}

func TestPlannedDifferencesNameTheWriteActionsThatStayInPega(t *testing.T) {
	// Keempat tombol tetap digambar, dan penggunanya harus tahu MENGAPA ia tidak bekerja.
	// Tombol yang diam tanpa keterangan dilaporkan sebagai kerusakan.
	found := false
	for _, difference := range inboxkomunikasicabang.PlannedDifferences {
		if strings.Contains(difference, "Kirim Pesan") &&
			strings.Contains(difference, "Selesai Komunikasi") {
			found = true
		}
	}
	require.True(t, found, "selisih terencana harus menyebut aksi tulis yang belum tersedia")
}
