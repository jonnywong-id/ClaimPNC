package inputacceptation_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
)

func TestErrorMessagesForLogs(t *testing.T) {
	require.Equal(t, "inputacceptation: permintaan tidak sah",
		inputacceptation.NewValidationError(nil).Error())
	require.Equal(t, "inputacceptation: pertama",
		inputacceptation.NewValidationError([]inputacceptation.Violation{
			{Message: "pertama"}, {Message: "kedua"},
		}).Error(), "hanya pelanggaran pertama yang masuk pesan log")

	require.Equal(t, "inputacceptation: isian x tidak dikenal bentuk layar",
		(&inputacceptation.UnknownFieldError{Field: "x"}).Error())
	require.Equal(t, "inputacceptation: tabel y tidak dikenal bentuk layar",
		(&inputacceptation.UnknownGridError{Grid: "y"}).Error())
}

func TestDetailAccessorsToleratesEmptyMaps(t *testing.T) {
	var empty inputacceptation.Detail
	require.Equal(t, "", empty.Get("claim_no"))
	require.Nil(t, empty.Rows(inputacceptation.GridAdjustment))

	detail, err := inputacceptation.NewDetail("CLMNP-1", "REF", "New", "ADMIN",
		map[string]string{"claim_no": "CLMNP-1"},
		map[string][]inputacceptation.GridRow{inputacceptation.GridAdjustment: {{"type": "Final"}}})
	require.NoError(t, err)
	require.Equal(t, "CLMNP-1", detail.Get("claim_no"))
	require.Equal(t, "Final", detail.Rows(inputacceptation.GridAdjustment)[0]["type"])

	_, err = inputacceptation.NewDetail("CLMNP-1", "REF", "New", "ADMIN", nil,
		map[string][]inputacceptation.GridRow{"tabel_karangan": nil})
	var unknown *inputacceptation.UnknownGridError
	require.ErrorAs(t, err, &unknown)
}
