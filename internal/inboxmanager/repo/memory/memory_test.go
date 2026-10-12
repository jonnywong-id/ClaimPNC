package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

func tabOf(t *testing.T, code string) inboxmanager.Tab {
	t.Helper()
	tab, known := inboxmanager.FindTab(code)
	require.True(t, known)
	return tab
}

func TestCountersDerivedFromRows(t *testing.T) {
	store := NewSampleStore()

	counters, err := store.Counters(context.Background(), inboxmanager.Caller{})
	require.NoError(t, err)
	require.Len(t, counters, 10, "Outstanding + sembilan antrean")

	byTab := map[string]inboxmanager.Counter{}
	for _, c := range counters {
		byTab[c.TabCode] = c
	}
	require.Equal(t, inboxmanager.TabOutstanding, counters[0].TabCode)
	require.Equal(t, 4, byTab[inboxmanager.TabOutstanding].Count, "jumlah baris panel pic")
	require.Empty(t, byTab[inboxmanager.TabOutstanding].Parent)
	require.Equal(t, 2, byTab[inboxmanager.TabMasterBengkel].Count)
	require.Equal(t, inboxmanager.TabApprovalMaster, byTab[inboxmanager.TabMasterBengkel].Parent)
	require.Equal(t, tabOf(t, inboxmanager.TabMasterBengkel).Name, byTab[inboxmanager.TabMasterBengkel].Label)
	require.Contains(t, byTab[inboxmanager.TabMasterSparepart].Unavailable, "SPAREPART_HE")
	require.Zero(t, byTab[inboxmanager.TabMasterSparepart].Count)

	_, hasProduktivitas := byTab[inboxmanager.TabProduktivitas]
	require.False(t, hasProduktivitas, "Produktivitas tidak punya pencacah")
}

func TestDashboardFillsPanelsAndEmptyIsNotNil(t *testing.T) {
	store := NewSampleStore()

	view, err := store.Dashboard(context.Background(),
		inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabKlaim)})
	require.NoError(t, err)
	require.Len(t, view.Panels, 2)
	require.Len(t, view.Panels[0].Rows, 2)
	require.Equal(t, "KEBAKARAN", view.Panels[1].Rows[0].Cells[inboxmanager.FieldPenyebab].Text)
	_, hasCause := view.Panels[0].Rows[0].Cells[inboxmanager.FieldPenyebab]
	require.False(t, hasCause)

	empty, err := NewStore().Dashboard(context.Background(),
		inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabOutstanding)})
	require.NoError(t, err)
	for _, panel := range empty.Panels {
		require.NotNil(t, panel.Rows)
		require.Empty(t, panel.Rows)
	}
}

func TestQueueSortedCopyAndUnavailable(t *testing.T) {
	store := NewStore()
	store.SetQueue(inboxmanager.TabMasterPanel, []inboxmanager.QueueRow{
		{Key: "B"}, {Key: "A"},
	})

	rows, err := store.Queue(context.Background(), inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabMasterPanel)})
	require.NoError(t, err)
	require.Equal(t, []inboxmanager.QueueRow{{Key: "A"}, {Key: "B"}}, rows)

	// Hasilnya salinan: mengubahnya tidak mengubah penyimpanan.
	rows[0].Key = "Z"
	again, err := store.Queue(context.Background(), inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabMasterPanel)})
	require.NoError(t, err)
	require.Equal(t, "A", again[0].Key)

	store.SetUnavailable(inboxmanager.TabMasterPanel, "view rusak")
	_, err = store.Queue(context.Background(), inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabMasterPanel)})
	require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)

	// Alasan kosong tidak menahan pembacaan.
	store.SetUnavailable(inboxmanager.TabMasterPanel, "")
	rows, err = store.Queue(context.Background(), inboxmanager.Query{Tab: tabOf(t, inboxmanager.TabMasterPanel)})
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestDecideRemovesOnlyFoundKeys(t *testing.T) {
	store := NewSampleStore()
	tab := tabOf(t, inboxmanager.TabMasterBengkel)

	changed, err := store.Decide(context.Background(), inboxmanager.Decision{
		Tab: tab, Keys: []string{"BGK-001", "TIDAK-ADA"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	rows, err := store.Queue(context.Background(), inboxmanager.Query{Tab: tab})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "BGK-002", rows[0].Key)
}

func TestLineBusinessForAndDefault(t *testing.T) {
	store := NewStrictStore()
	line, err := store.LineBusinessFor(context.Background(), "siapa")
	require.NoError(t, err)
	require.Empty(t, line, "penyimpanan ketat tanpa nilai bawaan")

	store.SetLineBusiness(" andika ", " PA ")
	line, err = store.LineBusinessFor(context.Background(), "ANDIKA")
	require.NoError(t, err)
	require.Equal(t, "PA", line)

	store.SetDefaultLineBusiness(" TRAVEL ")
	line, err = store.LineBusinessFor(context.Background(), "lain")
	require.NoError(t, err)
	require.Equal(t, "TRAVEL", line)

	sample := NewSampleStore()
	line, err = sample.LineBusinessFor(context.Background(), "siapapun")
	require.NoError(t, err)
	require.Equal(t, inboxmanager.LineNonMBU, line)
}
