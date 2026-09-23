package inboxadmin_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
)

func at(year int, month time.Month, day int) *time.Time {
	value := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &value
}

// ---------------------------------------------------------------- daftar tab

func TestEightTabsAreBuiltAndThreeAreNot(t *testing.T) {
	// Sebelas tab ada di Pega; tiga di antaranya — Komunikasi — dinyatakan Work Owner
	// 2026-09-20 sudah tidak dipakai dan sudah di-remark di aplikasi lama.
	require.Len(t, inboxadmin.Tabs(), 8)
	require.Len(t, inboxadmin.DisabledTabs, 3)
}

func TestDefaultTabIsAllCaseAdmin(t *testing.T) {
	// Keputusan Work Owner 2026-09-20. Ia berarti layar terbuka pada klaim MILIK petugas
	// itu sendiri, bukan pada seluruh klaim yang sedang berjalan.
	require.Equal(t, inboxadmin.TabAllCaseAdmin, inboxadmin.DefaultTab)

	tab, found := inboxadmin.FindTab(inboxadmin.DefaultTab)
	require.True(t, found)
	require.True(t, tab.ScopedToCaller)
}

func TestEveryTabHasColumns(t *testing.T) {
	for _, tab := range inboxadmin.Tabs() {
		require.NotEmptyf(t, tab.Columns, "tab %s (%s) tanpa kolom", tab.Code, tab.Name)
		require.NotEmptyf(t, tab.Name, "tab %s tanpa nama", tab.Code)
		require.NotEmptyf(t, tab.Description, "tab %s tanpa keterangan", tab.Code)
	}
}

func TestTabCodesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, tab := range inboxadmin.Tabs() {
		require.Falsef(t, seen[tab.Code], "kode tab %s dipakai dua kali", tab.Code)
		seen[tab.Code] = true
	}
}

func TestDisabledTabCodesAreNotBuilt(t *testing.T) {
	for _, disabled := range inboxadmin.DisabledTabs {
		_, found := inboxadmin.FindTab(disabled.Code)
		require.Falsef(t, found, "tab %s seharusnya tidak dibangun", disabled.Code)
	}
}

func TestBothUnregisteredTabsShowTheSameColumns(t *testing.T) {
	// Keduanya dilayani satu kueri di Pega; yang membedakannya hanya saringan kurir.
	// Kolom yang berbeda akan menjadi tanda salah satunya diubah tanpa yang lain.
	normal, found := inboxadmin.FindTab(inboxadmin.TabUnregisteredRCV)
	require.True(t, found)
	online, found := inboxadmin.FindTab(inboxadmin.TabRCVOnline)
	require.True(t, found)

	require.Equal(t, normal.Columns, online.Columns)
}

func TestBranchClaimLabelsProcessColumnAsClaimStatus(t *testing.T) {
	// Kueri `GetKlaimCabang` tidak membawa V_STS_CLAIM sama sekali, sehingga kolom
	// berjudul "Claim Status" pada tab ini sebenarnya berisi status PROSES. Judulnya
	// dipertahankan (`D-13`); yang dijaga uji ini adalah ia tidak diam-diam diarahkan ke
	// isian status klaim yang selalu kosong di tab ini.
	tab, found := inboxadmin.FindTab(inboxadmin.TabBranchClaim)
	require.True(t, found)

	for _, column := range tab.Columns {
		if column.Title == "Claim Status" {
			require.Equal(t, inboxadmin.FieldClaimPosition, column.Key)
			return
		}
	}
	t.Fatal("tab Branch Claim tidak punya kolom berjudul Claim Status")
}

// --------------------------------------------------------------------- Aging

func TestAgingCountsCalendarDays(t *testing.T) {
	now := time.Date(2026, time.September, 20, 10, 0, 0, 0, time.UTC)

	item := inboxadmin.WorkItem{
		ReportDate: at(2026, time.September, 15),
		InputDate:  at(2026, time.September, 18),
	}.WithAging(now)

	require.NotNil(t, item.ReportAgingDays)
	require.Equal(t, 5, *item.ReportAgingDays)

	require.NotNil(t, item.TotalAgingDays)
	require.Equal(t, 2, *item.TotalAgingDays)
}

