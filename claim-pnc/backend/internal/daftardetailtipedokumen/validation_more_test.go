package daftardetailtipedokumen_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetailtipedokumen"
)

// ID bisnis yang terlalu panjang dilaporkan sekali saja, menyebut ID-nya.
func TestCheckReportsOverlongBusinessIDOnce(t *testing.T) {
	long := strings.Repeat("9", daftardetailtipedokumen.MaxReferenceLength+1)
	err := daftardetailtipedokumen.Input{Businesses: []daftardetailtipedokumen.BusinessInput{
		{BusinessID: long}, {BusinessID: long},
	}}.Check()

	var validation *daftardetailtipedokumen.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violation, 1)
	require.Equal(t, daftardetailtipedokumen.FieldBusiness, validation.Violation[0].Field)
	require.Contains(t, validation.Violation[0].Message, long)
}

func TestValidationErrorMessage(t *testing.T) {
	require.EqualError(t, &daftardetailtipedokumen.ValidationError{}, "daftardetailtipedokumen: isian tidak sah")
	require.EqualError(t, &daftardetailtipedokumen.ValidationError{Violation: []daftardetailtipedokumen.Violation{
		{Field: "a", Message: "Satu."}, {Field: "b", Message: "Dua."},
	}}, "daftardetailtipedokumen: Satu.; Dua.")
}
