package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanagerreceivepucl"
	"claim-pnc/internal/inboxmanagerreceivepucl/repo/memory"
)

func mustQuery(t *testing.T, tab string) inboxmanagerreceivepucl.Query {
	t.Helper()
	q, err := inboxmanagerreceivepucl.NewQuery(
		inboxmanagerreceivepucl.QueryInput{Tab: tab},
		inboxmanagerreceivepucl.Caller{Login: "penyelia"},
	)
	require.NoError(t, err)
	return q
}

func caseIDs(items []inboxmanagerreceivepucl.WorkItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.CaseID)
	}
	return out
}

func TestSampleStoreReceiveTabFiltersWorkClassQueueAndGroupPanel(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(), mustQuery(t, inboxmanagerreceivepucl.TabReceive),
		inboxmanagerreceivepucl.Pagination{Page: 1, Size: 50})
	require.NoError(t, err)

	// Hanya berkas ReceiveDocument dari worklist yang Group Panel-nya terisi.
	require.Equal(t, page.Total, len(page.Items))
	for _, item := range page.Items {
		require.Contains(t, item.CaseID, "RCV-")
		require.NotEmpty(t, item.ClaimType, "Jenis Klaim diturunkan dari Group Panel")
	}
	require.NotContains(t, caseIDs(page.Items), "RCV-900005")
}

func TestSampleStoreRCLPUCLTabReadsWorkbasketAndExcludesCompleted(t *testing.T) {
	store := memory.NewSampleStore()

	page, err := store.List(context.Background(), mustQuery(t, inboxmanagerreceivepucl.TabRCLPUCL),
		inboxmanagerreceivepucl.Pagination{Page: 1, Size: 50})
	require.NoError(t, err)

	ids := caseIDs(page.Items)
	require.NotContains(t, ids, "PNC-800001", "baris worklist tidak masuk antrean bersama")
	require.NotContains(t, ids, "PNC-800004", "yang selesai dikeluarkan")
	for _, item := range page.Items {
		require.Contains(t, item.CaseID, "PNC-")
		require.Empty(t, item.ClaimType, "tab RCL/PUCL tidak menurunkan Jenis Klaim")
	}
}

func TestListOrdersNewestFirstThenByCaseID(t *testing.T) {
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	row := func(id string, when time.Time) memory.Row {
		return memory.Row{
			Item:       inboxmanagerreceivepucl.WorkItem{CaseID: id, Reference: "K " + id},
			WorkClass:  inboxmanagerreceivepucl.WorkClassReceiveDocument,
			GroupPanel: "006",
			CreatedAt:  when,
		}
	}
	store := memory.NewStore(
		row("B", at),
		row("A", at),
		row("C", at.Add(time.Hour)),
		// Antrean bersama tidak boleh masuk tab Receive.
		memory.Row{
			Item:           inboxmanagerreceivepucl.WorkItem{CaseID: "X"},
			WorkClass:      inboxmanagerreceivepucl.WorkClassReceiveDocument,
			GroupPanel:     "006",
			FromWorkbasket: true,
		},
	)

	page, err := store.List(context.Background(), mustQuery(t, inboxmanagerreceivepucl.TabReceive),
		inboxmanagerreceivepucl.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Equal(t, []string{"C", "A"}, caseIDs(page.Items))

	second, err := store.List(context.Background(), mustQuery(t, inboxmanagerreceivepucl.TabReceive),
		inboxmanagerreceivepucl.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, []string{"B"}, caseIDs(second.Items))
}

func TestRCLPUCLRequiresWorkbasketOperatorCaseInsensitive(t *testing.T) {
	store := memory.NewStore(
		memory.Row{
			Item:             inboxmanagerreceivepucl.WorkItem{CaseID: "OK"},
			WorkClass:        inboxmanagerreceivepucl.WorkClassClaim,
			FromWorkbasket:   true,
			AssignedOperator: "rclpucl",
		},
		memory.Row{
			Item:             inboxmanagerreceivepucl.WorkItem{CaseID: "LAIN"},
			WorkClass:        inboxmanagerreceivepucl.WorkClassClaim,
			FromWorkbasket:   true,
			AssignedOperator: "ANTREANLAIN",
		},
	)

	page, err := store.List(context.Background(), mustQuery(t, inboxmanagerreceivepucl.TabRCLPUCL),
		inboxmanagerreceivepucl.Pagination{})
	require.NoError(t, err)
	require.Equal(t, []string{"OK"}, caseIDs(page.Items))
}

func TestDocumentFindsReceiveDocumentAndDerivesClaimType(t *testing.T) {
	store := memory.NewSampleStore()

	doc, err := store.Document(context.Background(), "  ASM-FW-GCNMFW-WORK RCV-900001  ")
	require.NoError(t, err)
	require.Equal(t, "RCV-900001", doc.CaseID)
	require.Equal(t,
		inboxmanagerreceivepucl.ClaimTypeOf(inboxmanagerreceivepucl.GroupPanelPA), doc.ClaimType)
}

func TestDocumentRejectsEmptyReference(t *testing.T) {
	_, err := memory.NewSampleStore().Document(context.Background(), "   ")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrReferenceRequired)
}

func TestDocumentIgnoresClaimRowsAndUnknownReferences(t *testing.T) {
	store := memory.NewSampleStore()

	// Kunci milik klaim tidak boleh membuka layar kerja penerimaan dokumen.
	_, err := store.Document(context.Background(), "ASM-FW-GCNMFW-WORK PNC-800001")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrDocumentNotFound)

	_, err = store.Document(context.Background(), "TIDAK-ADA")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrDocumentNotFound)
}

func TestDocumentWithoutGroupPanelKeepsStoredClaimType(t *testing.T) {
	store := memory.NewStore(memory.Row{
		Item:      inboxmanagerreceivepucl.WorkItem{Reference: "K1"},
		WorkClass: inboxmanagerreceivepucl.WorkClassReceiveDocument,
		Document:  inboxmanagerreceivepucl.ReceiveDocument{CaseID: "RCV-1", ClaimType: "tersimpan"},
	})

	doc, err := store.Document(context.Background(), "K1")
	require.NoError(t, err)
	require.Equal(t, "tersimpan", doc.ClaimType)
}