func TestAgingIsNilWhenItsDateIsEmpty(t *testing.T) {
	// Nil dibedakan dari nol dengan sengaja: nol hari berarti "dilaporkan hari ini",
	// sedangkan tidak ada tanggal berarti kolomnya memang tidak berlaku di tab itu.
	item := inboxadmin.WorkItem{}.WithAging(time.Now())

	require.Nil(t, item.ReportAgingDays)
	require.Nil(t, item.TotalAgingDays)
	require.Nil(t, item.LODAgingDays)
	require.Nil(t, item.RequestAgingDays)
}

func TestAgingIgnoresClockTimeWithinTheSameDay(t *testing.T) {
	// Aturan berbasis hari kalender dihitung terhadap TANGGAL
	// (`08-TECHNICAL-STRATEGY.md` §4.4). Baris yang masuk pukul 23.00 dan dibaca pukul
	// 01.00 esok harinya berumur satu hari, bukan nol hari.
	entered := time.Date(2026, time.September, 19, 23, 0, 0, 0, time.UTC)
	read := time.Date(2026, time.September, 20, 1, 0, 0, 0, time.UTC)

	item := inboxadmin.WorkItem{InputDate: &entered}.WithAging(read)

	require.NotNil(t, item.TotalAgingDays)
	require.Equal(t, 1, *item.TotalAgingDays)
}

func TestAgingNeverGoesNegative(t *testing.T) {
	// Tanggal di masa depan memang ada di data warisan. "Minus tiga hari" tidak berarti
	// apa pun bagi petugas yang membaca kolom tenggat.
	now := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
	item := inboxadmin.WorkItem{InputDate: at(2026, time.September, 23)}.WithAging(now)

	require.NotNil(t, item.TotalAgingDays)
	require.Equal(t, 0, *item.TotalAgingDays)
}

// ---------------------------------------------------------------- paginasi

func TestPaginationCorrectsValuesOutOfRange(t *testing.T) {
	clean := inboxadmin.Pagination{Page: 0, Size: 0}.Normalize()
	require.Equal(t, 1, clean.Page)
	require.Equal(t, inboxadmin.DefaultPageSize, clean.Size)

	capped := inboxadmin.Pagination{Page: 3, Size: 5000}.Normalize()
	require.Equal(t, inboxadmin.MaxPageSize, capped.Size)
}

func TestDefaultPageSizeFollowsPegaGrid(t *testing.T) {
	// 25 adalah ukuran halaman grid di `Section/PNCInboxAdmin-Section.xml`. Ia tidak
	// disamakan dengan modul lain yang memakai 20.
	require.Equal(t, 25, inboxadmin.DefaultPageSize)
}

func TestSliceCutsTheRequestedPage(t *testing.T) {
	all := make([]inboxadmin.WorkItem, 0, 7)
	for _, code := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		all = append(all, inboxadmin.WorkItem{CaseID: code})
	}

	page := inboxadmin.Slice(all, inboxadmin.Pagination{Page: 2, Size: 3})

	require.Equal(t, 7, page.Total)
	require.Equal(t, 3, page.TotalPages())
	require.Len(t, page.Items, 3)
	require.Equal(t, "D", page.Items[0].CaseID)
	require.Equal(t, "F", page.Items[2].CaseID)
}

func TestSliceBeyondTheLastPageIsEmptyNotAnError(t *testing.T) {
	// Halaman di luar jangkauan dapat terjadi tanpa kesalahan siapa pun: petugas lain
	// menyelesaikan pekerjaannya, barisnya hilang dari antrean, dan halaman yang tadi ada
	// menjadi tidak ada. Menjawabnya dengan galat akan membuat layar tampak rusak.
	all := []inboxadmin.WorkItem{{CaseID: "A"}}

	page := inboxadmin.Slice(all, inboxadmin.Pagination{Page: 9, Size: 25})

	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.Equal(t, 1, page.Total)
}

