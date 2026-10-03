package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
)

func tab(t *testing.T, code string) inboxadmin.Tab {
	t.Helper()
	found, ok := inboxadmin.FindTab(code)
	require.True(t, ok, "tab %s tidak dikenal", code)
	return found
}

func date(day int) *time.Time {
	at := time.Date(2026, time.August, day, 0, 0, 0, 0, time.UTC)
	return &at
}

func caseIDs(items []inboxadmin.WorkItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.CaseID)
	}
	return ids
}

func list(t *testing.T, store *memory.Store, q inboxadmin.Query) []string {
	t.Helper()
	items, err := store.List(context.Background(), q)
	require.NoError(t, err)
	require.NotNil(t, items)
	return caseIDs(items)
}

// businessStore memuat satu baris per kombinasi Group Panel/kelompok bisnis di tab ALL.
func businessStore() *memory.Store {
	row := func(id, panel, group string) memory.Row {
		return memory.Row{
			Tab: inboxadmin.TabAll, GroupPanel: panel, BusinessGroupID: group,
			Item: inboxadmin.WorkItem{CaseID: id, PolicyNumber: "POL-" + id},
		}
	}
	return memory.NewStore(
		row("A-ANEKA", "003", "10001"),
		row("B-BONDING", "009", "10015"),
		row("C-PA", "002", "10002"),
		row("D-TRAVEL", "005", "10003"),
		row("E-MARINE", "004", "10004"),
	)
}

func TestBusinessFilterFollowsTheOldPredicate(t *testing.T) {
	store := businessStore()
	all := tab(t, inboxadmin.TabAll)

	cases := map[inboxadmin.BusinessLine][]string{
		inboxadmin.BusinessAll:     {"A-ANEKA", "B-BONDING", "C-PA", "D-TRAVEL", "E-MARINE"},
		inboxadmin.BusinessNonMBU:  {"A-ANEKA", "E-MARINE"},
		inboxadmin.BusinessBonding: {"B-BONDING"},
		inboxadmin.BusinessPA:      {"C-PA"},
		inboxadmin.BusinessTravel:  {"D-TRAVEL"},
		// Lini yang tidak dikenal tidak cocok dengan apa pun.
		inboxadmin.BusinessLine("ENTAH"): {},
	}

	for line, want := range cases {
		got := list(t, store, inboxadmin.Query{Tab: all, Business: line})
		require.Equalf(t, want, got, "lini %s", line)
	}
}

func TestKeywordTouchesCaseIDAndPolicyNumberCaseInsensitively(t *testing.T) {
	store := businessStore()
	all := tab(t, inboxadmin.TabAll)

	require.Equal(t, []string{"C-PA"},
		list(t, store, inboxadmin.Query{Tab: all, Business: inboxadmin.BusinessAll, Keyword: "c-pa"}))
	require.Equal(t, []string{"D-TRAVEL"},
		list(t, store, inboxadmin.Query{Tab: all, Business: inboxadmin.BusinessAll, Keyword: "pol-d"}))
	require.Empty(t,
		list(t, store, inboxadmin.Query{Tab: all, Business: inboxadmin.BusinessAll, Keyword: "tidak-ada"}))
}

func TestCourierSeparatesBothUnregisteredTabs(t *testing.T) {
	store := memory.NewStore(
		memory.Row{Tab: inboxadmin.TabUnregisteredRCV, Courier: "JNE",
			Item: inboxadmin.WorkItem{CaseID: "RCV-1"}},
		memory.Row{Tab: inboxadmin.TabUnregisteredRCV, Courier: "Auto Service",
			Item: inboxadmin.WorkItem{CaseID: "RCV-2"}},
		memory.Row{Tab: inboxadmin.TabRCVOnline, Courier: "Auto Service",
			Item: inboxadmin.WorkItem{CaseID: "RCV-3"}},
		memory.Row{Tab: inboxadmin.TabRCVOnline, Courier: "",
			Item: inboxadmin.WorkItem{CaseID: "RCV-4"}},
	)

	require.Equal(t, []string{"RCV-1"}, list(t, store, inboxadmin.Query{
		Tab: tab(t, inboxadmin.TabUnregisteredRCV), Business: inboxadmin.BusinessAll}))
	require.Equal(t, []string{"RCV-3"}, list(t, store, inboxadmin.Query{
		Tab: tab(t, inboxadmin.TabRCVOnline), Business: inboxadmin.BusinessAll}))
}

