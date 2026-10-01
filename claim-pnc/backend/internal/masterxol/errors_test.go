package masterxol_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
)

func TestNewValidationErrorNilWithoutViolation(t *testing.T) {
	require.NoError(t, masterxol.NewValidationError(nil))
}

func TestValidationErrorJoinsEveryViolation(t *testing.T) {
	err := masterxol.NewValidationError([]masterxol.Violation{
		{Field: masterxol.FieldName, Message: "wajib"},
		{Field: masterxol.FieldYear, Message: "salah"},
	})
	require.EqualError(t, err, "masterxol: validasi gagal — nama: wajib; tahun: salah")
}
