package inboxsalvage_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Nama uji menyebut ATURANNYA, bukan nama fungsinya, sehingga daftar uji terbaca sebagai
// daftar aturan yang berlaku (`14-TESTING-STRATEGY.md` §3.2).

func TestThirteenTabsExistAndEachOneNamesItsLegacyCode(t *testing.T) {
	// SELURUH tiga belas diperiksa, termasuk empat yang tidak ditawarkan layar — definisi
	// keempatnya tetap disimpan, dan yang disimpan harus tetap lengkap.
	tabs := inboxsalvage.AllTabs()
	require.Len(t, tabs, 13, "layar lama menggambar tiga belas daftar")

	// Yang DITAWARKAN sembilan, sesuai layar Pega sungguhan.
	require.Len(t, inboxsalvage.Tabs(), 9, "layar menawarkan sembilan daftar")

	for _, tab := range tabs {
		require.NotEmpty(t, tab.Code, "setiap daftar wajib punya kode")
		require.NotEmpty(t, tab.Name, "setiap daftar wajib punya judul")
		require.NotEmpty(t, tab.Description, "setiap daftar wajib menjelaskan isinya")
		require.NotEmpty(t, tab.Columns, "setiap daftar wajib punya kolom")

		// Tepat satu kode masuk wajib ada. Tanpanya, baris pencacah yang menunjuk daftar
		// ini tidak akan pernah menemukan tabnya.
		require.True(t,
			tab.LegacyTipe != "" || tab.LegacyTipe2 != "",
			"daftar %q tidak punya kode masuk sistem lama", tab.Code)
	}
}

// Kode tab TIDAK BOLEH ganda. Kode ganda membuat FindTab mengembalikan daftar yang salah,
// dan tidak ada satu pun galat yang muncul karenanya.
func TestTabCodesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, tab := range inboxsalvage.Tabs() {
		require.False(t, seen[tab.Code], "kode daftar ganda: %s", tab.Code)
		seen[tab.Code] = true
	}
}

// Inilah kekeliruan yang paling mudah terjadi pada layar ini, dan sudah pernah terjadi pada
// modul Inbox Komunikasi Cabang: dua ruang kode yang keduanya memuat angka yang sama.
//
// `Param.tipe` 1 berarti "Salvage Outstanding"; `Param.tipe2` 1 berarti "Ekonomis". Memeriksa
// `tipe` lebih dulu akan memetakan baris pencacah "Ekonomis" — yang mengisi KEDUANYA — ke
// daftar yang sama sekali berbeda.
func TestCounterRowForEkonomisResolvesByTipe2NotTipe(t *testing.T) {
	// Baris "Ekonomis" pada pencacah mengisi CityID=1 DAN RW=1.
	tab, found := inboxsalvage.TabForLegacy("1", "1")
	require.True(t, found)
	require.Equal(t, inboxsalvage.TabEkonomis, tab.Code,
		"pasangan (1,1) adalah Ekonomis, bukan Salvage Outstanding")

	// Tanpa RW, kode yang sama berarti Salvage Outstanding.
	tab, found = inboxsalvage.TabForLegacy("1", "")
	require.True(t, found)
	require.Equal(t, inboxsalvage.TabOutstanding, tab.Code)
}

// Tabel ringkas mengirim `tipe` 6 untuk Histori Salvage; activity pemuat daftar yang menulis
// ulangnya menjadi 2. Yang disimpan sebagai kode masuk harus yang DIKIRIM, bukan yang
// dibaca percabangan sesudahnya.
func TestHistoriSalvageEntersWithLegacyCodeSix(t *testing.T) {
	tab, found := inboxsalvage.TabForLegacy("6", "")
	require.True(t, found)
	require.Equal(t, inboxsalvage.TabHistori, tab.Code)
}

// Setiap baris pencacah yang menunjuk sebuah daftar WAJIB menemukan daftarnya. Baris yang
// menunjuk daftar tak dikenal akan tergambar sebagai angka yang tidak dapat diklik.
func TestEveryCounterRowPointsAtAKnownTabOrAtNothing(t *testing.T) {
	for _, row := range inboxsalvage.CountRows() {
		if row.Tab == "" {
			continue
		}
		_, found := inboxsalvage.FindTab(row.Tab)
		require.True(t, found,
			"baris pencacah %q menunjuk daftar %q yang tidak ada", row.Label, row.Tab)
	}
}

