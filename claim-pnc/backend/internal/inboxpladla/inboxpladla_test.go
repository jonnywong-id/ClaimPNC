package inboxpladla_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
)

func caller() inboxpladla.Caller {
	return inboxpladla.Caller{Login: "REASCONTOH"}
}

func codes() []string { return []string{"R901", "R900"} }

// Layar lama menawarkan TUJUH tampilan, bukan tiga.
//
// Seluruhnya dikendalikan satu nilai — `TempView.CityID`, yang `SetDataPLADLA` terima
// sebagai `param.tipe` — dan percabangannya terbaca langsung dari activity itu:
//
//	1 PLA · 2 PLA & DLA · 3 CLOSE CLAIM
//	4 komunikasi masuk · 5 terkirim belum dijawab · 6 terkirim sudah dijawab
//	7 DATA PLA DLA XOL KLAIM
//
// Ketiga yang terakhir pada baris kedua tidak pernah dibangun sampai 2026-09-28, dan
// ketiadaannya tidak terlihat dari layar: tampilan yang tidak punya tab tidak pernah
// diminta siapa pun.
func TestTheScreenOffersSevenViews(t *testing.T) {
	tabs := inboxpladla.Tabs()
	require.Len(t, tabs, 7)

	got := []string{}
	for _, tab := range tabs {
		got = append(got, tab.Code)
	}
	require.Equal(t, []string{
		"pla", "dla", "close",
		"komunikasi-masuk", "komunikasi-terkirim", "komunikasi-dijawab",
		"xol",
	}, got)
}

// Keenam daftar klaim memakai kolom yang SAMA; XOL punya kolomnya sendiri.
func TestEveryViewDescribesItself(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		require.NotEmpty(t, tab.Name, "%s", tab.Code)
		require.NotEmpty(t, tab.Description, "%s", tab.Code)

		if !tab.IsClaimList() {
			require.Len(t, tab.Columns, 4, "%s kolomnya bergeser", tab.Code)
			continue
		}
		require.Len(t, tab.Columns, 9, "%s kolomnya bergeser", tab.Code)
	}
}

// Judul kolomnya diambil APA ADANYA dari Pega, termasuk urutannya.
//
// Terbaca dari header grid pada `Section/InboxDLAReas_sect-Section.xml`. Menerjemahkannya
// melanggar `D-13`, dan di layar ini akibatnya lebih tajam daripada biasanya: pembacanya
// pihak LUAR, yang tidak dapat kita latih ulang.
func TestTheColumnTitlesFollowPegaWordForWord(t *testing.T) {
	pla, found := inboxpladla.FindTab("pla")
	require.True(t, found)

	titles := []string{}
	for _, column := range pla.Columns {
		titles = append(titles, column.Title)
	}

	require.Equal(t, []string{
		"Claim No", "Insured", "Policy No", "Business Name",
		"DOL", "Register Date", "PIC ASM", "PLA No", "Claim Progress",
	}, titles)
}

// Ketiga daftar komunikasi TIDAK mengecualikan lini bisnis apa pun.
//
// Ketiga daftar pemberitahuan mengecualikan Personal Accident dan Travel;
// `BrowseCommunicationReas` tidak memuat satu pun syarat itu. Menyeragamkannya akan
// MENGHILANGKAN klaim PA dari daftar komunikasi tanpa satu pun galat.
func TestOnlyTheAdviceListsExcludeBusinessLines(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		switch tab.Source {
		case inboxpladla.SourceAdvice:
			require.True(t, tab.ExcludesGroupPanel("002"), "%s", tab.Code)
			require.True(t, tab.ExcludesGroupPanel("005"), "%s", tab.Code)
		case inboxpladla.SourceCommunication:
			require.False(t, tab.ExcludesGroupPanel("002"), "%s", tab.Code)
			require.False(t, tab.ExcludesGroupPanel("005"), "%s", tab.Code)
		}
	}
}

