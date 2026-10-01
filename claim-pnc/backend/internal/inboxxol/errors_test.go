package inboxxol_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
)

func TestNewValidationErrorNilWhenNoViolation(t *testing.T) {
	require.NoError(t, inboxxol.NewValidationError(nil))
}

func TestValidationErrorListsEveryViolation(t *testing.T) {
	err := inboxxol.NewValidationError([]inboxxol.Violation{
		{Field: inboxxol.FieldYear, Message: "wajib"},
		{Field: inboxxol.FieldAdviceType, Message: "salah"},
	})
	require.EqualError(t, err, "inboxxol: validasi gagal — tahun: wajib; tipe: salah")
	var validation *inboxxol.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
}