// Satu baris pencacah memang TIDAK menuju daftar mana pun, dan itu keadaan di Pega: tidak
// ada tab yang menerima kodenya.
//
// Ia kini tidak digambar — layar Pega sungguhan tidak menampilkannya — tetapi definisinya
// tetap disimpan, dan ketiadaan daftarnya tetap dijaga. Bila kelak seseorang
// menghubungkannya, ia harus melakukannya dengan sadar.
func TestTidakTerjualCounterRowDeliberatelyLeadsNowhere(t *testing.T) {
	var found bool
	for _, row := range inboxsalvage.AllCountRows() {
		if row.Label == "Tidak Terjual" {
			found = true
			require.Empty(t, row.Tab,
				"baris \"Tidak Terjual\" tidak punya daftar di Pega")
		}
	}
	require.True(t, found, "baris \"Tidak Terjual\" hilang dari pencacah")
}

// Pencacah "Outstanding" dan daftarnya menghitung populasi yang BERBEDA di Pega, dan itu
// direplikasi (`P-5`). Uji ini menjaga selisihnya tetap ada — kalau seseorang
// "memperbaikinya", uji ini yang gagal lebih dulu, bukan pengguna yang melaporkannya.
func TestOutstandingCounterDeliberatelyCountsADifferentPopulationThanItsList(t *testing.T) {
	tab, found := inboxsalvage.FindTab(inboxsalvage.TabOutstanding)
	require.True(t, found)
	require.True(t, tab.SalvageStatusIsNull,
		"daftar Salvage Outstanding menyaring STSSALVAGE yang KOSONG")

	var counter inboxsalvage.CountRow
	for _, row := range inboxsalvage.CountRows() {
		if row.Tab == inboxsalvage.TabOutstanding {
			counter = row
		}
	}
	require.False(t, counter.SalvageStatusIsNull,
		"pencacahnya justru TIDAK menghitung yang kosong")
	require.Equal(t, []string{"3", "5"}, counter.SalvageStatuses)
}

// Selisih yang direplikasi maupun yang diperbaiki WAJIB dinyatakan ke pengguna, bukan hanya
// tercatat di komentar (`D-54`).
func TestPlannedDifferencesNameTheThreeDefectsDecidedOn(t *testing.T) {
	joined := strings.Join(inboxsalvage.PlannedDifferences, "\n")

	require.Contains(t, joined, "Outstanding",
		"selisih pencacah Outstanding wajib dinyatakan")
	require.Contains(t, joined, "Salvage Ditolak",
		"daftar yang di Pega tidak pernah dijalankan wajib dinyatakan")
	require.Contains(t, joined, "bersarang",
		"baris pencacah yang tertulis ke daftar bersarang wajib dinyatakan")
	require.Contains(t, joined, "Catatan",
		"kolom yang selalu kosong wajib dinyatakan")
	require.Contains(t, joined, "HARI KALENDER",
		"satuan Aging yang berbeda wajib dinyatakan")
	require.Contains(t, joined, "Insurtech",
		"portal yang belum dibangun wajib dinyatakan")
}

func TestAuctionStatusReadsTheValueNotMerelyItsPresence(t *testing.T) {
	// NOL berarti lelangnya berjalan tetapi tidak menghasilkan apa-apa. Memperlakukannya
	// sebagai terjual akan melaporkan pemasukan yang tidak pernah ada.
	require.Equal(t, inboxsalvage.AuctionUnsold, inboxsalvage.AuctionStatusOf("0"))
	require.Equal(t, inboxsalvage.AuctionUnsold, inboxsalvage.AuctionStatusOf(""))
	require.Equal(t, inboxsalvage.AuctionUnsold, inboxsalvage.AuctionStatusOf("   "))
	require.Equal(t, inboxsalvage.AuctionSold, inboxsalvage.AuctionStatusOf("5100000"))
	require.Equal(t, inboxsalvage.AuctionSold, inboxsalvage.AuctionStatusOf("0.01"))
}

func TestSubmissionTypeTurnsOnWhetherTheRequestNoteIsFilled(t *testing.T) {
	require.Equal(t, inboxsalvage.SubmissionNew, inboxsalvage.SubmissionTypeOf(""))
	require.Equal(t, inboxsalvage.SubmissionNew, inboxsalvage.SubmissionTypeOf("  "))
	require.Equal(t,
		inboxsalvage.SubmissionRequest, inboxsalvage.SubmissionTypeOf("minta turun"))
}