func TestTotalPagesIsAtLeastOne(t *testing.T) {
	page := inboxadmin.Slice(nil, inboxadmin.Pagination{Page: 1, Size: 25})
	require.Equal(t, 1, page.TotalPages())
}

// --------------------------------------------------------------- pembentukan Query

func TestEmptyTabBecomesTheDefaultTab(t *testing.T) {
	query, err := NewQueryForTest(t, inboxadmin.QueryInput{})
	require.NoError(t, err)
	require.Equal(t, inboxadmin.DefaultTab, query.Tab.Code)
}

func TestUnknownTabIsRejected(t *testing.T) {
	_, err := NewQueryForTest(t, inboxadmin.QueryInput{Tab: "99"})

	var validation *inboxadmin.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxadmin.FieldTab, validation.Violations[0].Field)
}

func TestDisabledTabIsRejectedWithItsReason(t *testing.T) {
	// Tautan lama yang masih menyimpan `CityID=4` akan sampai ke sini. Jawaban
	// "tab tidak dikenal" akan membuat orang mengira modulnya belum selesai.
	_, err := NewQueryForTest(t, inboxadmin.QueryInput{Tab: "4"})

	var validation *inboxadmin.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Violations[0].Message, "Not Answered")
	require.Contains(t, validation.Violations[0].Message, "remark")
}

func TestCallerWithoutLoginIsRejected(t *testing.T) {
	_, err := inboxadmin.NewQuery(inboxadmin.QueryInput{}, inboxadmin.Caller{Login: "   "})
	require.ErrorIs(t, err, inboxadmin.ErrCallerUnknown)
}

func TestEmptyBusinessLineMeansAll(t *testing.T) {
	query, err := NewQueryForTest(t, inboxadmin.QueryInput{Tab: inboxadmin.TabAll})
	require.NoError(t, err)
	require.Equal(t, inboxadmin.BusinessAll, query.Business)
}

func TestUnknownBusinessLineIsRejected(t *testing.T) {
	_, err := NewQueryForTest(t, inboxadmin.QueryInput{
		Tab:      inboxadmin.TabAll,
		Business: "MBU",
	})

	var validation *inboxadmin.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxadmin.FieldBusiness, validation.Violations[0].Field)
}

func TestFiltersAreDroppedOnTabsThatDoNotSupportThem(t *testing.T) {
	// Tab Status RCL/PUCL tidak punya kotak cari maupun dropdown lini bisnis di Pega.
	// Mengirimkan keduanya tidak boleh berakibat apa pun — dan yang dikembalikan adalah
	// keadaan sebenarnya, supaya layar dapat membersihkan kotaknya.
	query, err := NewQueryForTest(t, inboxadmin.QueryInput{
		Tab:      inboxadmin.TabRCLPUCL,
		Business: string(inboxadmin.BusinessPA),
		Keyword:  "PNC-1",
	})

	require.NoError(t, err)
	require.Equal(t, inboxadmin.BusinessAll, query.Business)
	require.Empty(t, query.Keyword)
}

func TestKeywordIsTrimmed(t *testing.T) {
	query, err := NewQueryForTest(t, inboxadmin.QueryInput{
		Tab:     inboxadmin.TabAll,
		Keyword: "  PNC-88  ",
	})

	require.NoError(t, err)
	require.Equal(t, "PNC-88", query.Keyword)
}

func TestBusinessLineLabelsAreFilled(t *testing.T) {
	lines := inboxadmin.BusinessLines()
	require.Len(t, lines, 5)

	for _, line := range lines {
		require.NotEmptyf(t, line.Label(), "lini bisnis %s tanpa label", line)
		require.NotEqualf(t, string(line), line.Label(),
			"lini bisnis %s belum punya label yang dapat dibaca pengguna", line)
	}
}

// NewQueryForTest membentuk permintaan dengan pemanggil yang selalu dikenali, sehingga uji
// di atas dapat berfokus pada isian yang sedang diujinya.
func NewQueryForTest(t *testing.T, input inboxadmin.QueryInput) (inboxadmin.Query, error) {
	t.Helper()
	return inboxadmin.NewQuery(input, inboxadmin.Caller{Login: "ADMINKLAIM"})
}
