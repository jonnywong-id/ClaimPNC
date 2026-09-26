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

// Ketiga daftar layar lama ada, dengan kode yang dipakai alamat.
func TestTheScreenOffersExactlyThreeLists(t *testing.T) {
	tabs := inboxpladla.Tabs()
	require.Len(t, tabs, 3)

	got := []string{}
	for _, tab := range tabs {
		got = append(got, tab.Code)
	}
	require.Equal(t, []string{"pla", "dla", "close"}, got)
}

// Setiap daftar punya keterangan dan kolom.
func TestEveryListDescribesItself(t *testing.T) {
	for _, tab := range inboxpladla.Tabs() {
		require.NotEmpty(t, tab.Name, "%s", tab.Code)
		require.NotEmpty(t, tab.Description, "%s", tab.Code)
		require.Len(t, tab.Columns, 9, "%s kolomnya bergeser", tab.Code)
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
