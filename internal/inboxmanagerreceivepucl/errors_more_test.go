package inboxmanagerreceivepucl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

func TestValidationErrorMessage(t *testing.T) {
	require.Equal(t, "inboxmanagerreceivepucl: isian tidak sah",
		inboxmanagerreceivepucl.NewValidationError(nil).Error())
	require.Equal(t, "inboxmanagerreceivepucl: tab: a; x: b",
		inboxmanagerreceivepucl.NewValidationError([]inboxmanagerreceivepucl.Violation{
			{Field: "tab", Message: "a"}, {Field: "x", Message: "b"},
		}).Error())
}

func TestPaginationNormalizeClampsRange(t *testing.T) {
	require.Equal(t, inboxmanagerreceivepucl.Pagination{Page: 1, Size: 50},
		inboxmanagerreceivepucl.Pagination{Page: -2, Size: 0}.Normalize())
	require.Equal(t, inboxmanagerreceivepucl.Pagination{Page: 3, Size: 100},
		inboxmanagerreceivepucl.Pagination{Page: 3, Size: 500}.Normalize())
}

func TestPageTotalPages(t *testing.T) {
	size := inboxmanagerreceivepucl.Pagination{Page: 1, Size: 10}
	require.Equal(t, 1, inboxmanagerreceivepucl.Page{Pagination: size}.TotalPages())
	require.Equal(t, 2, inboxmanagerreceivepucl.Page{Pagination: size, Total: 20}.TotalPages())
	require.Equal(t, 3, inboxmanagerreceivepucl.Page{Pagination: size, Total: 21}.TotalPages())
}

func TestSliceBeyondLastPageIsEmpty(t *testing.T) {
	all := []inboxmanagerreceivepucl.WorkItem{{CaseID: "a"}, {CaseID: "b"}}
	page := inboxmanagerreceivepucl.Slice(all, inboxmanagerreceivepucl.Pagination{Page: 5, Size: 1})
	require.Equal(t, 2, page.Total)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
}