// Ketiga daftar komunikasi menyaring PERCAKAPAN, bukan dokumen pemberitahuan.
func TestTheCommunicationListsFilterOnConversationsOnly(t *testing.T) {
	expected := map[string]struct {
		status string
		role   inboxpladla.CommunicationRole
	}{
		"komunikasi-masuk":    {"0", inboxpladla.RoleRecipient},
		"komunikasi-terkirim": {"0", inboxpladla.RoleSender},
		"komunikasi-dijawab":  {"1", inboxpladla.RoleSender},
	}

	for code, want := range expected {
		tab, found := inboxpladla.FindTab(code)
		require.True(t, found, "%s", code)

		require.Equal(t, inboxpladla.SourceCommunication, tab.Source, "%s", code)
		require.Equal(t, want.status, tab.CommunicationStatus, "%s", code)
		require.Equal(t, want.role, tab.CommunicationRole, "%s", code)

		require.Empty(t, tab.AdviceKindSent,
			"%s tidak boleh menuntut dokumen pemberitahuan terkirim", code)
		require.False(t, tab.ExcludeWhenDLASent, "%s", code)
	}
}

// Tampilan XOL BUKAN daftar klaim, dan layar mengetahuinya dari SERVER.
//
// Alternatifnya — layar mencocokkan `kode === 'xol'` — ditolak dengan alasan yang sama
// seperti senarai kolom: inventaris tampilan adalah hasil pembacaan export, dan
// menyalinnya ke layar berarti keputusan yang sama hidup di dua tempat.
func TestTheXOLViewIsNotAClaimList(t *testing.T) {
	xol, found := inboxpladla.FindTab("xol")
	require.True(t, found)

	require.False(t, xol.IsClaimList())
	require.Equal(t, inboxpladla.ViewXOL, xol.Kind)
	require.False(t, xol.HasDetailAction,
		"tampilan XOL tidak menggambar klaim, sehingga tidak punya tombol rincian")

	for _, tab := range inboxpladla.Tabs() {
		if tab.Code == "xol" {
			continue
		}
		require.True(t, tab.IsClaimList(), "%s", tab.Code)
		require.True(t, tab.HasDetailAction, "%s", tab.Code)
	}
}

// Daftar Close disaring PLA, bukan DLA — meski ia daftar klaim yang sudah selesai.
func TestTheCloseListIsFilteredByPLA(t *testing.T) {
	close, found := inboxpladla.FindTab("close")
	require.True(t, found)
	require.Equal(t, "pla", close.AdviceKindSent,
		"GetPNCList_PLADLAClose menyaring t_plalist, bukan t_dlalist")

	dla, found := inboxpladla.FindTab("dla")
	require.True(t, found)
	require.Equal(t, "dla", dla.AdviceKindSent)
}

// Hanya daftar PLA yang mengeluarkan klaim yang DLA-nya sudah terkirim.
//
// Itu yang membuat ketiga daftarnya tidak saling bertumpuk.
func TestOnlyThePLAListExcludesClaimsWhoseDLAWasSent(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		if tab.Code == "pla" {
			require.True(t, tab.ExcludeWhenDLASent)
			continue
		}
		require.False(t, tab.ExcludeWhenDLASent, "%s", tab.Code)
	}
}

// Hanya daftar DLA yang mengganti kode status menjadi `1139`.
func TestOnlyTheDLAListReplacesTheStatusCode(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		if tab.Code == "dla" {
			require.True(t, tab.PendingCloseBecomes1139)
			continue
		}
		require.False(t, tab.PendingCloseBecomes1139, "%s", tab.Code)
	}
}

// Hanya daftar Close yang mencocokkan SELURUH kode reasuradur.
//
// Perbedaan ini dipusatkan di EffectiveReinsurerCodes supaya ia terbaca di satu tempat
// alih-alih tersembunyi di tiga kueri sebagai `IN` versus `=`.
func TestOnlyTheCloseListUsesEveryReinsurerCode(t *testing.T) {
	forTab := func(code string) []string {
		query, err := inboxpladla.NewQuery(
			inboxpladla.QueryInput{Tab: code}, caller(), codes())
		require.NoError(t, err)
		return query.EffectiveReinsurerCodes()
	}

	require.Equal(t, []string{"R901", "R900"}, forTab("close"))
	require.Equal(t, []string{"R901"}, forTab("pla"),
		"daftar PLA hanya memakai kode tertinggi")
	require.Equal(t, []string{"R901"}, forTab("dla"),
		"daftar DLA hanya memakai kode tertinggi")
}

