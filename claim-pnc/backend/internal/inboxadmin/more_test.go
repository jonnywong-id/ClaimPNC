package inboxadmin_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
)

func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	err := inboxadmin.NewValidationError([]inboxadmin.Violation{
		{Field: inboxadmin.FieldTab, Message: "tab tidak dikenal"},
		{Field: inboxadmin.FieldBusiness, Message: "lini bisnis tidak dikenal"},
	})

	require.Equal(t,
		"inboxadmin: tab: tab tidak dikenal; bisnis: lini bisnis tidak dikenal", err.Error())
}

func TestValidationErrorWithoutViolationsHasGenericMessage(t *testing.T) {
	require.Equal(t, "inboxadmin: isian tidak sah", inboxadmin.NewValidationError(nil).Error())
}

func TestUnknownBusinessLineLabelIsItsOwnCode(t *testing.T) {
	// Kode yang tidak dikenal ditampilkan apa adanya, bukan dikosongkan.
	require.Equal(t, "ENTAH", inboxadmin.BusinessLine("ENTAH").Label())
}

func TestSliceShortensTheLastPartialPage(t *testing.T) {
	all := []inboxadmin.WorkItem{{CaseID: "A"}, {CaseID: "B"}, {CaseID: "C"}}

	page := inboxadmin.Slice(all, inboxadmin.Pagination{Page: 2, Size: 2})

	require.Equal(t, 3, page.Total)
	require.Equal(t, 2, page.TotalPages())
	require.Len(t, page.Items, 1)
	require.Equal(t, "C", page.Items[0].CaseID)
}
