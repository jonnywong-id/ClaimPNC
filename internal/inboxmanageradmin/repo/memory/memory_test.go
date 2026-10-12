package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/repo/memory"
)

func queryFor(t *testing.T, orgUnit string) inboxmanageradmin.Query {
	t.Helper()
	return inboxmanageradmin.Query{Tab: inboxmanageradmin.Tab{OrgUnit: orgUnit}}
}

func caseIDs(items []inboxmanageradmin.WorkItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.CaseID)
	}
	return result
}

func TestListFiltersClassOrgUnitAndStatus(t *testing.T) {
	store := memory.NewStrictStore()

	got, err := store.List(context.Background(), queryFor(t, inboxmanageradmin.OrgUnitNonMBU))
	require.NoError(t, err)
	// PNC-9003 selesai, RCV-7001 kelas lain: keduanya tersaring.
	require.Equal(t, []string{"PNC-9001", "PNC-9002"}, caseIDs(got))

	pa, err := store.List(context.Background(), queryFor(t, inboxmanageradmin.OrgUnitPA))
	require.NoError(t, err)
	// PNC-9103 ditolak dan tersaring.
	require.Equal(t, []string{"PNC-9101", "PNC-9102"}, caseIDs(pa))

	// Unit organisasi dibandingkan persis — huruf kecil tidak cocok.
	none, err := store.List(context.Background(), queryFor(t, "adminpnc"))
	require.NoError(t, err)
	require.Empty(t, none)
	require.NotNil(t, none, "hasil kosong adalah senarai, bukan nil")
}

func TestListDerivesClaimStatusFromWorkStatus(t *testing.T) {
	store := memory.NewStore(memory.Row{
		WorkClass:  memory.WorkClassPNC,
		OrgUnit:    "U",
		WorkStatus: "New",
		Item:       inboxmanageradmin.WorkItem{CaseID: "A", ClaimStatus: "dikarang"},
	})

	got, err := store.List(context.Background(), queryFor(t, "U"))
	require.NoError(t, err)
	require.Equal(t, inboxmanageradmin.DisplayStatusFor("New"), got[0].ClaimStatus)
}

func TestListOrdersNewestFirstWithCaseTieBreakAndEmptyDatesLast(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	sameAsLate := late

	row := func(id string, at *time.Time) memory.Row {
		return memory.Row{
			WorkClass: memory.WorkClassPNC, OrgUnit: "U", WorkStatus: "New",
			Item: inboxmanageradmin.WorkItem{CaseID: id, RegisteredAt: at},
		}
	}
	store := memory.NewStore(
		row("N2", nil),
		row("E1", &early),
		row("L2", &sameAsLate),
		row("N1", nil),
		row("L1", &late),
	)

	got, err := store.List(context.Background(), queryFor(t, "U"))
	require.NoError(t, err)
	require.Equal(t, []string{"L1", "L2", "E1", "N1", "N2"}, caseIDs(got))
}

func TestNewStoreCopiesRows(t *testing.T) {
	rows := []memory.Row{{WorkClass: memory.WorkClassPNC, OrgUnit: "U", Item: inboxmanageradmin.WorkItem{CaseID: "A"}}}
	store := memory.NewStore(rows...)
	rows[0].Item.CaseID = "diubah"

	got, err := store.List(context.Background(), queryFor(t, "U"))
	require.NoError(t, err)
	require.Equal(t, []string{"A"}, caseIDs(got))
}

func TestLineBusinessLookupIsCaseAndSpaceInsensitive(t *testing.T) {
	store := memory.NewStore()
	store.SetLineBusiness("  Petugas.Contoh ", " PA ")

	line, err := store.LineBusinessFor(context.Background(), "PETUGAS.CONTOH")
	require.NoError(t, err)
	require.Equal(t, "PA", line)

	unknown, err := store.LineBusinessFor(context.Background(), "orang.lain")
	require.NoError(t, err)
	require.Empty(t, unknown, "penyimpanan ketat tidak punya lini bisnis bawaan")
}

func TestZeroValueStoreAcceptsLineBusiness(t *testing.T) {
	// Peta lini bisnis dibentuk saat pertama dipakai, sehingga nilai nol tidak panik.
	store := &memory.Store{}
	store.SetLineBusiness("a", "TRAVEL")

	line, err := store.LineBusinessFor(context.Background(), "A")
	require.NoError(t, err)
	require.Equal(t, "TRAVEL", line)
}

func TestSampleStoreDefaultsToNonMBUButExplicitWins(t *testing.T) {
	store := memory.NewSampleStore()

	line, err := store.LineBusinessFor(context.Background(), "siapa.saja")
	require.NoError(t, err)
	require.Equal(t, inboxmanageradmin.LineNonMBU, line)

	store.SetLineBusiness("siapa.saja", inboxmanageradmin.LineTravel)
	line, err = store.LineBusinessFor(context.Background(), "siapa.saja")
	require.NoError(t, err)
	require.Equal(t, inboxmanageradmin.LineTravel, line)
}

func TestOrgUnitsListsDistinctSortedNonEmpty(t *testing.T) {
	store := memory.NewStore(append(memory.SampleRows(), memory.Row{OrgUnit: "  "})...)

	require.Equal(t, []string{"AdminLain", "AdminPA", "AdminPNC", "AdminTRAVEL"}, store.OrgUnits())
}
