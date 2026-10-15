package inboxautoclaim_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

func TestPremiumCheckQueryCleanTrims(t *testing.T) {
	clean, err := inboxautoclaim.PremiumCheckQuery{BusinessCode: " 01 ", SourceOfBusiness: " BRI "}.Clean()
	require.NoError(t, err)
	require.Equal(t, inboxautoclaim.PremiumCheckQuery{BusinessCode: "01", SourceOfBusiness: "BRI"}, clean)
}

func TestPremiumCheckQueryCleanReportsBothFields(t *testing.T) {
	_, err := inboxautoclaim.PremiumCheckQuery{BusinessCode: " "}.Clean()
	var validation *inboxautoclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []inboxautoclaim.Violation{
		{Field: "kode_bisnis", Message: "Nama Bisnis wajib dipilih."},
		{Field: "kode_sumber_bisnis", Message: "Sumber Bisnis wajib dipilih."},
	}, validation.Violation)
}
