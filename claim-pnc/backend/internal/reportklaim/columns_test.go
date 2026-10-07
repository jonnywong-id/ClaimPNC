package reportklaim

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseColumnsKeepsHeaderSpacing(t *testing.T) {
	got := parseColumns("\nNo  Klaim\tCaseID\n\nTgl \tReportDate\n")
	require.Equal(t, []Column{{Header: "No  Klaim", Field: "CaseID"}, {Header: "Tgl ", Field: "ReportDate"}}, got)
}

func TestParseColumnsRejectsBrokenRows(t *testing.T) {
	require.Panics(t, func() { parseColumns("tanpa tab") })
	require.Panics(t, func() { parseColumns("judul\t") })
}