// Permintaan tanpa daftar jatuh ke daftar bawaan.
func TestAnEmptyListCodeFallsBackToTheDefault(t *testing.T) {
	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{}, caller(), codes())

	require.NoError(t, err)
	require.Equal(t, inboxpladla.DefaultTab, query.Tab.Code)
}

// Daftar yang tidak dikenal DITOLAK.
func TestAnUnknownListIsRejected(t *testing.T) {
	_, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Tab: "pre-dla"}, caller(), codes())

	var validation *inboxpladla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxpladla.FieldTab, validation.Violations[0].Field)
}

// Permintaan tanpa identitas ditolak SEBELUM apa pun yang lain diperiksa.
//
// Di layar ini identitas bukan sekadar soal jejak: ia penyaring utama ketiga daftarnya.
func TestAQueryWithoutACallerIsRejected(t *testing.T) {
	_, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{}, inboxpladla.Caller{Login: "  "}, codes())

	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)
}

// Pemanggil yang bukan reasuradur DITOLAK, bukan dijawab daftar kosong.
//
// Keduanya berarti hal yang berbeda, dan hanya satu yang dapat ditindaklanjuti pengguna.
func TestACallerWithoutAReinsurerCodeIsRejected(t *testing.T) {
	_, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{}, caller(), nil)

	require.ErrorIs(t, err, inboxpladla.ErrCallerNotAReinsurer)
}

// Pencarian mencocokkan KUNCI KLAIM, dan tidak peka huruf besar-kecil.
func TestSearchMatchesTheWorkKeyCaseInsensitively(t *testing.T) {
	query, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Search: "pnc-2001"}, caller(), codes())
	require.NoError(t, err)

	require.True(t, query.Matches(inboxpladla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-2001",
	}))
	require.False(t, query.Matches(inboxpladla.Row{
		ClaimKey: "ASM-FW-GCNMFW-WORK PNC-9999",
	}))
}

// Kata kunci yang terlalu panjang ditolak sebelum sampai ke basis data.
func TestAnOverlongSearchKeywordIsRejected(t *testing.T) {
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'A'
	}

	_, err := inboxpladla.NewQuery(
		inboxpladla.QueryInput{Search: string(long)}, caller(), codes())

	var validation *inboxpladla.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Equal(t, inboxpladla.FieldSearch, validation.Violations[0].Field)
}

// Paginasi di luar rentang DIBETULKAN, bukan ditolak.
func TestPaginationIsCorrectedRatherThanRejected(t *testing.T) {
	require.Equal(t,
		inboxpladla.Pagination{Page: 1, Size: inboxpladla.DefaultPageSize},
		inboxpladla.Pagination{Page: -3, Size: 0}.Normalize())

	require.Equal(t,
		inboxpladla.Pagination{Page: 2, Size: inboxpladla.MaxPageSize},
		inboxpladla.Pagination{Page: 2, Size: 9999}.Normalize())
}

// Selisih terencana menyebut kebocoran XOL yang TIDAK dibawa.
//
// Ia yang paling penting dibaca di antara seluruh selisih modul ini: di Pega setiap
// reasuradur melihat ringkasan XOL milik satu mitra tertentu.
func TestPlannedDifferencesNameTheXOLLeakThatWasNotCarriedOver(t *testing.T) {
	joined := ""
	for _, item := range inboxpladla.PlannedDifferences {
		joined += item + "\n"
	}

	require.Contains(t, joined, "XOL")
	require.Contains(t, joined, "login Anda sendiri")
	require.Contains(t, joined, "D-15")
}
