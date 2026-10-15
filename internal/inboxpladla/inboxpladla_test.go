package inboxpladla_test

import (
	"errors"
	"strings"
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
func TestTheScreenOffersSixViews(t *testing.T) {
	tabs := inboxpladla.Tabs()
	require.Len(t, tabs, 6)

	got := []string{}
	for _, tab := range tabs {
		got = append(got, tab.Code)
	}
	require.Equal(t, []string{
		"pla", "dla", "close",
		"not-answered", "not-replied-from-asm", "replied-from-asm",
	}, got)
}

// Keenam daftar memakai kolom yang SAMA.
func TestEveryViewDescribesItself(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		require.NotEmpty(t, tab.Name, "%s", tab.Code)
		require.NotEmpty(t, tab.Description, "%s", tab.Code)

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
		"not-answered":         {"0", inboxpladla.RoleRecipient},
		"not-replied-from-asm": {"0", inboxpladla.RoleSender},
		"replied-from-asm":     {"1", inboxpladla.RoleSender},
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

// Tampilan XOL TIDAK ditawarkan, dan itu bukan kelalaian.
//
// Sectionnya memang memuat wadah berjudul "DATA PLA DLA XOL KLAIM", tetapi wadah itu
// bersyarat `pyContainerVisibleWhen = TempView.CityID==7` sementara `CityID` hanya pernah
// diisi dari `.CityID` sebuah baris tabel "Status / Jumlah" — dan tabel itu berisi enam
// baris. Literal `7` nol kemunculan sebagai nilai `tipe` di seluruh export.
//
// Ia sempat dibawa sebagai tab ketujuh, lalu sebagai panel permanen. Uji ini yang
// menahannya kembali: menambahkan tab ketujuh membuatnya merah.
func TestTheXOLViewIsNotOfferedAtAll(t *testing.T) {
	_, found := inboxpladla.FindTab("xol")
	require.False(t, found, "tampilan XOL tidak boleh ditawarkan")

	tabs := inboxpladla.Tabs()
	require.Len(t, tabs, 6, "layar ini punya ENAM daftar")

	for _, tab := range tabs {
		require.NotEqual(t, "xol", tab.Code)
		require.Len(t, tab.Columns, 9, "%s kolomnya bergeser", tab.Code)
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

// Selisih terbesar layar ini adalah satu TAMPILAN UTUH yang tidak dibawa.
//
// Ia harus terbaca pengguna di kaki layar, bukan hanya tercatat di kode: mitra yang
// terbiasa mencari "DATA PLA DLA XOL KLAIM" di Pega perlu tahu mengapa ia tidak ada —
// dan jawabannya bukan "belum dibangun" melainkan "di Pega pun ia tidak pernah tergambar".
func TestPlannedDifferencesSayTheXOLViewIsNotCarriedOver(t *testing.T) {
	joined := strings.Join(inboxpladla.PlannedDifferences, "\n")

	require.Contains(t, joined, "DATA PLA DLA XOL KLAIM")
	require.Contains(t, joined, "TIDAK dibawa")
	require.Contains(t, joined, "CityID==7")
}
