package inboxoutstanding_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

func TestFindDocumentStatusAcceptsKnownCodesCaseInsensitively(t *testing.T) {
	status, found := inboxoutstanding.FindDocumentStatus("  LENGKAP ")
	require.True(t, found)
	require.Equal(t, inboxoutstanding.StatusComplete, status)

	status, found = inboxoutstanding.FindDocumentStatus("tka")
	require.True(t, found)
	require.Equal(t, inboxoutstanding.StatusTKA, status)
}

// Kode tak dikenal DITOLAK, tidak diam-diam diartikan "semua".
func TestFindDocumentStatusRejectsUnknownCodes(t *testing.T) {
	status, found := inboxoutstanding.FindDocumentStatus("hampir-lengkap")
	require.False(t, found)
	require.Empty(t, status)
}

func TestDocumentStatusTitleFollowsTheLegacyLabel(t *testing.T) {
	require.Equal(t, "Complete documents", inboxoutstanding.StatusComplete.Title())
	require.Equal(t, "Documents not complete", inboxoutstanding.StatusIncomplete.Title())
	require.Equal(t, "ALL Case", inboxoutstanding.StatusAll.Title())

	// Status yang tidak dikenal digambar apa adanya.
	require.Equal(t, "entah", inboxoutstanding.DocumentStatus("entah").Title())
}

func TestOnlyThreeDocumentStatusesAreCountable(t *testing.T) {
	require.True(t, inboxoutstanding.StatusComplete.Countable())
	require.True(t, inboxoutstanding.StatusIncomplete.Countable())
	require.True(t, inboxoutstanding.StatusAll.Countable())

	require.False(t, inboxoutstanding.StatusLossAdjuster.Countable())
	require.False(t, inboxoutstanding.StatusTemporaryClose.Countable())
	require.False(t, inboxoutstanding.DocumentStatus("entah").Countable())
}

// Tab yang tidak dapat dihitung dikirim TANPA jumlah — bukan nol — dan urutannya mengikuti
// layar Pega.
func TestBuildSummaryKeepsTheLegacyOrderAndLeavesUncountableTabsWithoutACount(t *testing.T) {
	summary := inboxoutstanding.BuildSummary(map[inboxoutstanding.DocumentStatus]int{
		inboxoutstanding.StatusComplete:     3,
		inboxoutstanding.StatusAll:          5,
		inboxoutstanding.StatusLossAdjuster: 9, // tidak dapat dihitung: diabaikan
	}, 5)

	require.Equal(t, 5, summary.Total)
	require.Len(t, summary.Status, 9)

	codes := make([]inboxoutstanding.DocumentStatus, 0, len(summary.Status))
	for _, item := range summary.Status {
		codes = append(codes, item.Status)
	}
	require.Equal(t, []inboxoutstanding.DocumentStatus{
		inboxoutstanding.StatusComplete,
		inboxoutstanding.StatusIncomplete,
		inboxoutstanding.StatusTemporaryClose,
		inboxoutstanding.StatusDeadlineTemporaryClose,
		inboxoutstanding.StatusLossAdjuster,
		inboxoutstanding.StatusInternalSurveyor,
		inboxoutstanding.StatusAll,
		inboxoutstanding.StatusCommunication,
		inboxoutstanding.StatusTKA,
	}, codes)

	require.Equal(t, "Complete documents", summary.Status[0].Label)
	require.NotNil(t, summary.Status[0].Count)
	require.Equal(t, 3, *summary.Status[0].Count)

	// Dapat dihitung tetapi tidak diserahkan penyimpanan: tetap tanpa jumlah.
	require.Nil(t, summary.Status[1].Count)
	// Tidak dapat dihitung meski angkanya diserahkan.
	require.Nil(t, summary.Status[4].Count)
	require.Equal(t, 5, *summary.Status[6].Count)
}

func TestNormalizeLineBusinessRecognisesTheFourLines(t *testing.T) {
	require.Equal(t, inboxoutstanding.LinePA, inboxoutstanding.NormalizeLineBusiness(" pa "))
	require.Equal(t, inboxoutstanding.LineTravel, inboxoutstanding.NormalizeLineBusiness("travel"))
	require.Equal(t, inboxoutstanding.LineNonMBU, inboxoutstanding.NormalizeLineBusiness("NonMBU"))
	require.Equal(t, inboxoutstanding.LineBonding, inboxoutstanding.NormalizeLineBusiness("BONDING"))

	// Nilai asing menjadi tanpa cakupan, bukan ditolak.
	require.Equal(t, inboxoutstanding.LineUnknown, inboxoutstanding.NormalizeLineBusiness("MARINE"))
	require.Equal(t, inboxoutstanding.LineUnknown, inboxoutstanding.NormalizeLineBusiness(""))
}

func TestExportFilterNormalizeClampsTheBatchAndOffset(t *testing.T) {
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)

	clean := inboxoutstanding.ExportFilter{
		LineBusiness: "travel", From: &from, Limit: 0, Offset: -4,
	}.Normalize()
	require.Equal(t, inboxoutstanding.LineTravel, clean.LineBusiness)
	require.Equal(t, inboxoutstanding.DefaultLimit, clean.Limit)
	require.Zero(t, clean.Offset)
	require.Equal(t, &from, clean.From)

	big := inboxoutstanding.ExportFilter{Limit: 5000, Offset: 10}.Normalize()
	require.Equal(t, inboxoutstanding.MaxExportBatch, big.Limit)
	require.Equal(t, 10, big.Offset)
	require.Equal(t, inboxoutstanding.LineUnknown, big.LineBusiness)
}