func TestScopedTabsOnlyShowRowsOwnedByTheCaller(t *testing.T) {
	store := memory.NewStore(
		memory.Row{Tab: inboxadmin.TabAllCaseAdmin, Owner: "ADMINKLAIM",
			Item: inboxadmin.WorkItem{CaseID: "MINE"}},
		memory.Row{Tab: inboxadmin.TabAllCaseAdmin, Owner: "ORANGLAIN",
			Item: inboxadmin.WorkItem{CaseID: "THEIRS"}},
	)

	// Pencocokan login tidak peka huruf besar-kecil.
	got := list(t, store, inboxadmin.Query{
		Tab:      tab(t, inboxadmin.TabAllCaseAdmin),
		Business: inboxadmin.BusinessAll,
		Caller:   inboxadmin.Caller{Login: "adminklaim"},
	})
	require.Equal(t, []string{"MINE"}, got)
}

func TestDatedRowsAreSortedNewestFirstThenByCaseID(t *testing.T) {
	store := memory.NewStore(
		memory.Row{Tab: inboxadmin.TabAll, Item: inboxadmin.WorkItem{CaseID: "Z-OLD", InputDate: date(1)}},
		memory.Row{Tab: inboxadmin.TabAll, Item: inboxadmin.WorkItem{CaseID: "Y-NEW", InputDate: date(9)}},
		memory.Row{Tab: inboxadmin.TabAll, Item: inboxadmin.WorkItem{CaseID: "C-SAME", InputDate: date(9)}},
	)

	got := list(t, store, inboxadmin.Query{Tab: tab(t, inboxadmin.TabAll), Business: inboxadmin.BusinessAll})

	// Dua bertanggal sama diurutkan menurut Case ID.
	require.Equal(t, []string{"C-SAME", "Y-NEW", "Z-OLD"}, got)
}

func TestUndatedRowsAreSortedByCaseID(t *testing.T) {
	store := memory.NewStore(
		memory.Row{Tab: inboxadmin.TabAll, Item: inboxadmin.WorkItem{CaseID: "B-NODATE"}},
		memory.Row{Tab: inboxadmin.TabAll, Item: inboxadmin.WorkItem{CaseID: "A-NODATE"}},
	)

	got := list(t, store, inboxadmin.Query{Tab: tab(t, inboxadmin.TabAll), Business: inboxadmin.BusinessAll})
	require.Equal(t, []string{"A-NODATE", "B-NODATE"}, got)
}

func TestEmptyStoreReturnsEmptyListNotNil(t *testing.T) {
	got := list(t, memory.NewStore(), inboxadmin.Query{
		Tab: tab(t, inboxadmin.TabRCLPUCL), Business: inboxadmin.BusinessAll})
	require.Empty(t, got)
}

func TestSampleStoreServesEveryBuiltTab(t *testing.T) {
	// Contoh bawaan memuat sedikitnya satu baris untuk setiap tab yang dibangun bagi
	// pemiliknya, sehingga layar lokal tidak pernah tampak kosong tanpa alasan.
	store := memory.NewSampleStore()
	caller := inboxadmin.Caller{Login: memory.SampleOwner}

	for _, built := range inboxadmin.Tabs() {
		got := list(t, store, inboxadmin.Query{
			Tab: built, Business: inboxadmin.BusinessAll, Caller: caller})
		require.NotEmptyf(t, got, "tab %s (%s) kosong", built.Code, built.Name)
	}
}

func TestSampleScopedTabHidesTheOtherOwnersRow(t *testing.T) {
	store := memory.NewSampleStore()

	mine := list(t, store, inboxadmin.Query{
		Tab: tab(t, inboxadmin.TabAllCaseAdmin), Business: inboxadmin.BusinessAll,
		Caller: inboxadmin.Caller{Login: memory.SampleOwner}})
	stranger := list(t, store, inboxadmin.Query{
		Tab: tab(t, inboxadmin.TabAllCaseAdmin), Business: inboxadmin.BusinessAll,
		Caller: inboxadmin.Caller{Login: "TIDAKADA"}})

	require.NotEmpty(t, mine)
	require.Empty(t, stranger)
}