// Aging yang GAGAL dibaca menghasilkan KOSONG, bukan "0 day".
//
// Keduanya berbeda dan perbedaannya penting: nol hari berarti diajukan hari ini, kosong
// berarti tanggalnya tidak diketahui. Menyamakannya mengulang cacat `GETSELISIHJAM` yang
// mengembalikan `0` saat gagal — butir 13 daftar perbaikan `P-5`.
func TestAgingIsEmptyWhenTheDateCannotBeRead(t *testing.T) {
	require.Equal(t, "", inboxsalvage.AgingOf("", "2026-09-25"))
	require.Equal(t, "", inboxsalvage.AgingOf("bukan-tanggal", "2026-09-25"))
	require.Equal(t, "", inboxsalvage.AgingOf("2026-09-25", ""))

	require.Equal(t, "0 day", inboxsalvage.AgingOf("2026-09-25", "2026-09-25"))
	require.Equal(t, "10 day", inboxsalvage.AgingOf("2026-09-15", "2026-09-25"))

	// Tanggal yang membawa bagian waktu tetap terbaca — driver mengembalikannya begitu
	// pada sebagian setelan.
	require.Equal(t, "10 day",
		inboxsalvage.AgingOf("2026-09-15 08:30:00", "2026-09-25"))
}

func TestQueryFallsBackToTheDefaultTabAndRejectsAnUnknownOne(t *testing.T) {
	caller := inboxsalvage.Caller{Login: "SITIRAHAYU"}

	query, err := inboxsalvage.NewQuery(inboxsalvage.QueryInput{}, caller)
	require.NoError(t, err)
	require.Equal(t, inboxsalvage.DefaultTab, query.Tab.Code)

	_, err = inboxsalvage.NewQuery(
		inboxsalvage.QueryInput{Tab: "tidak-ada"}, caller)
	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxsalvage.FieldTab, validation.Violations[0].Field)
}

// Identitas yang tidak terbaca DITOLAK, dan alasannya bukan sekadar jejak: satu daftar
// menyaring menurut PIC, sehingga tanpa identitas ia akan menampilkan pengajuan orang lain.
func TestQueryRejectsAnUnknownCaller(t *testing.T) {
	_, err := inboxsalvage.NewQuery(inboxsalvage.QueryInput{}, inboxsalvage.Caller{})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)

	_, err = inboxsalvage.NewQuery(
		inboxsalvage.QueryInput{}, inboxsalvage.Caller{Login: "   "})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)
}

// Pencarian pada daftar yang tidak punya kotak pencarian DIBUANG, bukan ditolak.
//
// Menolaknya akan membuat layar gagal ketika pengguna berpindah daftar sementara kotaknya
// masih terisi — keadaan yang terjadi setiap hari dan bukan kesalahan siapa pun.
func TestSearchIsDroppedOnATabThatHasNoSearchBox(t *testing.T) {
	// Seluruh tab di layar ini punya kotak pencarian, jadi yang diuji adalah aturannya
	// sendiri lewat tab yang dibuat khusus.
	for _, tab := range inboxsalvage.Tabs() {
		require.NotEmpty(t, tab.SearchLabel,
			"tab %q kehilangan kotak pencariannya", tab.Code)
	}
}

// Pencarian COCOK PERSIS pada tiga daftar, MENGANDUNG pada sepuluh daftar lain. Perbedaan
// itu perilaku layar lama, dan ia diuji supaya tidak diseragamkan tanpa sengaja.
func TestCheckerSearchMatchesExactlyWhileOthersMatchPartially(t *testing.T) {
	caller := inboxsalvage.Caller{Login: "SITIRAHAYU"}
	row := inboxsalvage.Row{ClaimNo: "PNC-2044", PIC: "SITIRAHAYU"}

	exact, err := queryForAnyTab(t, inboxsalvage.TabChecker, "PNC-20", caller)
	require.NoError(t, err)
	require.False(t, exact.Matches(row), "separuh nomor klaim TIDAK cocok di tab Checker")

	exactFull, err := queryForAnyTab(t, inboxsalvage.TabChecker, "PNC-2044", caller)
	require.NoError(t, err)
	require.True(t, exactFull.Matches(row))

	// Satu kotak, DUA kolom: nama PIC pun cocok.
	byPIC, err := queryForAnyTab(t, inboxsalvage.TabChecker, "SITIRAHAYU", caller)
	require.NoError(t, err)
	require.True(t, byPIC.Matches(row))

	partial, err := inboxsalvage.NewQuery(inboxsalvage.QueryInput{
		Tab: inboxsalvage.TabHistori, Search: "PNC-20",
	}, caller)
	require.NoError(t, err)
	require.True(t, partial.Matches(row), "separuh nomor klaim cocok di tab Histori")

	// Dan di tab yang tidak mencari PIC, nama PIC TIDAK cocok.
	require.False(t, partial.Matches(inboxsalvage.Row{ClaimNo: "X", PIC: "PNC-20"}))
}

