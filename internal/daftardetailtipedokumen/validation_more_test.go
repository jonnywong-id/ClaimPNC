package daftardetailtipedokumen_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetailtipedokumen"
)

// Kode tipe dokumen yang terlalu panjang menyebut nilainya di dalam pesan, supaya
// petugas tahu isian mana yang dimaksud tanpa menebak.
func TestCheckReportsOverlongDocumentTypeWithItsValue(t *testing.T) {
	long := strings.Repeat("9", daftardetailtipedokumen.MaxReferenceLength+1)
	err := daftardetailtipedokumen.Input{DocumentTypeID: long}.Check()

	var validation *daftardetailtipedokumen.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violation, 1)
	require.Equal(t, daftardetailtipedokumen.FieldDocumentType, validation.Violation[0].Field)
}

func TestValidationErrorMessage(t *testing.T) {
	require.EqualError(t, &daftardetailtipedokumen.ValidationError{}, "daftardetailtipedokumen: isian tidak sah")
	require.EqualError(t, &daftardetailtipedokumen.ValidationError{Violation: []daftardetailtipedokumen.Violation{
		{Field: "a", Message: "Satu."}, {Field: "b", Message: "Dua."},
	}}, "daftardetailtipedokumen: Satu.; Dua.")
}
