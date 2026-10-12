package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/inputacceptation/repo/memory"
)

func TestFindIsCaseInsensitiveAndReportsMissingClaims(t *testing.T) {
	store := memory.NewSampleStore()

	detail, err := store.Find(context.Background(), inputacceptation.Query{ClaimID: " clmnp-1001 "})
	require.NoError(t, err)
	require.Equal(t, "CLMNP-1001", detail.ClaimID)
	require.NotEmpty(t, detail.Values)

	empty, err := store.Find(context.Background(), inputacceptation.Query{ClaimID: "CLMNP-1002"})
	require.NoError(t, err, "klaim tanpa dokumen tetap ditemukan")
	require.Equal(t, "CLMNP-1002", empty.ClaimID)

	_, err = store.Find(context.Background(), inputacceptation.Query{ClaimID: "CLMNP-9999"})
	require.ErrorIs(t, err, inputacceptation.ErrNotFound)
}

func TestSampleDetailsCoverBothQueues(t *testing.T) {
	ids := []string{}
	for _, detail := range memory.SampleDetails() {
		ids = append(ids, detail.ClaimID)
	}
	require.ElementsMatch(t, []string{"CLMNP-1001", "CLMNP-1002", "CLMNP-2001", "CLMNP-2002"}, ids)
}

func TestSaveIsRefusedWhileTheTableBelongsToPega(t *testing.T) {
	err := memory.NewStore().Save(context.Background(), inputacceptation.SaveCommand{})
	require.ErrorIs(t, err, inputacceptation.ErrWriteNotOwned)
}