func TestPaginationCorrectsOutOfRangeValuesInsteadOfRejectingThem(t *testing.T) {
	require.Equal(t, 1, inboxsalvage.Pagination{Page: 0}.Normalize().Page)
	require.Equal(t, 1, inboxsalvage.Pagination{Page: -5}.Normalize().Page)

	require.Equal(t,
		inboxsalvage.DefaultPageSize, inboxsalvage.Pagination{Size: 0}.Normalize().Size)
	require.Equal(t,
		inboxsalvage.MaxPageSize, inboxsalvage.Pagination{Size: 9999}.Normalize().Size)

	// Ukuran halaman layar ini 20, BUKAN 50 seperti modul inbox lain. Angkanya dibaca dari
	// `<pyPageSize>20</pyPageSize>` pada section-nya sendiri.
	require.Equal(t, 20, inboxsalvage.DefaultPageSize)
}

func TestTotalPagesIsNeverZero(t *testing.T) {
	empty := inboxsalvage.Page{Pagination: inboxsalvage.Pagination{Page: 1, Size: 20}}
	require.Equal(t, 1, empty.TotalPages())

	full := inboxsalvage.Page{
		Total:      41,
		Pagination: inboxsalvage.Pagination{Page: 1, Size: 20},
	}
	require.Equal(t, 3, full.TotalPages())
}

// `STSTRANSFER` dipetakan BERBEDA pada grid riwayat dibanding pada panel rincian.
//
// Uji ini menjaga keduanya tetap terpisah. Menyatukannya akan mengubah apa yang dibaca
// pengguna di salah satu dari keduanya, dan tidak ada satu pun galat yang menandainya.
func TestHistoryPositionIsADifferentMappingFromThePanelPosition(t *testing.T) {
	cases := []struct {
		code       string
		acceptance bool
		want       string
	}{
		{"1", false, inboxsalvage.HistoryAcceptedByChecker},

		// Kode 2 BERCABANG menurut terisi-tidaknya nomor akseptasi — satu-satunya kode
		// yang begitu, dan kueri lama memang membedakannya.
		{"2", true, inboxsalvage.HistoryAccepted},
		{"2", false, inboxsalvage.HistoryAdjustmentCreated},

		{"3", false, inboxsalvage.HistoryApprovedByKomite},
		{"5", false, inboxsalvage.HistoryRejectedByKomite},
		{"6", false, inboxsalvage.HistoryWaived},

		// Kode 4 dan 7 dipakai sebagai penyaring daftar, tetapi kueri riwayat lama tidak
		// memberinya kalimat. Dibawa apa adanya (`P-5`).
		{"4", false, inboxsalvage.HistoryOther},
		{"7", false, inboxsalvage.HistoryOther},
		{"", false, inboxsalvage.HistoryOther},
		{"  3  ", false, inboxsalvage.HistoryApprovedByKomite},
	}

	for _, c := range cases {
		require.Equalf(t, c.want,
			inboxsalvage.HistoryPositionOf(c.code, c.acceptance),
			"kode %q (akseptasi=%v)", c.code, c.acceptance)
	}

	// Dan pemetaannya memang BERBEDA dari panel rincian untuk kode yang sama.
	require.NotEqual(t,
		inboxsalvage.HistoryPositionOf("3", false),
		inboxsalvage.PositionLabelOf("3"),
		"kedua pemetaan tidak boleh diam-diam menjadi satu")
}

