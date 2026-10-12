package laporanhasilai_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/laporanhasilai"
)

func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	err := laporanhasilai.NewValidationError([]laporanhasilai.Violation{
		{Field: laporanhasilai.FieldDateFrom, Message: "kosong"},
		{Field: laporanhasilai.FieldDateTo, Message: "kosong juga"},
	})
	require.Equal(t, "laporanhasilai: dari: kosong; sampai: kosong juga", err.Error())
}

func TestValidationErrorWithoutViolationsHasGenericMessage(t *testing.T) {
	require.Equal(t, "laporanhasilai: isian tidak sah",
		laporanhasilai.NewValidationError(nil).Error())
}