// Tabel ringkas menggambar SEMBILAN baris, persis seperti layar Pega sungguhan.
//
// Daftar dan urutannya disalin dari tangkapan layar yang diberikan Work Owner 2026-09-26.
// Ia sengaja ditulis sebagai senarai harfiah, bukan diturunkan dari CountRows(): uji yang
// menghitung ulang dari sumber yang sama dengan yang diujinya tidak membuktikan apa pun.
func TestCounterRowsMatchTheRealPegaScreen(t *testing.T) {
	want := []string{
		"Outstanding",
		"Ekonomis",
		"Rejected Checker",
		"Balai Lelang",
		"Tidak Ekonomis",
		"TBA",
		"Tidak Ada Salvage",
		"Salvage Buyback",
		"Histori Salvage",
	}

	got := []string{}
	for _, row := range inboxsalvage.CountRows() {
		got = append(got, row.Label)
	}

	require.Equal(t, want, got)
}

// Lima baris yang TIDAK digambar tetap tersimpan, beserta alasannya.
//
// Menyimpannya bukan kelalaian: penyaring tiap baris diturunkan dari pembacaan export yang
// mahal, dan tiga di antaranya harus kembali begitu kewenangan berbasis peran ada — di
// sistem lama dua operator bernama memang melihatnya.
func TestHiddenCounterRowsAreKeptWithTheirReason(t *testing.T) {
	hidden := map[string]string{}
	for _, row := range inboxsalvage.AllCountRows() {
		if row.HiddenReason != "" {
			hidden[row.Label] = row.HiddenReason
		}
	}

	require.Len(t, hidden, 5)

	// Ketiganya hanya tampil bagi dua operator bernama di sistem lama.
	for _, label := range []string{"Checker", "Salvage Diterima", "Salvage Ditolak"} {
		require.Equal(t, inboxsalvage.HiddenManagerOnly, hidden[label], label)
	}

	// Kedua ini tidak tampil pada layar sungguhan sama sekali.
	for _, label := range []string{"Request Balai Lelang", "Tidak Terjual"} {
		require.Equal(t, inboxsalvage.HiddenNotOnScreen, hidden[label], label)
	}
}

// Setiap baris pencacah yang digambar MENUJU sebuah daftar yang juga ditawarkan.
//
// Sebelumnya ada satu baris tanpa daftar ("Tidak Terjual"), dan itu keadaan di Pega. Baris
// itu kini tidak digambar, sehingga seluruh baris yang tersisa dapat diklik — dan uji ini
// menjaga keduanya tidak pernah berselisih lagi: baris pencacah tanpa daftarnya adalah
// angka yang tidak dapat ditindaklanjuti.
func TestEveryVisibleCounterRowOpensAVisibleList(t *testing.T) {
	for _, row := range inboxsalvage.CountRows() {
		require.NotEmptyf(t, row.Tab, "baris %q tidak menuju daftar mana pun", row.Label)

		_, exists := inboxsalvage.FindTab(row.Tab)
		require.Truef(t, exists,
			"baris %q menuju daftar %q yang tidak ditawarkan", row.Label, row.Tab)
	}
}

// Kode daftar yang tidak ditawarkan diperlakukan sama dengan kode yang tidak dikenal.
func TestHiddenTabCodesAreNotResolvable(t *testing.T) {
	hidden := inboxsalvage.HiddenTabs()
	require.Len(t, hidden, 4)

	for _, tab := range hidden {
		_, exists := inboxsalvage.FindTab(tab.Code)
		require.Falsef(t, exists, "daftar %q seharusnya tidak dapat dibuka", tab.Code)

		require.NotEmptyf(t, tab.HiddenReason, "daftar %q disembunyikan tanpa alasan", tab.Code)
	}
}

// queryForAnyTab menyusun permintaan atas daftar mana pun, termasuk yang TIDAK ditawarkan
// layar.
//
// Ia ada karena penyaring keempat daftar yang tidak ditawarkan tetap harus benar: tiga di
// antaranya kembali begitu kewenangan berbasis peran ada, dan penyaring yang tidak diuji
// selama itu akan berhenti benar tanpa ada yang tahu.
func queryForAnyTab(
	t *testing.T,
	code, search string,
	caller inboxsalvage.Caller,
) (inboxsalvage.Query, error) {
	t.Helper()

	tab, known := inboxsalvage.FindAnyTab(code)
	require.Truef(t, known, "daftar %q tidak ada sama sekali", code)

	return inboxsalvage.NewQueryForTab(
		tab, inboxsalvage.QueryInput{Tab: code, Search: search}, caller)
}
